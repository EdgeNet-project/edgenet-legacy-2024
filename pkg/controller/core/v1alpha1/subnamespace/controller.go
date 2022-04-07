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
	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	registrationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/core/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/core/v1alpha1"
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
	applied                = "Applied"
	provisioned            = "Provisioned"
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
			subnamespace := obj.(*corev1alpha1.SubNamespace)
			if subnamespace.Spec.Expiry != nil && time.Until(subnamespace.Spec.Expiry.Time) > 0 {
				controller.enqueueSubNamespaceAfter(obj, time.Until(subnamespace.Spec.Expiry.Time))
			}
			controller.enqueueSubNamespace(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			newSubnamespace := new.(*corev1alpha1.SubNamespace)
			oldSubnamespace := old.(*corev1alpha1.SubNamespace)
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
			subnamespace := obj.(*corev1alpha1.SubNamespace)
			if subnamespace.Status.State == established {
				namespace, err := controller.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), subnamespace.GetNamespace(), metav1.GetOptions{})
				if err != nil {
					klog.Infoln(err)
					return
				}
				namespaceLabels := namespace.GetLabels()

				childNameHashed := subnamespace.GenerateChildName(namespaceLabels["edge-net.io/cluster-uid"])
				if childExists, childOwned := controller.validateChildOwnership(namespace, subnamespace.GetMode(), childNameHashed); childExists && childOwned {
					switch subnamespace.GetMode() {
					case "workspace":
						controller.kubeclientset.CoreV1().Namespaces().Delete(context.TODO(), childNameHashed, metav1.DeleteOptions{})
					case "subtenant":
						controller.edgenetclientset.CoreV1alpha1().Tenants().Delete(context.TODO(), childNameHashed, metav1.DeleteOptions{})
					}
				} else {
					return
				}
				// TODO: Return resources when there is a slice
				if parentResourceQuota, err := controller.kubeclientset.CoreV1().ResourceQuotas(subnamespace.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-quota", namespaceLabels["edge-net.io/kind"]), metav1.GetOptions{}); err == nil {
					returnedQuota := make(map[corev1.ResourceName]resource.Quantity)
					for key, value := range parentResourceQuota.Spec.Hard {
						remainingQuota := value.DeepCopy()
						remainingQuota.Add(subnamespace.RetrieveQuantity(key))
						returnedQuota[key] = remainingQuota
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
	klog.Infof("Processing object: %s", object.GetName())

	namespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), object.GetNamespace(), metav1.GetOptions{})
	if err != nil {
		return
	}

	subnamespaceRaw, err := c.subnamespacesLister.SubNamespaces(object.GetNamespace()).List(labels.Everything())
	if err == nil {
		for _, subnamespaceRow := range subnamespaceRaw {
			if subnamespaceRow.Spec.Workspace != nil && subnamespaceRow.Spec.Workspace.Sync {
				c.enqueueSubNamespaceAfter(subnamespaceRow, 30*time.Second)
			}
		}
	}

	objectLabels := object.GetLabels()
	if ownerRef := metav1.GetControllerOf(namespace); ownerRef != nil && objectLabels["edge-net.io/generated"] == "true" {
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
				if subnamespaceRow.Spec.Workspace != nil && subnamespaceRow.Spec.Workspace.Sync {
					childNameHashed := subnamespaceRow.GenerateChildName(parentnamespaceLabels["edge-net.io/cluster-uid"])
					if childExist, childOwned := c.validateChildOwnership(parentnamespace, subnamespaceRow.GetMode(), childNameHashed); childExist && childOwned {
						c.enqueueSubNamespace(subnamespaceRow)
					}
				}
			}
		}
		return
	}
}

