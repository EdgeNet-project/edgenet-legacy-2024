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
	"reflect"
	"time"

	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	tenantv1alpha "github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha/tenant"
	"github.com/EdgeNet-project/edgenet/pkg/controller/registration/v1alpha/emailverification"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/registration/v1alpha"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/registration/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"
	"github.com/EdgeNet-project/edgenet/pkg/permission"
	"github.com/EdgeNet-project/edgenet/pkg/registration"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
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
	messageResourceSynced = "Tenant Request synced successfully"
	create                = "create"
	update                = "update"
	delete                = "delete"
	failure               = "Failure"
	issue                 = "Malfunction"
	success               = "Successful"
	approved              = "Approved"
	established           = "Established"
)

// Dictionary of status messages
var statusDict = map[string]string{
	"tenant-approved": "Tenant request has been approved",
	"tenant-failed":   "Tenant successfully failed",
	"tenant-taken":    "Tenant name, %s, is already taken",
	"email-ok":        "Verification email sent",
	"email-fail":      "Couldn't send verification email",
	"email-exist":     "Email address, %s, already exists for another user account",
	"email-used-reg":  "Email address, %s, has already been used in a user registration request",
	"email-used-auth": "Email address, %s, has already been used in another tenant request",
}

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
			newTenantRequest := new.(*registrationv1alpha.TenantRequest)
			oldTenantRequest := old.(*registrationv1alpha.TenantRequest)
			if reflect.DeepEqual(newTenantRequest.Spec, oldTenantRequest.Spec) {
				return
			}

			controller.enqueueTenantRequest(new)
		},
	})

	permission.Clientset = kubeclientset
	permission.EdgenetClientset = edgenetclientset
	registration.Clientset = kubeclientset
	registration.EdgenetClientset = edgenetclientset

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
	go c.RunExpiryController()

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

	if tenantrequest.Status.State != approved {
		c.applyProcedure(tenantrequest)
	}
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

func (c *Controller) applyProcedure(tenantRequestCopy *registrationv1alpha.TenantRequest) {
	oldStatus := tenantRequestCopy.Status
	statusUpdate := func() {
		if !reflect.DeepEqual(oldStatus, tenantRequestCopy.Status) {
			c.edgenetclientset.RegistrationV1alpha().TenantRequests().UpdateStatus(context.TODO(), tenantRequestCopy, metav1.UpdateOptions{})
		}
	}
	defer statusUpdate()
	// Flush the status
	tenantRequestCopy.Status = registrationv1alpha.TenantRequestStatus{}
	tenantRequestCopy.Status.Expiry = oldStatus.Expiry

	if tenantRequestCopy.Spec.Approved {
		created := registration.CreateTenant(tenantRequestCopy)
		if created {
			tenantRequestCopy.Status.State = approved
			tenantRequestCopy.Status.Message = []string{statusDict["tenant-approved"]}
			go func() {
				timeout := time.After(60 * time.Second)
				ticker := time.Tick(1 * time.Second)
			check:
				for {
					select {
					case <-timeout:
						break check
					case <-ticker:
						if tenant, err := c.edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantRequestCopy.GetName(), metav1.GetOptions{}); err == nil && tenant.Status.State == established {
							user := registrationv1alpha.UserRequest{}
							user.SetName(tenantRequestCopy.Spec.Contact.Username)
							user.Spec.Tenant = tenantRequestCopy.GetName()
							user.Spec.Email = tenantRequestCopy.Spec.Contact.Email
							user.Spec.FirstName = tenantRequestCopy.Spec.Contact.FirstName
							user.Spec.LastName = tenantRequestCopy.Spec.Contact.LastName
							user.Spec.Role = "Owner"
							user.SetLabels(map[string]string{"edge-net.io/user-template-hash": util.GenerateRandomString(6)})
							permission.ConfigureTenantPermissions(tenant, user.DeepCopy(), tenantv1alpha.SetAsOwnerReference(tenant))
							break check
						}
					}
				}
			}()
		} else {
			c.sendEmail("tenant-creation-failure", tenantRequestCopy)
			tenantRequestCopy.Status.State = failure
			tenantRequestCopy.Status.Message = []string{statusDict["tenant-failed"]}
		}
	} else {
		if tenantRequestCopy.Status.Expiry == nil {
			// Set the approval timeout which is 72 hours
			tenantRequestCopy.Status.Expiry = &metav1.Time{
				Time: time.Now().Add(72 * time.Hour),
			}
		}
		exists, _ := util.Contains(tenantRequestCopy.Status.Message, statusDict["email-ok"])
		if !exists {
			emailVerificationHandler := emailverification.Handler{}
			emailVerificationHandler.Init(c.kubeclientset, c.edgenetclientset)
			created := emailVerificationHandler.Create(tenantRequestCopy, SetAsOwnerReference(tenantRequestCopy))
			if created {
				// Update the status as successful
				tenantRequestCopy.Status.State = success
				tenantRequestCopy.Status.Message = []string{statusDict["email-ok"]}
			} else {
				// TODO: Define error message more precisely
				tenantRequestCopy.Status.State = issue
				tenantRequestCopy.Status.Message = []string{statusDict["email-fail"]}
			}
		}
	}
}

