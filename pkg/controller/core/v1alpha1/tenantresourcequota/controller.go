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

package tenantresourcequota

import (
	"context"
	"fmt"
	"reflect"
	"sync"
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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "tenantresourcequota-controller"

// Definitions of the state of the tenantresourcequota resource
const (
	backoffLimit = 3

	successSynced           = "Synced"
	successApplied          = "Applied"
	successTraversalStarted = "Started"
	successTuned            = "Tuned"
	successDeleted          = "Deleted"
	successRemoved          = "Removed"
	warningNotFound         = "Not Found"

	messageResourceSynced   = "Tenant Resource Quota synced successfully"
	messageTraversalStarted = "Namespace traversal initiated successfully"
	messageTuned            = "Core resource quota tuned"
	messageDeleted          = "Last created subnamespace deleted to balance resource consumption"
	messageRemoved          = "Expired Claim / Drop removed smoothly"
	messageNotFound         = "There is no resource quota in the core namespace"
	messageNotUpdated       = "Resource quota cannot be updated"
	messageQuotaCreated     = "Core resource quota created"
	messageReconciliation   = "Reconciliation in progress"
	messageApplied          = "Tenant Resource Quota applied to tenant's namespaces"
)

type traverseStatus struct {
	deleted bool
	failed  bool
	done    bool
}

// Controller is the controller implementation for Tenant Resource Quota resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	nodesLister corelisters.NodeLister
	nodesSynced cache.InformerSynced

	tenantresourcequotasLister listers.TenantResourceQuotaLister
	tenantresourcequotasSynced cache.InformerSynced

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
	nodeInformer coreinformers.NodeInformer,
	tenantresourcequotaInformer informers.TenantResourceQuotaInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:              kubeclientset,
		edgenetclientset:           edgenetclientset,
		nodesLister:                nodeInformer.Lister(),
		nodesSynced:                nodeInformer.Informer().HasSynced,
		tenantresourcequotasLister: tenantresourcequotaInformer.Lister(),
		tenantresourcequotasSynced: tenantresourcequotaInformer.Informer().HasSynced,
		workqueue:                  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "TenantResourceQuotas"),
		recorder:                   recorder,
	}

	klog.V(4).Infoln("Setting up event handlers")
	// Set up an event handler for when Tenant Resource Quota resources change
	var getClosestExpiryDate = func(stale bool, objects ...map[string]corev1alpha1.ResourceTuning) (*metav1.Time, bool) {
		var closestDate *metav1.Time
		expiryDateExists := false
		for _, obj := range objects {
			for _, value := range obj {
				if value.Expiry != nil && time.Until(value.Expiry.Time) > 0 {
					if stale || !expiryDateExists || closestDate.Sub(value.Expiry.Time) >= 0 {
						expiryDateExists = true
						closestDate = value.Expiry
					}
				}
			}
		}
		return closestDate, expiryDateExists
	}
	tenantresourcequotaInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			tenantResourceQuota := obj.(*corev1alpha1.TenantResourceQuota)
			if expiryDate, exists := getClosestExpiryDate(false, tenantResourceQuota.Spec.Claim, tenantResourceQuota.Spec.Drop); exists {
				controller.enqueueTenantResourceQuotaAfter(tenantResourceQuota, time.Until(expiryDate.Time))
			}
			controller.enqueueTenantResourceQuota(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			newTenantResourceQuota := new.(*corev1alpha1.TenantResourceQuota)
			oldTenantResourceQuota := old.(*corev1alpha1.TenantResourceQuota)
			if newExpiryDate, exists := getClosestExpiryDate(false, newTenantResourceQuota.Spec.Claim, newTenantResourceQuota.Spec.Drop); exists {
				if previousExpiryDate, exists := getClosestExpiryDate(true, oldTenantResourceQuota.Spec.Claim, oldTenantResourceQuota.Spec.Drop); !exists ||
					(exists && previousExpiryDate.Sub(newExpiryDate.Time) > 0) {
					controller.enqueueTenantResourceQuotaAfter(newTenantResourceQuota, time.Until(newExpiryDate.Time))
				}
			}
			controller.enqueueTenantResourceQuota(new)
		},
	})

	return controller
}

