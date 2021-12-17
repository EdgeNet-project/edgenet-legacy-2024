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

package emailverification

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/access"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/registration/v1alpha"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/registration/v1alpha"

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

const controllerAgentName = "emailverification-controller"

// Definitions of the state of the emailverification resource
const (
	successSynced         = "Synced"
	messageResourceSynced = "Email Verification synced successfully"
	create                = "create"
	update                = "update"
	delete                = "delete"
	verified              = "Verified"
)

// Controller is the controller implementation for Email Verification resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	emailverificationsLister listers.EmailVerificationLister
	emailverificationsSynced cache.InformerSynced

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
	emailverificationInformer informers.EmailVerificationInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:            kubeclientset,
		edgenetclientset:         edgenetclientset,
		emailverificationsLister: emailverificationInformer.Lister(),
		emailverificationsSynced: emailverificationInformer.Informer().HasSynced,
		workqueue:                workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "EmailVerifications"),
		recorder:                 recorder,
	}

	klog.V(4).Infoln("Setting up event handlers")
	// Set up an event handler for when Email Verification resources change
	emailverificationInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueEmailVerification,
		UpdateFunc: func(old, new interface{}) {
			newEmailVerification := new.(*registrationv1alpha.EmailVerification)
			oldEmailVerification := old.(*registrationv1alpha.EmailVerification)
			if reflect.DeepEqual(newEmailVerification.Spec, oldEmailVerification.Spec) {
				return
			}

			controller.enqueueEmailVerification(new)
		},
	})

	access.Clientset = kubeclientset
	access.EdgenetClientset = edgenetclientset

	return controller
}

// Run will set up the event handlers for the types of email verification and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.V(4).Infoln("Starting Email Verification controller")

	klog.V(4).Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.emailverificationsSynced); !ok {
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
// converge the two. It then updates the Status block of the Email Verification
// resource with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	emailverification, err := c.emailverificationsLister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("emailverification '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	if emailverification.Status.State != verified {
		c.applyProcedure(emailverification)
	}
	c.recorder.Event(emailverification, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueEmailVerification takes a EmailVerification resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than EmailVerification.
func (c *Controller) enqueueEmailVerification(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *Controller) applyProcedure(emailverificationCopy *registrationv1alpha.EmailVerification) {
	if emailverificationCopy.Spec.Verified {
		emailverificationCopy.Status.State = verified
		c.edgenetclientset.RegistrationV1alpha().EmailVerifications().UpdateStatus(context.TODO(), emailverificationCopy, metav1.UpdateOptions{})

		c.statusUpdate(emailverificationCopy.GetLabels())
	} else {
		if emailverificationCopy.Status.Expiry == nil {
			// Set the email verification timeout which is 24 hours
			emailverificationCopy.Status.Expiry = &metav1.Time{
				Time: time.Now().Add(24 * time.Hour),
			}
			c.edgenetclientset.RegistrationV1alpha().EmailVerifications().UpdateStatus(context.TODO(), emailverificationCopy, metav1.UpdateOptions{})
		}
	}
}

// statusUpdate to update the objects that are relevant the request and send email
func (c *Controller) statusUpdate(labels map[string]string) {
	// Update the status of request related to email verification
	if strings.ToLower(labels["edge-net.io/registration"]) == "tenant" {
		tenantRequest, _ := c.edgenetclientset.RegistrationV1alpha().TenantRequests().Get(context.TODO(), labels["edge-net.io/tenant"], metav1.GetOptions{})
		// TO-DO: Check dubious activity here
		// labels := tenantRequest.GetLabels()
		tenantRequest.Status.EmailVerified = true
		c.edgenetclientset.RegistrationV1alpha().TenantRequests().UpdateStatus(context.TODO(), tenantRequest, metav1.UpdateOptions{})
		// Send email to inform admins of the cluster
		//access.SendEmailVerificationNotification("tenant-email-verified-alert", labels["edge-net.io/tenant"], tenantRequest.Spec.Contact.Username,
		//	fmt.Sprintf("%s %s", tenantRequest.Spec.Contact.FirstName, tenantRequest.Spec.Contact.LastName), "", "")
	} else if strings.ToLower(labels["edge-net.io/registration"]) == "user" {
		userRequestObj, _ := c.edgenetclientset.RegistrationV1alpha().UserRequests().Get(context.TODO(), labels["edge-net.io/username"], metav1.GetOptions{})
		userRequestObj.Status.EmailVerified = true
		c.edgenetclientset.RegistrationV1alpha().UserRequests().UpdateStatus(context.TODO(), userRequestObj, metav1.UpdateOptions{})
		// Send email to inform edgenet tenant admins and authorized users
		//access.SendEmailVerificationNotification("user-email-verified-alert", labels["edge-net.io/tenant"], labels["edge-net.io/username"],
		//	fmt.Sprintf("%s %s", userRequestObj.Spec.FirstName, userRequestObj.Spec.LastName), "", "")
	} else if strings.ToLower(labels["edge-net.io/registration"]) == "email" {
		acceptableUsePolicy, _ := c.edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), labels["edge-net.io/username"], metav1.GetOptions{})
		acceptableUsePolicy.Spec.Accepted = true
		c.edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), acceptableUsePolicy, metav1.UpdateOptions{})

		// TO-DO: Get user contact information
		// Send email to inform user
		// c.sendEmail("user-email-verified-notification", labels["edge-net.io/tenant"], "", labels["edge-net.io/username"],
		// fmt.Sprintf("%s %s", userObj.Spec.FirstName, userObj.Spec.LastName), userObj.Spec.Email, "")
	}
}

