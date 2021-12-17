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

package userrequest

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/access"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	tenantv1alpha "github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha/tenant"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/registration/v1alpha"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/registration/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	log "github.com/sirupsen/logrus"
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

const controllerAgentName = "userrequest-controller"

// Definitions of the state of the userrequest resource
const (
	successSynced         = "Synced"
	messageResourceSynced = "User Request synced successfully"
	failure               = "Failure"
	issue                 = "Malfunction"
	success               = "Successful"
	approved              = "Approved"
)

// Dictionary of status messages
var statusDict = map[string]string{
	"user-approved":     "User request has been approved",
	"user-failed":       "User creation failed",
	"email-ok":          "Verification email sent",
	"email-fail":        "Couldn't send verification email",
	"email-exist":       "Email address, %s, already exists for another user account",
	"email-existregist": "Email address, %s, already exists for another user registration request",
	"email-existauth":   "Email address, %s, already exists for another tenant request",
	"username-exist":    "Username, %s, already exists for another user account",
	"role-failed":       "Cluster role generation failed",
}

// Controller is the controller implementation for User Request resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	userrequestsLister listers.UserRequestLister
	userrequestsSynced cache.InformerSynced

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
	userrequestInformer informers.UserRequestInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:      kubeclientset,
		edgenetclientset:   edgenetclientset,
		userrequestsLister: userrequestInformer.Lister(),
		userrequestsSynced: userrequestInformer.Informer().HasSynced,
		workqueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "UserRequests"),
		recorder:           recorder,
	}

	klog.V(4).Infoln("Setting up event handlers")
	// Set up an event handler for when User Request resources change
	userrequestInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueUserRequest,
		UpdateFunc: func(old, new interface{}) {
			newUserRequest := new.(*registrationv1alpha.UserRequest)
			oldUserRequest := old.(*registrationv1alpha.UserRequest)
			if reflect.DeepEqual(newUserRequest.Spec, oldUserRequest.Spec) {
				return
			}

			controller.enqueueUserRequest(new)
		},
	})

	access.Clientset = kubeclientset
	access.EdgenetClientset = edgenetclientset

	return controller
}

// Run will set up the event handlers for the types of user request and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.V(4).Infoln("Starting User Request controller")

	klog.V(4).Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.userrequestsSynced); !ok {
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
// converge the two. It then updates the Status block of the User Request
// resource with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	userrequest, err := c.userrequestsLister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("userrequest '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	if userrequest.Status.State != approved {
		c.applyProcedure(userrequest)
	}
	c.recorder.Event(userrequest, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueUserRequest takes a UserRequest resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than UserRequest.
func (c *Controller) enqueueUserRequest(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *Controller) applyProcedure(userRequestCopy *registrationv1alpha.UserRequest) {
	oldStatus := userRequestCopy.Status
	statusUpdate := func() {
		if !reflect.DeepEqual(oldStatus, userRequestCopy.Status) {
			c.edgenetclientset.RegistrationV1alpha().UserRequests().UpdateStatus(context.TODO(), userRequestCopy, metav1.UpdateOptions{})
		}
	}
	defer statusUpdate()
	// Flush the status
	userRequestCopy.Status = registrationv1alpha.UserRequestStatus{}
	userRequestCopy.Status.Expiry = oldStatus.Expiry

	tenant, _ := c.edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), strings.ToLower(userRequestCopy.Spec.Tenant), metav1.GetOptions{})
	// Check if the tenant is active
	if tenant.Spec.Enabled {
		if userRequestCopy.Spec.Approved {
			userRequestCopy.SetLabels(map[string]string{"edge-net.io/user-template-hash": util.GenerateRandomString(6)})
			access.ConfigureTenantPermissions(tenant, userRequestCopy, tenantv1alpha.SetAsOwnerReference(tenant))

			if aupFailure, _ := util.Contains(tenant.Status.Message, fmt.Sprintf(statusDict["aup-rolebinding-failure"], userRequestCopy.Spec.Email)); !aupFailure {
				if certFailure, _ := util.Contains(tenant.Status.Message, fmt.Sprintf(statusDict["cert-failure"], userRequestCopy.Spec.Email)); !certFailure {
					if kubeconfigFailure, _ := util.Contains(tenant.Status.Message, fmt.Sprintf(statusDict["kubeconfig-failure"], userRequestCopy.Spec.Email)); !kubeconfigFailure {
						userRequestCopy.Status.State = approved
						userRequestCopy.Status.Message = []string{statusDict["user-approved"]}
						return
					}
				}
			}
			//c.sendEmail(userRequestCopy, tenant.GetName(), "user-creation-failure")
			userRequestCopy.Status.State = failure
			userRequestCopy.Status.Message = []string{statusDict["user-failed"]}
		} else {
			if oldStatus.Expiry == nil {
				// Set the approval timeout which is 72 hours
				userRequestCopy.Status.Expiry = &metav1.Time{
					Time: time.Now().Add(72 * time.Hour),
				}
			}
			exists, _ := util.Contains(userRequestCopy.Status.Message, statusDict["email-ok"])
			if !exists {
				created := access.CreateEmailVerification(userRequestCopy, SetAsOwnerReference(userRequestCopy))
				if created {
					// Update the status as successful
					userRequestCopy.Status.State = success
					userRequestCopy.Status.Message = []string{statusDict["email-ok"]}
				} else {
					userRequestCopy.Status.State = issue
					userRequestCopy.Status.Message = []string{statusDict["email-fail"]}
				}
			}
			ownerReferences := SetAsOwnerReference(userRequestCopy)
			if err := access.CreateObjectSpecificClusterRole(tenant.GetName(), "registration.edgenet.io", "userrequests", userRequestCopy.GetName(), "owner", []string{"get", "update", "patch"}, ownerReferences); err != nil && !errors.IsAlreadyExists(err) {
				log.Infof("Couldn't create user request cluster role %s, %s: %s", tenant.GetName(), userRequestCopy.GetName(), err)
				// TODO: Provide err information at the status
			}

			if acceptableUsePolicyRaw, err := c.edgenetclientset.CoreV1alpha().AcceptableUsePolicies().List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/generated=true,edge-net.io/tenant=%s,edge-net.io/identity=true", tenant.GetName())}); err == nil {
				for _, acceptableUsePolicyRow := range acceptableUsePolicyRaw.Items {
					aupLabels := acceptableUsePolicyRow.GetLabels()
					if aupLabels != nil && aupLabels["edge-net.io/username"] != "" && aupLabels["edge-net.io/role"] != "" {
						if aupLabels["edge-net.io/role"] == "Owner" || aupLabels["edge-net.io/role"] == "Admin" {
							clusterRoleName := fmt.Sprintf("edgenet:%s:userrequests:%s-%s", tenant.GetName(), userRequestCopy.GetName(), "owner")
							roleBindLabels := map[string]string{"edge-net.io/tenant": tenant.GetName(), "edge-net.io/username": aupLabels["edge-net.io/username"], "edge-net.io/user-template-hash": aupLabels["edge-net.io/user-template-hash"]}
							if err := access.CreateObjectSpecificClusterRoleBinding(tenant.GetName(), clusterRoleName, fmt.Sprintf("%s-%s", aupLabels["edge-net.io/username"], aupLabels["edge-net.io/user-template-hash"]), acceptableUsePolicyRow.Spec.Email, roleBindLabels, ownerReferences); err != nil {
								// TODO: Define the error precisely
								userRequestCopy.Status.State = failure
								userRequestCopy.Status.Message = []string{statusDict["role-failed"]}
							}
						}
					}
				}
			}
		}
	} else {
		c.edgenetclientset.RegistrationV1alpha().UserRequests().Delete(context.TODO(), userRequestCopy.GetName(), metav1.DeleteOptions{})
	}
}