// Run will set up the event handlers for the types of tenant resource quota and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.V(4).Infoln("Starting Tenant Resource Quota controller")

	klog.V(4).Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.nodesSynced,
		c.tenantresourcequotasSynced); !ok {
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
// converge the two. It then updates the Status block of the Tenant Resource Quota
// resource with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	tenantresourcequota, err := c.tenantresourcequotasLister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("tenantresourcequota '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	c.processTenantResourceQuota(tenantresourcequota.DeepCopy())

	c.recorder.Event(tenantresourcequota, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueTenantResourceQuota takes a TenantResourceQuota resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than TenantResourceQuota.
func (c *Controller) enqueueTenantResourceQuota(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// enqueueTenantResourceQuotaAfter takes a TenantResourceQuota resource and converts it into a namespace/name
// string which is then put onto the work queue after the expiry date of a claim/drop to delete the so-said claim/drop.
// This method should *not* be passed resources of any type other than TenantResourceQuota.
func (c *Controller) enqueueTenantResourceQuotaAfter(obj interface{}, after time.Duration) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.AddAfter(key, after)
}

func (c *Controller) processTenantResourceQuota(tenantResourceQuotaCopy *corev1alpha1.TenantResourceQuota) {
	if exceedsBackoffLimit := tenantResourceQuotaCopy.Status.Failed >= backoffLimit; exceedsBackoffLimit {
		c.cleanup(tenantResourceQuotaCopy)
		return
	}

	multitenancyManager := multitenancy.NewManager(c.kubeclientset, c.edgenetclientset)
	permitted, _, parentNamespaceLabels := multitenancyManager.EligibilityCheck(tenantResourceQuotaCopy.GetName())
	if permitted {
		if expired := tenantResourceQuotaCopy.DropExpiredItems(); expired {
			c.recorder.Event(tenantResourceQuotaCopy, corev1.EventTypeNormal, successRemoved, messageRemoved)
			tenantResourceQuotaCopy.Status.State = corev1alpha1.StatusReconciliation
			tenantResourceQuotaCopy.Status.Message = messageReconciliation
			c.updateStatus(context.TODO(), tenantResourceQuotaCopy)
			return
		}

		switch tenantResourceQuotaCopy.Status.State {
		case corev1alpha1.StatusApplied:
			c.reconcile(tenantResourceQuotaCopy, parentNamespaceLabels["edge-net.io/cluster-uid"])
		case corev1alpha1.StatusQuotaCreated:
			if ok := c.tuneHierarchicalResourceQuota(tenantResourceQuotaCopy, parentNamespaceLabels["edge-net.io/cluster-uid"]); !ok {
				c.recorder.Event(tenantResourceQuotaCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageNotUpdated)
				tenantResourceQuotaCopy.Status.State = corev1alpha1.StatusFailed
				tenantResourceQuotaCopy.Status.Message = messageNotUpdated
				return
			}
			c.recorder.Event(tenantResourceQuotaCopy, corev1.EventTypeNormal, corev1alpha1.StatusApplied, messageApplied)
			tenantResourceQuotaCopy.Status.State = corev1alpha1.StatusApplied
			tenantResourceQuotaCopy.Status.Message = messageApplied
			c.updateStatus(context.TODO(), tenantResourceQuotaCopy)
		default:
			// The initial resource quota in the core namespace is equal to the defined tenant resource quota.
			resourceQuota := corev1.ResourceQuota{}
			resourceQuota.Name = "core-quota"
			resourceQuota.Spec = corev1.ResourceQuotaSpec{
				Hard: tenantResourceQuotaCopy.Spec.Claim["initial"].ResourceList,
			}
			if _, err := c.kubeclientset.CoreV1().ResourceQuotas(tenantResourceQuotaCopy.GetName()).Create(context.TODO(), resourceQuota.DeepCopy(), metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
				c.recorder.Event(tenantResourceQuotaCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageNotFound)
				tenantResourceQuotaCopy.Status.State = corev1alpha1.StatusFailed
				tenantResourceQuotaCopy.Status.Message = messageNotFound
				c.updateStatus(context.TODO(), tenantResourceQuotaCopy)
				return
			}
			c.recorder.Event(tenantResourceQuotaCopy, corev1.EventTypeNormal, corev1alpha1.StatusQuotaCreated, messageQuotaCreated)
			tenantResourceQuotaCopy.Status.State = corev1alpha1.StatusQuotaCreated
			tenantResourceQuotaCopy.Status.Message = messageQuotaCreated
			c.updateStatus(context.TODO(), tenantResourceQuotaCopy)
		}
	}
}

func (c *Controller) reconcile(tenantResourceQuotaCopy *corev1alpha1.TenantResourceQuota, clusterUID string) {
	if ok := c.tuneHierarchicalResourceQuota(tenantResourceQuotaCopy, clusterUID); !ok {
		tenantResourceQuotaCopy.Status.State = corev1alpha1.StatusQuotaCreated
		tenantResourceQuotaCopy.Status.Message = messageQuotaCreated
	}
	if _, err := c.kubeclientset.CoreV1().ResourceQuotas(tenantResourceQuotaCopy.GetName()).Get(context.TODO(), "core-quota", metav1.GetOptions{}); err != nil {
		tenantResourceQuotaCopy.Status.State = corev1alpha1.StatusReconciliation
		tenantResourceQuotaCopy.Status.Message = messageReconciliation
	}
	if tenantResourceQuotaCopy.Status.State != corev1alpha1.StatusApplied {
		c.updateStatus(context.TODO(), tenantResourceQuotaCopy)
	}
}

func (c *Controller) tuneHierarchicalResourceQuota(tenantResourceQuotaCopy *corev1alpha1.TenantResourceQuota, clusterUID string) bool {
	c.recorder.Event(tenantResourceQuotaCopy, corev1.EventTypeNormal, successTraversalStarted, messageTraversalStarted)
	ok := true
	statusChannel := make(chan traverseStatus, 1)
	go c.traverse(tenantResourceQuotaCopy.GetName(), "core", clusterUID, tenantResourceQuotaCopy.Fetch(), statusChannel)
traverseNamespaces:
	for {
		select {
		case status, ok := <-statusChannel:
			if !ok || (ok && status.done) {
				break traverseNamespaces
			}
			if status.deleted {
				c.recorder.Event(tenantResourceQuotaCopy, corev1.EventTypeWarning, successDeleted, messageDeleted)
			}
			if status.failed {
				ok = false
			}
		}
	}
	close(statusChannel)
	return ok
}

func (c *Controller) traverse(namespace, namespaceKind, clusterUID string, remainingQuotaResourceList map[corev1.ResourceName]resource.Quantity, statusChannel chan<- traverseStatus) {
	// This task becomes expensive when the hierarchy chain is gigantic with a substantial depth.
	// So Goroutines come into play.
	var wg sync.WaitGroup
	isDeleted, isFailed := c.tuneResourceQuota(namespace, namespaceKind, remainingQuotaResourceList)
	statusChannel <- traverseStatus{deleted: isDeleted, failed: isFailed}
	if !isFailed {
		subNamespaceRaw, _ := c.edgenetclientset.CoreV1alpha1().SubNamespaces(namespace).List(context.TODO(), metav1.ListOptions{})
		if len(subNamespaceRaw.Items) != 0 {
			for _, subnamespaceRow := range subNamespaceRaw.Items {
				if subnamespaceRow.Spec.Workspace != nil {
					wg.Add(1)
					go func(subnamespace corev1alpha1.SubNamespace) {
						defer wg.Done()
						c.traverse(subnamespace.GenerateChildName(clusterUID), "sub", clusterUID, subnamespace.GetResourceAllocation(), statusChannel)
					}(subnamespaceRow)
				}
			}
			wg.Wait()
		}
	}
	if namespaceKind == "core" {
		statusChannel <- traverseStatus{done: true}
	}
}

func (c *Controller) tuneResourceQuota(namespace, namespaceKind string, remainingQuotaResourceList map[corev1.ResourceName]resource.Quantity) (bool, bool) {
	if resourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(namespace).Get(context.TODO(), fmt.Sprintf("%s-quota", namespaceKind), metav1.GetOptions{}); err == nil {
		remainingQuotaResourceList, lastInSubnamespace, isQuotaSufficient := c.subtractSubnamespaceQuotas(namespace, remainingQuotaResourceList)
		if !isQuotaSufficient {
			c.edgenetclientset.CoreV1alpha1().SubNamespaces(namespace).Delete(context.TODO(), lastInSubnamespace, metav1.DeleteOptions{})
		}
		if !reflect.DeepEqual(remainingQuotaResourceList, resourceQuota.Spec.Hard) {
			resourceQuota.Spec.Hard = remainingQuotaResourceList
			if _, err := c.kubeclientset.CoreV1().ResourceQuotas(namespace).Update(context.TODO(), resourceQuota, metav1.UpdateOptions{}); err != nil {
				return !isQuotaSufficient, true
			}
		}
		return !isQuotaSufficient, false
	}
	return false, false
}

func (c *Controller) subtractSubnamespaceQuotas(namespace string, remainingQuotaResourceList map[corev1.ResourceName]resource.Quantity) (map[corev1.ResourceName]resource.Quantity, string, bool) {
	var lastInDate metav1.Time
	var lastInSubnamespace string
	if subnamespaceRaw, err := c.edgenetclientset.CoreV1alpha1().SubNamespaces(namespace).List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, subnamespaceRow := range subnamespaceRaw.Items {
			if subnamespaceRow.Status.State == corev1alpha1.StatusEstablished || subnamespaceRow.Status.State == corev1alpha1.StatusQuotaSet || subnamespaceRow.Status.State == corev1alpha1.StatusSubnamespaceCreated || subnamespaceRow.Status.State == corev1alpha1.StatusPartitioned {
				if lastInDate.IsZero() || subnamespaceRow.GetCreationTimestamp().After(lastInDate.Time) {
					lastInSubnamespace = subnamespaceRow.GetName()
					lastInDate = subnamespaceRow.GetCreationTimestamp()
				}
				for remainingQuotaResource, remainingQuotaQuantity := range remainingQuotaResourceList {
					childQuota := subnamespaceRow.RetrieveQuantity(remainingQuotaResource)
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

func (c *Controller) cleanup(tenantResourceQuotaCopy *corev1alpha1.TenantResourceQuota) {

}

// updateStatus calls the API to update the tenant resource quota status.
func (c *Controller) updateStatus(ctx context.Context, tenantResourceQuotaCopy *corev1alpha1.TenantResourceQuota) {
	if tenantResourceQuotaCopy.Status.State == corev1alpha1.StatusFailed {
		tenantResourceQuotaCopy.Status.Failed++
	}
	if _, err := c.edgenetclientset.CoreV1alpha1().TenantResourceQuotas().UpdateStatus(ctx, tenantResourceQuotaCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}
