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

// TODO: Clean up the code

package subnamespace

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/access"
	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/core/v1alpha"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/core/v1alpha"
	namespacev1 "github.com/EdgeNet-project/edgenet/pkg/namespace"

	"github.com/google/uuid"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	networkinginformers "k8s.io/client-go/informers/networking/v1"
	rbacinformers "k8s.io/client-go/informers/rbac/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	networkinglisters "k8s.io/client-go/listers/networking/v1"
	rbaclisters "k8s.io/client-go/listers/rbac/v1"
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
	failureCreation        = "Not Created"
	messageCreationFail    = "Subsidiary namespace cannot be created"
	failureInheritance     = "Not Inherited"
	messageInheritanceFail = "Inheritance from parent to child failed"
	failureBinding         = "Binding Failed"
	messageBindingFailed   = "Role binding failed"
	failureCollision       = "Name Collision"
	messageCollision       = "Name is not available. Please choose another one."
	failureSlice           = "Slice Unready"
	messageSlice           = "Slice is not ready to be used."
	failure                = "Failure"
	established            = "Established"
	bound                  = "Bound"
)

// Controller is the controller implementation for Subsidiary Namespace resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	subnamespacesLister listers.SubNamespaceLister
	subnamespacesSynced cache.InformerSynced

	rolesLister           rbaclisters.RoleLister
	rolesSynced           cache.InformerSynced
	rolebindingsLister    rbaclisters.RoleBindingLister
	rolebindingsSynced    cache.InformerSynced
	networkpoliciesLister networkinglisters.NetworkPolicyLister
	networkpoliciesSynced cache.InformerSynced
	limitrangesLister     corelisters.LimitRangeLister
	limitrangesSynced     cache.InformerSynced
	secretsLister         corelisters.SecretLister
	secretsSynced         cache.InformerSynced
	configmapsLister      corelisters.ConfigMapLister
	configmapsSynced      cache.InformerSynced
	serviceaccountsLister corelisters.ServiceAccountLister
	serviceaccountsSynced cache.InformerSynced

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
	roleInformer rbacinformers.RoleInformer,
	rolebindingInformer rbacinformers.RoleBindingInformer,
	networkpolicyInformer networkinginformers.NetworkPolicyInformer,
	limitrangeInformer coreinformers.LimitRangeInformer,
	secretInformer coreinformers.SecretInformer,
	configmapInformer coreinformers.ConfigMapInformer,
	serviceaccountInformer coreinformers.ServiceAccountInformer,
	subnamespaceInformer informers.SubNamespaceInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:         kubeclientset,
		edgenetclientset:      edgenetclientset,
		rolesLister:           roleInformer.Lister(),
		rolesSynced:           roleInformer.Informer().HasSynced,
		rolebindingsLister:    rolebindingInformer.Lister(),
		rolebindingsSynced:    roleInformer.Informer().HasSynced,
		networkpoliciesLister: networkpolicyInformer.Lister(),
		networkpoliciesSynced: networkpolicyInformer.Informer().HasSynced,
		limitrangesLister:     limitrangeInformer.Lister(),
		limitrangesSynced:     limitrangeInformer.Informer().HasSynced,
		secretsLister:         secretInformer.Lister(),
		secretsSynced:         secretInformer.Informer().HasSynced,
		configmapsLister:      configmapInformer.Lister(),
		configmapsSynced:      configmapInformer.Informer().HasSynced,
		serviceaccountsLister: serviceaccountInformer.Lister(),
		serviceaccountsSynced: serviceaccountInformer.Informer().HasSynced,
		subnamespacesLister:   subnamespaceInformer.Lister(),
		subnamespacesSynced:   subnamespaceInformer.Informer().HasSynced,
		workqueue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "SubNamespaces"),
		recorder:              recorder,
	}

	klog.Infoln("Setting up event handlers")
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
			if reflect.DeepEqual(newSubnamespace.Spec, oldSubnamespace.Spec) && (newSubnamespace.Spec.Expiry == nil || time.Until(newSubnamespace.Spec.Expiry.Time) > 0) {
				return
			} else {
				controller.enqueueSubNamespace(new)
				if (oldSubnamespace.Spec.Expiry == nil && newSubnamespace.Spec.Expiry != nil) ||
					(oldSubnamespace.Spec.Expiry != nil && newSubnamespace.Spec.Expiry != nil && !oldSubnamespace.Spec.Expiry.Time.Equal(newSubnamespace.Spec.Expiry.Time) && time.Until(newSubnamespace.Spec.Expiry.Time) > 0) {
					controller.enqueueSubNamespaceAfter(new, time.Until(newSubnamespace.Spec.Expiry.Time))
				}
			}
		}, DeleteFunc: func(obj interface{}) {
			subnamespace := obj.(*corev1alpha.SubNamespace)
			namespace, err := controller.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), subnamespace.GetNamespace(), metav1.GetOptions{})
			if err != nil {
				klog.Infoln(err)
				return
			}
			namespaceLabels := namespace.GetLabels()

			childNameHashed := subnamespace.GenerateChildName(namespaceLabels["edge-net.io/cluster-uid"])
			sliceExists := false
			switch subnamespace.GetMode() {
			case "workspace":
				if childExists, childOwned := controller.validateChildOwnership(namespace, subnamespace.GetMode(), childNameHashed); childExists && childOwned {
					controller.kubeclientset.CoreV1().Namespaces().Delete(context.TODO(), childNameHashed, metav1.DeleteOptions{})
				} else {
					return
				}
				if subnamespace.Spec.Workspace.SliceClaim != nil {
					sliceExists = true
				}
			case "subtenant":
				if childExists, childOwned := controller.validateChildOwnership(namespace, subnamespace.GetMode(), childNameHashed); childExists && childOwned {
					controller.edgenetclientset.CoreV1alpha().Tenants().Delete(context.TODO(), childNameHashed, metav1.DeleteOptions{})
				} else {
					return
				}
				if subnamespace.Spec.Subtenant.SliceClaim != nil {
					sliceExists = true
				}
			}

			if !sliceExists {
				if parentResourceQuota, err := controller.kubeclientset.CoreV1().ResourceQuotas(subnamespace.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-quota", namespaceLabels["edge-net.io/kind"]), metav1.GetOptions{}); err == nil {
					returnedQuota := make(map[corev1.ResourceName]resource.Quantity)
					for key, value := range parentResourceQuota.Spec.Hard {
						resourceDemand := subnamespace.RetrieveQuantityValue(key)
						availableQuota := value.Value()
						returnedQuota[key] = *resource.NewQuantity(availableQuota+resourceDemand, parentResourceQuota.Spec.Hard[key].Format)
					}
					parentResourceQuotaCopy := parentResourceQuota.DeepCopy()
					parentResourceQuotaCopy.Spec.Hard = returnedQuota
					controller.kubeclientset.CoreV1().ResourceQuotas(parentResourceQuota.GetNamespace()).Update(context.TODO(), parentResourceQuotaCopy, metav1.UpdateOptions{})
				}
			}
		},
	})

	roleInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newObj := new.(*rbacv1.Role)
			oldObj := old.(*rbacv1.Role)
			if newObj.ResourceVersion == oldObj.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	rolebindingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newObj := new.(*rbacv1.RoleBinding)
			oldObj := old.(*rbacv1.RoleBinding)
			if newObj.ResourceVersion == oldObj.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	networkpolicyInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newObj := new.(*networkingv1.NetworkPolicy)
			oldObj := old.(*networkingv1.NetworkPolicy)
			if newObj.ResourceVersion == oldObj.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	limitrangeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newObj := new.(*corev1.LimitRange)
			oldObj := old.(*corev1.LimitRange)
			if newObj.ResourceVersion == oldObj.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	secretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newObj := new.(*corev1.Secret)
			oldObj := old.(*corev1.Secret)
			if newObj.ResourceVersion == oldObj.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	configmapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newObj := new.(*corev1.ConfigMap)
			oldObj := old.(*corev1.ConfigMap)
			if newObj.ResourceVersion == oldObj.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	serviceaccountInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newObj := new.(*corev1.ServiceAccount)
			oldObj := old.(*corev1.ServiceAccount)
			if newObj.ResourceVersion == oldObj.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	access.Clientset = kubeclientset
	access.EdgenetClientset = edgenetclientset

	return controller
}