func (c *Controller) processSubNamespace(subnamespaceCopy *corev1alpha1.SubNamespace) {
	if subnamespaceCopy.Spec.Expiry != nil && time.Until(subnamespaceCopy.Spec.Expiry.Time) <= 0 {
		c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, successExpired, messageExpired)
		c.edgenetclientset.CoreV1alpha1().SubNamespaces(subnamespaceCopy.GetNamespace()).Delete(context.TODO(), subnamespaceCopy.GetName(), metav1.DeleteOptions{})
		return
	}
	oldStatus := subnamespaceCopy.Status
	statusUpdate := func() {
		if !reflect.DeepEqual(oldStatus, subnamespaceCopy.Status) {
			if _, err := c.edgenetclientset.CoreV1alpha1().SubNamespaces(subnamespaceCopy.GetNamespace()).UpdateStatus(context.TODO(), subnamespaceCopy, metav1.UpdateOptions{}); err != nil {
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
		if tenant, err := c.edgenetclientset.CoreV1alpha1().Tenants().Get(context.TODO(), strings.ToLower(namespaceLabels["edge-net.io/tenant"]), metav1.GetOptions{}); err == nil {
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
		var childResourceQuota map[corev1.ResourceName]resource.Quantity
		annotations := namespace.GetAnnotations()

		childNameHashed := subnamespaceCopy.GenerateChildName(namespaceLabels["edge-net.io/cluster-uid"])
		childExist, childOwned := c.validateChildOwnership(namespace, subnamespaceCopy.GetMode(), childNameHashed)
		if childExist && !childOwned {
			c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureCollision, messageCollision)
			subnamespaceCopy.Status.State = failure
			subnamespaceCopy.Status.Message = messageCollision
			return
		}

		if sliceclaim := subnamespaceCopy.GetSliceClaim(); sliceclaim != nil {
			if isBound, isApplied := c.checkSliceClaim(subnamespaceCopy.GetNamespace(), *sliceclaim); (!isBound && !isApplied) || (isApplied && subnamespaceCopy.Status.State != established) {
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureSlice, messageSlice)
				subnamespaceCopy.Status.State = failure
				subnamespaceCopy.Status.Message = failureSlice
				return
			}
			annotations = map[string]string{"scheduler.alpha.kubernetes.io/node-selector": fmt.Sprintf("edge-net.io/access=private,edge-net.io/slice=%s", *sliceclaim)}
		}

		switch subnamespaceCopy.GetMode() {
		case "workspace":
			if subResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(childNameHashed).Get(context.TODO(), "sub-quota", metav1.GetOptions{}); err == nil {
				childResourceQuota = subResourceQuota.Spec.Hard
			}
			labels = map[string]string{"edge-net.io/generated": "true", "edge-net.io/kind": "sub", "edge-net.io/tenant": namespaceLabels["edge-net.io/tenant"],
				"edge-net.io/owner": subnamespaceCopy.GetName(), "edge-net.io/parent-namespace": subnamespaceCopy.GetNamespace()}
		case "subtenant":
			if subtenantResourceQuota, err := c.edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), childNameHashed, metav1.GetOptions{}); err == nil {
				assignedQuota := subtenantResourceQuota.Fetch()
				childResourceQuota = assignedQuota
			}
		}

		if parentResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(subnamespaceCopy.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-quota", namespaceLabels["edge-net.io/kind"]), metav1.GetOptions{}); err == nil {
			if sufficientQuota := c.tuneParentResourceQuota(subnamespaceCopy, parentResourceQuota, childResourceQuota); !sufficientQuota {
				return
			}
		}

		ownerReferences := namespacev1.SetAsOwnerReference(namespace)
		childInitiated := c.constructSubsidiaryNamespace(subnamespaceCopy, childNameHashed, childExist, annotations, labels, ownerReferences)
		if !childInitiated {
			return
		}

		if subnamespaceCopy.GetResourceAllocation() != nil || subnamespaceCopy.GetSliceClaim() != nil {
			quotaApplied := c.applyChildResourceQuota(subnamespaceCopy, childNameHashed)
			if !quotaApplied {
				// TODO: Error handling
				if parentResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(subnamespaceCopy.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-quota", namespaceLabels["edge-net.io/kind"]), metav1.GetOptions{}); err == nil {
					c.returnParentResourceQuota(subnamespaceCopy, parentResourceQuota)
				}
				c.tareChildResourceQuota(subnamespaceCopy, childNameHashed)
				return
			}
		}

		if subnamespaceCopy.Spec.Workspace != nil {
			done := c.handleInheritance(subnamespaceCopy, childNameHashed)
			if !done {
				return
			}
		}

		subnamespaceCopy.Status.State = established
		subnamespaceCopy.Status.Message = messageFormed
		c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, successFormed, messageFormed)
	}
}

func (c *Controller) checkSliceClaim(namespace, name string) (bool, bool) {
	if sliceClaim, err := c.edgenetclientset.CoreV1alpha1().SliceClaims(namespace).Get(context.TODO(), name, metav1.GetOptions{}); err == nil {
		if sliceClaim.Status.State == bound {
			return true, false
		} else if sliceClaim.Status.State == applied {
			return false, true
		} else {
			return false, false
		}
	}
	return false, false
}

func (c *Controller) checkSlice(name string) bool {
	if slice, err := c.edgenetclientset.CoreV1alpha1().Slices().Get(context.TODO(), name, metav1.GetOptions{}); err == nil {
		if slice.Status.State == provisioned {
			return true
		}
	} else {
		klog.Infoln(err)
	}
	return false
}

func (c *Controller) validateChildOwnership(parentNamespace *corev1.Namespace, mode, childName string) (bool, bool) {
	var checkOwnerReferences = func(ownerReferences []metav1.OwnerReference) (bool, bool) {
		for _, ownerReference := range ownerReferences {
			if ownerReference.Kind == "Namespace" && ownerReference.UID == parentNamespace.GetUID() && ownerReference.Name == parentNamespace.GetName() {
				return true, true
			}
		}
		return true, false
	}
	if mode == "workspace" {
		if childNamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childName, metav1.GetOptions{}); err == nil {
			return checkOwnerReferences(childNamespace.GetOwnerReferences())
		}
	} else {
		if subtenant, err := c.edgenetclientset.CoreV1alpha1().Tenants().Get(context.TODO(), childName, metav1.GetOptions{}); err == nil {
			return checkOwnerReferences(subtenant.GetOwnerReferences())
		}
	}
	return false, false
}

