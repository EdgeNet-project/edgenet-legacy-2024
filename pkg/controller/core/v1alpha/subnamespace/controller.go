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

package subnamespace

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/core/v1alpha"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/core/v1alpha"
	namespacev1 "github.com/EdgeNet-project/edgenet/pkg/namespace"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/google/uuid"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "subnamespace-controller"

// Definitions of the state of the subnamespace resource
const (
	successSynced          = "Synced"
	messageResourceSynced  = "Subsidiary namespace synced successfully"
	successFormed          = "Formed"
	messageFormed          = "Subsidiary namespace formed successfully"
	successExpired         = "Expired"
	messageExpired         = "Subsidiary namespace deleted successfully"
	successWiped           = "Wiped"
	messageWiped           = "Mode change wiped previous subsidiary namespace"
	successApplied         = "Applied"
	messageApplied         = "Child quota applied successfully"
	successQuotaCheck      = "Checked"
	messageQuotaCheck      = "The parent has sufficient quota"
	failureQuotaShortage   = "Shortage"
	messageQuotaShortage   = "Insufficient quota at the parent"
	failureUpdate          = "Not Updated"
	messageUpdateFail      = "Parent quota cannot be updated"
	failureApplied         = "Not Applied"
	messageApplyFail       = "Child quota cannot be applied"
	failureNotWiped        = "Not Wiped"
	messageNotWiped        = "An error occurred while wiping previous subsidiary namespace"
	failureCreation        = "Not Created"
	messageCreationFail    = "Subsidiary namespace cannot be created"
	failureInheritance     = "Not Inherited"
	messageInheritanceFail = "Inheritance from parent to child failed"
	failure                = "Failure"
	established            = "Established"
)

// Controller is the controller implementation for Subsidiary Namespace resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	subnamespacesLister listers.SubNamespaceLister
	subnamespacesSynced cache.InformerSynced

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
	subnamespaceInformer informers.SubNamespaceInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:       kubeclientset,
		edgenetclientset:    edgenetclientset,
		subnamespacesLister: subnamespaceInformer.Lister(),
		subnamespacesSynced: subnamespaceInformer.Informer().HasSynced,
		workqueue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "SubNamespaces"),
		recorder:            recorder,
	}

	klog.V(4).Infoln("Setting up event handlers")
	// Set up an event handler for when Subsidiary Namespace resources change
	subnamespaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			subnamespace := obj.(*corev1alpha.SubNamespace)
			if subnamespace.Spec.Expiry != nil && time.Until(subnamespace.Spec.Expiry.Time) > 0 {
				controller.enqueueSubNamespaceAfter(obj, time.Until(subnamespace.Spec.Expiry.Time))
			}
			controller.enqueueSubNamespace(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			newSubnamespace := new.(*corev1alpha.SubNamespace)
			oldSubnamespace := old.(*corev1alpha.SubNamespace)
			if (oldSubnamespace.Spec.Expiry == nil && newSubnamespace.Spec.Expiry != nil) ||
				(oldSubnamespace.Spec.Expiry != nil && newSubnamespace.Spec.Expiry != nil && !oldSubnamespace.Spec.Expiry.Time.Equal(newSubnamespace.Spec.Expiry.Time) && time.Until(newSubnamespace.Spec.Expiry.Time) > 0) {
				controller.enqueueSubNamespaceAfter(new, time.Until(newSubnamespace.Spec.Expiry.Time))
			}
			if reflect.DeepEqual(newSubnamespace.Spec, oldSubnamespace.Spec) && (newSubnamespace.Spec.Expiry == nil || time.Until(newSubnamespace.Spec.Expiry.Time) > 0) {
				return
			}
			controller.enqueueSubNamespace(new)
		}, DeleteFunc: func(obj interface{}) {
			subnamespace := obj.(*corev1alpha.SubNamespace)
			if subnamespace.Status.Child != nil {
				switch subnamespace.Status.Child.Kind {
				case "Tenant":
					controller.edgenetclientset.CoreV1alpha().Tenants().Delete(context.TODO(), subnamespace.Status.Child.Name, metav1.DeleteOptions{})
				case "Namespace":
					controller.kubeclientset.CoreV1().Namespaces().Delete(context.TODO(), subnamespace.Status.Child.Name, metav1.DeleteOptions{})
				default:
					return
				}
				namespace, err := controller.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), subnamespace.GetNamespace(), metav1.GetOptions{})
				if err != nil {
					klog.V(4).Infoln(err)
					return
				}
				namespaceLabels := namespace.GetLabels()
				if parentResourceQuota, err := controller.kubeclientset.CoreV1().ResourceQuotas(subnamespace.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-quota", namespaceLabels["edge-net.io/kind"]), metav1.GetOptions{}); err == nil {
					parentResourceQuotaCopy := parentResourceQuota.DeepCopy()
					for key, value := range parentResourceQuotaCopy.Spec.Hard {
						if _, elementExists := subnamespace.Spec.ResourceAllocation[key]; elementExists {
							resourceDemand := subnamespace.Spec.ResourceAllocation[key]
							availableQuota := value.Value()
							parentResourceQuotaCopy.Spec.Hard[key] = *resource.NewQuantity(availableQuota+resourceDemand.Value(), parentResourceQuota.Spec.Hard[key].Format)
						}
					}
					controller.kubeclientset.CoreV1().ResourceQuotas(parentResourceQuota.GetNamespace()).Update(context.TODO(), parentResourceQuotaCopy, metav1.UpdateOptions{})
				}
			}
		},
	})

	return controller
}

