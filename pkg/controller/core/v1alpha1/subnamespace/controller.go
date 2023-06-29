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
	"reflect"
	"strings"
	"time"

	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	registrationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/core/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/core/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/multitenancy"

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
	backoffLimit = 3

	successSynced        = "Synced"
	successExpired       = "Expired"
	successSlice         = "Slice Ready"
	failureQuotaShortage = "Shortage"
	failureUpdate        = "Not Updated"
	failureApplied       = "Not Applied"
	failureCreation      = "Not Created"
	failureInheritance   = "Not Inherited"
	failureBinding       = "Binding Failed"
	failureCollision     = "Name Collision"
	failureSlice         = "Slice Unready"

	messageResourceSynced      = "Subsidiary namespace synced successfully"
	messageEstablished         = "Subsidiary namespace established"
	messageExpired             = "Subsidiary namespace deleted"
	messageQuotaCheck          = "The parent has sufficient quota"
	messageApplyFail           = "Child quota cannot be applied"
	messageCreation            = "Subsidiary namespace created"
	messageCreationFail        = "Subsidiary namespace cannot be created"
	messageNSUpdateFail        = "Subsidiary namespace cannot be updated"
	messageInheritanceFail     = "Inheritance from parent to child failed"
	messageCollision           = "Name is not available. Please choose another one."
	messageSubnamespaceDeleted = "Last created child subnamespace has been deleted due to insufficient quota "
	messageParentQuotaShortage = "Insufficient quota at the parent"
	messageUpdateFail          = "Quota cannot be updated"
	messageSliceFailure        = "Slice is not ready to be used"
	messageSliceReady          = "Slice is ready"
	messageBindingFailed       = "Role binding failed"
	messagePartitioned         = "Parent resource quota has been partitioned among its children and itself"
	messageApplied             = "Child quota applied successfully"
	messageReconciliation      = "Reconciliation in progress"
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

	multitenancyManager *multitenancy.Manager

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

	multitenancyManager := multitenancy.NewManager(kubeclientset, edgenetclientset)

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
		multitenancyManager:   multitenancyManager,
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
			controller.enqueueSubNamespace(new)
			if (oldSubnamespace.Spec.Expiry == nil && newSubnamespace.Spec.Expiry != nil) ||
				(oldSubnamespace.Spec.Expiry != nil && newSubnamespace.Spec.Expiry != nil && !oldSubnamespace.Spec.Expiry.Time.Equal(newSubnamespace.Spec.Expiry.Time) && time.Until(newSubnamespace.Spec.Expiry.Time) > 0) {
				controller.enqueueSubNamespaceAfter(new, time.Until(newSubnamespace.Spec.Expiry.Time))
			}
		}, DeleteFunc: func(obj interface{}) {
			subnamespace := obj.(*corev1alpha1.SubNamespace)
			controller.cleanup(subnamespace)
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
	if exceedsBackoffLimit := subnamespaceCopy.Status.Failed >= backoffLimit; exceedsBackoffLimit {
		c.cleanup(subnamespaceCopy)
		return
	}

	permitted, parentNamespace, parentNamespaceLabels := c.multitenancyManager.EligibilityCheck(subnamespaceCopy.GetNamespace())
	if permitted {
		var childNameHashed string
		if subnamespaceCopy.Status.Child != nil {
			childNameHashed = *subnamespaceCopy.Status.Child
		} else {
			childNameHashed = subnamespaceCopy.GenerateChildName(parentNamespaceLabels["edge-net.io/cluster-uid"])
			if hasConflict := c.checkNamespaceCollision(subnamespaceCopy, parentNamespace, childNameHashed); hasConflict {
				return
			}
		}

		switch subnamespaceCopy.Status.State {
		case corev1alpha1.StatusEstablished:
			if sliceclaimName := subnamespaceCopy.GetSliceClaim(); sliceclaimName != nil {
				if sliceclaimCopy, ok := c.checkSliceClaim(subnamespaceCopy.GetNamespace(), *sliceclaimName); sliceclaimCopy == nil || !ok {
					c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureSlice, messageSliceFailure)
					subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
					subnamespaceCopy.Status.Message = failureSlice
					c.updateStatus(context.TODO(), subnamespaceCopy)
					return
				}
			}
			c.reconcile(subnamespaceCopy, parentNamespace, childNameHashed)
		case corev1alpha1.StatusQuotaSet:
			if subnamespaceCopy.Spec.Workspace != nil {
				if isInherited := c.handleInheritance(subnamespaceCopy, childNameHashed); !isInherited {
					return
				}
			}
			c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, corev1alpha1.StatusEstablished, messageEstablished)
			subnamespaceCopy.Status.State = corev1alpha1.StatusEstablished
			subnamespaceCopy.Status.Message = messageEstablished
			c.updateStatus(context.TODO(), subnamespaceCopy)
		case corev1alpha1.StatusSubnamespaceCreated:
			if subnamespaceCopy.GetResourceAllocation() != nil {
				remainingQuotaResourceList, isQuotaSufficient, isReconciled := c.reconcileWithChildQuota(subnamespaceCopy, childNameHashed)
				if !isReconciled {
					if !isQuotaSufficient {
						return
					}
					switch subnamespaceCopy.GetMode() {
					case "workspace":
						resourceQuota := corev1.ResourceQuota{}
						resourceQuota.SetName("sub-quota")
						resourceQuota.Spec = corev1.ResourceQuotaSpec{
							Hard: remainingQuotaResourceList,
						}
						if _, err := c.kubeclientset.CoreV1().ResourceQuotas(childNameHashed).Create(context.TODO(), resourceQuota.DeepCopy(), metav1.CreateOptions{}); err != nil {
							if errors.IsAlreadyExists(err) {
								remainingChildResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(childNameHashed).Get(context.TODO(), resourceQuota.GetName(), metav1.GetOptions{})
								if err != nil {
									c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureApplied, messageApplyFail)
									subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
									subnamespaceCopy.Status.Message = failureApplied
									c.updateStatus(context.TODO(), subnamespaceCopy)
									return
								}
								remainingChildResourceQuota.Spec.Hard = remainingQuotaResourceList
								if _, err := c.kubeclientset.CoreV1().ResourceQuotas(childNameHashed).Update(context.TODO(), remainingChildResourceQuota, metav1.UpdateOptions{}); err != nil {
									c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureApplied, messageApplyFail)
									subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
									subnamespaceCopy.Status.Message = failureApplied
									c.updateStatus(context.TODO(), subnamespaceCopy)
									klog.Infoln(err)
									return
								}
							} else {
								c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureApplied, messageApplyFail)
								subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
								subnamespaceCopy.Status.Message = failureApplied
								c.updateStatus(context.TODO(), subnamespaceCopy)
								klog.Infoln(err)
								return
							}
						}
					case "subtenant":
						if subtenant, err := c.edgenetclientset.CoreV1alpha1().Tenants().Get(context.TODO(), childNameHashed, metav1.GetOptions{}); err == nil {
							claim := corev1alpha1.ResourceTuning{
								ResourceList: remainingQuotaResourceList,
							}
							applied := make(chan error)
							go c.multitenancyManager.ApplyTenantResourceQuota(childNameHashed, []metav1.OwnerReference{subtenant.MakeOwnerReference()}, claim, applied)
							if err := <-applied; err != nil {
								c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureApplied, messageApplyFail)
								subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
								subnamespaceCopy.Status.Message = failureApplied
								c.updateStatus(context.TODO(), subnamespaceCopy)
								klog.Infoln(err)
								return
							}
						} else {
							c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureApplied, messageApplyFail)
							subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
							subnamespaceCopy.Status.Message = failureApplied
							c.updateStatus(context.TODO(), subnamespaceCopy)
							klog.Infoln(err)
							return
						}
					}
				}
			}
			c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, corev1alpha1.StatusQuotaSet, messageApplied)
			subnamespaceCopy.Status.State = corev1alpha1.StatusQuotaSet
			subnamespaceCopy.Status.Message = messageApplied
			c.updateStatus(context.TODO(), subnamespaceCopy)
		case corev1alpha1.StatusPartitioned:
			ownerReferences := []metav1.OwnerReference{multitenancy.MakeOwnerReferenceForNamespace(parentNamespace)}
			if isCreated := c.makeSubsidiaryNamespace(subnamespaceCopy, parentNamespaceLabels["edge-net.io/tenant"], childNameHashed, parentNamespace.GetAnnotations(), ownerReferences); !isCreated {
				return
			}
			c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, corev1alpha1.StatusPartitioned, messageCreation)
			subnamespaceCopy.Status.Child = &childNameHashed
			subnamespaceCopy.Status.State = corev1alpha1.StatusSubnamespaceCreated
			subnamespaceCopy.Status.Message = messageCreation
			c.updateStatus(context.TODO(), subnamespaceCopy)
		default:
			if sliceclaimName := subnamespaceCopy.GetSliceClaim(); sliceclaimName != nil {
				sliceclaimCopy, ok := c.checkSliceClaim(subnamespaceCopy.GetNamespace(), *sliceclaimName)
				if !ok {
					c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureSlice, messageSliceFailure)
					subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
					subnamespaceCopy.Status.Message = failureSlice
					c.updateStatus(context.TODO(), subnamespaceCopy)
					return
				}
				if subnamespaceCopy.GetResourceAllocation() == nil {
					childQuota := make(map[corev1.ResourceName]resource.Quantity)
					labelSelector := fmt.Sprintf("edge-net.io/access=public,edge-net.io/pre-reservation=%s", *sliceclaimName)
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
						subnamespaceCopy.SetResourceAllocation(childQuota)
						if _, err := c.edgenetclientset.CoreV1alpha1().SubNamespaces(subnamespaceCopy.GetNamespace()).Update(context.TODO(), subnamespaceCopy, metav1.UpdateOptions{}); err == nil {
							c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, successSlice, messageSliceReady)
							return
						}
					}
					c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureSlice, messageSliceFailure)
					subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
					subnamespaceCopy.Status.Message = failureSlice
					c.updateStatus(context.TODO(), subnamespaceCopy)
					return
				}
				sliceclaimOwnerReferences := sliceclaimCopy.GetOwnerReferences()
				subnamespaceControllerRef := subnamespaceCopy.MakeOwnerReference()
				takeControl := false
				subnamespaceControllerRef.Controller = &takeControl
				sliceclaimOwnerReferences = append(sliceclaimOwnerReferences, subnamespaceControllerRef)
				sliceclaimCopy.SetOwnerReferences(sliceclaimOwnerReferences)
				if _, err := c.edgenetclientset.CoreV1alpha1().SliceClaims(subnamespaceCopy.GetNamespace()).Update(context.TODO(), sliceclaimCopy, metav1.UpdateOptions{}); err != nil {
					klog.Infoln(err)
					c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureSlice, messageSliceFailure)
					subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
					subnamespaceCopy.Status.Message = failureSlice
					c.updateStatus(context.TODO(), subnamespaceCopy)
					return
				}
			}
			if isPartitioned := c.partitionParentQuota(subnamespaceCopy, parentNamespace); !isPartitioned {
				return
			}
			c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, corev1alpha1.StatusPartitioned, messagePartitioned)
			subnamespaceCopy.Status.State = corev1alpha1.StatusPartitioned
			subnamespaceCopy.Status.Message = messagePartitioned
			c.updateStatus(context.TODO(), subnamespaceCopy)
		}
	}
}

