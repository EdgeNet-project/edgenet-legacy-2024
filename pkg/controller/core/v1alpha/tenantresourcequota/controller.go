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
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/access"
	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/core/v1alpha"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/node"

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
	successSynced         = "Synced"
	messageResourceSynced = "Tenant Resource Quota synced successfully"
	successApplied        = "Applied"
	messageApplied        = "Tenant Resource Quota applied successfully"
	create                = "create"
	update                = "update"
	success               = "Applied"
	trueStr               = "True"
	falseStr              = "False"
	unknownStr            = "Unknown"
)

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
	var getClosestExpiryDate = func(stale bool, objects ...map[string]corev1alpha.ResourceTuning) (*metav1.Time, bool) {
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
			tenantResourceQuota := obj.(*corev1alpha.TenantResourceQuota)
			if expiryDate, exists := getClosestExpiryDate(false, tenantResourceQuota.Spec.Claim, tenantResourceQuota.Spec.Drop); exists {
				controller.enqueueTenantResourceQuotaAfter(tenantResourceQuota, time.Until(expiryDate.Time))
			}
			controller.enqueueTenantResourceQuota(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			newTenantResourceQuota := new.(*corev1alpha.TenantResourceQuota)
			oldTenantResourceQuota := old.(*corev1alpha.TenantResourceQuota)
			if newExpiryDate, exists := getClosestExpiryDate(false, newTenantResourceQuota.Spec.Claim, newTenantResourceQuota.Spec.Drop); exists {
				if previousExpiryDate, exists := getClosestExpiryDate(true, oldTenantResourceQuota.Spec.Claim, oldTenantResourceQuota.Spec.Drop); !exists || (exists && previousExpiryDate.Sub(newExpiryDate.Time) > 0) {
					controller.enqueueTenantResourceQuotaAfter(newTenantResourceQuota, time.Until(newExpiryDate.Time))
				}
			}
			controller.enqueueTenantResourceQuota(new)
		},
	})

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			nodeObj := obj.(*corev1.Node)
			for key := range nodeObj.Labels {
				if key == "node-role.kubernetes.io/master" {
					return
				}
			}
			ready := node.GetConditionReadyStatus(nodeObj)
			if ready == trueStr {
				for _, owner := range nodeObj.GetOwnerReferences() {
					if owner.Kind == "Tenant" {
						tenantResourceQuota, err := edgenetclientset.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), owner.Name, metav1.GetOptions{})
						if err == nil {
							cpuAward := resource.NewQuantity(int64(float64(nodeObj.Status.Capacity.Cpu().Value())*1.5), resource.BinarySI).DeepCopy()
							memoryAward := resource.NewQuantity(int64(float64(nodeObj.Status.Capacity.Memory().Value())*1.3), resource.BinarySI).DeepCopy()
							if _, elementExists := tenantResourceQuota.Spec.Claim[nodeObj.GetName()]; elementExists {
								if tenantResourceQuota.Spec.Claim[nodeObj.GetName()].ResourceList["cpu"] != cpuAward ||
									tenantResourceQuota.Spec.Claim[nodeObj.GetName()].ResourceList["memory"] != memoryAward {
									tenantResourceQuota.Spec.Claim[nodeObj.GetName()].ResourceList["cpu"] = cpuAward
									tenantResourceQuota.Spec.Claim[nodeObj.GetName()].ResourceList["memory"] = memoryAward
									edgenetclientset.CoreV1alpha().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuota, metav1.UpdateOptions{})
								}
							} else {
								claim := corev1alpha.ResourceTuning{
									ResourceList: corev1.ResourceList{
										corev1.ResourceCPU:    cpuAward,
										corev1.ResourceMemory: memoryAward,
									},
								}
								tenantResourceQuota.Spec.Claim[nodeObj.GetName()] = claim
								edgenetclientset.CoreV1alpha().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuota, metav1.UpdateOptions{})
							}
						}
					}
				}
			}
		},
		UpdateFunc: func(old, new interface{}) {
			oldObj := old.(*corev1.Node)
			newObj := new.(*corev1.Node)
			oldReady := node.GetConditionReadyStatus(oldObj)
			newReady := node.GetConditionReadyStatus(newObj)
			if (oldReady == falseStr && newReady == trueStr) ||
				(oldReady == unknownStr && newReady == trueStr) {
				for _, owner := range newObj.GetOwnerReferences() {
					if owner.Kind == "Tenant" {
						tenantResourceQuota, err := edgenetclientset.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), owner.Name, metav1.GetOptions{})
						if err == nil {
							cpuAward := resource.NewQuantity(int64(float64(newObj.Status.Capacity.Cpu().Value())*1.5), resource.BinarySI).DeepCopy()
							memoryAward := resource.NewQuantity(int64(float64(newObj.Status.Capacity.Memory().Value())*1.3), resource.BinarySI).DeepCopy()
							if _, elementExists := tenantResourceQuota.Spec.Claim[newObj.GetName()]; elementExists {
								if tenantResourceQuota.Spec.Claim[newObj.GetName()].ResourceList["cpu"] != cpuAward ||
									tenantResourceQuota.Spec.Claim[newObj.GetName()].ResourceList["memory"] != memoryAward {
									tenantResourceQuota.Spec.Claim[newObj.GetName()].ResourceList["cpu"] = cpuAward
									tenantResourceQuota.Spec.Claim[newObj.GetName()].ResourceList["memory"] = memoryAward
									edgenetclientset.CoreV1alpha().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuota, metav1.UpdateOptions{})
								}
							} else {
								claim := corev1alpha.ResourceTuning{
									ResourceList: corev1.ResourceList{
										corev1.ResourceCPU:    cpuAward,
										corev1.ResourceMemory: memoryAward,
									},
								}
								tenantResourceQuota.Spec.Claim[newObj.GetName()] = claim
								edgenetclientset.CoreV1alpha().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuota, metav1.UpdateOptions{})
							}
						}
					}
				}
			} else if (oldReady == trueStr && newReady == falseStr) ||
				(oldReady == trueStr && newReady == unknownStr) {
				for _, owner := range newObj.GetOwnerReferences() {
					if owner.Kind == "Tenant" {
						tenantResourceQuota, err := edgenetclientset.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), owner.Name, metav1.GetOptions{})
						if err == nil {
							if _, elementExists := tenantResourceQuota.Spec.Claim[newObj.GetName()]; elementExists {
								delete(tenantResourceQuota.Spec.Claim, newObj.GetName())
								edgenetclientset.CoreV1alpha().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuota, metav1.UpdateOptions{})
							}
						}
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			nodeObj := obj.(*corev1.Node)
			ready := node.GetConditionReadyStatus(nodeObj)
			if ready == trueStr {
				for _, owner := range nodeObj.GetOwnerReferences() {
					if owner.Kind == "Tenant" {
						tenantResourceQuota, err := edgenetclientset.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), owner.Name, metav1.GetOptions{})
						if err == nil {
							if _, elementExists := tenantResourceQuota.Spec.Claim[nodeObj.GetName()]; elementExists {
								delete(tenantResourceQuota.Spec.Claim, nodeObj.GetName())
								edgenetclientset.CoreV1alpha().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuota, metav1.UpdateOptions{})

							}
						}
					}
				}
			}
		},
	})

	access.Clientset = kubeclientset
	access.EdgenetClientset = edgenetclientset

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