// Run will set up the event handlers for the types of subsidiary namespace and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.V(4).Infoln("Starting Subsidiary Namespace controller")

	klog.V(4).Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.subnamespacesSynced); !ok {
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
// converge the two. It then updates the Status block of the Subsidiary Namespace
// resource with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	subnamespace, err := c.subnamespacesLister.SubNamespaces(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("subnamespace '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	c.processSubNamespace(subnamespace.DeepCopy())
	c.recorder.Event(subnamespace, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueSubNamespace takes a Subsidiary Namespace resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Subsidiary Namespace.
func (c *Controller) enqueueSubNamespace(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// enqueueSubNamespaceAfter takes a Subsidiary Namespace resource and converts it into a namespace/name
// string which is then put onto the work queue after the expiry date to be deleted.
// This method should *not* be passed resources of any type other than TenantResourceQuota.
func (c *Controller) enqueueSubNamespaceAfter(obj interface{}, after time.Duration) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.AddAfter(key, after)
}

func (c *Controller) processSubNamespace(subnamespaceCopy *corev1alpha.SubNamespace) {
	if subnamespaceCopy.Spec.Expiry != nil && time.Until(subnamespaceCopy.Spec.Expiry.Time) <= 0 {
		c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, successExpired, messageExpired)
		c.edgenetclientset.CoreV1alpha().SubNamespaces(subnamespaceCopy.GetNamespace()).Delete(context.TODO(), subnamespaceCopy.GetName(), metav1.DeleteOptions{})
		return
	}
	oldStatus := subnamespaceCopy.Status
	statusUpdate := func() {
		if !reflect.DeepEqual(oldStatus, subnamespaceCopy.Status) {
			if _, err := c.edgenetclientset.CoreV1alpha().SubNamespaces(subnamespaceCopy.GetNamespace()).UpdateStatus(context.TODO(), subnamespaceCopy, metav1.UpdateOptions{}); err != nil {
				klog.V(4).Infoln(err)
			}
		}
	}
	defer statusUpdate()

	// Below code checks whether namespace, where role request made, is local to the cluster or is propagated along with a federated deployment.
	// If another cluster propagates the namespace, we skip checking the owner tenant's status as the Selective Deployment entity manages this life-cycle.
	permitted := false
	systemNamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err != nil {
		klog.V(4).Infoln(err)
		return
	}
	namespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), subnamespaceCopy.GetNamespace(), metav1.GetOptions{})
	if err != nil {
		klog.V(4).Infoln(err)
		return
	}
	namespaceLabels := namespace.GetLabels()
	if systemNamespace.GetUID() != types.UID(namespaceLabels["edge-net.io/cluster-uid"]) {
		permitted = true
	} else {
		tenant, err := c.edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), strings.ToLower(namespaceLabels["edge-net.io/tenant"]), metav1.GetOptions{})
		if err != nil {
			klog.V(4).Infoln(err)
			return
		}
		if tenant.GetUID() == types.UID(namespaceLabels["edge-net.io/tenant-uid"]) && tenant.Spec.Enabled {
			permitted = true
		}
	}

	if permitted {
		if strings.ToLower(subnamespaceCopy.Spec.Mode) == "hierarchy" {
			if subnamespaceCopy.Status.Child != nil && subnamespaceCopy.Status.Child.Kind == "Tenant" {
				if err := c.edgenetclientset.CoreV1alpha().Tenants().Delete(context.TODO(), subnamespaceCopy.Status.Child.Name, metav1.DeleteOptions{}); err != nil {
					c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureNotWiped, messageNotWiped)
					subnamespaceCopy.Status.State = failure
					subnamespaceCopy.Status.Message = messageNotWiped
					return
				}
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, successWiped, messageWiped)
			}
			log.Println(time.Now())
			log.Println("In Switch")
			var childName string
			childExists := false
			if subnamespaceCopy.Status.Child != nil && subnamespaceCopy.Status.Child.Kind == "Namespace" {
				childName = subnamespaceCopy.Status.Child.Name
				childExists = true
			} else if subnamespaceCopy.Spec.Scope == nil || *subnamespaceCopy.Spec.Scope == "local" {
				childName = fmt.Sprintf("%s-%s", namespaceLabels["edge-net.io/tenant"], subnamespaceCopy.GetName())
			} else if *subnamespaceCopy.Spec.Scope == "federation" {
				childName = fmt.Sprintf("%s-%s-%s", namespaceLabels["edge-net.io/cluster-uid"], namespaceLabels["edge-net.io/tenant"], subnamespaceCopy.GetName())
			}
			log.Println(time.Now())
			log.Println("Child Name Set")
			if parentResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(subnamespaceCopy.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-quota", namespaceLabels["edge-net.io/kind"]), metav1.GetOptions{}); err == nil {
				sufficientQuota := false
				if subResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(childName).Get(context.TODO(), "sub-quota", metav1.GetOptions{}); err == nil {
					sufficientQuota = c.tuneParentResourceQuota(subnamespaceCopy, parentResourceQuota, subResourceQuota.Spec.Hard)
				} else {
					sufficientQuota = c.tuneParentResourceQuota(subnamespaceCopy, parentResourceQuota, nil)
				}
				if !sufficientQuota {
					return
				}
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, successQuotaCheck, messageQuotaCheck)
			}
			log.Println(time.Now())
			log.Println("Parent Resource Quota Tuned")
			ownerReferences := namespacev1.SetAsOwnerReference(namespace)
			childNamespaceLabels := map[string]string{"edge-net.io/generated": "true", "edge-net.io/kind": "sub", "edge-net.io/tenant": namespaceLabels["edge-net.io/tenant"],
				"edge-net.io/owner": subnamespaceCopy.GetName(), "edge-net.io/parent-namespace": subnamespaceCopy.GetNamespace()}
			childName, namespaceCreated := c.generateSubsidiaryNamespace(subnamespaceCopy, childName, childExists, childNamespaceLabels, ownerReferences)
			if !namespaceCreated {
				return
			}
			log.Println(time.Now())
			log.Println("Namespace Created")
			if subnamespaceCopy.Spec.ResourceAllocation != nil {
				quotaApplied := c.applyChildResourceQuota(subnamespaceCopy, childName)
				if !quotaApplied {
					c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureApplied, messageApplyFail)
					subnamespaceCopy.Status.State = failure
					subnamespaceCopy.Status.Message = failureApplied
					return
				}
			}
			log.Println(time.Now())
			log.Println("Quota Applied")
			done := c.handleInheritance(subnamespaceCopy, childName)
			if !done {
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureInheritance, messageInheritanceFail)
				subnamespaceCopy.Status.State = failure
				subnamespaceCopy.Status.Message = messageInheritanceFail
				return
			}
			log.Println(time.Now())
			log.Println("Inheritance OK")
			if subnamespaceCopy.Status.Child == nil {
				subnamespaceCopy.Status.Child = new(corev1alpha.Child)
			}
			subnamespaceCopy.Status.Child.Kind = "Namespace"
			subnamespaceCopy.Status.Child.Name = childName
		} else if strings.ToLower(subnamespaceCopy.Spec.Mode) == "vendor" {
			if subnamespaceCopy.Status.Child != nil && subnamespaceCopy.Status.Child.Kind == "Namespace" {
				if err := c.kubeclientset.CoreV1().Namespaces().Delete(context.TODO(), subnamespaceCopy.Status.Child.Name, metav1.DeleteOptions{}); err != nil {
					c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureNotWiped, messageNotWiped)
					subnamespaceCopy.Status.State = failure
					subnamespaceCopy.Status.Message = messageNotWiped
					return
				}
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, successWiped, messageWiped)
			}
		}
		subnamespaceCopy.Status.State = established
		subnamespaceCopy.Status.Message = messageFormed
		c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, successFormed, messageFormed)
	}
}

