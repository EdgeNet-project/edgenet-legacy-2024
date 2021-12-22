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

package acceptableusepolicy

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/access"
	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/core/v1alpha"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/core/v1alpha"

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

const controllerAgentName = "aup-controller"

// Definitions of the state of the acceptableusepolicy resource
const (
	successSynced         = "Synced"
	messageResourceSynced = "Acceptable Use Policy synced successfully"
	successAgreed         = "Agreed"
	messageAgreed         = "Acceptable Use Policy agreed successfully"
	warningNotAgreed      = "Not Agreed"
	messageNotAgreed      = "Waiting for the Acceptable Use Policy to be agreed"
	warningRevoked        = "Revoked"
	messageRevoked        = "Acceptable Use Policy revoked and user access restricted"
	warningUpdate         = "Not Updates"
	messageUpdate         = "Failed to update the status of associated resources"
	failure               = "Failure"
	pending               = "Pending"
	success               = "Successful"
)

// Controller is the controller implementation for Acceptable Use Policy resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	acceptableusepoliciesLister listers.AcceptableUsePolicyLister
	acceptableusepoliciesSynced cache.InformerSynced

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
	acceptableusepolicyInformer informers.AcceptableUsePolicyInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:               kubeclientset,
		edgenetclientset:            edgenetclientset,
		acceptableusepoliciesLister: acceptableusepolicyInformer.Lister(),
		acceptableusepoliciesSynced: acceptableusepolicyInformer.Informer().HasSynced,
		workqueue:                   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "AcceptableUsePolicies"),
		recorder:                    recorder,
	}

	klog.V(4).Infoln("Setting up event handlers")
	// Set up an event handler for when Acceptable Use Policy resources change
	acceptableusepolicyInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueAcceptableUsePolicy,
		UpdateFunc: func(old, new interface{}) {
			newAcceptableUsePolicy := new.(*corev1alpha.AcceptableUsePolicy)
			oldAcceptableUsePolicy := old.(*corev1alpha.AcceptableUsePolicy)
			if reflect.DeepEqual(newAcceptableUsePolicy.Spec, oldAcceptableUsePolicy.Spec) {
				return
			}
			controller.enqueueAcceptableUsePolicy(new)
		},
	})

	access.Clientset = kubeclientset
	access.EdgenetClientset = edgenetclientset

	return controller
}