func (c *Controller) reconcile(subnamespaceCopy *corev1alpha1.SubNamespace, parentNamespace *corev1.Namespace, childNameHashed string) {
	if subnamespaceCopy.GetResourceAllocation() != nil {
		if _, isQuotaSufficient, isReconciled := c.reconcileWithChildQuota(subnamespaceCopy, childNameHashed); !isReconciled || !isQuotaSufficient {
			subnamespaceCopy.Status.State = corev1alpha1.StatusSubnamespaceCreated
			subnamespaceCopy.Status.Message = messageReconciliation
		}
	}
	if isReconciled := c.reconcileWithOwnerPermissions(subnamespaceCopy, childNameHashed); !isReconciled {
		subnamespaceCopy.Status.State = corev1alpha1.StatusPartitioned
		subnamespaceCopy.Status.Message = messageReconciliation
	}
	if _, isReconciled := c.reconcileWithParentQuota(subnamespaceCopy, parentNamespace); !isReconciled {
		subnamespaceCopy.Status.State = corev1alpha1.StatusReconciliation
		subnamespaceCopy.Status.Message = messageReconciliation
	}
	if subnamespaceCopy.Status.State != corev1alpha1.StatusEstablished {
		c.updateStatus(context.TODO(), subnamespaceCopy)
		return
	}
	if subnamespaceCopy.Spec.Workspace != nil && subnamespaceCopy.Spec.Workspace.Sync {
		klog.Infoln("SYNCING")
		c.handleInheritance(subnamespaceCopy, childNameHashed)
	}
}