// sendEmail to send notification to participants
/*func (c *Controller) sendEmail(userRequest *registrationv1alpha.UserRequest, tenantName, subject string) {
	// Set the HTML template variables
	contentData := mailer.CommonContentData{}
	contentData.CommonData.Tenant = tenantName
	contentData.CommonData.Username = userRequest.GetName()
	contentData.CommonData.Name = fmt.Sprintf("%s %s", userRequest.Spec.FirstName, userRequest.Spec.LastName)
	contentData.CommonData.Email = []string{userRequest.Spec.Email}
	mailer.Send(subject, contentData)
}*/

// RunExpiryController puts a procedure in place to turn accepted policies into not accepted
func (c *Controller) RunExpiryController() {
	var closestExpiry time.Time
	terminated := make(chan bool)
	newExpiry := make(chan time.Time)
	defer close(terminated)
	defer close(newExpiry)

	watchUserRequest, err := c.edgenetclientset.RegistrationV1alpha().UserRequests().Watch(context.TODO(), metav1.ListOptions{})
	if err == nil {
		watchEvents := func(watchUserRequest watch.Interface, newExpiry *chan time.Time) {
			// Watch the events of user request object
			// Get events from watch interface
			for userRequestEvent := range watchUserRequest.ResultChan() {
				// Get updated user request object
				updatedUserRequest, status := userRequestEvent.Object.(*registrationv1alpha.UserRequest)
				if status {
					if updatedUserRequest.Status.Expiry != nil {
						*newExpiry <- updatedUserRequest.Status.Expiry.Time
					}
				}
			}
		}
		go watchEvents(watchUserRequest, &newExpiry)
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
			userRequestRaw, err := c.edgenetclientset.RegistrationV1alpha().UserRequests().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				// TO-DO: Provide more information on error
				klog.V(4).Infoln(err)
			}
			for _, userRequestRow := range userRequestRaw.Items {
				if userRequestRow.Status.Expiry != nil && userRequestRow.Status.Expiry.Time.Sub(time.Now()) <= 0 {
					c.edgenetclientset.RegistrationV1alpha().UserRequests().Delete(context.TODO(), userRequestRow.GetName(), metav1.DeleteOptions{})
				} else if userRequestRow.Status.Expiry != nil && userRequestRow.Status.Expiry.Time.Sub(time.Now()) > 0 {
					if closestExpiry.Sub(time.Now()) <= 0 || closestExpiry.Sub(userRequestRow.Status.Expiry.Time) > 0 {
						closestExpiry = userRequestRow.Status.Expiry.Time
						klog.V(4).Infof("ExpiryController: Closest expiry date is %v after the expiration of a user request", closestExpiry)
					}
				}
			}

			if closestExpiry.Sub(time.Now()) <= 0 {
				closestExpiry = time.Now().AddDate(1, 0, 0)
				klog.V(4).Infof("ExpiryController: Closest expiry date is %v after the expiration of a user request", closestExpiry)
			}
		case <-terminated:
			watchUserRequest.Stop()
			break infiniteLoop
		}
	}
}

// SetAsOwnerReference put the userrequest as owner
func SetAsOwnerReference(userRequest *registrationv1alpha.UserRequest) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newNamespaceRef := *metav1.NewControllerRef(userRequest, registrationv1alpha.SchemeGroupVersion.WithKind("UserRequest"))
	takeControl := false
	newNamespaceRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newNamespaceRef)
	return ownerReferences
}