func (c *Controller) tuneParentResourceQuota(subnamespaceCopy *corev1alpha1.SubNamespace, parentResourceQuota *corev1.ResourceQuota, childResourceQuota map[corev1.ResourceName]resource.Quantity) bool {
	remainingQuota := make(map[corev1.ResourceName]resource.Quantity)
	if slice := subnamespaceCopy.GetSliceClaim(); slice == nil || slice != nil && subnamespaceCopy.GetResourceAllocation() != nil {
		for key, value := range parentResourceQuota.Spec.Hard {
			availableQuota := value.DeepCopy()
			if _, elementExists := childResourceQuota[key]; childResourceQuota != nil && elementExists {
				availableQuota.Add(childResourceQuota[key])
			}
			if availableQuota.Cmp(subnamespaceCopy.RetrieveQuantity(key)) == -1 {
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureQuotaShortage, messageQuotaShortage)
				subnamespaceCopy.Status.State = failure
				subnamespaceCopy.Status.Message = messageQuotaShortage
				return false
			} else {
				availableQuota.Sub(subnamespaceCopy.RetrieveQuantity(key))
				remainingQuota[key] = availableQuota
			}
		}
	} else {
		/*isProvisioned := false
		if subnamespaceCopy.Status.State == established {
			ticker := time.NewTicker(500 * time.Millisecond)
			done := make(chan bool)
			go func() {
			checkLoop:
				for {
					select {
					case <-done:
						return
					case <-ticker.C:
						isProvisioned = c.checkSlice(*slice)
						if isProvisioned {
							break checkLoop
						}
					}
				}
			}()
			time.Sleep(5 * time.Second)
			ticker.Stop()
			done <- true
		} else {
			isProvisioned = c.checkSlice(*slice)
		}*/

		labelSelector := fmt.Sprintf("edge-net.io/access=public,edge-net.io/pre-reservation=%s", *slice)
		/*if isProvisioned {
			labelSelector = fmt.Sprintf("edge-net.io/access=private,edge-net.io/slice=%s", *slice)
		}*/
		if nodeRaw, err := c.kubeclientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector}); err == nil {
			for key, value := range parentResourceQuota.Spec.Hard {
				availableQuota := value.DeepCopy()
				for _, nodeRow := range nodeRaw.Items {
					if _, elementExists := nodeRow.Status.Capacity[key]; elementExists {
						if availableQuota.Cmp(nodeRow.Status.Capacity[key]) == -1 {
							return false
						} else {
							availableQuota.Sub(nodeRow.Status.Capacity[key])
							remainingQuota[key] = availableQuota
						}
					}
				}
			}
		} else {
			return false
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

func (c *Controller) returnParentResourceQuota(subnamespaceCopy *corev1alpha1.SubNamespace, parentResourceQuota *corev1.ResourceQuota) bool {
	returnedQuota := make(map[corev1.ResourceName]resource.Quantity)
	for key, value := range parentResourceQuota.Spec.Hard {
		remainingQuota := value.DeepCopy()
		remainingQuota.Add(subnamespaceCopy.RetrieveQuantity(key))
		returnedQuota[key] = remainingQuota
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

func (c *Controller) tareChildResourceQuota(subnamespaceCopy *corev1alpha1.SubNamespace, childNameHashed string) bool {
	switch subnamespaceCopy.GetMode() {
	case "workspace":
		if subResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(childNameHashed).Get(context.TODO(), "sub-quota", metav1.GetOptions{}); err == nil {
			taredQuota := make(map[corev1.ResourceName]resource.Quantity)
			for key := range subResourceQuota.Spec.Hard {
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
		if subtenantResourceQuota, err := c.edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), childNameHashed, metav1.GetOptions{}); err == nil {
			taredQuota := make(map[corev1.ResourceName]resource.Quantity)
			for key := range subtenantResourceQuota.Spec.Claim["initial"].ResourceList {
				taredQuota[key] = *resource.NewQuantity(0, subtenantResourceQuota.Spec.Claim["initial"].ResourceList[key].Format)
			}
			subtenantResourceQuotaCopy := subtenantResourceQuota.DeepCopy()
			claim := corev1alpha1.ResourceTuning{
				ResourceList: taredQuota,
			}
			subtenantResourceQuotaCopy.Spec.Claim["initial"] = claim
			if _, err := c.edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Update(context.TODO(), subtenantResourceQuotaCopy, metav1.UpdateOptions{}); err != nil {
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureUpdate, messageUpdateFail)
				subnamespaceCopy.Status.State = failure
				subnamespaceCopy.Status.Message = messageUpdateFail
				return false
			}
		}
	}
	return true
}

func (c *Controller) constructSubsidiaryNamespace(subnamespaceCopy *corev1alpha1.SubNamespace, childName string, childExists bool, annotations, labels map[string]string, ownerReferences []metav1.OwnerReference) bool {
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

		objectName := "edgenet:workspace:owner"
		if subnamespaceCopy.Spec.Workspace.Owner != nil {
			if roleBinding, err := c.kubeclientset.RbacV1().RoleBindings(childName).Get(context.TODO(), objectName, metav1.GetOptions{}); err == nil {
				roleBindingCopy := roleBinding.DeepCopy()
				roleBindingCopy.Subjects = []rbacv1.Subject{{Kind: "User", Name: subnamespaceCopy.Spec.Workspace.Owner.Email, APIGroup: "rbac.authorization.k8s.io"}}
				if _, err := c.kubeclientset.RbacV1().RoleBindings(childName).Update(context.TODO(), roleBindingCopy, metav1.UpdateOptions{}); err != nil {
					c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
					subnamespaceCopy.Status.State = failure
					subnamespaceCopy.Status.Message = messageBindingFailed
					return false
				}
			} else {
				roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: "edgenet:tenant-owner"}
				rbSubjects := []rbacv1.Subject{{Kind: "User", Name: subnamespaceCopy.Spec.Workspace.Owner.Email, APIGroup: "rbac.authorization.k8s.io"}}
				roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: objectName, Namespace: childName},
					Subjects: rbSubjects, RoleRef: roleRef}
				if _, err := c.kubeclientset.RbacV1().RoleBindings(childName).Create(context.TODO(), roleBind, metav1.CreateOptions{}); err != nil {
					c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
					subnamespaceCopy.Status.State = failure
					subnamespaceCopy.Status.Message = messageBindingFailed
					return false
				}
			}
		} else {
			c.kubeclientset.RbacV1().RoleBindings(childName).Delete(context.TODO(), objectName, metav1.DeleteOptions{})
		}
	case "subtenant":
		if !childExists {
			tenantRequest := new(registrationv1alpha1.TenantRequest)
			tenantRequest.SetName(childName)
			tenantRequest.SetAnnotations(annotations)
			tenantRequest.SetLabels(labels)
			tenantRequest.SetOwnerReferences(ownerReferences)
			tenantRequest.Spec.Contact = subnamespaceCopy.Spec.Subtenant.Owner
			if err := access.CreateTenant(tenantRequest); err != nil {
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureCreation, messageCreationFail)
				subnamespaceCopy.Status.State = failure
				subnamespaceCopy.Status.Message = messageCreationFail
				return false
			}
		} else {
			if subtenant, err := c.edgenetclientset.CoreV1alpha1().Tenants().Get(context.TODO(), childName, metav1.GetOptions{}); err == nil {
				subtenantCopy := subtenant.DeepCopy()
				subtenantCopy.Spec.Contact = subnamespaceCopy.Spec.Subtenant.Owner
				// TODO: Error handling
				_, err = c.edgenetclientset.CoreV1alpha1().Tenants().Update(context.TODO(), subtenantCopy, metav1.UpdateOptions{})
				klog.Infoln(err)
			} else {
				klog.Infoln(err)
			}
		}
	}
	return true
}