func (c *Controller) reconcileWithChildQuota(subnamespaceCopy *corev1alpha1.SubNamespace, childNameHashed string) (map[corev1.ResourceName]resource.Quantity, bool, bool) {
	remainingQuotaResourceList, lastInSubnamespace, isQuotaSufficient := c.subtractSubnamespaceQuotas(subnamespaceCopy, childNameHashed, subnamespaceCopy.GetResourceAllocation())
	if !isQuotaSufficient {
		c.edgenetclientset.CoreV1alpha1().SubNamespaces(childNameHashed).Delete(context.TODO(), lastInSubnamespace, metav1.DeleteOptions{})
		c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageSubnamespaceDeleted)
		c.enqueueSubNamespaceAfter(subnamespaceCopy, time.Minute)
		return nil, false, false
	}

	var childQuotaResourceList = make(map[corev1.ResourceName]resource.Quantity)
	switch subnamespaceCopy.GetMode() {
	case "subtenant":
		currentChildResourceQuota, err := c.edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), childNameHashed, metav1.GetOptions{})
		if err != nil {
			return remainingQuotaResourceList, true, false
		}
		childQuotaResourceList = currentChildResourceQuota.Fetch()
	default:
		currentChildResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(childNameHashed).Get(context.TODO(), "sub-quota", metav1.GetOptions{})
		if err != nil {
			return remainingQuotaResourceList, true, false
		}
		childQuotaResourceList = currentChildResourceQuota.Spec.Hard
	}

	if len(remainingQuotaResourceList) != len(childQuotaResourceList) {
		return remainingQuotaResourceList, true, false
	}
	for resourceName, remainingQuantity := range remainingQuotaResourceList {
		if childQuantity, elementExists := childQuotaResourceList[resourceName]; elementExists {
			if !remainingQuantity.Equal(childQuantity) {
				return remainingQuotaResourceList, true, false
			}
		} else {
			return remainingQuotaResourceList, true, false
		}
	}
	return nil, true, true
}