// Run will set up the event handlers for the types of subsidiary namespace and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Infoln("Starting Subsidiary Namespace controller")

	klog.Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.subnamespacesSynced); !ok {
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

// handleObject will take any resource implementing metav1.Object and attempt
// to find the SubNamespace resource that 'owns' its namespace. It does this by
// looking at the objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that SubNamespace resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	objectLabels := object.GetLabels()
	if objectLabels["edge-net.io/generated"] != "true" {
		return
	}
	klog.Infof("Processing object: %s", object.GetName())

	namespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), object.GetNamespace(), metav1.GetOptions{})
	if err != nil {
		return
	}

	subnamespaceRaw, err := c.subnamespacesLister.SubNamespaces(object.GetNamespace()).List(labels.Everything())
	if err == nil {
		for _, subnamespaceRow := range subnamespaceRaw {
			c.enqueueSubNamespaceAfter(subnamespaceRow, 30*time.Second)
		}
	}

	if ownerRef := metav1.GetControllerOf(namespace); ownerRef != nil {
		if ownerRef.Kind != "Namespace" {
			return
		}

		parentnamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
		if err != nil {
			return
		}
		parentnamespaceLabels := parentnamespace.GetLabels()

		subnamespaceRaw, err = c.subnamespacesLister.SubNamespaces(ownerRef.Name).List(labels.Everything())
		if err != nil {
			klog.Infof("ignoring orphaned object '%s' of subnamespace '%s'", object.GetSelfLink(), ownerRef.Name)
		} else {
			for _, subnamespaceRow := range subnamespaceRaw {
				childNameHashed := subnamespaceRow.GenerateChildName(parentnamespaceLabels["edge-net.io/cluster-uid"])
				if childExist, childOwned := c.validateChildOwnership(parentnamespace, subnamespaceRow.GetMode(), childNameHashed); childExist && childOwned {
					c.enqueueSubNamespace(subnamespaceRow)
				}
			}
		}
		return
	}
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
				klog.Infoln(err)
			}
		}
	}
	defer statusUpdate()

	// Below code checks whether namespace, where role request made, is local to the cluster or is propagated along with a federated deployment.
	// If another cluster propagates the namespace, we skip checking the owner tenant's status as the Selective Deployment entity manages this life-cycle.
	permitted := false
	systemNamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err != nil {
		klog.Infoln(err)
		return
	}
	namespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), subnamespaceCopy.GetNamespace(), metav1.GetOptions{})
	if err != nil {
		klog.Infoln(err)
		return
	}
	namespaceLabels := namespace.GetLabels()
	if systemNamespace.GetUID() != types.UID(namespaceLabels["edge-net.io/cluster-uid"]) {
		permitted = true
	} else {
		if tenant, err := c.edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), strings.ToLower(namespaceLabels["edge-net.io/tenant"]), metav1.GetOptions{}); err == nil {
			if tenant.GetUID() == types.UID(namespaceLabels["edge-net.io/tenant-uid"]) && tenant.Spec.Enabled {
				permitted = true
			}
		} else {
			klog.Infoln(err)
			return
		}
	}

	if permitted {
		var labels = map[string]string{"edge-net.io/generated": "true", "edge-net.io/kind": "sub"}
		var annotations = map[string]string{"scheduler.alpha.kubernetes.io/node-selector": "edge-net.io/access=public,edge-net.io/slice=none"}
		var childResourceQuota map[corev1.ResourceName]resource.Quantity

		childNameHashed := subnamespaceCopy.GenerateChildName(namespaceLabels["edge-net.io/cluster-uid"])
		childExist, childOwned := c.validateChildOwnership(namespace, subnamespaceCopy.GetMode(), childNameHashed)
		if childExist && !childOwned {
			c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureCollision, messageCollision)
			subnamespaceCopy.Status.State = failure
			subnamespaceCopy.Status.Message = messageCollision
			return
		}

		ownerReferences := namespacev1.SetAsOwnerReference(namespace)
		sliceClaimExists := false
		var sliceName string
		switch subnamespaceCopy.GetMode() {
		case "workspace":
			if subnamespaceCopy.Spec.Workspace.SliceClaim != nil {
				sliceName = *subnamespaceCopy.Spec.Workspace.SliceClaim
				sliceClaimExists = true
			} else {
				if subResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(childNameHashed).Get(context.TODO(), "sub-quota", metav1.GetOptions{}); err == nil {
					childResourceQuota = subResourceQuota.Spec.Hard
				}
			}
			labels = map[string]string{"edge-net.io/generated": "true", "edge-net.io/kind": "sub", "edge-net.io/tenant": namespaceLabels["edge-net.io/tenant"],
				"edge-net.io/owner": subnamespaceCopy.GetName(), "edge-net.io/parent-namespace": subnamespaceCopy.GetNamespace()}
		case "subtenant":
			if subnamespaceCopy.Spec.Subtenant.SliceClaim != nil {
				sliceName = *subnamespaceCopy.Spec.Workspace.SliceClaim
				sliceClaimExists = true
			} else {
				if subtenantResourceQuota, err := c.edgenetclientset.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), childNameHashed, metav1.GetOptions{}); err == nil {
					_, assignedQuota := subtenantResourceQuota.Fetch()
					childResourceQuota = assignedQuota
				}
			}
		}

		if !sliceClaimExists {
			if parentResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(subnamespaceCopy.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-quota", namespaceLabels["edge-net.io/kind"]), metav1.GetOptions{}); err == nil {
				if sufficientQuota := c.tuneParentResourceQuota(subnamespaceCopy, parentResourceQuota, childResourceQuota); !sufficientQuota {
					return
				}
			}
		} else {
			if isBound := c.checkSliceClaim(subnamespaceCopy.GetNamespace(), sliceName); !isBound {
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureSlice, messageSlice)
				subnamespaceCopy.Status.State = failure
				subnamespaceCopy.Status.Message = failureSlice
				return
			}
			annotations = map[string]string{"scheduler.alpha.kubernetes.io/node-selector": fmt.Sprintf("edge-net.io/access=private,edge-net.io/slice=%s", *subnamespaceCopy.Spec.Workspace.SliceClaim)}
		}
		childInitiated := c.constructSubsidiaryNamespace(subnamespaceCopy, childNameHashed, childExist, annotations, labels, ownerReferences)
		if !childInitiated {
			// TODO: Error handling
			if parentResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(subnamespaceCopy.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-quota", namespaceLabels["edge-net.io/kind"]), metav1.GetOptions{}); err == nil {
				c.returnParentResourceQuota(subnamespaceCopy, parentResourceQuota)
			}
			c.tareChildResourceQuota(subnamespaceCopy, childNameHashed)
			return
		}

		subnamespaceCopy.Status.State = established
		subnamespaceCopy.Status.Message = messageFormed
		c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, successFormed, messageFormed)
	}
}