// Run will set up the event handlers for the types of acceptable use policy and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.V(4).Infoln("Starting Acceptable Use Policy controller")

	klog.V(4).Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.acceptableusepoliciesSynced); !ok {
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
// converge the two. It then updates the Status block of the Acceptable Use Policy
// resource with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	acceptableusepolicy, err := c.acceptableusepoliciesLister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("acceptableusepolicy '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	c.processAcceptableUsePolicy(acceptableusepolicy.DeepCopy())

	c.recorder.Event(acceptableusepolicy, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueAcceptableUsePolicy takes an AcceptableUsePolicy resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than AcceptableUsePolicy.
func (c *Controller) enqueueAcceptableUsePolicy(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *Controller) processAcceptableUsePolicy(acceptableUsePolicyCopy *corev1alpha.AcceptableUsePolicy) {
	oldStatus := acceptableUsePolicyCopy.Status
	statusUpdate := func() {
		if !reflect.DeepEqual(oldStatus, acceptableUsePolicyCopy.Status) {
			c.edgenetclientset.CoreV1alpha().AcceptableUsePolicies().UpdateStatus(context.TODO(), acceptableUsePolicyCopy, metav1.UpdateOptions{})
		}
	}
	defer statusUpdate()
	if acceptableUsePolicyCopy.Spec.Accepted {
		if acceptableUsePolicyCopy.Status.State == success && acceptableUsePolicyCopy.Status.Message == messageAgreed {
			return
		}
		c.recorder.Event(acceptableUsePolicyCopy, corev1.EventTypeNormal, successAgreed, messageAgreed)
		acceptableUsePolicyCopy.Status.State = success
		acceptableUsePolicyCopy.Status.Message = messageAgreed

		if roleRequestRaw, err := c.edgenetclientset.RegistrationV1alpha().RoleRequests("").List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/acceptable-use-policy=%s", acceptableUsePolicyCopy.GetName())}); err == nil {
			for _, roleRequestRow := range roleRequestRaw.Items {
				ownerReferences := roleRequestRow.GetOwnerReferences()
				for _, ownerReference := range ownerReferences {
					if ownerReference.UID == acceptableUsePolicyCopy.GetUID() {
						roleRequestCopy := roleRequestRow.DeepCopy()
						roleRequestCopy.Status.PolicyAgreed = &acceptableUsePolicyCopy.Spec.Accepted
						if _, err := c.edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequestCopy.GetNamespace()).UpdateStatus(context.TODO(), roleRequestCopy, metav1.UpdateOptions{}); err != nil {
							c.recorder.Event(acceptableUsePolicyCopy, corev1.EventTypeWarning, warningUpdate, messageUpdate)
							klog.V(4).Infoln(err)
						}
					}
				}
			}
		}

		if tenantRequestRaw, err := c.edgenetclientset.RegistrationV1alpha().TenantRequests().List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/acceptable-use-policy=%s", acceptableUsePolicyCopy.GetName())}); err == nil {
			for _, tenantRequestRow := range tenantRequestRaw.Items {
				ownerReferences := tenantRequestRow.GetOwnerReferences()
				for _, ownerReference := range ownerReferences {
					if ownerReference.UID == acceptableUsePolicyCopy.GetUID() {
						tenantRequestCopy := tenantRequestRow.DeepCopy()
						tenantRequestCopy.Status.PolicyAgreed = &acceptableUsePolicyCopy.Spec.Accepted
						if _, err := c.edgenetclientset.RegistrationV1alpha().TenantRequests().UpdateStatus(context.TODO(), tenantRequestCopy, metav1.UpdateOptions{}); err != nil {
							c.recorder.Event(acceptableUsePolicyCopy, corev1.EventTypeWarning, warningUpdate, messageUpdate)
							klog.V(4).Infoln(err)
						}
					}
				}
			}
		}

		if tenantRaw, err := c.edgenetclientset.CoreV1alpha().Tenants().List(context.TODO(), metav1.ListOptions{}); err == nil {
			for _, tenantRow := range tenantRaw.Items {
				if acceptableUsePolicyCopy.Spec.Email == tenantRow.Spec.Contact.Email {
					tenantCopy := tenantRow.DeepCopy()
					tenantCopy.Status.PolicyAgreed[acceptableUsePolicyCopy.GetName()] = acceptableUsePolicyCopy.Spec.Accepted
					if _, err := c.edgenetclientset.CoreV1alpha().Tenants().UpdateStatus(context.TODO(), tenantCopy, metav1.UpdateOptions{}); err != nil {
						c.recorder.Event(acceptableUsePolicyCopy, corev1.EventTypeWarning, warningUpdate, messageUpdate)
						klog.V(4).Infoln(err)
					}
				}
			}
		}
	} else if !acceptableUsePolicyCopy.Spec.Accepted {
		if acceptableUsePolicyCopy.Status.State == success {
			c.recorder.Event(acceptableUsePolicyCopy, corev1.EventTypeWarning, warningRevoked, messageRevoked)
			acceptableUsePolicyCopy.Status.State = failure
			acceptableUsePolicyCopy.Status.Message = messageRevoked
			// TODO: Remove all rolebindings of user

		} else {
			if (acceptableUsePolicyCopy.Status.State == pending && acceptableUsePolicyCopy.Status.Message == messageNotAgreed) ||
				(acceptableUsePolicyCopy.Status.State == failure && acceptableUsePolicyCopy.Status.Message == messageRevoked) {
				return
			}
			c.recorder.Event(acceptableUsePolicyCopy, corev1.EventTypeWarning, warningNotAgreed, messageNotAgreed)
			acceptableUsePolicyCopy.Status.State = pending
			acceptableUsePolicyCopy.Status.Message = messageNotAgreed
			// Notify User to Agree ON AUP
		}
	}
}

// SetAsOwnerReference put the rolerequest as owner
func SetAsOwnerReference(roleRequest *corev1alpha.AcceptableUsePolicy) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newNamespaceRef := *metav1.NewControllerRef(roleRequest, corev1alpha.SchemeGroupVersion.WithKind("AcceptableUsePolicy"))
	takeControl := true
	newNamespaceRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newNamespaceRef)
	return ownerReferences
}