// sendEmail to send notification to participants
func (c *Controller) sendEmail(subject string, tenantRequest *registrationv1alpha.TenantRequest) {
	// Set the HTML template variables
	var contentData = mailer.CommonContentData{}
	contentData.CommonData.Tenant = tenantRequest.GetName()
	contentData.CommonData.Username = tenantRequest.Spec.Contact.Username
	contentData.CommonData.Name = fmt.Sprintf("%s %s", tenantRequest.Spec.Contact.FirstName, tenantRequest.Spec.Contact.LastName)
	contentData.CommonData.Email = []string{tenantRequest.Spec.Contact.Email}
	mailer.Send(subject, contentData)
}

// RunExpiryController puts a procedure in place to turn accepted policies into not accepted
func (c *Controller) RunExpiryController() {
	var closestExpiry time.Time
	terminated := make(chan bool)
	newExpiry := make(chan time.Time)
	defer close(terminated)
	defer close(newExpiry)

	watchTenantRequest, err := c.edgenetclientset.RegistrationV1alpha().TenantRequests().Watch(context.TODO(), metav1.ListOptions{})
	if err == nil {
		watchEvents := func(watchTenantRequest watch.Interface, newExpiry *chan time.Time) {
			// Watch the events of tenant request object
			// Get events from watch interface
			for tenantRequestEvent := range watchTenantRequest.ResultChan() {
				// Get updated tenant request object
				updatedTenantRequest, status := tenantRequestEvent.Object.(*registrationv1alpha.TenantRequest)
				if status {
					if updatedTenantRequest.Status.Expiry != nil {
						*newExpiry <- updatedTenantRequest.Status.Expiry.Time
					}
				}
			}
		}
		go watchEvents(watchTenantRequest, &newExpiry)
	} else {
		go c.RunExpiryController()
		terminated <- true
	}

infiniteLoop:
	for {
		// Wait on multiple channel operations
		select {
		case timeout := <-newExpiry:
			if closestExpiry.Sub(timeout) > 0 {
				closestExpiry = timeout
				klog.V(4).Infof("ExpiryController: Closest expiry date is %v", closestExpiry)
			}
		case <-time.After(time.Until(closestExpiry)):
			tenantRequestRaw, err := c.edgenetclientset.RegistrationV1alpha().TenantRequests().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				// TODO: Provide more information on error
				klog.V(4).Infoln(err)
			}
			for _, tenantRequestRow := range tenantRequestRaw.Items {
				if tenantRequestRow.Status.Expiry != nil && tenantRequestRow.Status.Expiry.Time.Sub(time.Now()) <= 0 {
					c.edgenetclientset.RegistrationV1alpha().TenantRequests().Delete(context.TODO(), tenantRequestRow.GetName(), metav1.DeleteOptions{})
				} else if tenantRequestRow.Status.Expiry != nil && tenantRequestRow.Status.Expiry.Time.Sub(time.Now()) > 0 {
					if closestExpiry.Sub(time.Now()) <= 0 || closestExpiry.Sub(tenantRequestRow.Status.Expiry.Time) > 0 {
						closestExpiry = tenantRequestRow.Status.Expiry.Time
						klog.V(4).Infof("ExpiryController: Closest expiry date is %v after the expiration of a tenant request", closestExpiry)
					}
				}
			}

			if closestExpiry.Sub(time.Now()) <= 0 {
				closestExpiry = time.Now().AddDate(1, 0, 0)
				klog.V(4).Infof("ExpiryController: Closest expiry date is %v after the expiration of a tenant request", closestExpiry)
			}
		case <-terminated:
			watchTenantRequest.Stop()
			break infiniteLoop
		}
	}
}

// SetAsOwnerReference put the tenantrequest as owner
func SetAsOwnerReference(tenantRequest *registrationv1alpha.TenantRequest) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newNamespaceRef := *metav1.NewControllerRef(tenantRequest, registrationv1alpha.SchemeGroupVersion.WithKind("TenantRequest"))
	takeControl := false
	newNamespaceRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newNamespaceRef)
	return ownerReferences
}