func (c *Controller) applyChildResourceQuota(subnamespaceCopy *corev1alpha1.SubNamespace, childName string) bool {
	var childQuota map[corev1.ResourceName]resource.Quantity
	var slice *string

	if slice = subnamespaceCopy.GetSliceClaim(); slice != nil && subnamespaceCopy.GetResourceAllocation() == nil {
		childQuota = make(map[corev1.ResourceName]resource.Quantity)
		/*isProvisioned := false
		if subnamespaceCopy.Status.State == established {
			ticker := time.NewTicker(500 * time.Millisecond)
			done := make(chan bool)
			go func() {
			checkLoop:
				for {
					select {
					case <-done:
						return
					case <-ticker.C:
						isProvisioned = c.checkSlice(*slice)
						if isProvisioned {
							break checkLoop
						}
					}
				}
			}()
			time.Sleep(5 * time.Second)
			ticker.Stop()
			done <- true
		} else {
			isProvisioned = c.checkSlice(*slice)
		}*/
		labelSelector := fmt.Sprintf("edge-net.io/access=public,edge-net.io/pre-reservation=%s", *slice)
		/*if isProvisioned {
			labelSelector = fmt.Sprintf("edge-net.io/access=private,edge-net.io/slice=%s", *slice)
		}*/
		if nodeRaw, err := c.kubeclientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector}); err == nil {
			for _, nodeRow := range nodeRaw.Items {
				for key, capacity := range nodeRow.Status.Capacity {
					if _, elementExists := childQuota[key]; elementExists {
						resourceQuantity := childQuota[key]
						resourceQuantity.Add(capacity)
						childQuota[key] = resourceQuantity
					} else {
						childQuota[key] = capacity
					}
				}
			}
		} else {
			return false
		}
	}

	switch subnamespaceCopy.GetMode() {
	case "workspace":
		if childResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(childName).Get(context.TODO(), "sub-quota", metav1.GetOptions{}); err == nil {
			childResourceQuotaCopy := childResourceQuota.DeepCopy()
			childResourceQuotaCopy.Spec.Hard = subnamespaceCopy.GetResourceAllocation()
			if slice != nil && subnamespaceCopy.GetResourceAllocation() == nil {
				childResourceQuotaCopy.Spec.Hard = childQuota
			}
			if _, err := c.kubeclientset.CoreV1().ResourceQuotas(childName).Update(context.TODO(), childResourceQuotaCopy, metav1.UpdateOptions{}); err != nil {
				klog.Infoln(err)
				return false
			}
		} else {
			resourceQuota := corev1.ResourceQuota{}
			resourceQuota.Name = "sub-quota"
			resourceQuota.Spec = corev1.ResourceQuotaSpec{
				Hard: subnamespaceCopy.GetResourceAllocation(),
			}
			if slice != nil && subnamespaceCopy.GetResourceAllocation() == nil {
				resourceQuota.Spec = corev1.ResourceQuotaSpec{
					Hard: childQuota,
				}
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
		if subtenant, err := c.edgenetclientset.CoreV1alpha1().Tenants().Get(context.TODO(), childName, metav1.GetOptions{}); err == nil {
			claim := corev1alpha1.ResourceTuning{
				ResourceList: subnamespaceCopy.GetResourceAllocation(),
			}
			if slice != nil && subnamespaceCopy.GetResourceAllocation() == nil {
				claim = corev1alpha1.ResourceTuning{
					ResourceList: childQuota,
				}
			}

			applied := make(chan error)
			go access.ApplyTenantResourceQuota(childName, []metav1.OwnerReference{subtenant.MakeOwnerReference()}, claim, applied)
			if err := <-applied; err != nil {
				klog.Infoln(err)
				return false
			}
		}
	}

	c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, successApplied, messageApplied)
	return true
}