func (c *Controller) tuneParentResourceQuota(subnamespaceCopy *corev1alpha.SubNamespace, parentResourceQuota *corev1.ResourceQuota, childResourceQuota map[corev1.ResourceName]resource.Quantity) bool {
	remainingQuota := make(map[corev1.ResourceName]resource.Quantity)
	for key, value := range parentResourceQuota.Spec.Hard {
		if _, elementExists := subnamespaceCopy.Spec.ResourceAllocation[key]; elementExists {
			resourceDemand := subnamespaceCopy.Spec.ResourceAllocation[key]
			availableQuota := value.Value()
			if _, elementExists := childResourceQuota[key]; elementExists && childResourceQuota != nil {
				appliedQuota := childResourceQuota[key]
				availableQuota += appliedQuota.Value()
			}
			if availableQuota < resourceDemand.Value() {
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureQuotaShortage, messageQuotaShortage)
				subnamespaceCopy.Status.State = failure
				subnamespaceCopy.Status.Message = messageQuotaShortage
				return false
			} else {
				remainingQuota[key] = *resource.NewQuantity(availableQuota-resourceDemand.Value(), parentResourceQuota.Spec.Hard[key].Format)
			}
		}
	}

	parentResourceQuotaCopy := parentResourceQuota.DeepCopy()
	parentResourceQuotaCopy.Spec.Hard = remainingQuota
	if _, err := c.kubeclientset.CoreV1().ResourceQuotas(parentResourceQuota.GetNamespace()).Update(context.TODO(), parentResourceQuotaCopy, metav1.UpdateOptions{}); err != nil {
		c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureUpdate, messageUpdateFail)
		subnamespaceCopy.Status.State = failure
		subnamespaceCopy.Status.Message = messageUpdateFail
		return false
	}
	return true
}