func (c *Controller) processTenantResourceQuota(tenantResourceQuotaCopy *corev1alpha.TenantResourceQuota) {
	oldStatus := tenantResourceQuotaCopy.Status
	statusUpdate := func() {
		if !reflect.DeepEqual(oldStatus, tenantResourceQuotaCopy.Status) {
			if _, err := c.edgenetclientset.CoreV1alpha().TenantResourceQuotas().UpdateStatus(context.TODO(), tenantResourceQuotaCopy, metav1.UpdateOptions{}); err != nil {
				klog.V(4).Infoln(err)
			}
		}
	}
	defer statusUpdate()

	if tenant, err := c.edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantResourceQuotaCopy.GetName(), metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			c.edgenetclientset.CoreV1alpha().TenantResourceQuotas().Delete(context.TODO(), tenant.GetName(), metav1.DeleteOptions{})
		}
	} else {
		claimList, expiredClaim := removeExpiredItems(tenantResourceQuotaCopy.Spec.Claim)
		dropList, expiredDrop := removeExpiredItems(tenantResourceQuotaCopy.Spec.Drop)
		if expiredClaim || expiredDrop {
			tenantResourceQuotaCopy.Spec.Claim = claimList
			tenantResourceQuotaCopy.Spec.Drop = dropList
			if tenantResourceQuotaUpdated, err := c.edgenetclientset.CoreV1alpha().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuotaCopy, metav1.UpdateOptions{}); err == nil {
				tenantResourceQuotaCopy = tenantResourceQuotaUpdated.DeepCopy()
			}
		}

		if tenant.Spec.Enabled {
			tenantResourceQuotaCopy.Status.State = success
			tenantResourceQuotaCopy.Status.Message = messageApplied

			c.tuneResourceQuota(tenant.GetName(), tenantResourceQuotaCopy)
		}
	}
}

