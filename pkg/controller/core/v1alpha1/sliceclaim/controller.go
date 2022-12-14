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

package sliceclaim

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/core/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/core/v1alpha1"
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

const controllerAgentName = "sliceclaim-controller"

// Definitions of the state of the sliceclaim resource
const (
	backoffLimit = 3

	successSynced        = "Synced"
	successClaimed       = "Slice Claimed"
	successApplied       = "Applied"
	successQuotaCheck    = "Checked"
	successBound         = "Bound"
	failureQuotaShortage = "Shortage"
	failureBound         = "Already Bound"
	failureBinding       = "Binding Failed"
	failureCreation      = "Creation Failed"
	pendingSlice         = "Not Bound"

	messageResourceSynced = "Slice claim synced successfully"
	messageClaimed        = "Slice claimed successfully"
	messageApplied        = "Slice claim has applied successfully"
	messageQuotaCheck     = "The parent has sufficient quota"
	messageBound          = "Slice is bound successfully"
	messageQuotaShortage  = "Insufficient quota at the parent"
	messageBoundAlready   = "Slice is bound to another claim already"
	messageBindingFailed  = "Slice binding failed"
	messageCreationFailed = "Slice creation failed"
	messageWaiting        = "Waiting for the slice"
	messageReconciliation = "Reconciliation in progress"
)

// Controller is the controller implementation for Slice Claimresources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	sliceclaimsLister listers.SliceClaimLister
	sliceclaimsSynced cache.InformerSynced

	subnamespacesLister listers.SubNamespaceLister
	subnamespacesSynced cache.InformerSynced

	provisioning string

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
	subnamespaceInformer informers.SubNamespaceInformer,
	sliceclaimInformer informers.SliceClaimInformer,
	provisioning string) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:       kubeclientset,
		edgenetclientset:    edgenetclientset,
		subnamespacesLister: subnamespaceInformer.Lister(),
		subnamespacesSynced: subnamespaceInformer.Informer().HasSynced,
		sliceclaimsLister:   sliceclaimInformer.Lister(),
		sliceclaimsSynced:   sliceclaimInformer.Informer().HasSynced,
		workqueue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "SliceClaims"),
		recorder:            recorder,
		provisioning:        provisioning,
	}

	klog.Infoln("Setting up event handlers")
	// Set up an event handler for when Slice Claimresources change
	sliceclaimInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueSliceClaim,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueSliceClaim(new)
		},
	})

	subnamespaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleSubNamespace,
		UpdateFunc: func(old, new interface{}) {
			controller.handleSubNamespace(new)
		},
		DeleteFunc: controller.handleSubNamespace,
	})

	return controller
}

// Run will set up the event handlers for the types of slice claim and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Infoln("Starting Slice Claimcontroller")

	klog.Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.subnamespacesSynced,
		c.sliceclaimsSynced); !ok {
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
// converge the two. It then updates the Status block of the Slice Claim
// resource with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	sliceclaim, err := c.sliceclaimsLister.SliceClaims(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("sliceclaim '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}
	c.processSliceClaim(sliceclaim.DeepCopy())

	c.recorder.Event(sliceclaim, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueSliceClaim takes a Slice Claimresource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Slice Claim.
func (c *Controller) enqueueSliceClaim(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// handleSubNamespace will take any resource implementing corev1alpha1.SubNamespace and attempt
// to find the SliceClaim resource that 'owns' it. It does this by looking at the
// objects SliceClaimRef field. It then enqueues that SliceClaim resource to be processed.
// If the object does not have an appropriate SliceClaimRef, it will simply be skipped.
func (c *Controller) handleSubNamespace(obj interface{}) {
	var subnamespace *corev1alpha1.SubNamespace
	var ok bool
	if subnamespace, ok = obj.(*corev1alpha1.SubNamespace); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding subnamespace, invalid type"))
			return
		}
		subnamespace, ok = tombstone.Obj.(*corev1alpha1.SubNamespace)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding subnamespace tombstone, invalid type"))
			return
		}
		klog.Infof("Recovered deleted subnamespace '%s' from tombstone", subnamespace.GetName())
	}
	klog.Infof("Processing subnamespace: %s", subnamespace.GetName())
	if subnamespace.Status.State == corev1alpha1.StatusEstablished || subnamespace.Status.State == corev1alpha1.StatusQuotaSet || subnamespace.Status.State == corev1alpha1.StatusSubnamespaceCreated || subnamespace.Status.State == corev1alpha1.StatusPartitioned {
		return
	}
	if name := subnamespace.GetSliceClaim(); name != nil {
		/*if ownerRef := metav1.GetControllerOf(&subnamespace.ObjectMeta); ownerRef != nil {
			if !(ownerRef.Kind == "SliceClaim" && ownerRef.Name == *name) {
				return
			}
		}*/
		sliceclaim, err := c.sliceclaimsLister.SliceClaims(subnamespace.GetNamespace()).Get(*name)
		if err != nil {
			klog.Infof("ignoring orphaned object '%s' of slice claim '%s'", subnamespace.GetSelfLink(), *name)
			return
		}
		c.enqueueSliceClaim(sliceclaim)
	}
}