func (c *Controller) generateSubsidiaryNamespace(subnamespaceCopy *corev1alpha.SubNamespace, childName string, childExists bool, labels map[string]string, ownerReferences []metav1.OwnerReference) (string, bool) {
	childNamespaceObj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: childName, OwnerReferences: ownerReferences}}
	childNamespaceObj.SetLabels(labels)
	if _, err := c.kubeclientset.CoreV1().Namespaces().Create(context.TODO(), childNamespaceObj, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureCreation, messageCreationFail)
		subnamespaceCopy.Status.State = failure
		subnamespaceCopy.Status.Message = messageCreationFail
		return childName, false
	} else if err != nil && errors.IsAlreadyExists(err) && !childExists {
		childName = fmt.Sprintf("%s-%s", childName, util.GenerateRandomString(6))
		childNamespaceObj.SetName(childName)
		if _, err := c.kubeclientset.CoreV1().Namespaces().Create(context.TODO(), childNamespaceObj, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureCreation, messageCreationFail)
			subnamespaceCopy.Status.State = failure
			subnamespaceCopy.Status.Message = messageCreationFail
			return childName, false
		}
		//childName, namespaceCreated := c.generateSubsidiaryNamespace(subnamespaceCopy, childName, childExists, labels, ownerReferences)
		//return childName, namespaceCreated
	}
	return childName, true
}

func (c *Controller) applyChildResourceQuota(subnamespaceCopy *corev1alpha.SubNamespace, childName string) bool {
	resourceQuota := corev1.ResourceQuota{}
	resourceQuota.Name = "sub-quota"
	resourceQuota.Spec = corev1.ResourceQuotaSpec{
		Hard: subnamespaceCopy.Spec.ResourceAllocation,
	}
	if _, err := c.kubeclientset.CoreV1().ResourceQuotas(childName).Create(context.TODO(), resourceQuota.DeepCopy(), metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		klog.V(4).Infoln(err)
		return false
	} else if errors.IsAlreadyExists(err) {
		if childResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(childName).Get(context.TODO(), "sub-quota", metav1.GetOptions{}); err != nil {
			klog.V(4).Infoln(err)
			return false
		} else {
			childResourceQuotaCopy := childResourceQuota.DeepCopy()
			childResourceQuotaCopy.Spec.Hard = subnamespaceCopy.Spec.ResourceAllocation
			if _, err := c.kubeclientset.CoreV1().ResourceQuotas(childName).Update(context.TODO(), childResourceQuotaCopy, metav1.UpdateOptions{}); err != nil {
				klog.V(4).Infoln(err)
				return false
			}
		}
	}
	c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, successApplied, messageApplied)
	return true
}