func removeExpiredItems(obj map[string]corev1alpha.ResourceTuning) (map[string]corev1alpha.ResourceTuning, bool) {
	expired := false
	for key, value := range obj {
		if value.Expiry != nil && time.Until(value.Expiry.Time) <= 0 {
			expired = true
			delete(obj, key)
		}
	}
	return obj, expired
}

func (c *Controller) NamespaceTraversal(coreNamespace string) (int64, int64, *corev1alpha.SubNamespace) {
	// Get the total consumption that all namespaces do in tenant
	var aggregatedCPU, aggregatedMemory int64 = 0, 0
	var lastInDate metav1.Time
	var lastInSubNamespace *corev1alpha.SubNamespace
	c.traverse(coreNamespace, coreNamespace, &aggregatedCPU, &aggregatedMemory, lastInSubNamespace, &lastInDate)
	return aggregatedCPU, aggregatedMemory, lastInSubNamespace
}

func (c *Controller) traverse(coreNamespace, namespace string, aggregatedCPU *int64, aggregatedMemory *int64, lastInSubNamespace *corev1alpha.SubNamespace, lastInDate *metav1.Time) {
	c.aggregateQuota(namespace, aggregatedCPU, aggregatedMemory)
	subNamespaceRaw, _ := c.edgenetclientset.CoreV1alpha().SubNamespaces(namespace).List(context.TODO(), metav1.ListOptions{})
	if len(subNamespaceRaw.Items) != 0 {
		for _, subNamespaceRow := range subNamespaceRaw.Items {
			if lastInDate.IsZero() || lastInDate.Sub(subNamespaceRow.GetCreationTimestamp().Time) >= 0 {
				lastInSubNamespace = subNamespaceRow.DeepCopy()
				*lastInDate = subNamespaceRow.GetCreationTimestamp()
			}
			subNamespaceStr := fmt.Sprintf("%s-%s", coreNamespace, subNamespaceRow.GetName())
			c.traverse(coreNamespace, subNamespaceStr, aggregatedCPU, aggregatedMemory, lastInSubNamespace, lastInDate)
		}
	}
}

func (c *Controller) aggregateQuota(namespace string, aggregatedCPU *int64, aggregatedMemory *int64) {
	resourceQuotasRaw, _ := c.kubeclientset.CoreV1().ResourceQuotas(namespace).List(context.TODO(), metav1.ListOptions{})
	if len(resourceQuotasRaw.Items) != 0 {
		for _, resourceQuotasRow := range resourceQuotasRaw.Items {
			*aggregatedCPU += resourceQuotasRow.Spec.Hard.Cpu().Value()
			*aggregatedMemory += resourceQuotasRow.Spec.Hard.Memory().Value()
		}
	}
}