func (c *Controller) checkSliceClaim(namespace, name string) bool {
	if sliceClaim, err := c.edgenetclientset.CoreV1alpha().SliceClaims(namespace).Get(context.TODO(), name, metav1.GetOptions{}); err != nil {
		if sliceClaim.Status.State != bound {
			return false
		}
	} else {
		return false
	}
	return true
}

func (c *Controller) validateChildOwnership(parentNamespace *corev1.Namespace, mode, childName string) (bool, bool) {
	if mode == "workspace" {
		if childNamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childName, metav1.GetOptions{}); err == nil {
			for _, ownerReference := range childNamespace.GetOwnerReferences() {
				if ownerReference.Kind == "Namespace" && ownerReference.UID == parentNamespace.GetUID() && ownerReference.Name == parentNamespace.GetName() {
					return true, true
				}
			}
			return true, false
		}
	} else {
		if subtenant, err := c.edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), childName, metav1.GetOptions{}); err == nil {
			for _, ownerReference := range subtenant.GetOwnerReferences() {
				if ownerReference.Kind == "Namespace" && ownerReference.UID == parentNamespace.GetUID() && ownerReference.Name == parentNamespace.GetName() {
					return true, true
				}
			}
			return true, false
		}
	}
	return false, false
}

func (c *Controller) tuneParentResourceQuota(subnamespaceCopy *corev1alpha.SubNamespace, parentResourceQuota *corev1.ResourceQuota, childResourceQuota map[corev1.ResourceName]resource.Quantity) bool {
	remainingQuota := make(map[corev1.ResourceName]resource.Quantity)
	for key, value := range parentResourceQuota.Spec.Hard {
		resourceDemand := subnamespaceCopy.RetrieveQuantityValue(key)
		availableQuota := value.Value()

		if _, elementExists := childResourceQuota[key]; childResourceQuota != nil && elementExists {
			appliedQuota := childResourceQuota[key]
			availableQuota += appliedQuota.Value()
		}

		if availableQuota < resourceDemand {
			c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureQuotaShortage, messageQuotaShortage)
			subnamespaceCopy.Status.State = failure
			subnamespaceCopy.Status.Message = messageQuotaShortage
			return false
		} else {
			remainingQuota[key] = *resource.NewQuantity(availableQuota-resourceDemand, parentResourceQuota.Spec.Hard[key].Format)
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

	c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, successQuotaCheck, messageQuotaCheck)
	return true
}