func (c *Controller) handleInheritance(subnamespaceCopy *corev1alpha1.SubNamespace, childNamespace string) bool {
	done := true
	if subnamespaceCopy.Spec.Workspace.Inheritance["rbac"] {
		if parentRaw, err := c.kubeclientset.RbacV1().Roles(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil {
			var childItems []rbacv1.Role
			if childRaw, err := c.kubeclientset.RbacV1().Roles(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"}); err == nil {
				childItems = childRaw.Items
			}
			inheritance := Inheritance{}
			inheritance.Child = make([]interface{}, len(childItems))
			for k, v := range childItems {
				inheritance.Child[k] = v.DeepCopy()
			}
			inheritance.Parent = make([]interface{}, len(parentRaw.Items))
			for k, v := range parentRaw.Items {
				inheritance.Parent[k] = v.DeepCopy()
			}
			createList, updateList, deleteList := inheritance.GetOperationList()
			if len(createList) > 0 {
				for _, obj := range createList {
					role := obj.(*rbacv1.Role)
					if _, err := c.kubeclientset.RbacV1().Roles(childNamespace).Create(context.TODO(), role, metav1.CreateOptions{}); err != nil {
						if !errors.IsAlreadyExists(err) {
							done = false
							klog.Infoln(err)
						} else {
							// TODO: Warning
						}
					}
				}
			}
			if len(updateList) > 0 {
				for _, obj := range updateList {
					childRole := obj.(*rbacv1.Role)
					if _, err := c.kubeclientset.RbacV1().Roles(childNamespace).Update(context.TODO(), childRole, metav1.UpdateOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					}
				}
			}
			if len(deleteList) > 0 {
				for objName := range deleteList {
					if err := c.kubeclientset.RbacV1().Roles(childNamespace).Delete(context.TODO(), objName, metav1.DeleteOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					}
				}
			}

		}
		if parentRaw, err := c.kubeclientset.RbacV1().RoleBindings(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil {
			var childItems []rbacv1.RoleBinding
			if childRaw, err := c.kubeclientset.RbacV1().RoleBindings(childNamespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"}); err == nil {
				childItems = childRaw.Items
			}
			inheritance := Inheritance{}
			inheritance.Child = make([]interface{}, len(childItems))
			for k, v := range childItems {
				inheritance.Child[k] = v.DeepCopy()
			}
			inheritance.Parent = make([]interface{}, len(parentRaw.Items))
			for k, v := range parentRaw.Items {
				inheritance.Parent[k] = v.DeepCopy()
			}
			createList, updateList, deleteList := inheritance.GetOperationList()
			if len(createList) > 0 {
				for _, obj := range createList {
					role := obj.(*rbacv1.RoleBinding)
					if _, err := c.kubeclientset.RbacV1().RoleBindings(childNamespace).Create(context.TODO(), role, metav1.CreateOptions{}); err != nil {
						if !errors.IsAlreadyExists(err) {
							done = false
							klog.Infoln(err)
						} else {
							// TODO: Warning
						}
					}
				}
			}
			if len(updateList) > 0 {
				for _, obj := range updateList {
					childRole := obj.(*rbacv1.RoleBinding)
					if _, err := c.kubeclientset.RbacV1().RoleBindings(childNamespace).Update(context.TODO(), childRole, metav1.UpdateOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					}
				}
			}
			if len(deleteList) > 0 {
				for objName := range deleteList {
					if err := c.kubeclientset.RbacV1().RoleBindings(childNamespace).Delete(context.TODO(), objName, metav1.DeleteOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					}
				}
			}
		}
	} else {
		c.kubeclientset.RbacV1().Roles(childNamespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"})
		c.kubeclientset.RbacV1().RoleBindings(childNamespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"})
	}
	if subnamespaceCopy.Spec.Workspace.Inheritance["networkpolicy"] {
		if parentRaw, err := c.kubeclientset.NetworkingV1().NetworkPolicies(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil {
			var childItems []networkingv1.NetworkPolicy
			if childRaw, err := c.kubeclientset.NetworkingV1().NetworkPolicies(childNamespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"}); err == nil {
				childItems = childRaw.Items
			}
			inheritance := Inheritance{}
			inheritance.Child = make([]interface{}, len(childItems))
			for k, v := range childItems {
				inheritance.Child[k] = v.DeepCopy()
			}
			inheritance.Parent = make([]interface{}, len(parentRaw.Items))
			for k, v := range parentRaw.Items {
				inheritance.Parent[k] = v.DeepCopy()
			}
			createList, updateList, deleteList := inheritance.GetOperationList()
			if len(createList) > 0 {
				for _, obj := range createList {
					role := obj.(*networkingv1.NetworkPolicy)
					if _, err := c.kubeclientset.NetworkingV1().NetworkPolicies(childNamespace).Create(context.TODO(), role, metav1.CreateOptions{}); err != nil {
						if !errors.IsAlreadyExists(err) {
							done = false
							klog.Infoln(err)
						} else {
							// TODO: Warning
						}
					}
				}
			}
			if len(updateList) > 0 {
				for _, obj := range updateList {
					childRole := obj.(*networkingv1.NetworkPolicy)
					if _, err := c.kubeclientset.NetworkingV1().NetworkPolicies(childNamespace).Update(context.TODO(), childRole, metav1.UpdateOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					}
				}
			}
			if len(deleteList) > 0 {
				for objName := range deleteList {
					if err := c.kubeclientset.NetworkingV1().NetworkPolicies(childNamespace).Delete(context.TODO(), objName, metav1.DeleteOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					}
				}
			}
		}
	} else {
		c.kubeclientset.NetworkingV1().NetworkPolicies(childNamespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"})
	}
	if subnamespaceCopy.Spec.Workspace.Inheritance["limitrange"] {
		if parentRaw, err := c.kubeclientset.CoreV1().LimitRanges(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil {
			var childItems []corev1.LimitRange
			if childRaw, err := c.kubeclientset.CoreV1().LimitRanges(childNamespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"}); err == nil {
				childItems = childRaw.Items
			}
			inheritance := Inheritance{}
			inheritance.Child = make([]interface{}, len(childItems))
			for k, v := range childItems {
				inheritance.Child[k] = v.DeepCopy()
			}
			inheritance.Parent = make([]interface{}, len(parentRaw.Items))
			for k, v := range parentRaw.Items {
				inheritance.Parent[k] = v.DeepCopy()
			}
			createList, updateList, deleteList := inheritance.GetOperationList()
			if len(createList) > 0 {
				for _, obj := range createList {
					role := obj.(*corev1.LimitRange)
					if _, err := c.kubeclientset.CoreV1().LimitRanges(childNamespace).Create(context.TODO(), role, metav1.CreateOptions{}); err != nil {
						if !errors.IsAlreadyExists(err) {
							done = false
							klog.Infoln(err)
						} else {
							// TODO: Warning
						}
					}
				}
			}
			if len(updateList) > 0 {
				for _, obj := range updateList {
					childRole := obj.(*corev1.LimitRange)
					if _, err := c.kubeclientset.CoreV1().LimitRanges(childNamespace).Update(context.TODO(), childRole, metav1.UpdateOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					}
				}
			}
			if len(deleteList) > 0 {
				for objName := range deleteList {
					if err := c.kubeclientset.CoreV1().LimitRanges(childNamespace).Delete(context.TODO(), objName, metav1.DeleteOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					}
				}
			}
		}
	} else {
		c.kubeclientset.CoreV1().LimitRanges(childNamespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"})
	}
	if subnamespaceCopy.Spec.Workspace.Inheritance["secret"] {
		if parentRaw, err := c.kubeclientset.CoreV1().Secrets(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil {
			var childItems []corev1.Secret
			if childRaw, err := c.kubeclientset.CoreV1().Secrets(childNamespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"}); err == nil {
				childItems = childRaw.Items
			}
			inheritance := Inheritance{}
			inheritance.Child = make([]interface{}, len(childItems))
			for k, v := range childItems {
				inheritance.Child[k] = v.DeepCopy()
			}
			inheritance.Parent = make([]interface{}, len(parentRaw.Items))
			for k, v := range parentRaw.Items {
				inheritance.Parent[k] = v.DeepCopy()
			}
			createList, updateList, deleteList := inheritance.GetOperationList()
			if len(createList) > 0 {
				for _, obj := range createList {
					role := obj.(*corev1.Secret)
					if _, err := c.kubeclientset.CoreV1().Secrets(childNamespace).Create(context.TODO(), role, metav1.CreateOptions{}); err != nil {
						if !errors.IsAlreadyExists(err) {
							done = false
							klog.Infoln(err)
						} else {
							// TODO: Warning
						}
					}
				}
			}
			if len(updateList) > 0 {
				for _, obj := range updateList {
					childRole := obj.(*corev1.Secret)
					if _, err := c.kubeclientset.CoreV1().Secrets(childNamespace).Update(context.TODO(), childRole, metav1.UpdateOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					}
				}
			}
			if len(deleteList) > 0 {
				for objName := range deleteList {
					if err := c.kubeclientset.CoreV1().Secrets(childNamespace).Delete(context.TODO(), objName, metav1.DeleteOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					}
				}
			}
		}
	} else {
		c.kubeclientset.CoreV1().Secrets(childNamespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"})
	}
	if subnamespaceCopy.Spec.Workspace.Inheritance["configmap"] {
		if parentRaw, err := c.kubeclientset.CoreV1().ConfigMaps(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil {
			var childItems []corev1.ConfigMap
			if childRaw, err := c.kubeclientset.CoreV1().ConfigMaps(childNamespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"}); err == nil {
				childItems = childRaw.Items
			}
			inheritance := Inheritance{}
			inheritance.Child = make([]interface{}, len(childItems))
			for k, v := range childItems {
				inheritance.Child[k] = v.DeepCopy()
			}
			inheritance.Parent = make([]interface{}, len(parentRaw.Items))
			for k, v := range parentRaw.Items {
				inheritance.Parent[k] = v.DeepCopy()
			}
			createList, updateList, deleteList := inheritance.GetOperationList()
			if len(createList) > 0 {
				for _, obj := range createList {
					role := obj.(*corev1.ConfigMap)
					if _, err := c.kubeclientset.CoreV1().ConfigMaps(childNamespace).Create(context.TODO(), role, metav1.CreateOptions{}); err != nil {
						if !errors.IsAlreadyExists(err) {
							done = false
							klog.Infoln(err)
						} else {
							// TODO: Warning
						}
					}
				}
			}
			if len(updateList) > 0 {
				for _, obj := range updateList {
					childRole := obj.(*corev1.ConfigMap)
					if _, err := c.kubeclientset.CoreV1().ConfigMaps(childNamespace).Update(context.TODO(), childRole, metav1.UpdateOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					}
				}
			}
			if len(deleteList) > 0 {
				for objName := range deleteList {
					if err := c.kubeclientset.CoreV1().ConfigMaps(childNamespace).Delete(context.TODO(), objName, metav1.DeleteOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					}
				}
			}
		}
	} else {
		c.kubeclientset.CoreV1().ConfigMaps(childNamespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"})
	}
	if subnamespaceCopy.Spec.Workspace.Inheritance["serviceaccount"] {
		if parentRaw, err := c.kubeclientset.CoreV1().ServiceAccounts(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil {
			var childItems []corev1.ServiceAccount
			if childRaw, err := c.kubeclientset.CoreV1().ServiceAccounts(childNamespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"}); err == nil {
				childItems = childRaw.Items
			}
			inheritance := Inheritance{}
			inheritance.Child = make([]interface{}, len(childItems))
			for k, v := range childItems {
				inheritance.Child[k] = v.DeepCopy()
			}
			inheritance.Parent = make([]interface{}, len(parentRaw.Items))
			for k, v := range parentRaw.Items {
				inheritance.Parent[k] = v.DeepCopy()
			}
			createList, updateList, deleteList := inheritance.GetOperationList()
			if len(createList) > 0 {
				for _, obj := range createList {
					role := obj.(*corev1.ServiceAccount)
					if _, err := c.kubeclientset.CoreV1().ServiceAccounts(childNamespace).Create(context.TODO(), role, metav1.CreateOptions{}); err != nil {
						if !errors.IsAlreadyExists(err) {
							done = false
							klog.Infoln(err)
						} else {
							// TODO: Warning
						}
					}
				}
			}
			if len(updateList) > 0 {
				for _, obj := range updateList {
					childRole := obj.(*corev1.ServiceAccount)
					if _, err := c.kubeclientset.CoreV1().ServiceAccounts(childNamespace).Update(context.TODO(), childRole, metav1.UpdateOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					}
				}
			}
			if len(deleteList) > 0 {
				for objName := range deleteList {
					if err := c.kubeclientset.CoreV1().ServiceAccounts(childNamespace).Delete(context.TODO(), objName, metav1.DeleteOptions{}); err != nil {
						done = false
						klog.Infoln(err)
					}
				}
			}
		}
	} else {
		c.kubeclientset.CoreV1().ServiceAccounts(childNamespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"})
	}

	if !done {
		c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureInheritance, messageInheritanceFail)
		subnamespaceCopy.Status.State = failure
		subnamespaceCopy.Status.Message = messageInheritanceFail
	}
	return done
}

type Inheritance struct {
	Child          []interface{}
	Parent         []interface{}
	ChildNamespace string
}

func (i Inheritance) GetOperationList() ([]interface{}, []interface{}, map[string]interface{}) {
	var createList []interface{}
	var updateList []interface{}
	comparisonSlice := make(map[string]interface{})
	for _, childObj := range i.Child {
		comparisonSlice[childObj.(metav1.Object).GetName()] = childObj
	}
	for _, parentObj := range i.Parent {
		if _, ok := comparisonSlice[parentObj.(metav1.Object).GetName()]; ok {
			childObj := i.prepareForUpdate(comparisonSlice[parentObj.(metav1.Object).GetName()], parentObj)
			if childObj != nil {
				updateList = append(updateList, childObj)
			}
			delete(comparisonSlice, parentObj.(metav1.Object).GetName())
		} else {
			childObj := i.prepareForCreate(parentObj)
			createList = append(createList, childObj)
		}
	}
	return createList, updateList, comparisonSlice
}

func (i Inheritance) prepareForCreate(obj interface{}) interface{} {
	obj.(metav1.Object).SetNamespace(i.ChildNamespace)
	obj.(metav1.Object).SetUID(types.UID(uuid.New().String()))
	obj.(metav1.Object).SetResourceVersion("")
	obj.(metav1.Object).SetLabels(map[string]string{"edge-net.io/generated": "true"})
	return obj
}

func (i Inheritance) prepareForUpdate(childObj, parentObj interface{}) interface{} {
	var childForUpdate interface{}
	switch parentObjectForUpdate := parentObj.(type) {
	case *rbacv1.Role:
		childRoleCopy := childObj.(*rbacv1.Role)
		parentLabels := parentObjectForUpdate.GetLabels()
		if parentLabels == nil {
			parentLabels = make(map[string]string)
		}
		parentLabels["edge-net.io/generated"] = "true"
		if !reflect.DeepEqual(childRoleCopy.Rules, parentObjectForUpdate.Rules) || !reflect.DeepEqual(childRoleCopy.GetLabels(), parentObjectForUpdate.GetLabels()) {
			childRoleCopy.Rules = parentObjectForUpdate.Rules
			childRoleCopy.SetLabels(parentLabels)
			childForUpdate = childRoleCopy
		}
	case *rbacv1.RoleBinding:
		childRoleBindingCopy := childObj.(*rbacv1.RoleBinding)
		parentLabels := parentObjectForUpdate.GetLabels()
		if parentLabels == nil {
			parentLabels = make(map[string]string)
		}
		parentLabels["edge-net.io/generated"] = "true"
		if !reflect.DeepEqual(childRoleBindingCopy.RoleRef, parentObjectForUpdate.RoleRef) || !reflect.DeepEqual(childRoleBindingCopy.Subjects, parentObjectForUpdate.Subjects) || !reflect.DeepEqual(childRoleBindingCopy.GetLabels(), parentObjectForUpdate.GetLabels()) {
			childRoleBindingCopy.RoleRef = parentObjectForUpdate.RoleRef
			childRoleBindingCopy.Subjects = parentObjectForUpdate.Subjects
			childRoleBindingCopy.SetLabels(parentLabels)
			childForUpdate = childRoleBindingCopy
		}
	case *networkingv1.NetworkPolicy:
		childNetworkPolicyCopy := childObj.(*networkingv1.NetworkPolicy)
		parentLabels := parentObjectForUpdate.GetLabels()
		if parentLabels == nil {
			parentLabels = make(map[string]string)
		}
		parentLabels["edge-net.io/generated"] = "true"
		if !reflect.DeepEqual(childNetworkPolicyCopy.Spec, parentObjectForUpdate.Spec) || !reflect.DeepEqual(childNetworkPolicyCopy.GetLabels(), parentObjectForUpdate.GetLabels()) {
			childNetworkPolicyCopy.Spec = parentObjectForUpdate.Spec
			childNetworkPolicyCopy.SetLabels(parentLabels)
			childForUpdate = childNetworkPolicyCopy
		}
	case *corev1.LimitRange:
		childLimitRangeCopy := childObj.(*corev1.LimitRange)
		parentLabels := parentObjectForUpdate.GetLabels()
		if parentLabels == nil {
			parentLabels = make(map[string]string)
		}
		parentLabels["edge-net.io/generated"] = "true"
		if !reflect.DeepEqual(childLimitRangeCopy.Spec, parentObjectForUpdate.Spec) || !reflect.DeepEqual(childLimitRangeCopy.GetLabels(), parentObjectForUpdate.GetLabels()) {
			childLimitRangeCopy.Spec = parentObjectForUpdate.Spec
			childLimitRangeCopy.SetLabels(parentLabels)
			childForUpdate = childLimitRangeCopy
		}
	case *corev1.Secret:
		childSecretCopy := childObj.(*corev1.Secret)
		parentLabels := parentObjectForUpdate.GetLabels()
		if parentLabels == nil {
			parentLabels = make(map[string]string)
		}
		parentLabels["edge-net.io/generated"] = "true"
		if !reflect.DeepEqual(childSecretCopy.Type, parentObjectForUpdate.Type) || !reflect.DeepEqual(childSecretCopy.Data, parentObjectForUpdate.Data) ||
			!reflect.DeepEqual(childSecretCopy.StringData, parentObjectForUpdate.StringData) || !reflect.DeepEqual(childSecretCopy.Immutable, parentObjectForUpdate.Immutable) ||
			!reflect.DeepEqual(childSecretCopy.GetLabels(), parentObjectForUpdate.GetLabels()) {
			childSecretCopy.Type = parentObjectForUpdate.Type
			childSecretCopy.Data = parentObjectForUpdate.Data
			childSecretCopy.StringData = parentObjectForUpdate.StringData
			childSecretCopy.Immutable = parentObjectForUpdate.Immutable
			childSecretCopy.SetLabels(parentLabels)
			childForUpdate = childSecretCopy
		}
	case corev1.ConfigMap:
		childConfigMapCopy := childObj.(*corev1.ConfigMap)
		parentLabels := parentObjectForUpdate.GetLabels()
		if parentLabels == nil {
			parentLabels = make(map[string]string)
		}
		parentLabels["edge-net.io/generated"] = "true"
		if !reflect.DeepEqual(childConfigMapCopy.BinaryData, parentObjectForUpdate.BinaryData) || !reflect.DeepEqual(childConfigMapCopy.Data, parentObjectForUpdate.Data) ||
			!reflect.DeepEqual(childConfigMapCopy.Immutable, parentObjectForUpdate.Immutable) || !reflect.DeepEqual(childConfigMapCopy.GetLabels(), parentObjectForUpdate.GetLabels()) {
			childConfigMapCopy.BinaryData = parentObjectForUpdate.BinaryData
			childConfigMapCopy.Data = parentObjectForUpdate.Data
			childConfigMapCopy.Immutable = parentObjectForUpdate.Immutable
			childConfigMapCopy.SetLabels(parentLabels)
			childForUpdate = childConfigMapCopy
		}
	case *corev1.ServiceAccount:
		childServiceAccountCopy := childObj.(*corev1.ServiceAccount)
		parentLabels := parentObjectForUpdate.GetLabels()
		if parentLabels == nil {
			parentLabels = make(map[string]string)
		}
		parentLabels["edge-net.io/generated"] = "true"
		if !reflect.DeepEqual(childServiceAccountCopy.AutomountServiceAccountToken, parentObjectForUpdate.AutomountServiceAccountToken) || !reflect.DeepEqual(childServiceAccountCopy.ImagePullSecrets, parentObjectForUpdate.ImagePullSecrets) ||
			!reflect.DeepEqual(childServiceAccountCopy.Secrets, parentObjectForUpdate.Secrets) || !reflect.DeepEqual(childServiceAccountCopy.GetLabels(), parentObjectForUpdate.GetLabels()) {
			childServiceAccountCopy.AutomountServiceAccountToken = parentObjectForUpdate.AutomountServiceAccountToken
			childServiceAccountCopy.ImagePullSecrets = parentObjectForUpdate.ImagePullSecrets
			childServiceAccountCopy.Secrets = parentObjectForUpdate.Secrets
			childServiceAccountCopy.SetLabels(parentLabels)
			childForUpdate = childServiceAccountCopy
		}
	}
	return childForUpdate
}
