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

package rolerequest

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
	multitenancy "github.com/EdgeNet-project/edgenet/pkg/multitenancy"

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

const controllerAgentName = "rolerequest-controller"

// Definitions of the state of the rolerequest resource
const (
	successSynced  = "Synced"
	successFound   = "Found"
	failureFound   = "Not Found"
	failureBinding = "Binding Failed"

	messageResourceSynced   = "Role Request synced successfully"
	messageRoleBound        = "Requested Role / Cluster Role is bound"
	messageRoleFound        = "Requested Role / Cluster Role found"
	messageRoleNotFound     = "Requested Role / Cluster Role does not exist"
	messageRoleApproved     = "Requested Role / Cluster Role approved successfully"
	messagePending          = "Waiting for approval"
	messageBindingFailed    = "Role binding failed"
	messageOwnershipFailure = "Role Request ownership cannot be granted"
)

// Controller is the controller implementation for Role Request resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	rolerequestsLister listers.RoleRequestLister
	rolerequestsSynced cache.InformerSynced

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
	rolerequestInformer informers.RoleRequestInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:      kubeclientset,
		edgenetclientset:   edgenetclientset,
		rolerequestsLister: rolerequestInformer.Lister(),
		rolerequestsSynced: rolerequestInformer.Informer().HasSynced,
		workqueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "RoleRequests"),
		recorder:           recorder,
	}

	klog.Infoln("Setting up event handlers")
	// Set up an event handler for when Role Request resources change
	rolerequestInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueRoleRequest,
		UpdateFunc: func(old, new interface{}) {
			newRoleRequest := new.(*registrationv1alpha1.RoleRequest)
			oldRoleRequest := old.(*registrationv1alpha1.RoleRequest)
			if (oldRoleRequest.Status.Expiry == nil && newRoleRequest.Status.Expiry != nil) ||
				(oldRoleRequest.Status.Expiry != nil && newRoleRequest.Status.Expiry != nil && !oldRoleRequest.Status.Expiry.Time.Equal(newRoleRequest.Status.Expiry.Time)) {
				controller.enqueueRoleRequestAfter(newRoleRequest, time.Until(newRoleRequest.Status.Expiry.Time))
			}
			controller.enqueueRoleRequest(new)
		},
	})

	return controller
}