func (c *Controller) returnParentResourceQuota(subnamespaceCopy *corev1alpha.SubNamespace, parentResourceQuota *corev1.ResourceQuota) bool {
	returnedQuota := make(map[corev1.ResourceName]resource.Quantity)
	for key, value := range parentResourceQuota.Spec.Hard {
		resourceDemand := subnamespaceCopy.RetrieveQuantityValue(key)
		remainingQuota := value.Value()
		returnedQuota[key] = *resource.NewQuantity(remainingQuota+resourceDemand, parentResourceQuota.Spec.Hard[key].Format)
	}

	parentResourceQuotaCopy := parentResourceQuota.DeepCopy()
	parentResourceQuotaCopy.Spec.Hard = returnedQuota
	if _, err := c.kubeclientset.CoreV1().ResourceQuotas(parentResourceQuota.GetNamespace()).Update(context.TODO(), parentResourceQuotaCopy, metav1.UpdateOptions{}); err != nil {
		c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureUpdate, messageUpdateFail)
		subnamespaceCopy.Status.State = failure
		subnamespaceCopy.Status.Message = messageUpdateFail
		return false
	}

	c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, successQuotaCheck, messageQuotaCheck)
	return true
}

func (c *Controller) tareChildResourceQuota(subnamespaceCopy *corev1alpha.SubNamespace, childNameHashed string) bool {
	switch subnamespaceCopy.GetMode() {
	case "workspace":
		if subResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(childNameHashed).Get(context.TODO(), "sub-quota", metav1.GetOptions{}); err == nil {
			taredQuota := make(map[corev1.ResourceName]resource.Quantity)
			for key, _ := range subResourceQuota.Spec.Hard {
				taredQuota[key] = *resource.NewQuantity(0, subResourceQuota.Spec.Hard[key].Format)
			}
			subResourceQuotaCopy := subResourceQuota.DeepCopy()
			subResourceQuotaCopy.Spec.Hard = taredQuota
			if _, err := c.kubeclientset.CoreV1().ResourceQuotas(subResourceQuota.GetNamespace()).Update(context.TODO(), subResourceQuotaCopy, metav1.UpdateOptions{}); err != nil {
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureUpdate, messageUpdateFail)
				subnamespaceCopy.Status.State = failure
				subnamespaceCopy.Status.Message = messageUpdateFail
				return false
			}
		}
	case "subtenant":
		if subtenantResourceQuota, err := c.edgenetclientset.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), childNameHashed, metav1.GetOptions{}); err == nil {
			taredQuota := make(map[corev1.ResourceName]resource.Quantity)
			for key, _ := range subtenantResourceQuota.Spec.Claim["initial"].ResourceList {
				taredQuota[key] = *resource.NewQuantity(0, subtenantResourceQuota.Spec.Claim["initial"].ResourceList[key].Format)
			}
			subtenantResourceQuotaCopy := subtenantResourceQuota.DeepCopy()
			claim := corev1alpha.ResourceTuning{
				ResourceList: taredQuota,
			}
			subtenantResourceQuotaCopy.Spec.Claim["initial"] = claim
			if _, err := c.edgenetclientset.CoreV1alpha().TenantResourceQuotas().Update(context.TODO(), subtenantResourceQuotaCopy, metav1.UpdateOptions{}); err != nil {
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureUpdate, messageUpdateFail)
				subnamespaceCopy.Status.State = failure
				subnamespaceCopy.Status.Message = messageUpdateFail
				return false
			}
		}
	}
	return true
}