func (c *Controller) reconcileWithOwnerPermissions(subnamespaceCopy *corev1alpha1.SubNamespace, childNameHashed string) bool {
	switch subnamespaceCopy.GetMode() {
	case "subtenant":
		subtenant, err := c.edgenetclientset.CoreV1alpha1().Tenants().Get(context.TODO(), childNameHashed, metav1.GetOptions{})
		if err != nil {
			return false
		}
		if subtenant.Spec.Contact == subnamespaceCopy.Spec.Subtenant.Owner {
			return true
		}
	default:
		if subnamespaceCopy.Spec.Workspace.Owner != nil {
			roleBinding, err := c.kubeclientset.RbacV1().RoleBindings(childNameHashed).Get(context.TODO(), "edgenet:workspace:owner", metav1.GetOptions{})
			if err != nil {
				return false
			}
			if roleBinding.RoleRef.Kind == "ClusterRole" && roleBinding.RoleRef.Name == corev1alpha1.TenantOwnerClusterRoleName {
				for _, subject := range roleBinding.Subjects {
					if subject.Kind == "User" && subject.Name == subnamespaceCopy.Spec.Workspace.Owner.Email {
						return true
					}
				}
			}
		} else {
			return true
		}
	}
	return false
}

func (c *Controller) reconcileWithParentQuota(subnamespaceCopy *corev1alpha1.SubNamespace, parentNamespace *corev1.Namespace) (*corev1.ResourceQuota, bool) {
	parentNamespaceLabels := parentNamespace.GetLabels()
	currentParentResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(parentNamespace.GetName()).Get(context.TODO(), fmt.Sprintf("%s-quota", parentNamespaceLabels["edge-net.io/kind"]), metav1.GetOptions{})
	if err != nil {
		return nil, true
	}
	if subnamespaceCopy.GetResourceAllocation() == nil {
		return nil, false
	}
	var parentQuotaResourceList = make(corev1.ResourceList)
	if strings.ToLower(parentNamespaceLabels["edge-net.io/kind"]) == "core" {
		if parentResourceQuota, err := c.edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), parentNamespace.GetName(), metav1.GetOptions{}); err == nil {
			parentQuotaResourceList = parentResourceQuota.Fetch()
		}
	} else {
		if parentNamespaceOwner, err := c.edgenetclientset.CoreV1alpha1().SubNamespaces(parentNamespaceLabels["edge-net.io/parent-namespace"]).Get(context.TODO(), parentNamespaceLabels["edge-net.io/owner"], metav1.GetOptions{}); err == nil {
			parentQuotaResourceList = parentNamespaceOwner.GetResourceAllocation()
		}
	}
	remainingQuotaResourceList, _, isQuotaSufficient := c.subtractSubnamespaceQuotas(subnamespaceCopy, parentNamespace.GetName(), parentQuotaResourceList)
	if !isQuotaSufficient {
		return nil, false
	}

	if len(remainingQuotaResourceList) != len(currentParentResourceQuota.Spec.Hard) {
		currentParentResourceQuota.Spec.Hard = remainingQuotaResourceList
		return currentParentResourceQuota, false
	}
	for resourceName, remainingQuantity := range remainingQuotaResourceList {
		if childQuantity, elementExists := currentParentResourceQuota.Spec.Hard[resourceName]; elementExists {
			if !remainingQuantity.Equal(childQuantity) {
				currentParentResourceQuota.Spec.Hard = remainingQuotaResourceList
				return currentParentResourceQuota, false
			}
		} else {
			currentParentResourceQuota.Spec.Hard = remainingQuotaResourceList
			return currentParentResourceQuota, false
		}
	}
	return nil, true
}