// RunExpiryController puts a procedure in place to remove requests by verification or timeout
func (c *Controller) RunExpiryController() {
	var closestExpiry time.Time
	terminated := make(chan bool)
	newExpiry := make(chan time.Time)
	defer close(terminated)
	defer close(newExpiry)

	watchEmailVerifiation, err := c.edgenetclientset.RegistrationV1alpha().EmailVerifications().Watch(context.TODO(), metav1.ListOptions{})
	if err == nil {
		watchEvents := func(watchEmailVerifiation watch.Interface, newExpiry *chan time.Time) {
			// Watch the events of user request object
			// Get events from watch interface
			for emailVerificationEvent := range watchEmailVerifiation.ResultChan() {
				// Get updated user request object
				updatedEmailVerification, status := emailVerificationEvent.Object.(*registrationv1alpha.EmailVerification)
				if status {
					if updatedEmailVerification.Status.Expiry != nil {
						*newExpiry <- updatedEmailVerification.Status.Expiry.Time
					}
				}
			}
		}
		go watchEvents(watchEmailVerifiation, &newExpiry)
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
				log.Printf("ExpiryController: Closest expiry date is %v", closestExpiry)
			}
		case <-time.After(time.Until(closestExpiry)):
			emailVerificationRaw, err := c.edgenetclientset.RegistrationV1alpha().EmailVerifications().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				// TO-DO: Provide more information on error
				log.Println(err)
			}
			for _, emailVerificationRow := range emailVerificationRaw.Items {
				if emailVerificationRow.Status.Expiry != nil && emailVerificationRow.Status.Expiry.Time.Sub(time.Now()) <= 0 {
					c.edgenetclientset.RegistrationV1alpha().EmailVerifications().Delete(context.TODO(), emailVerificationRow.GetName(), metav1.DeleteOptions{})
				} else if emailVerificationRow.Status.Expiry != nil && emailVerificationRow.Status.Expiry.Time.Sub(time.Now()) > 0 {
					if closestExpiry.Sub(time.Now()) <= 0 || closestExpiry.Sub(emailVerificationRow.Status.Expiry.Time) > 0 {
						closestExpiry = emailVerificationRow.Status.Expiry.Time
						log.Printf("ExpiryController: Closest expiry date is %v after the expiration of a user request", closestExpiry)
					}
				}
			}

			if closestExpiry.Sub(time.Now()) <= 0 {
				closestExpiry = time.Now().AddDate(1, 0, 0)
				log.Printf("ExpiryController: Closest expiry date is %v after the expiration of a user request", closestExpiry)
			}
		case <-terminated:
			watchEmailVerifiation.Stop()
			break infiniteLoop
		}
	}
}