// Run will set up the event handlers for the types of role request and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Infoln("Starting Role Request controller")

	klog.Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.rolerequestsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Infoln("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Infoln("Started workers")
	<-stopCh
	klog.Infoln("Shutting down workers")

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
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Role Request
// resource with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	rolerequest, err := c.rolerequestsLister.RoleRequests(namespace).Get(name)

	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("rolerequest '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	c.processRoleRequest(rolerequest.DeepCopy())
	c.recorder.Event(rolerequest, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueRoleRequest takes a RoleRequest resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than RoleRequest.
func (c *Controller) enqueueRoleRequest(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// enqueueRoleRequestAfter takes a RoleRequest resource and converts it into a namespace/name
// string which is then put onto the work queue after the expiry date to be deleted. This method should *not* be
// passed resources of any type other than RoleRequest.
func (c *Controller) enqueueRoleRequestAfter(obj interface{}, after time.Duration) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.AddAfter(key, after)
}

func (c *Controller) processRoleRequest(roleRequestCopy *registrationv1alpha1.RoleRequest) {
	if roleRequestCopy.Status.Expiry == nil {
		// Set the approval timeout which is 72 hours
		roleRequestCopy.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
	} else if time.Until(roleRequestCopy.Status.Expiry.Time) <= 0 {
		c.edgenetclientset.RegistrationV1alpha1().RoleRequests(roleRequestCopy.GetNamespace()).Delete(context.TODO(), roleRequestCopy.GetName(), metav1.DeleteOptions{})
		return
	}

	multitenancyManager := multitenancy.NewManager(c.kubeclientset, c.edgenetclientset)
	permitted, _, _ := multitenancyManager.EligibilityCheck(roleRequestCopy.GetNamespace())
	if permitted {
		// Below is to ensure that the requested Role / ClusterRole exists before moving forward in the procedure.
		// If not, the status of the object falls into an error state.
		roleExists := c.checkForRequestedRole(roleRequestCopy)
		if !roleExists {
			return
		}

		switch roleRequestCopy.Status.State {
		case registrationv1alpha1.StatusBound:
			c.recorder.Event(roleRequestCopy, corev1.EventTypeNormal, registrationv1alpha1.StatusBound, messageRoleBound)
		case registrationv1alpha1.StatusApproved:
			// The following section handles role binding. There are two basic logical steps here.
			// Check if role binding already exists; if not, create a role binding for the user.
			// If role binding exists, check if the user already holds the role. If not, pin the role to the user.

			roleRef := rbacv1.RoleRef{Kind: roleRequestCopy.Spec.RoleRef.Kind, Name: roleRequestCopy.Spec.RoleRef.Name}
			rbSubjects := []rbacv1.Subject{{Kind: "User", Name: roleRequestCopy.Spec.Email, APIGroup: "rbac.authorization.k8s.io"}}
			requestedBinding := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: roleRequestCopy.Spec.RoleRef.Name, Namespace: roleRequestCopy.GetNamespace()},
				Subjects: rbSubjects, RoleRef: roleRef}
			requestedBindingLabels := map[string]string{"edge-net.io/generated": "true"}
			requestedBinding.SetLabels(requestedBindingLabels)
			if _, err := c.kubeclientset.RbacV1().RoleBindings(requestedBinding.GetNamespace()).Create(context.TODO(), requestedBinding, metav1.CreateOptions{}); err != nil {
				if !errors.IsAlreadyExists(err) {
					c.recorder.Event(roleRequestCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
					return
				}

				if roleBinding, err := c.kubeclientset.RbacV1().RoleBindings(requestedBinding.GetNamespace()).Get(context.TODO(), requestedBinding.GetName(), metav1.GetOptions{}); err == nil {
					isBound := false
					for _, subjectRow := range roleBinding.Subjects {
						if subjectRow.Kind == "User" && subjectRow.Name == roleRequestCopy.Spec.Email {
							isBound = true
							break
						}
					}
					if !isBound {
						roleBindingCopy := roleBinding.DeepCopy()
						roleBindingCopy.Subjects = append(roleBindingCopy.Subjects, rbacv1.Subject{Kind: "User", Name: roleRequestCopy.Spec.Email, APIGroup: "rbac.authorization.k8s.io"})
						if _, err := c.kubeclientset.RbacV1().RoleBindings(roleBindingCopy.GetNamespace()).Update(context.TODO(), roleBindingCopy, metav1.UpdateOptions{}); err != nil {
							c.recorder.Event(roleBindingCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
							return
						}
					}
				} else {
					c.recorder.Event(roleRequestCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
					return
				}

			}

			roleRequestCopy.Status.State = registrationv1alpha1.StatusBound
			roleRequestCopy.Status.Message = messageRoleBound
			c.updateStatus(context.TODO(), roleRequestCopy)
		case registrationv1alpha1.StatusPending:
			if roleRequestCopy.Spec.Approved {
				c.recorder.Event(roleRequestCopy, corev1.EventTypeNormal, registrationv1alpha1.StatusApproved, messageRoleApproved)
				roleRequestCopy.Status.State = registrationv1alpha1.StatusApproved
				roleRequestCopy.Status.Message = messageRoleApproved
				c.updateStatus(context.TODO(), roleRequestCopy)
			}
		default:
			if ownershipGranted := c.grantRequestOwnership(roleRequestCopy); !ownershipGranted {
				return
			}

			roleRequestCopy.Status.State = registrationv1alpha1.StatusPending
			roleRequestCopy.Status.Message = messagePending
			c.updateStatus(context.TODO(), roleRequestCopy)
		}
	} else {
		c.edgenetclientset.RegistrationV1alpha1().RoleRequests(roleRequestCopy.GetNamespace()).Delete(context.TODO(), roleRequestCopy.GetName(), metav1.DeleteOptions{})
	}
}

func (c *Controller) grantRequestOwnership(roleRequestCopy *registrationv1alpha1.RoleRequest) bool {
	objectName := fmt.Sprintf("edgenet:%s:%s", "rolerequest", roleRequestCopy.GetName())
	policyRule := []rbacv1.PolicyRule{{APIGroups: []string{"registration.edgenet.io"}, Resources: []string{"rolerequests"}, ResourceNames: []string{roleRequestCopy.GetName()}, Verbs: []string{"get", "update", "patch", "delete"}},
		{APIGroups: []string{"registration.edgenet.io"}, Resources: []string{fmt.Sprintf("%s/status", "rolerequests")}, ResourceNames: []string{roleRequestCopy.GetName()}, Verbs: []string{"get", "list", "watch"}},
	}
	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: objectName, OwnerReferences: []metav1.OwnerReference{roleRequestCopy.MakeOwnerReference()}},
		Rules: policyRule}
	if _, err := c.kubeclientset.RbacV1().Roles(roleRequestCopy.GetNamespace()).Create(context.TODO(), role, metav1.CreateOptions{}); err == nil || errors.IsAlreadyExists(err) {
		roleRef := rbacv1.RoleRef{Kind: "Role", Name: objectName}
		rbSubjects := []rbacv1.Subject{{Kind: "User", Name: roleRequestCopy.Spec.Email, APIGroup: "rbac.authorization.k8s.io"}}
		roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: objectName},
			Subjects: rbSubjects, RoleRef: roleRef}
		roleBind.ObjectMeta.OwnerReferences = []metav1.OwnerReference{roleRequestCopy.MakeOwnerReference()}
		if _, err := c.kubeclientset.RbacV1().RoleBindings(roleRequestCopy.GetNamespace()).Create(context.TODO(), roleBind, metav1.CreateOptions{}); err == nil || errors.IsAlreadyExists(err) {
			return true
		}
		klog.Infof("Couldn't create %s  role binding: %s", objectName, err)
	} else {
		klog.Infof("Couldn't create %s role: %s", objectName, err)
	}

	if roleRequestCopy.Status.State != registrationv1alpha1.StatusFailed {
		roleRequestCopy.Status.State = registrationv1alpha1.StatusFailed
		roleRequestCopy.Status.Message = messageOwnershipFailure
		c.updateStatus(context.TODO(), roleRequestCopy)
	}

	return false
}