func (c *Controller) partitionParentQuota(subnamespaceCopy *corev1alpha1.SubNamespace, parentNamespace *corev1.Namespace) bool {
	if currentParentResourceQuota, isReconciled := c.reconcileWithParentQuota(subnamespaceCopy, parentNamespace); !isReconciled {
		if currentParentResourceQuota != nil {
			if _, err := c.kubeclientset.CoreV1().ResourceQuotas(parentNamespace.GetName()).Update(context.TODO(), currentParentResourceQuota, metav1.UpdateOptions{}); err != nil {
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureSlice, messageUpdateFail)
				subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
				subnamespaceCopy.Status.Message = messageUpdateFail
				c.updateStatus(context.TODO(), subnamespaceCopy)
				return false
			}
		} else {
			c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureQuotaShortage, messageParentQuotaShortage)
			subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
			subnamespaceCopy.Status.Message = messageParentQuotaShortage
			c.updateStatus(context.TODO(), subnamespaceCopy)
			return false
		}
	}
	return true
}

func (c *Controller) subtractSubnamespaceQuotas(subnamespaceCopy *corev1alpha1.SubNamespace, namespace string, remainingQuotaResourceList map[corev1.ResourceName]resource.Quantity) (map[corev1.ResourceName]resource.Quantity, string, bool) {
	var lastInDate metav1.Time
	var lastInSubnamespace string
	if subnamespaceRaw, err := c.edgenetclientset.CoreV1alpha1().SubNamespaces(namespace).List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, subnamespaceRow := range subnamespaceRaw.Items {
			if (subnamespaceRow.GetUID() == subnamespaceCopy.GetUID() && !(subnamespaceCopy.Status.Failed >= backoffLimit && subnamespaceCopy.Status.State == corev1alpha1.StatusFailed)) ||
				subnamespaceRow.Status.State == corev1alpha1.StatusEstablished || subnamespaceRow.Status.State == corev1alpha1.StatusQuotaSet || subnamespaceRow.Status.State == corev1alpha1.StatusSubnamespaceCreated || subnamespaceRow.Status.State == corev1alpha1.StatusPartitioned {
				if lastInDate.IsZero() || subnamespaceRow.GetCreationTimestamp().After(lastInDate.Time) {
					lastInSubnamespace = subnamespaceRow.GetName()
					lastInDate = subnamespaceRow.GetCreationTimestamp()
				}
				for remainingQuotaResource, remainingQuotaQuantity := range remainingQuotaResourceList {
					childQuota := subnamespaceRow.RetrieveQuantity(remainingQuotaResource)
					if subnamespaceRow.GetUID() == subnamespaceCopy.GetUID() {
						childQuota = subnamespaceCopy.RetrieveQuantity(remainingQuotaResource)
					}
					if remainingQuotaQuantity.Cmp(childQuota) == -1 {
						return remainingQuotaResourceList, lastInSubnamespace, false
					}
					remainingQuotaQuantity.Sub(childQuota)
					remainingQuotaResourceList[remainingQuotaResource] = remainingQuotaQuantity
				}
			}
		}
	}
	return remainingQuotaResourceList, lastInSubnamespace, true
}