// calculateTenantQuota adds the resources defined in claims, and subtracts those in drops to calculate the tenant resource quota.
func calculateTenantQuota(tenantResourceQuota *corev1alpha.TenantResourceQuota) (int64, int64) {
	var cpuQuota int64
	var memoryQuota int64
	if len(tenantResourceQuota.Spec.Claim) > 0 {
		for _, claim := range tenantResourceQuota.Spec.Claim {
			if claim.Expiry == nil || (claim.Expiry != nil && claim.Expiry.Time.Sub(time.Now()) >= 0) {
				cpuResource := claim.ResourceList["cpu"]
				cpuQuota += cpuResource.Value()
				memoryResource := claim.ResourceList["memory"]
				memoryQuota += memoryResource.Value()
			}
		}
	}
	if len(tenantResourceQuota.Spec.Drop) > 0 {
		for _, drop := range tenantResourceQuota.Spec.Drop {
			if drop.Expiry == nil || (drop.Expiry != nil && drop.Expiry.Time.Sub(time.Now()) >= 0) {
				cpuResource := drop.ResourceList["cpu"]
				cpuQuota -= cpuResource.Value()
				memoryResource := drop.ResourceList["memory"]
				memoryQuota -= memoryResource.Value()
			}
		}
	}
	return cpuQuota, memoryQuota
}

func (c *Controller) tuneResourceQuota(coreNamespace string, tenantResourceQuota *corev1alpha.TenantResourceQuota) {
	aggregatedCPU, aggregatedMemory, lastInSubNamespace := c.NamespaceTraversal(coreNamespace)
	cpuQuota, memoryQuota := calculateTenantQuota(tenantResourceQuota)
	if cpuQuota < aggregatedCPU || memoryQuota < aggregatedMemory {
		cpuShortage := aggregatedCPU - cpuQuota
		memoryShortage := aggregatedMemory - memoryQuota
		coreResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(coreNamespace).Get(context.TODO(), "core-quota", metav1.GetOptions{})
		if err == nil {
			coreCPUQuota := coreResourceQuota.Spec.Hard.Cpu().DeepCopy()
			coreMemoryQuota := coreResourceQuota.Spec.Hard.Memory().DeepCopy()
			if coreCPUQuota.Value() >= cpuShortage && coreMemoryQuota.Value() >= memoryShortage {
				coreCPUQuota.Set(coreCPUQuota.Value() - cpuShortage)
				coreResourceQuota.Spec.Hard["cpu"] = coreCPUQuota
				coreMemoryQuota.Set(coreMemoryQuota.Value() - memoryShortage)
				coreResourceQuota.Spec.Hard["memory"] = coreMemoryQuota
				c.kubeclientset.CoreV1().ResourceQuotas(coreNamespace).Update(context.TODO(), coreResourceQuota, metav1.UpdateOptions{})
			} else {
				if lastInSubNamespace != nil {
					c.edgenetclientset.CoreV1alpha().SubNamespaces(lastInSubNamespace.GetNamespace()).Delete(context.TODO(), lastInSubNamespace.GetName(), metav1.DeleteOptions{})
					time.Sleep(200 * time.Millisecond)
					defer c.tuneResourceQuota(coreNamespace, tenantResourceQuota)
				}
			}
		}
	} else if cpuQuota > aggregatedCPU || memoryQuota > aggregatedMemory {
		cpuLacune := cpuQuota - aggregatedCPU
		memoryLacune := memoryQuota - aggregatedMemory
		coreResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(coreNamespace).Get(context.TODO(), "core-quota", metav1.GetOptions{})
		if err == nil {
			coreCPUQuota := coreResourceQuota.Spec.Hard.Cpu().DeepCopy()
			coreMemoryQuota := coreResourceQuota.Spec.Hard.Memory().DeepCopy()
			coreCPUQuota.Set(coreCPUQuota.Value() + cpuLacune)
			coreResourceQuota.Spec.Hard["cpu"] = coreCPUQuota
			coreMemoryQuota.Set(coreMemoryQuota.Value() + memoryLacune)
			coreResourceQuota.Spec.Hard["memory"] = coreMemoryQuota
			c.kubeclientset.CoreV1().ResourceQuotas(coreNamespace).Update(context.TODO(), coreResourceQuota, metav1.UpdateOptions{})
		}
	}
}