func (c *Controller) checkForRequestedRole(roleRequestCopy *registrationv1alpha1.RoleRequest) bool {
	if roleRequestCopy.Spec.RoleRef.Kind == "ClusterRole" {
		if clusterRoleRaw, err := c.kubeclientset.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{}); err == nil {
			for _, clusterRoleRow := range clusterRoleRaw.Items {
				if clusterRoleRow.GetName() == roleRequestCopy.Spec.RoleRef.Name {
					c.recorder.Event(roleRequestCopy, corev1.EventTypeNormal, successFound, messageRoleFound)
					return true
				}
			}
		}
	} else if roleRequestCopy.Spec.RoleRef.Kind == "Role" {
		if roleRaw, err := c.kubeclientset.RbacV1().Roles(roleRequestCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil {
			for _, roleRow := range roleRaw.Items {
				if roleRow.GetName() == roleRequestCopy.Spec.RoleRef.Name {
					c.recorder.Event(roleRequestCopy, corev1.EventTypeNormal, successFound, messageRoleFound)
					return true
				}
			}
		}
	}

	c.recorder.Event(roleRequestCopy, corev1.EventTypeWarning, failureFound, messageRoleNotFound)
	roleRequestCopy.Status.State = registrationv1alpha1.StatusFailed
	roleRequestCopy.Status.Message = messageRoleNotFound
	return false
}

// updateStatus calls the API to update the role request status.
func (c *Controller) updateStatus(ctx context.Context, roleRequestCopy *registrationv1alpha1.RoleRequest) {
	if _, err := c.edgenetclientset.RegistrationV1alpha1().RoleRequests(roleRequestCopy.GetNamespace()).UpdateStatus(ctx, roleRequestCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}