func (c *Controller) processSliceClaim(sliceclaimCopy *corev1alpha1.SliceClaim) {
	if exceedsBackoffLimit := sliceclaimCopy.Status.Failed >= backoffLimit; exceedsBackoffLimit {
		return
	}

	multitenancyManager := multitenancy.NewManager(c.kubeclientset, c.edgenetclientset)
	permitted, _, namespaceLabels := multitenancyManager.EligibilityCheck(sliceclaimCopy.GetNamespace())
	if permitted {
		switch sliceclaimCopy.Status.State {
		case corev1alpha1.StatusEmployed:
			controllerRef := metav1.GetControllerOf(sliceclaimCopy)
			if controllerRef == nil || (controllerRef != nil && controllerRef.Kind != "Slice") {
				c.recorder.Event(sliceclaimCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
				sliceclaimCopy.Status.State = corev1alpha1.StatusFailed
				sliceclaimCopy.Status.Message = messageBindingFailed
				c.updateStatus(context.TODO(), sliceclaimCopy)
				return
			}
			if isAllocated, isSufficient := c.checkResourceAllocation(sliceclaimCopy, fmt.Sprintf("%s-quota", namespaceLabels["edge-net.io/kind"])); !isSufficient || !isAllocated {
				c.recorder.Event(sliceclaimCopy, corev1.EventTypeNormal, corev1alpha1.StatusReconciliation, messageReconciliation)
				sliceclaimCopy.Status.State = corev1alpha1.StatusReconciliation
				sliceclaimCopy.Status.Message = messageReconciliation
				c.updateStatus(context.TODO(), sliceclaimCopy)
				return
			}
		case corev1alpha1.StatusBound:
			isAllocated, isSufficient := c.checkResourceAllocation(sliceclaimCopy, fmt.Sprintf("%s-quota", namespaceLabels["edge-net.io/kind"]))
			if !isSufficient {
				return
			}
			if isAllocated {
				c.recorder.Event(sliceclaimCopy, corev1.EventTypeNormal, successApplied, messageApplied)
				sliceclaimCopy.Status.State = corev1alpha1.StatusEmployed
				sliceclaimCopy.Status.Message = messageApplied
				c.updateStatus(context.TODO(), sliceclaimCopy)
				return
			}
		case corev1alpha1.StatusRequested:
			if _, isSufficient := c.checkResourceAllocation(sliceclaimCopy, fmt.Sprintf("%s-quota", namespaceLabels["edge-net.io/kind"])); !isSufficient {
				return
			}
			if slice, err := c.edgenetclientset.CoreV1alpha1().Slices().Get(context.TODO(), sliceclaimCopy.Spec.SliceName, metav1.GetOptions{}); err == nil && slice.Spec.ClaimRef != nil && slice.Spec.ClaimRef.UID == sliceclaimCopy.GetUID() {
				if slice.Status.State == corev1alpha1.StatusBound {
					c.recorder.Event(sliceclaimCopy, corev1.EventTypeNormal, successBound, messageBound)
					sliceclaimCopy.Status.State = corev1alpha1.StatusBound
					sliceclaimCopy.Status.Message = messageBound
					c.updateStatus(context.TODO(), sliceclaimCopy)
				}
				return
			}
			c.recorder.Event(sliceclaimCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
			sliceclaimCopy.Status.State = corev1alpha1.StatusFailed
			sliceclaimCopy.Status.Message = messageBindingFailed
			c.updateStatus(context.TODO(), sliceclaimCopy)
		case corev1alpha1.StatusPending:
			if _, isSufficient := c.checkResourceAllocation(sliceclaimCopy, fmt.Sprintf("%s-quota", namespaceLabels["edge-net.io/kind"])); !isSufficient {
				return
			}
			if slice, err := c.edgenetclientset.CoreV1alpha1().Slices().Get(context.TODO(), sliceclaimCopy.Spec.SliceName, metav1.GetOptions{}); err == nil {
				if isBound := c.bindSlice(slice.DeepCopy(), sliceclaimCopy.Spec.SliceClassName, sliceclaimCopy.Spec.NodeSelector, sliceclaimCopy.MakeObjectReference()); isBound {
					c.recorder.Event(sliceclaimCopy, corev1.EventTypeNormal, successClaimed, messageClaimed)
					sliceclaimCopy.Status.State = corev1alpha1.StatusRequested
					sliceclaimCopy.Status.Message = messageWaiting
					c.updateStatus(context.TODO(), sliceclaimCopy)
					return
				}
				c.recorder.Event(sliceclaimCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
				sliceclaimCopy.Status.State = corev1alpha1.StatusFailed
				sliceclaimCopy.Status.Message = messageBindingFailed
				c.updateStatus(context.TODO(), sliceclaimCopy)
				return
			}

			if strings.EqualFold(c.provisioning, corev1alpha1.DynamicStr) {
				if isCreated := c.createSlice(sliceclaimCopy.Spec.SliceName, sliceclaimCopy.Spec.SliceClassName, sliceclaimCopy.Spec.NodeSelector, sliceclaimCopy.MakeObjectReference(), sliceclaimCopy.Spec.SliceExpiry); isCreated {
					c.recorder.Event(sliceclaimCopy, corev1.EventTypeNormal, successClaimed, messageClaimed)
					sliceclaimCopy.Status.State = corev1alpha1.StatusRequested
					sliceclaimCopy.Status.Message = messageWaiting
					c.updateStatus(context.TODO(), sliceclaimCopy)
					return
				}
				c.recorder.Event(sliceclaimCopy, corev1.EventTypeWarning, failureCreation, messageCreationFailed)
				sliceclaimCopy.Status.State = corev1alpha1.StatusFailed
				sliceclaimCopy.Status.Message = messageCreationFailed
				c.updateStatus(context.TODO(), sliceclaimCopy)
			}
		default:
			if _, isSufficient := c.checkResourceAllocation(sliceclaimCopy, fmt.Sprintf("%s-quota", namespaceLabels["edge-net.io/kind"])); !isSufficient {
				return
			}
			c.recorder.Event(sliceclaimCopy, corev1.EventTypeWarning, pendingSlice, messageWaiting)
			sliceclaimCopy.Status.State = corev1alpha1.StatusPending
			sliceclaimCopy.Status.Message = messageWaiting
			c.updateStatus(context.TODO(), sliceclaimCopy)
		}
	}
}

func (c *Controller) checkSubnamespace(namespace string, ownerReferences []metav1.OwnerReference) bool {
	for _, ownerReference := range ownerReferences {
		if ownerReference.Kind == "SubNamespace" {
			if subnamespaceCopy, err := c.edgenetclientset.CoreV1alpha1().SubNamespaces(namespace).Get(context.TODO(), ownerReference.Name, metav1.GetOptions{}); err == nil {
				if subnamespaceCopy.GetResourceAllocation() != nil && (subnamespaceCopy.Status.State == corev1alpha1.StatusEstablished || subnamespaceCopy.Status.State == corev1alpha1.StatusQuotaSet || subnamespaceCopy.Status.State == corev1alpha1.StatusSubnamespaceCreated || subnamespaceCopy.Status.State == corev1alpha1.StatusPartitioned) {
					return true
				}
			}
		}
	}
	return false
}

func (c *Controller) checkResourceAllocation(sliceclaimCopy *corev1alpha1.SliceClaim, quotaName string) (bool, bool) {
	isControlled := c.checkSubnamespace(sliceclaimCopy.GetNamespace(), sliceclaimCopy.GetOwnerReferences())
	if !isControlled {
		if hasEnoughQuota := c.checkResourceQuota(sliceclaimCopy.Spec.NodeSelector.Resources.Limits, sliceclaimCopy.Spec.NodeSelector.Count, sliceclaimCopy.GetNamespace(), quotaName); !hasEnoughQuota {
			c.recorder.Event(sliceclaimCopy, corev1.EventTypeWarning, failureQuotaShortage, messageQuotaShortage)
			sliceclaimCopy.Status.State = corev1alpha1.StatusFailed
			sliceclaimCopy.Status.Message = messageQuotaShortage
			c.updateStatus(context.TODO(), sliceclaimCopy)
			return false, false
		}
		c.recorder.Event(sliceclaimCopy, corev1.EventTypeNormal, successQuotaCheck, messageQuotaCheck)
		return false, true
	}
	return true, true
}

func (c *Controller) createSlice(sliceName, sliceclaimClass string, sliceclaimNodeSelector corev1alpha1.NodeSelector, sliceclaimRef *corev1.ObjectReference, expiry *metav1.Time) bool {
	slice := new(corev1alpha1.Slice)
	slice.SetName(sliceName)
	slice.Spec.SliceClassName = sliceclaimClass
	slice.Spec.NodeSelector = sliceclaimNodeSelector
	slice.Spec.ClaimRef = sliceclaimRef
	slice.Status.Expiry = expiry
	if _, err := c.edgenetclientset.CoreV1alpha1().Slices().Create(context.TODO(), slice, metav1.CreateOptions{}); err == nil {
		return true
	}
	return false
}

func (c *Controller) bindSlice(sliceCopy *corev1alpha1.Slice, sliceclaimClass string, sliceclaimNodeSelector corev1alpha1.NodeSelector, sliceclaimRef *corev1.ObjectReference) bool {
	if sliceCopy.Status.State == corev1alpha1.StatusReserved && sliceCopy.Spec.SliceClassName == sliceclaimClass && reflect.DeepEqual(sliceCopy.Spec.NodeSelector, sliceclaimNodeSelector) {
		if sliceCopy.Spec.ClaimRef == nil {
			sliceCopy.Spec.ClaimRef = sliceclaimRef
			if _, err := c.edgenetclientset.CoreV1alpha1().Slices().Update(context.TODO(), sliceCopy, metav1.UpdateOptions{}); err == nil {
				return true
			}
		} else {
			if sliceCopy.Spec.ClaimRef.UID == sliceclaimRef.UID {
				return true
			}
		}
	}
	return false
}

func (c *Controller) checkResourceQuota(sliceclaimResourceLimits corev1.ResourceList, nodeCount int, parentNamespace, parentQuotaName string) bool {
	if parentResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(parentNamespace).Get(context.TODO(), parentQuotaName, metav1.GetOptions{}); err == nil {
		var resourceDemandList = make(corev1.ResourceList)
		for key, value := range sliceclaimResourceLimits {
			resourceDemand := value.DeepCopy()
			for i := 0; i < nodeCount; i++ {
				if _, elementExists := resourceDemandList[key]; elementExists {
					resourceDemand.Add(value)
					resourceDemandList[key] = resourceDemand
				} else {
					resourceDemandList[key] = resourceDemand
				}
			}
		}
		for key, value := range parentResourceQuota.Spec.Hard {
			availableQuota := value.DeepCopy()
			if _, elementExists := resourceDemandList[key]; elementExists {
				if availableQuota.Cmp(resourceDemandList[key]) == -1 {
					return false
				}
				availableQuota.Sub(resourceDemandList[key])
			}
		}
	}
	return true
}

// updateStatus calls the API to update the slice claim status.
func (c *Controller) updateStatus(ctx context.Context, sliceclaimCopy *corev1alpha1.SliceClaim) {
	if sliceclaimCopy.Status.State == corev1alpha1.StatusFailed {
		sliceclaimCopy.Status.Failed++
	}
	if _, err := c.edgenetclientset.CoreV1alpha1().SliceClaims(sliceclaimCopy.GetNamespace()).UpdateStatus(ctx, sliceclaimCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}
