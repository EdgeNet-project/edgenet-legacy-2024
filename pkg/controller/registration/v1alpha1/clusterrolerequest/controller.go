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
	"reflect"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/access"
	registrationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/registration/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/registration/v1alpha1"

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
	successSynced          = "Synced"
	messageResourceSynced  = "Cluster Role Request synced successfully"
	successFound           = "Found"
	messageRoleFound       = "Requested Role / Cluster Role found successfully"
	failureFound           = "Not Found"
	messageRoleNotFound    = "Requested Role / Cluster Role does not exist"
	warningApproved        = "Not Approved"
	messageRoleNotApproved = "Waiting for Requested Role / Cluster Role to be approved"
	successApproved        = "Approved"
	messageRoleApproved    = "Requested Role / Cluster Role approved successfully"
	failureBinding         = "Binding Failed"
	messageBindingFailed   = "Role binding failed"
	failure                = "Failure"
	pending                = "Pending"
	approved               = "Approved"
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
				!oldClusterRoleRequest.Status.Expiry.Time.Equal(newClusterRoleRequest.Status.Expiry.Time) {
				controller.enqueueClusterRoleRequestAfter(newClusterRoleRequest, time.Until(newClusterRoleRequest.Status.Expiry.Time))
			}
			controller.enqueueClusterRoleRequest(new)
		},
	})

	access.Clientset = kubeclientset
	access.EdgenetClientset = edgenetclientset

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

	if clusterrolerequest.Status.State != approved {
		c.processClusterRoleRequest(clusterrolerequest.DeepCopy())
	}
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
	oldStatus := clusterRoleRequestCopy.Status
	statusUpdate := func() {
		if !reflect.DeepEqual(oldStatus, clusterRoleRequestCopy.Status) {
			if _, err := c.edgenetclientset.RegistrationV1alpha1().ClusterRoleRequests().UpdateStatus(context.TODO(), clusterRoleRequestCopy, metav1.UpdateOptions{}); err != nil {
				klog.V(4).Infoln(err)
			}
		}
	}
	if clusterRoleRequestCopy.Status.Expiry == nil {
		// Set the approval timeout which is 72 hours
		clusterRoleRequestCopy.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
	} else if time.Until(clusterRoleRequestCopy.Status.Expiry.Time) <= 0 {
		c.edgenetclientset.RegistrationV1alpha1().ClusterRoleRequests().Delete(context.TODO(), clusterRoleRequestCopy.GetName(), metav1.DeleteOptions{})
		return
	}
	defer statusUpdate()

	// Below is to ensure that the requested Role / ClusterRole exists before moving forward in the procedure.
	// If not, the status of the object falls into an error state.
	roleExists := c.checkForRequestedRole(clusterRoleRequestCopy)
	if !roleExists {
		return
	}

	if !clusterRoleRequestCopy.Spec.Approved {
		if clusterRoleRequestCopy.Status.State == pending && clusterRoleRequestCopy.Status.Message == messageRoleNotApproved {
			return
		}
		c.recorder.Event(clusterRoleRequestCopy, corev1.EventTypeWarning, warningApproved, messageRoleNotApproved)
		clusterRoleRequestCopy.Status.State = pending
		clusterRoleRequestCopy.Status.Message = messageRoleNotApproved
	} else {
		c.recorder.Event(clusterRoleRequestCopy, corev1.EventTypeNormal, successApproved, messageRoleApproved)
		clusterRoleRequestCopy.Status.State = approved
		clusterRoleRequestCopy.Status.Message = messageRoleApproved

		// The following section handles cluster role binding. There are two basic logical steps here.
		// Check if cluster role binding already exists; if not, create a cluster role binding for the user.
		// If cluster role binding exists, check if the user already holds the role. If not, pin the cluster role to the user.
		if clusterRoleBindingRaw, err := c.kubeclientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"}); err == nil {
			// TODO: Simplfy below
			clusterRoleBindingExists := false
			clusterRoleBound := false
			for _, clusterRoleBindingRow := range clusterRoleBindingRaw.Items {
				if clusterRoleBindingRow.GetName() == clusterRoleRequestCopy.Spec.RoleName && clusterRoleBindingRow.RoleRef.Name == clusterRoleRequestCopy.Spec.RoleName {
					clusterRoleBindingExists = true
					for _, subjectRow := range clusterRoleBindingRow.Subjects {
						if subjectRow.Kind == "User" && subjectRow.Name == clusterRoleRequestCopy.Spec.Email {
							break
						}
					}
					if !clusterRoleBound {
						clusterRoleBindingCopy := clusterRoleBindingRow.DeepCopy()
						clusterRoleBindingCopy.Subjects = append(clusterRoleBindingCopy.Subjects, rbacv1.Subject{Kind: "User", Name: clusterRoleRequestCopy.Spec.Email, APIGroup: "rbac.authorization.k8s.io"})
						if _, err := c.kubeclientset.RbacV1().ClusterRoleBindings().Update(context.TODO(), clusterRoleBindingCopy, metav1.UpdateOptions{}); err != nil {
							c.recorder.Event(clusterRoleBindingCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
							clusterRoleRequestCopy.Status.State = failure
							clusterRoleRequestCopy.Status.Message = messageBindingFailed
							klog.V(4).Infoln(err)
						}
						break
					}
				}
			}
			if !clusterRoleBindingExists {
				roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: clusterRoleRequestCopy.Spec.RoleName}
				rbSubjects := []rbacv1.Subject{{Kind: "User", Name: clusterRoleRequestCopy.Spec.Email, APIGroup: "rbac.authorization.k8s.io"}}
				clusterRoleBind := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: clusterRoleRequestCopy.Spec.RoleName},
					Subjects: rbSubjects, RoleRef: roleRef}
				clusterRoleBindLabels := map[string]string{"edge-net.io/generated": "true"}
				clusterRoleBind.SetLabels(clusterRoleBindLabels)
				if _, err := c.kubeclientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), clusterRoleBind, metav1.CreateOptions{}); err != nil {
					c.recorder.Event(clusterRoleRequestCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
					clusterRoleRequestCopy.Status.State = failure
					clusterRoleRequestCopy.Status.Message = messageBindingFailed
					klog.V(4).Infoln(err)
				}
			}
		}
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
	clusterRoleRequestCopy.Status.State = failure
	clusterRoleRequestCopy.Status.Message = messageRoleNotFound
	return false
}