func (c *Controller) handleInheritance(subnamespaceCopy *corev1alpha.SubNamespace, childNamespace string) bool {
	done := true
	if roleRaw, err := c.kubeclientset.RbacV1().Roles(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Inheritance["rbac"] {
		for _, roleRow := range roleRaw.Items {
			role := roleRow.DeepCopy()
			role.SetNamespace(childNamespace)
			role.SetUID(types.UID(uuid.New().String()))
			if _, err := c.kubeclientset.RbacV1().Roles(childNamespace).Create(context.TODO(), role, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
				done = false
			}
		}
	}
	if roleBindingRaw, err := c.kubeclientset.RbacV1().RoleBindings(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Inheritance["rbac"] {
		for _, roleBindingRow := range roleBindingRaw.Items {
			roleBinding := roleBindingRow.DeepCopy()
			roleBinding.SetNamespace(childNamespace)
			roleBinding.SetUID(types.UID(uuid.New().String()))
			if _, err := c.kubeclientset.RbacV1().RoleBindings(childNamespace).Create(context.TODO(), roleBinding, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
				done = false
			}
		}
	}
	if networkPolicyRaw, err := c.kubeclientset.NetworkingV1().NetworkPolicies(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Inheritance["networkpolicy"] {
		for _, networkPolicyRow := range networkPolicyRaw.Items {
			networkPolicy := networkPolicyRow.DeepCopy()
			networkPolicy.SetNamespace(childNamespace)
			networkPolicy.SetUID(types.UID(uuid.New().String()))
			if _, err := c.kubeclientset.NetworkingV1().NetworkPolicies(childNamespace).Create(context.TODO(), networkPolicy, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
				done = false
			}
		}
	}
	if limitRangeRaw, err := c.kubeclientset.CoreV1().LimitRanges(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Inheritance["limitrange"] {
		for _, limitRangeRow := range limitRangeRaw.Items {
			limitRange := limitRangeRow.DeepCopy()
			limitRange.SetNamespace(childNamespace)
			limitRange.SetUID(types.UID(uuid.New().String()))
			if _, err := c.kubeclientset.CoreV1().LimitRanges(childNamespace).Create(context.TODO(), limitRange, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
				done = false
			}
		}
	}
	if secretRaw, err := c.kubeclientset.CoreV1().Secrets(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Inheritance["secret"] {
		for _, secretRow := range secretRaw.Items {
			secret := secretRow.DeepCopy()
			secret.SetNamespace(childNamespace)
			secret.SetUID(types.UID(uuid.New().String()))
			if _, err := c.kubeclientset.CoreV1().Secrets(childNamespace).Create(context.TODO(), secret, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
				done = false
			}
		}
	}
	if configMapRaw, err := c.kubeclientset.CoreV1().ConfigMaps(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Inheritance["configmap"] {
		for _, configMapRow := range configMapRaw.Items {
			configMap := configMapRow.DeepCopy()
			configMap.SetNamespace(childNamespace)
			configMap.SetUID(types.UID(uuid.New().String()))
			if _, err := c.kubeclientset.CoreV1().ConfigMaps(childNamespace).Create(context.TODO(), configMap, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
				done = false
			}
		}
	}
	if serviceAccountRaw, err := c.kubeclientset.CoreV1().ServiceAccounts(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Inheritance["serviceaccount"] {
		for _, serviceAccountRow := range serviceAccountRaw.Items {
			serviceAccount := serviceAccountRow.DeepCopy()
			serviceAccount.SetNamespace(childNamespace)
			serviceAccount.SetUID(types.UID(uuid.New().String()))
			if _, err := c.kubeclientset.CoreV1().ServiceAccounts(childNamespace).Create(context.TODO(), serviceAccount, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
				done = false
			}
		}
	}
	return done
}