func (c *Controller) checkSliceClaim(namespace, name string) (*corev1alpha1.SliceClaim, bool) {
	if sliceclaimCopy, err := c.edgenetclientset.CoreV1alpha1().SliceClaims(namespace).Get(context.TODO(), name, metav1.GetOptions{}); err == nil {
		if sliceclaimCopy.Status.State == corev1alpha1.StatusBound || sliceclaimCopy.Status.State == corev1alpha1.StatusEmployed {
			return sliceclaimCopy, true
		}
		return sliceclaimCopy, false
	}
	return nil, false
}

func (c *Controller) checkNamespaceCollision(subnamespaceCopy *corev1alpha1.SubNamespace, parentNamespace *corev1.Namespace, childNameHashed string) bool {
	var checkOwnerReferences = func(ownerReferences []metav1.OwnerReference) bool {
		for _, ownerReference := range ownerReferences {
			if ownerReference.Kind == "Namespace" && ownerReference.UID == parentNamespace.GetUID() && ownerReference.Name == parentNamespace.GetName() {
				return false
			}
		}
		c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureCollision, messageCollision)
		subnamespaceCopy.Status.Failed = backoffLimit - 1
		subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
		subnamespaceCopy.Status.Message = messageCollision
		c.updateStatus(context.TODO(), subnamespaceCopy)
		return true
	}
	if subnamespaceCopy.GetMode() == "workspace" {
		if childNamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childNameHashed, metav1.GetOptions{}); err == nil {
			return checkOwnerReferences(childNamespace.GetOwnerReferences())
		}
	} else {
		if subtenant, err := c.edgenetclientset.CoreV1alpha1().Tenants().Get(context.TODO(), childNameHashed, metav1.GetOptions{}); err == nil {
			return checkOwnerReferences(subtenant.GetOwnerReferences())
		}
	}
	return false
}

func (c *Controller) validateChildOwnership(parentNamespace *corev1.Namespace, mode, childNameHashed string) (bool, bool) {
	var checkOwnerReferences = func(ownerReferences []metav1.OwnerReference) (bool, bool) {
		for _, ownerReference := range ownerReferences {
			if ownerReference.Kind == "Namespace" && ownerReference.UID == parentNamespace.GetUID() && ownerReference.Name == parentNamespace.GetName() {
				return true, true
			}
		}
		return true, false
	}
	if mode == "workspace" {
		if childNamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childNameHashed, metav1.GetOptions{}); err == nil {
			return checkOwnerReferences(childNamespace.GetOwnerReferences())
		}
	} else {
		if subtenant, err := c.edgenetclientset.CoreV1alpha1().Tenants().Get(context.TODO(), childNameHashed, metav1.GetOptions{}); err == nil {
			return checkOwnerReferences(subtenant.GetOwnerReferences())
		}
	}
	return false, false
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
		subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
		subnamespaceCopy.Status.Message = messageUpdateFail
		return false
	}

	c.recorder.Event(subnamespaceCopy, corev1.EventTypeNormal, corev1alpha1.StatusPartitioned, messageQuotaCheck)
	return true
}