func (c *Controller) constructSubsidiaryNamespace(subnamespaceCopy *corev1alpha.SubNamespace, childName string, childExists bool, annotations, labels map[string]string, ownerReferences []metav1.OwnerReference) bool {
	switch subnamespaceCopy.GetMode() {
	case "workspace":
		if !childExists {
			childNamespaceObj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: childName, OwnerReferences: ownerReferences}}
			childNamespaceObj.SetName(childName)
			childNamespaceObj.SetAnnotations(annotations)
			childNamespaceObj.SetLabels(labels)
			if _, err := c.kubeclientset.CoreV1().Namespaces().Create(context.TODO(), childNamespaceObj, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureCreation, messageCreationFail)
				subnamespaceCopy.Status.State = failure
				subnamespaceCopy.Status.Message = messageCreationFail
				return false
			}
		}

		if subnamespaceCopy.Spec.Workspace.Owner != nil {
			objectName := "edgenet:workspace:owner"
			if roleBinding, err := c.kubeclientset.RbacV1().RoleBindings(subnamespaceCopy.GetNamespace()).Get(context.TODO(), objectName, metav1.GetOptions{}); err == nil {
				roleBindingCopy := roleBinding.DeepCopy()
				roleBindingCopy.Subjects = []rbacv1.Subject{{Kind: "User", Name: subnamespaceCopy.Spec.Workspace.Owner.Email, APIGroup: "rbac.authorization.k8s.io"}}
				if _, err := c.kubeclientset.RbacV1().RoleBindings(subnamespaceCopy.GetNamespace()).Update(context.TODO(), roleBindingCopy, metav1.UpdateOptions{}); err != nil {
					c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
					subnamespaceCopy.Status.State = failure
					subnamespaceCopy.Status.Message = messageBindingFailed
					klog.Infoln(err)
					return false
				}
			} else {
				roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: "edgenet:workspace:owner"}
				rbSubjects := []rbacv1.Subject{{Kind: "User", Name: subnamespaceCopy.Spec.Workspace.Owner.Email, APIGroup: "rbac.authorization.k8s.io"}}
				roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: objectName, Namespace: subnamespaceCopy.GetNamespace()},
					Subjects: rbSubjects, RoleRef: roleRef}
				roleBindLabels := map[string]string{"edge-net.io/generated": "true"}
				roleBind.SetLabels(roleBindLabels)
				if _, err := c.kubeclientset.RbacV1().RoleBindings(subnamespaceCopy.GetNamespace()).Create(context.TODO(), roleBind, metav1.CreateOptions{}); err != nil {
					c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
					subnamespaceCopy.Status.State = failure
					subnamespaceCopy.Status.Message = messageBindingFailed
					klog.Infoln(err)
					return false
				}
			}
		}

		if subnamespaceCopy.Spec.Workspace.ResourceAllocation != nil {
			quotaApplied := c.applyChildResourceQuota(subnamespaceCopy, childName, ownerReferences)
			if !quotaApplied {
				return false
			}
		}

		done := c.handleInheritance(subnamespaceCopy, childName)
		if !done {
			return false
		}
	case "subtenant":
		if !childExists {
			// Separate tenant creation and tenant resource quota creation
			tenantRequest := new(registrationv1alpha.TenantRequest)
			tenantRequest.SetName(childName)
			tenantRequest.SetAnnotations(annotations)
			tenantRequest.SetLabels(labels)
			tenantRequest.SetOwnerReferences(ownerReferences)
			tenantRequest.Spec.Contact = subnamespaceCopy.Spec.Subtenant.Owner
			tenantRequest.Spec.ResourceAllocation = subnamespaceCopy.Spec.Subtenant.ResourceAllocation
			if err := access.CreateTenant(tenantRequest); err != nil {
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureCreation, messageCreationFail)
				subnamespaceCopy.Status.State = failure
				subnamespaceCopy.Status.Message = messageCreationFail
				return false
			}
		} else {
			if subnamespaceCopy.Spec.Subtenant.ResourceAllocation != nil {
				quotaApplied := c.applyChildResourceQuota(subnamespaceCopy, childName, ownerReferences)
				if !quotaApplied {
					return false
				}
			}
		}
	}
	return true
}

