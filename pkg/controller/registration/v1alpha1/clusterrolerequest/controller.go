/*
Copyright 2021 Contributors to the EdgeNet project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clusterrolerequest

import (
	"context"
	"fmt"
	"time"

	registrationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/registration/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/registration/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/multitenancy"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "clusterrolerequest-controller"

// Definitions of the state of the clusterrolerequest resource
const (
	successSynced  = "Synced"
	successFound   = "Found"
	failureFound   = "Not Found"
	failureBinding = "Binding Failed"

	messageResourceSynced   = "Cluster Role Request synced successfully"
	messageRoleBound        = "Requested Cluster Role is bound"
	messageRoleApproved     = "Requested Cluster Role approved"
	messageRoleFound        = "Requested Cluster Role found"
	messageRoleNotFound     = "Requested Cluster Role does not exist"
	messagePending          = "Waiting for approval"
	messageBindingFailed    = "Role binding failed"
	messageOwnershipFailure = "Cluster Role Request ownership cannot be granted"
)

// Controller is the controller implementation for Cluster Role Request resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	clusterrolerequestsLister listers.ClusterRoleRequestLister
	clusterrolerequestsSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new controller
func NewController(
	kubeclientset kubernetes.Interface,
	edgenetclientset clientset.Interface,
	clusterrolerequestInformer informers.ClusterRoleRequestInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:             kubeclientset,
		edgenetclientset:          edgenetclientset,
		clusterrolerequestsLister: clusterrolerequestInformer.Lister(),
		clusterrolerequestsSynced: clusterrolerequestInformer.Informer().HasSynced,
		workqueue:                 workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ClusterRoleRequests"),
		recorder:                  recorder,
	}

	klog.V(4).Infoln("Setting up event handlers")
	// Set up an event handler for when Cluster Role Request resources change
	clusterrolerequestInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueClusterRoleRequest,
		UpdateFunc: func(old, new interface{}) {
			newClusterRoleRequest := new.(*registrationv1alpha1.ClusterRoleRequest)
			oldClusterRoleRequest := old.(*registrationv1alpha1.ClusterRoleRequest)
			if (oldClusterRoleRequest.Status.Expiry == nil && newClusterRoleRequest.Status.Expiry != nil) ||
				(oldClusterRoleRequest.Status.Expiry != nil && newClusterRoleRequest.Status.Expiry != nil && !oldClusterRoleRequest.Status.Expiry.Time.Equal(newClusterRoleRequest.Status.Expiry.Time)) {
				controller.enqueueClusterRoleRequestAfter(newClusterRoleRequest, time.Until(newClusterRoleRequest.Status.Expiry.Time))
			}
			controller.enqueueClusterRoleRequest(new)
		},
	})

	return controller
}

// Run will set up the event handlers for the types of cluster role request and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.V(4).Infoln("Starting Cluster Role Request controller")

	klog.V(4).Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.clusterrolerequestsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.V(4).Infoln("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.V(4).Infoln("Started workers")
	<-stopCh
	klog.V(4).Infoln("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		c.workqueue.Forget(obj)
		klog.V(4).Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Cluster Role Request
// resource with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	clusterrolerequest, err := c.clusterrolerequestsLister.Get(name)

	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("clusterrolerequest '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	c.processClusterRoleRequest(clusterrolerequest.DeepCopy())
	c.recorder.Event(clusterrolerequest, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueClusterRoleRequest takes a ClusterRoleRequest resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than ClusterRoleRequest.
func (c *Controller) enqueueClusterRoleRequest(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// enqueueClusterRoleRequestAfter takes a ClusterRoleRequest resource and converts it into a namespace/name
// string which is then put onto the work queue after the expiry date to be deleted. This method should *not* be
// passed resources of any type other than ClusterRoleRequest.
func (c *Controller) enqueueClusterRoleRequestAfter(obj interface{}, after time.Duration) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.AddAfter(key, after)
}

func (c *Controller) processClusterRoleRequest(clusterRoleRequestCopy *registrationv1alpha1.ClusterRoleRequest) {
	if clusterRoleRequestCopy.Status.Expiry == nil {
		// Set the approval timeout which is 72 hours
		clusterRoleRequestCopy.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
	} else if time.Until(clusterRoleRequestCopy.Status.Expiry.Time) <= 0 {
		c.edgenetclientset.RegistrationV1alpha1().ClusterRoleRequests().Delete(context.TODO(), clusterRoleRequestCopy.GetName(), metav1.DeleteOptions{})
		return
	}

	// Below is to ensure that the requested ClusterRole exists before moving forward in the procedure.
	// If not, the status of the object falls into an error state.
	if roleExists := c.checkForRequestedRole(clusterRoleRequestCopy); !roleExists {
		return
	}

	switch clusterRoleRequestCopy.Status.State {
	case registrationv1alpha1.StatusBound:
		c.recorder.Event(clusterRoleRequestCopy, corev1.EventTypeNormal, registrationv1alpha1.StatusBound, messageRoleBound)
	case registrationv1alpha1.StatusApproved:
		// The following section handles cluster role binding. There are two basic logical steps here.
		// Try to create a cluster role binding for the user.
		// If cluster role binding exists, check if the user already holds the role. If not, pin the cluster role to the user.
		roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: clusterRoleRequestCopy.Spec.RoleName}
		rbSubjects := []rbacv1.Subject{{Kind: "User", Name: clusterRoleRequestCopy.Spec.Email, APIGroup: "rbac.authorization.k8s.io"}}
		requestedBinding := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: clusterRoleRequestCopy.Spec.RoleName},
			Subjects: rbSubjects, RoleRef: roleRef}
		requestedBindingLabels := map[string]string{"edge-net.io/generated": "true"}
		requestedBinding.SetLabels(requestedBindingLabels)
		if _, err := c.kubeclientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), requestedBinding, metav1.CreateOptions{}); err != nil {
			if !errors.IsAlreadyExists(err) {
				c.recorder.Event(clusterRoleRequestCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
				return
			}

			if clusterRoleBinding, err := c.kubeclientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), requestedBinding.GetName(), metav1.GetOptions{}); err != nil {
				c.recorder.Event(clusterRoleRequestCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
				return
			} else {
				isBound := false
				for _, subjectRow := range clusterRoleBinding.Subjects {
					if subjectRow.Kind == "User" && subjectRow.Name == clusterRoleRequestCopy.Spec.Email {
						isBound = true
						break
					}
				}
				if !isBound {
					clusterRoleBindingCopy := clusterRoleBinding.DeepCopy()
					clusterRoleBindingCopy.Subjects = append(clusterRoleBindingCopy.Subjects, rbacv1.Subject{Kind: "User", Name: clusterRoleRequestCopy.Spec.Email, APIGroup: "rbac.authorization.k8s.io"})
					if _, err := c.kubeclientset.RbacV1().ClusterRoleBindings().Update(context.TODO(), clusterRoleBindingCopy, metav1.UpdateOptions{}); err != nil {
						c.recorder.Event(clusterRoleBindingCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
						return
					}
				}
			}
		}

		clusterRoleRequestCopy.Status.State = registrationv1alpha1.StatusBound
		clusterRoleRequestCopy.Status.Message = messageRoleBound
		c.updateStatus(context.TODO(), clusterRoleRequestCopy)
	case registrationv1alpha1.StatusPending:
		if clusterRoleRequestCopy.Spec.Approved {
			c.recorder.Event(clusterRoleRequestCopy, corev1.EventTypeNormal, registrationv1alpha1.StatusApproved, messageRoleApproved)
			clusterRoleRequestCopy.Status.State = registrationv1alpha1.StatusApproved
			clusterRoleRequestCopy.Status.Message = messageRoleApproved
			c.updateStatus(context.TODO(), clusterRoleRequestCopy)
		}
	default:
		multitenancyManager := multitenancy.NewManager(c.kubeclientset, c.edgenetclientset)
		if err := multitenancyManager.GrantObjectOwnership("registration.edgenet.io", "clusterrolerequests", clusterRoleRequestCopy.GetName(), clusterRoleRequestCopy.Spec.Email, []metav1.OwnerReference{clusterRoleRequestCopy.MakeOwnerReference()}); err != nil {
			clusterRoleRequestCopy.Status.State = registrationv1alpha1.StatusFailed
			clusterRoleRequestCopy.Status.Message = messageOwnershipFailure
			c.updateStatus(context.TODO(), clusterRoleRequestCopy)
			return
		}

		clusterRoleRequestCopy.Status.State = registrationv1alpha1.StatusPending
		clusterRoleRequestCopy.Status.Message = messagePending
		c.updateStatus(context.TODO(), clusterRoleRequestCopy)
	}
}

func (c *Controller) checkForRequestedRole(clusterRoleRequestCopy *registrationv1alpha1.ClusterRoleRequest) bool {
	if clusterRoleRaw, err := c.kubeclientset.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, clusterRoleRow := range clusterRoleRaw.Items {
			if clusterRoleRow.GetName() == clusterRoleRequestCopy.Spec.RoleName {
				c.recorder.Event(clusterRoleRequestCopy, corev1.EventTypeNormal, successFound, messageRoleFound)
				return true
			}
		}
	}

	c.recorder.Event(clusterRoleRequestCopy, corev1.EventTypeWarning, failureFound, messageRoleNotFound)

	if clusterRoleRequestCopy.Status.State != registrationv1alpha1.StatusFailed {
		clusterRoleRequestCopy.Status.State = registrationv1alpha1.StatusFailed
		clusterRoleRequestCopy.Status.Message = messageRoleNotFound
		c.updateStatus(context.TODO(), clusterRoleRequestCopy)
	}

	return false
}

// updateStatus calls the API to update the cluster role request status.
func (c *Controller) updateStatus(ctx context.Context, clusterRoleRequestCopy *registrationv1alpha1.ClusterRoleRequest) {
	if _, err := c.edgenetclientset.RegistrationV1alpha1().ClusterRoleRequests().UpdateStatus(ctx, clusterRoleRequestCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}