func (c *Controller) makeSubsidiaryNamespace(subnamespaceCopy *corev1alpha1.SubNamespace, tenant, childNameHashed string, parentAnnotations map[string]string, ownerReferences []metav1.OwnerReference) bool {
	var annotations map[string]string
	if parentAnnotations != nil {
		if value, elementExists := parentAnnotations["scheduler.alpha.kubernetes.io/node-selector"]; elementExists {
			if value != "edge-net.io/access=public,edge-net.io/slice=none" {
				annotations = map[string]string{"scheduler.alpha.kubernetes.io/node-selector": parentAnnotations["scheduler.alpha.kubernetes.io/node-selector"]}
			}
		}
	}
	if annotations == nil {
		if sliceclaim := subnamespaceCopy.GetSliceClaim(); sliceclaim != nil {
			annotations = map[string]string{"scheduler.alpha.kubernetes.io/node-selector": fmt.Sprintf("edge-net.io/access=private,edge-net.io/slice=%s", *sliceclaim)}
		}
	}
	switch subnamespaceCopy.GetMode() {
	case "workspace":
		labels := map[string]string{"edge-net.io/generated": "true", "edge-net.io/kind": "sub", "edge-net.io/tenant": tenant,
			"edge-net.io/owner": subnamespaceCopy.GetName(), "edge-net.io/parent-namespace": subnamespaceCopy.GetNamespace()}
		childNamespaceObj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: childNameHashed, OwnerReferences: ownerReferences}}
		childNamespaceObj.SetName(childNameHashed)
		childNamespaceObj.SetAnnotations(annotations)
		childNamespaceObj.SetLabels(labels)
		if _, err := c.kubeclientset.CoreV1().Namespaces().Create(context.TODO(), childNamespaceObj, metav1.CreateOptions{}); err != nil {
			if errors.IsAlreadyExists(err) {
				childNamespace, _ := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childNamespaceObj.GetName(), metav1.GetOptions{})
				childNamespace.SetAnnotations(annotations)
				childNamespace.SetLabels(labels)
				if _, err := c.kubeclientset.CoreV1().Namespaces().Update(context.TODO(), childNamespace, metav1.UpdateOptions{}); err != nil {
					c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureUpdate, messageNSUpdateFail)
					subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
					subnamespaceCopy.Status.Message = messageNSUpdateFail
					return false
				}
			} else {
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureCreation, messageCreationFail)
				subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
				subnamespaceCopy.Status.Message = messageCreationFail
				c.updateStatus(context.TODO(), subnamespaceCopy)
				return false
			}
		}

		objectName := "edgenet:workspace:owner"
		if subnamespaceCopy.Spec.Workspace.Owner != nil {
			roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: corev1alpha1.TenantOwnerClusterRoleName}
			rbSubjects := []rbacv1.Subject{{Kind: "User", Name: subnamespaceCopy.Spec.Workspace.Owner.Email, APIGroup: "rbac.authorization.k8s.io"}}
			roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: objectName, Namespace: childNameHashed},
				Subjects: rbSubjects, RoleRef: roleRef}
			if roleBinding, err := c.kubeclientset.RbacV1().RoleBindings(childNameHashed).Create(context.TODO(), roleBind, metav1.CreateOptions{}); err != nil {
				if errors.IsAlreadyExists(err) {
					roleBindingCopy := roleBinding.DeepCopy()
					roleBindingCopy.Subjects = []rbacv1.Subject{{Kind: "User", Name: subnamespaceCopy.Spec.Workspace.Owner.Email, APIGroup: "rbac.authorization.k8s.io"}}
					if _, err := c.kubeclientset.RbacV1().RoleBindings(childNameHashed).Update(context.TODO(), roleBindingCopy, metav1.UpdateOptions{}); err != nil {
						c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
						subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
						subnamespaceCopy.Status.Message = messageBindingFailed
						c.updateStatus(context.TODO(), subnamespaceCopy)
						return false
					}
				} else {
					c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
					subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
					subnamespaceCopy.Status.Message = messageBindingFailed
					c.updateStatus(context.TODO(), subnamespaceCopy)
					return false
				}
			}
		} else {
			c.kubeclientset.RbacV1().RoleBindings(childNameHashed).Delete(context.TODO(), objectName, metav1.DeleteOptions{})
		}
	case "subtenant":
		labels := map[string]string{"edge-net.io/generated": "true", "edge-net.io/kind": "core", "edge-net.io/tenant": tenant,
			"edge-net.io/owner": subnamespaceCopy.GetName(), "edge-net.io/parent-namespace": subnamespaceCopy.GetNamespace()}
		tenantRequest := new(registrationv1alpha1.TenantRequest)
		tenantRequest.SetName(childNameHashed)
		tenantRequest.SetAnnotations(annotations)
		tenantRequest.SetLabels(labels)
		tenantRequest.SetOwnerReferences(ownerReferences)
		tenantRequest.Spec.Contact = subnamespaceCopy.Spec.Subtenant.Owner
		if err := c.multitenancyManager.CreateTenant(tenantRequest); err != nil {
			if errors.IsAlreadyExists(err) {
				if subtenant, err := c.edgenetclientset.CoreV1alpha1().Tenants().Get(context.TODO(), childNameHashed, metav1.GetOptions{}); err == nil {
					subtenantCopy := subtenant.DeepCopy()
					subtenantCopy.Spec.Contact = subnamespaceCopy.Spec.Subtenant.Owner
					if _, err = c.edgenetclientset.CoreV1alpha1().Tenants().Update(context.TODO(), subtenantCopy, metav1.UpdateOptions{}); err != nil {
						c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
						subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
						subnamespaceCopy.Status.Message = messageBindingFailed
						c.updateStatus(context.TODO(), subnamespaceCopy)
					}
				} else {
					c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
					subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
					subnamespaceCopy.Status.Message = messageBindingFailed
					c.updateStatus(context.TODO(), subnamespaceCopy)
					klog.Infoln(err)
				}
			} else {
				c.recorder.Event(subnamespaceCopy, corev1.EventTypeWarning, failureCreation, messageCreationFail)
				subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
				subnamespaceCopy.Status.Message = messageCreationFail
				return false
			}
		}
	}
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
		subnamespaceCopy.Status.State = corev1alpha1.StatusFailed
		subnamespaceCopy.Status.Message = messageInheritanceFail
		c.updateStatus(context.TODO(), subnamespaceCopy)
	}
	return done
}