func (c *Controller) applyChildResourceQuota(subnamespaceCopy *corev1alpha.SubNamespace, childName string, ownerReferences []metav1.OwnerReference) bool {
	switch subnamespaceCopy.GetMode() {
	case "workspace":
		if childResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(childName).Get(context.TODO(), "sub-quota", metav1.GetOptions{}); err == nil {
			childResourceQuotaCopy := childResourceQuota.DeepCopy()
			childResourceQuotaCopy.Spec.Hard = subnamespaceCopy.Spec.Workspace.ResourceAllocation
			if _, err := c.kubeclientset.CoreV1().ResourceQuotas(childName).Update(context.TODO(), childResourceQuotaCopy, metav1.UpdateOptions{}); err != nil {
				klog.Infoln(err)
				return false
			}
		} else {
			resourceQuota := corev1.ResourceQuota{}
			resourceQuota.Name = "sub-quota"
			resourceQuota.Spec = corev1.ResourceQuotaSpec{
				Hard: subnamespaceCopy.Spec.Workspace.ResourceAllocation,
			}
			if _, err := c.kubeclientset.CoreV1().ResourceQuotas(childName).Create(context.TODO(), resourceQuota.DeepCopy(), metav1.CreateOptions{}); err != nil {
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureApplied, messageApplyFail)
				subnamespaceCopy.Status.State = failure
				subnamespaceCopy.Status.Message = failureApplied
				klog.Infoln(err)
				return false
			}
		}
	case "subtenant":
		claim := corev1alpha.ResourceTuning{
			ResourceList: subnamespaceCopy.Spec.Subtenant.ResourceAllocation,
		}
		access.ApplyTenantResourceQuota(childName, nil, claim)
	}

	c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, successApplied, messageApplied)
	return true
}

