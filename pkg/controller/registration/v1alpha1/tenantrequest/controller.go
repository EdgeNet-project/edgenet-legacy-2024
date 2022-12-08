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

package tenantrequest

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

const controllerAgentName = "tenantrequest-controller"

// Definitions of the state of the tenantrequest resource
const (
	successSynced         = "Synced"
	successApproved       = "Approved"
	failureTenantCreation = "Creation Failed"
	failureTenantExists   = "Conflicting"

	messageResourceSynced   = "Tenant Request synced successfully"
	messageApproved         = "Tenant request approved successfully"
	messageCreationFailed   = "Tenant creation failed"
	messageExists           = "Tenant already exists"
	messageCreated          = "Tenant created successfully"
	messagePending          = "Waiting for approval"
	messageOwnershipFailure = "Cluster Role Request ownership cannot be granted"
)

// Controller is the controller implementation for Tenant Request resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	tenantrequestsLister listers.TenantRequestLister
	tenantrequestsSynced cache.InformerSynced

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
	tenantrequestInformer informers.TenantRequestInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:        kubeclientset,
		edgenetclientset:     edgenetclientset,
		tenantrequestsLister: tenantrequestInformer.Lister(),
		tenantrequestsSynced: tenantrequestInformer.Informer().HasSynced,
		workqueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "TenantRequests"),
		recorder:             recorder,
	}

	klog.V(4).Infoln("Setting up event handlers")
	// Set up an event handler for when Tenant Request resources change
	tenantrequestInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueTenantRequest,
		UpdateFunc: func(old, new interface{}) {
			newTenantRequest := new.(*registrationv1alpha1.TenantRequest)
			oldTenantRequest := old.(*registrationv1alpha1.TenantRequest)
			if (oldTenantRequest.Status.Expiry == nil && newTenantRequest.Status.Expiry != nil) ||
				(oldTenantRequest.Status.Expiry != nil && newTenantRequest.Status.Expiry != nil && !oldTenantRequest.Status.Expiry.Time.Equal(newTenantRequest.Status.Expiry.Time)) {
				controller.enqueueTenantRequestAfter(newTenantRequest, time.Until(newTenantRequest.Status.Expiry.Time))
				return
			}
			controller.enqueueTenantRequest(new)
		},
	})

	return controller
}

// Run will set up the event handlers for the types of tenant request and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.V(4).Infoln("Starting Tenant Request controller")

	klog.V(4).Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.tenantrequestsSynced); !ok {
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
// converge the two. It then updates the Status block of the Tenant Request
// resource with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	tenantrequest, err := c.tenantrequestsLister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("tenantrequest '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	c.processTenantRequest(tenantrequest.DeepCopy())
	c.recorder.Event(tenantrequest, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueTenantRequest takes a TenantRequest resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than TenantRequest.
func (c *Controller) enqueueTenantRequest(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// enqueueTenantRequestAfter takes a TenantRequest resource and converts it into a namespace/name
// string which is then put onto the work queue after the expiry date to be deleted. This method should *not* be
// passed resources of any type other than TenantRequest.
func (c *Controller) enqueueTenantRequestAfter(obj interface{}, after time.Duration) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.AddAfter(key, after)
}

func (c *Controller) processTenantRequest(tenantRequestCopy *registrationv1alpha1.TenantRequest) {
	if tenantRequestCopy.Status.Expiry == nil {
		// Set the approval timeout which is 72 hours
		tenantRequestCopy.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
	} else if time.Until(tenantRequestCopy.Status.Expiry.Time) <= 0 {
		c.edgenetclientset.RegistrationV1alpha1().TenantRequests().Delete(context.TODO(), tenantRequestCopy.GetName(), metav1.DeleteOptions{})
		return
	}
	if tenant, err := c.edgenetclientset.CoreV1alpha1().Tenants().Get(context.TODO(), tenantRequestCopy.GetName(), metav1.GetOptions{}); err == nil {
		labels := tenant.GetLabels()
		if labels["edge-net.io/request-uid"] != string(tenantRequestCopy.GetUID()) && tenantRequestCopy.Status.State != registrationv1alpha1.StatusFailed {
			c.recorder.Event(tenantRequestCopy, corev1.EventTypeWarning, failureTenantExists, messageExists)
			tenantRequestCopy.Status.State = registrationv1alpha1.StatusFailed
			tenantRequestCopy.Status.Message = messageExists
			c.updateStatus(context.TODO(), tenantRequestCopy)
			return
		}
	}

	switch tenantRequestCopy.Status.State {
	case registrationv1alpha1.StatusCreated:
		c.recorder.Event(tenantRequestCopy, corev1.EventTypeNormal, registrationv1alpha1.StatusCreated, messageCreated)
	case registrationv1alpha1.StatusApproved:
		multitenancyManager := multitenancy.NewManager(c.kubeclientset, c.edgenetclientset)
		if err := multitenancyManager.CreateTenant(tenantRequestCopy); err == nil || errors.IsAlreadyExists(err) {
			c.recorder.Event(tenantRequestCopy, corev1.EventTypeNormal, registrationv1alpha1.StatusCreated, messageCreated)
		} else {
			klog.Infoln(err)
			c.recorder.Event(tenantRequestCopy, corev1.EventTypeWarning, failureTenantCreation, messageCreationFailed)
			return
		}

		tenantRequestCopy.Status.State = registrationv1alpha1.StatusCreated
		tenantRequestCopy.Status.Message = messageCreated
		c.updateStatus(context.TODO(), tenantRequestCopy)
	case registrationv1alpha1.StatusPending:
		if tenantRequestCopy.Spec.Approved {
			c.recorder.Event(tenantRequestCopy, corev1.EventTypeNormal, registrationv1alpha1.StatusApproved, messageApproved)
			tenantRequestCopy.Status.State = registrationv1alpha1.StatusApproved
			tenantRequestCopy.Status.Message = messageApproved
			c.updateStatus(context.TODO(), tenantRequestCopy)
		}
	default:
		multitenancyManager := multitenancy.NewManager(c.kubeclientset, c.edgenetclientset)
		if err := multitenancyManager.GrantObjectOwnership("registration.edgenet.io", "tenantrequests", tenantRequestCopy.GetName(), tenantRequestCopy.Spec.Contact.Email, []metav1.OwnerReference{tenantRequestCopy.MakeOwnerReference()}); err != nil {
			tenantRequestCopy.Status.State = registrationv1alpha1.StatusFailed
			tenantRequestCopy.Status.Message = messageOwnershipFailure
			c.updateStatus(context.TODO(), tenantRequestCopy)
			return
		}

		tenantRequestCopy.Status.State = registrationv1alpha1.StatusPending
		tenantRequestCopy.Status.Message = messagePending
		c.updateStatus(context.TODO(), tenantRequestCopy)
	}
}

// updateStatus calls the API to update the cluster role request status.
func (c *Controller) updateStatus(ctx context.Context, tenantRequestCopy *registrationv1alpha1.TenantRequest) {
	if _, err := c.edgenetclientset.RegistrationV1alpha1().TenantRequests().UpdateStatus(ctx, tenantRequestCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}