// Inheritance is a struct to manage inheritance between parent and child
type Inheritance struct {
	Child          []interface{}
	Parent         []interface{}
	ChildNamespace string
}

// GetOperationList returns the list of objects to create, update and delete
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

func (c *Controller) cleanup(subnamespaceCopy *corev1alpha1.SubNamespace) {
	parentNamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), subnamespaceCopy.GetNamespace(), metav1.GetOptions{})
	if err != nil {
		klog.Infoln(err)
		return
	}
	parentNamespaceLabels := parentNamespace.GetLabels()
	childNameHashed := subnamespaceCopy.GenerateChildName(parentNamespaceLabels["edge-net.io/cluster-uid"])
	if childExists, childOwned := c.validateChildOwnership(parentNamespace, subnamespaceCopy.GetMode(), childNameHashed); childExists && childOwned {
		switch subnamespaceCopy.GetMode() {
		case "workspace":
			c.kubeclientset.CoreV1().Namespaces().Delete(context.TODO(), childNameHashed, metav1.DeleteOptions{})
		case "subtenant":
			c.edgenetclientset.CoreV1alpha1().Tenants().Delete(context.TODO(), childNameHashed, metav1.DeleteOptions{})
		}
	} else {
		return
	}
	c.partitionParentQuota(subnamespaceCopy, parentNamespace)
}

// updateStatus calls the API to update the subnamespace status.
func (c *Controller) updateStatus(ctx context.Context, subnamespaceCopy *corev1alpha1.SubNamespace) {
	if subnamespaceCopy.Status.State == corev1alpha1.StatusFailed {
		subnamespaceCopy.Status.Failed++
	}
	if _, err := c.edgenetclientset.CoreV1alpha1().SubNamespaces(subnamespaceCopy.GetNamespace()).UpdateStatus(ctx, subnamespaceCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}