func (c *Controller) handleInheritance(subnamespaceCopy *corev1alpha.SubNamespace, childNamespace string) bool {
	// TODO: What if deleted?
	done := true
	if subnamespaceCopy.Spec.Workspace != nil {
		if roleRaw, err := c.kubeclientset.RbacV1().Roles(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Workspace.Inheritance["rbac"] {
			for _, roleRow := range roleRaw.Items {
				role := roleRow.DeepCopy()
				role.SetNamespace(childNamespace)
				role.SetUID(types.UID(uuid.New().String()))
				role.ResourceVersion = ""
				if _, err := c.kubeclientset.RbacV1().Roles(childNamespace).Create(context.TODO(), role, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
					done = false
					klog.Infoln(err)
				} else if errors.IsAlreadyExists(err) && subnamespaceCopy.Spec.Workspace.Sync {
					if existingRole, err := c.kubeclientset.RbacV1().Roles(childNamespace).Get(context.TODO(), role.GetName(), metav1.GetOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					} else {
						if !reflect.DeepEqual(role.Rules, existingRole.Rules) || !reflect.DeepEqual(role.GetLabels(), existingRole.GetLabels()) {
							existingRole.Rules = role.Rules
							existingRole.SetLabels(role.GetLabels())
							if _, err := c.kubeclientset.RbacV1().Roles(childNamespace).Update(context.TODO(), existingRole, metav1.UpdateOptions{}); err != nil {
								done = false
								klog.Infoln(err)
							}
						}
					}
				}
			}
		}
		if roleBindingRaw, err := c.kubeclientset.RbacV1().RoleBindings(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Workspace.Inheritance["rbac"] {
			for _, roleBindingRow := range roleBindingRaw.Items {
				roleBinding := roleBindingRow.DeepCopy()
				roleBinding.SetNamespace(childNamespace)
				roleBinding.SetUID(types.UID(uuid.New().String()))
				roleBinding.ResourceVersion = ""
				if _, err := c.kubeclientset.RbacV1().RoleBindings(childNamespace).Create(context.TODO(), roleBinding, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
					done = false
					klog.Infoln(err)
				} else if errors.IsAlreadyExists(err) && subnamespaceCopy.Spec.Workspace.Sync {
					if existingRoleBinding, err := c.kubeclientset.RbacV1().RoleBindings(childNamespace).Get(context.TODO(), roleBinding.GetName(), metav1.GetOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					} else {
						if !reflect.DeepEqual(roleBinding.RoleRef, existingRoleBinding.RoleRef) || !reflect.DeepEqual(roleBinding.Subjects, existingRoleBinding.Subjects) || !reflect.DeepEqual(roleBinding.GetLabels(), existingRoleBinding.GetLabels()) {
							existingRoleBinding.RoleRef = roleBinding.RoleRef
							existingRoleBinding.Subjects = roleBinding.Subjects
							existingRoleBinding.SetLabels(roleBinding.GetLabels())
							if _, err := c.kubeclientset.RbacV1().RoleBindings(childNamespace).Update(context.TODO(), existingRoleBinding, metav1.UpdateOptions{}); err != nil {
								done = false
								klog.Infoln(err)
							}
						}
					}
				}
			}
		}
		if networkPolicyRaw, err := c.kubeclientset.NetworkingV1().NetworkPolicies(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Workspace.Inheritance["networkpolicy"] {
			for _, networkPolicyRow := range networkPolicyRaw.Items {
				networkPolicy := networkPolicyRow.DeepCopy()
				networkPolicy.SetNamespace(childNamespace)
				networkPolicy.SetUID(types.UID(uuid.New().String()))
				networkPolicy.ResourceVersion = ""
				if _, err := c.kubeclientset.NetworkingV1().NetworkPolicies(childNamespace).Create(context.TODO(), networkPolicy, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
					done = false
				} else if errors.IsAlreadyExists(err) && subnamespaceCopy.Spec.Workspace.Sync {
					if existingNetworkPolicy, err := c.kubeclientset.NetworkingV1().NetworkPolicies(childNamespace).Get(context.TODO(), networkPolicy.GetName(), metav1.GetOptions{}); err != nil {
						done = false
					} else {
						if !reflect.DeepEqual(networkPolicy.Spec, existingNetworkPolicy.Spec) || !reflect.DeepEqual(networkPolicy.GetLabels(), existingNetworkPolicy.GetLabels()) {
							existingNetworkPolicy.Spec = networkPolicy.Spec
							existingNetworkPolicy.SetLabels(networkPolicy.GetLabels())
							if _, err := c.kubeclientset.NetworkingV1().NetworkPolicies(childNamespace).Update(context.TODO(), existingNetworkPolicy, metav1.UpdateOptions{}); err != nil {
								done = false
								klog.Infoln(err)
							}
						}
					}
				}
			}
		}
		if limitRangeRaw, err := c.kubeclientset.CoreV1().LimitRanges(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Workspace.Inheritance["limitrange"] {
			for _, limitRangeRow := range limitRangeRaw.Items {
				limitRange := limitRangeRow.DeepCopy()
				limitRange.SetNamespace(childNamespace)
				limitRange.SetUID(types.UID(uuid.New().String()))
				limitRange.ResourceVersion = ""
				if _, err := c.kubeclientset.CoreV1().LimitRanges(childNamespace).Create(context.TODO(), limitRange, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
					done = false
				} else if errors.IsAlreadyExists(err) && subnamespaceCopy.Spec.Workspace.Sync {
					if existingLimitRange, err := c.kubeclientset.CoreV1().LimitRanges(childNamespace).Get(context.TODO(), limitRange.GetName(), metav1.GetOptions{}); err != nil {
						done = false
					} else {
						if !reflect.DeepEqual(limitRange.Spec, existingLimitRange.Spec) || !reflect.DeepEqual(limitRange.GetLabels(), existingLimitRange.GetLabels()) {
							existingLimitRange.Spec = limitRange.Spec
							existingLimitRange.SetLabels(limitRange.GetLabels())
							if _, err := c.kubeclientset.CoreV1().LimitRanges(childNamespace).Update(context.TODO(), existingLimitRange, metav1.UpdateOptions{}); err != nil {
								done = false
								klog.Infoln(err)
							}
						}
					}
				}
			}
		}
		if secretRaw, err := c.kubeclientset.CoreV1().Secrets(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Workspace.Inheritance["secret"] {
			for _, secretRow := range secretRaw.Items {
				secret := secretRow.DeepCopy()
				secret.SetNamespace(childNamespace)
				secret.SetUID(types.UID(uuid.New().String()))
				secret.ResourceVersion = ""
				if _, err := c.kubeclientset.CoreV1().Secrets(childNamespace).Create(context.TODO(), secret, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
					done = false
				} else if errors.IsAlreadyExists(err) && subnamespaceCopy.Spec.Workspace.Sync {
					if existingSecret, err := c.kubeclientset.CoreV1().Secrets(childNamespace).Get(context.TODO(), secret.GetName(), metav1.GetOptions{}); err != nil {
						done = false
					} else {
						if !reflect.DeepEqual(secret.Type, existingSecret.Type) || !reflect.DeepEqual(secret.Data, existingSecret.Data) || !reflect.DeepEqual(secret.StringData, existingSecret.StringData) || !reflect.DeepEqual(secret.Immutable, existingSecret.Immutable) || !reflect.DeepEqual(secret.GetLabels(), existingSecret.GetLabels()) {
							existingSecret.Type = secret.Type
							existingSecret.Data = secret.Data
							existingSecret.StringData = secret.StringData
							existingSecret.Immutable = secret.Immutable
							existingSecret.SetLabels(secret.GetLabels())
							if _, err := c.kubeclientset.CoreV1().Secrets(childNamespace).Update(context.TODO(), existingSecret, metav1.UpdateOptions{}); err != nil {
								done = false
								klog.Infoln(err)
							}
						}
					}
				}
			}
		}
		if configMapRaw, err := c.kubeclientset.CoreV1().ConfigMaps(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Workspace.Inheritance["configmap"] {
			for _, configMapRow := range configMapRaw.Items {
				configMap := configMapRow.DeepCopy()
				configMap.SetNamespace(childNamespace)
				configMap.SetUID(types.UID(uuid.New().String()))
				configMap.ResourceVersion = ""
				if _, err := c.kubeclientset.CoreV1().ConfigMaps(childNamespace).Create(context.TODO(), configMap, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
					done = false
				} else if errors.IsAlreadyExists(err) && subnamespaceCopy.Spec.Workspace.Sync {
					if existingConfigMap, err := c.kubeclientset.CoreV1().ConfigMaps(childNamespace).Get(context.TODO(), configMap.GetName(), metav1.GetOptions{}); err != nil {
						done = false
					} else {
						if !reflect.DeepEqual(configMap.BinaryData, existingConfigMap.BinaryData) || !reflect.DeepEqual(configMap.Data, existingConfigMap.Data) || !reflect.DeepEqual(configMap.Immutable, existingConfigMap.Immutable) || !reflect.DeepEqual(configMap.GetLabels(), existingConfigMap.GetLabels()) {
							existingConfigMap.BinaryData = configMap.BinaryData
							existingConfigMap.Data = configMap.Data
							existingConfigMap.Immutable = configMap.Immutable
							existingConfigMap.SetLabels(configMap.GetLabels())
							if _, err := c.kubeclientset.CoreV1().ConfigMaps(childNamespace).Update(context.TODO(), existingConfigMap, metav1.UpdateOptions{}); err != nil {
								done = false
								klog.Infoln(err)
							}
						}
					}
				}
			}
		}
		if serviceAccountRaw, err := c.kubeclientset.CoreV1().ServiceAccounts(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Workspace.Inheritance["serviceaccount"] {
			for _, serviceAccountRow := range serviceAccountRaw.Items {
				serviceAccount := serviceAccountRow.DeepCopy()
				serviceAccount.SetNamespace(childNamespace)
				serviceAccount.SetUID(types.UID(uuid.New().String()))
				serviceAccount.ResourceVersion = ""
				if _, err := c.kubeclientset.CoreV1().ServiceAccounts(childNamespace).Create(context.TODO(), serviceAccount, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
					done = false
				} else if errors.IsAlreadyExists(err) && subnamespaceCopy.Spec.Workspace.Sync {
					if existingServiceAccount, err := c.kubeclientset.CoreV1().ServiceAccounts(childNamespace).Get(context.TODO(), serviceAccount.GetName(), metav1.GetOptions{}); err != nil {
						done = false
					} else {
						if !reflect.DeepEqual(serviceAccount.AutomountServiceAccountToken, existingServiceAccount.AutomountServiceAccountToken) || !reflect.DeepEqual(serviceAccount.ImagePullSecrets, existingServiceAccount.ImagePullSecrets) || !reflect.DeepEqual(serviceAccount.Secrets, existingServiceAccount.Secrets) || !reflect.DeepEqual(serviceAccount.GetLabels(), existingServiceAccount.GetLabels()) {
							existingServiceAccount.AutomountServiceAccountToken = serviceAccount.AutomountServiceAccountToken
							existingServiceAccount.ImagePullSecrets = serviceAccount.ImagePullSecrets
							existingServiceAccount.Secrets = serviceAccount.Secrets
							existingServiceAccount.SetLabels(serviceAccount.GetLabels())
							if _, err := c.kubeclientset.CoreV1().ServiceAccounts(childNamespace).Update(context.TODO(), existingServiceAccount, metav1.UpdateOptions{}); err != nil {
								done = false
								klog.Infoln(err)
							}
						}
					}
				}
			}
		}
	}
	if !done {
		c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureInheritance, messageInheritanceFail)
		subnamespaceCopy.Status.State = failure
		subnamespaceCopy.Status.Message = messageInheritanceFail
	}
	return done
}
