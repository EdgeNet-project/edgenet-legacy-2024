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
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/access"
	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/core/v1alpha"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/namespace"
	"github.com/google/uuid"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

const controllerAgentName = "subnamespace-controller"

// Definitions of the state of the subnamespace resource
const (
	successSynced         = "Synced"
	messageResourceSynced = "Subsidiary Namespace synced successfully"
	failure               = "Failure"
	established           = "Established"
)

// Dictionary of status messages
var statusDict = map[string]string{
	"subnamespace-ok":      "Subsidiary Namespace successfully established",
	"subnamespace-failure": "Subsidiary Namespace cannot be created",
	"namespace-exists":     "Name of the namespace, %s, conflicts with another one in the tenant",
	"quota-exceeded":       "Tenant resource quota exceeded",
}

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
		AddFunc: controller.enqueueSubNamespace,
		UpdateFunc: func(old, new interface{}) {
			newSubNamespace := new.(*corev1alpha.SubNamespace)
			oldSubNamespace := old.(*corev1alpha.SubNamespace)
			if reflect.DeepEqual(newSubNamespace.Spec, oldSubNamespace.Spec) {
				return
			}

			controller.enqueueSubNamespace(new)
		},
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

	c.applyProcedure(subnamespace)
	c.recorder.Event(subnamespace, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueSubNamespace takes a SubNamespace resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than SubNamespace.
func (c *Controller) enqueueSubNamespace(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *Controller) applyProcedure(subnamespaceCopy *corev1alpha.SubNamespace) {
	tenantEnabled := false
	parentNamespace, _ := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), subnamespaceCopy.GetNamespace(), metav1.GetOptions{})
	labels := parentNamespace.GetLabels()

	if labels != nil && labels["edge-net.io/tenant"] != "" {
		if tenant, err := c.edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), labels["edge-net.io/tenant"], metav1.GetOptions{}); err == nil {
			tenantEnabled = tenant.Spec.Enabled
		}
	} else {
		return
	}

	subNamespaceStr := fmt.Sprintf("%s-%s", labels["edge-net.io/tenant"], subnamespaceCopy.GetName())
	if tenantEnabled {
		oldStatus := subnamespaceCopy.Status
		statusUpdate := func() {
			if !reflect.DeepEqual(oldStatus, subnamespaceCopy.Status) {
				c.edgenetclientset.CoreV1alpha().SubNamespaces(subnamespaceCopy.GetNamespace()).UpdateStatus(context.TODO(), subnamespaceCopy, metav1.UpdateOptions{})
			}
		}
		defer statusUpdate()
		// Flush the status
		subnamespaceCopy.Status = corev1alpha.SubNamespaceStatus{}

		cpuResource := resource.MustParse(subnamespaceCopy.Spec.Resources.CPU)
		cpuDemand := cpuResource.Value()
		memoryResource := resource.MustParse(subnamespaceCopy.Spec.Resources.Memory)
		memoryDemand := memoryResource.Value()
		if parentResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(subnamespaceCopy.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-quota", labels["edge-net.io/kind"]), metav1.GetOptions{}); err == nil {
			if parentResourceQuota.Spec.Hard.Cpu().Value() < cpuDemand && parentResourceQuota.Spec.Hard.Memory().Value() < memoryDemand {
				subnamespaceCopy.Status.State = failure
				subnamespaceCopy.Status.Message = []string{statusDict["quota-exceeded"]}
			} else {
				childNamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), subNamespaceStr, metav1.GetOptions{})
				if err == nil {
					childNamespaceLabels := childNamespace.GetLabels()
					if childNamespaceLabels["edge-net.io/owner"] != subnamespaceCopy.GetName() && childNamespaceLabels["edge-net.io/ownerNamespace"] != subnamespaceCopy.GetNamespace() {
						// TODO: Error handling
						subnamespaceCopy.Status.State = failure
						subnamespaceCopy.Status.Message = []string{fmt.Sprintf(statusDict["namespace-exists"], subNamespaceStr)}
						return
					}
				} else {
					ownerReferences := namespace.SetAsOwnerReference(parentNamespace)
					childNamespaceObj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: subNamespaceStr, OwnerReferences: ownerReferences}}
					namespaceLabels := map[string]string{"edge-net.io/generated": "true", "edge-net.io/kind": "sub", "edge-net.io/tenant": labels["edge-net.io/tenant"], "edge-net.io/owner": subnamespaceCopy.GetName(), "edge-net.io/ownerNamespace": subnamespaceCopy.GetNamespace()}
					childNamespaceObj.SetLabels(namespaceLabels)

					childNamespace, err = c.kubeclientset.CoreV1().Namespaces().Create(context.TODO(), childNamespaceObj, metav1.CreateOptions{})
					if err != nil {
						// TODO: Error handling
						subnamespaceCopy.Status.State = failure
						subnamespaceCopy.Status.Message = []string{statusDict["subnamespace-failure"]}
					}
				}

				remainingCPUQuota := cpuDemand
				remainingMemoryQuota := memoryDemand

				parentQuotaCPU := parentResourceQuota.Spec.Hard.Cpu().Value()
				parentQuotaMemory := parentResourceQuota.Spec.Hard.Memory().Value()
				parentQuotaCPU -= cpuDemand
				parentQuotaMemory -= memoryDemand
				if subResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(childNamespace.GetName()).Get(context.TODO(), "sub-quota", metav1.GetOptions{}); err == nil {
					parentQuotaCPU += subResourceQuota.Spec.Hard.Cpu().Value()
					parentQuotaMemory += subResourceQuota.Spec.Hard.Memory().Value()

					var traverseSubnamespace = func() (int64, int64, *corev1alpha.SubNamespace) {
						subNamespaceRaw, _ := c.edgenetclientset.CoreV1alpha().SubNamespaces(childNamespace.GetName()).List(context.TODO(), metav1.ListOptions{})
						var aggregatedCPU, aggregatedMemory int64 = 0, 0
						var lastInDate metav1.Time
						var lastInSubNamespace *corev1alpha.SubNamespace
						if len(subNamespaceRaw.Items) != 0 {
							for _, subNamespaceRow := range subNamespaceRaw.Items {
								if subNamespaceRow.Status.State != failure {
									subCPUResource := resource.MustParse(subNamespaceRow.Spec.Resources.CPU)
									aggregatedCPU += subCPUResource.Value()
									subMemoryResource := resource.MustParse(subNamespaceRow.Spec.Resources.Memory)
									aggregatedMemory += subMemoryResource.Value()

									if lastInDate.IsZero() || lastInDate.Sub(subNamespaceRow.GetCreationTimestamp().Time) >= 0 {
										lastInSubNamespace = subNamespaceRow.DeepCopy()
										lastInDate = subNamespaceRow.GetCreationTimestamp()
									}
								}
							}
						}
						return aggregatedCPU, aggregatedMemory, lastInSubNamespace
					}
					aggregatedCPU, aggregatedMemory, lastInSubNamespace := traverseSubnamespace()
					parentQuotaCPU += aggregatedCPU
					parentQuotaMemory += aggregatedMemory

					var tuneResourceQuota func(aggregatedCPU, aggregatedMemory int64, lastInSubNamespace *corev1alpha.SubNamespace) (int64, int64)
					tuneResourceQuota = func(aggregatedCPU, aggregatedMemory int64, lastInSubNamespace *corev1alpha.SubNamespace) (int64, int64) {
						var tunedCPU, tunedMemory int64 = aggregatedCPU, aggregatedMemory
						if cpuDemand < aggregatedCPU || memoryDemand < aggregatedMemory {
							if lastInSubNamespace != nil {
								c.edgenetclientset.CoreV1alpha().SubNamespaces(lastInSubNamespace.GetNamespace()).Delete(context.TODO(), lastInSubNamespace.GetName(), metav1.DeleteOptions{})
								aggregatedCPU, aggregatedMemory, lastInSubNamespace = traverseSubnamespace()
								tunedCPU, tunedMemory = tuneResourceQuota(aggregatedCPU, aggregatedMemory, lastInSubNamespace)
							}
						}
						return tunedCPU, tunedMemory
					}
					tunedCPU, tunedMemory := tuneResourceQuota(aggregatedCPU, aggregatedMemory, lastInSubNamespace)
					remainingCPUQuota -= tunedCPU
					remainingMemoryQuota -= tunedMemory

					subResourceQuota.Spec.Hard["cpu"] = *resource.NewQuantity(remainingCPUQuota, resource.DecimalSI)
					subResourceQuota.Spec.Hard["memory"] = *resource.NewQuantity(remainingMemoryQuota, resource.BinarySI)
					c.kubeclientset.CoreV1().ResourceQuotas(childNamespace.GetName()).Update(context.TODO(), subResourceQuota, metav1.UpdateOptions{})
				} else {
					subResourceQuota := &corev1.ResourceQuota{}
					subResourceQuota.Name = "sub-quota"
					subResourceQuota.Spec = corev1.ResourceQuotaSpec{
						Hard: map[corev1.ResourceName]resource.Quantity{
							"cpu":    cpuResource,
							"memory": memoryResource,
						},
					}
					c.kubeclientset.CoreV1().ResourceQuotas(childNamespace.GetName()).Create(context.TODO(), subResourceQuota, metav1.CreateOptions{})
				}

				parentResourceQuota.Spec.Hard["cpu"] = *resource.NewQuantity(parentQuotaCPU, resource.DecimalSI)
				parentResourceQuota.Spec.Hard["memory"] = *resource.NewQuantity(parentQuotaMemory, resource.BinarySI)
				c.kubeclientset.CoreV1().ResourceQuotas(parentResourceQuota.GetNamespace()).Update(context.TODO(), parentResourceQuota, metav1.UpdateOptions{})

				if roleRaw, err := c.kubeclientset.RbacV1().Roles(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Inheritance.RBAC {
					// TODO: Provide err information at the status
					for _, roleRow := range roleRaw.Items {
						role := roleRow.DeepCopy()
						role.SetNamespace(childNamespace.GetName())
						role.SetUID(types.UID(uuid.New().String()))
						c.kubeclientset.RbacV1().Roles(childNamespace.GetName()).Create(context.TODO(), role, metav1.CreateOptions{})
					}
				}
				if roleBindingRaw, err := c.kubeclientset.RbacV1().RoleBindings(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Inheritance.RBAC {
					// TODO: Provide err information at the status
					for _, roleBindingRow := range roleBindingRaw.Items {
						roleBinding := roleBindingRow.DeepCopy()
						roleBinding.SetNamespace(childNamespace.GetName())
						roleBinding.SetUID(types.UID(uuid.New().String()))
						c.kubeclientset.RbacV1().RoleBindings(childNamespace.GetName()).Create(context.TODO(), roleBinding, metav1.CreateOptions{})
					}
				}
				if networkPolicyRaw, err := c.kubeclientset.NetworkingV1().NetworkPolicies(subnamespaceCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespaceCopy.Spec.Inheritance.NetworkPolicy {
					// TODO: Provide err information at the status
					for _, networkPolicyRow := range networkPolicyRaw.Items {
						networkPolicy := networkPolicyRow.DeepCopy()
						networkPolicy.SetNamespace(childNamespace.GetName())
						networkPolicy.SetUID(types.UID(uuid.New().String()))
						c.kubeclientset.NetworkingV1().NetworkPolicies(childNamespace.GetName()).Create(context.TODO(), networkPolicy, metav1.CreateOptions{})
					}
				}
				subnamespaceCopy.Status.State = established
				subnamespaceCopy.Status.Message = []string{statusDict["subnamespace-ok"]}
			}
		}
	}
}

// RunExpiryController puts a procedure in place to turn accepted policies into not accepted
func (c *Controller) RunExpiryController() {
	var closestExpiry time.Time
	terminated := make(chan bool)
	newExpiry := make(chan time.Time)
	defer close(terminated)
	defer close(newExpiry)

	watchSubNamespace, err := c.edgenetclientset.CoreV1alpha().SubNamespaces("").Watch(context.TODO(), metav1.ListOptions{})
	if err == nil {
		watchEvents := func(watchSubNamespace watch.Interface, newExpiry *chan time.Time) {
			// Watch the events of subsidiary namespace object
			// Get events from watch interface
			for subNamespaceEvent := range watchSubNamespace.ResultChan() {
				// Get updated subsidiary namespace object
				updatedSubNamespace, status := subNamespaceEvent.Object.(*corev1alpha.SubNamespace)
				if status {
					if subNamespaceEvent.Type == "DELETED" {
						parentNamespace, _ := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), updatedSubNamespace.GetNamespace(), metav1.GetOptions{})
						parentNamespaceLabels := parentNamespace.GetLabels()
						if parentNamespaceLabels != nil && parentNamespaceLabels["edge-net.io/tenant"] != "" {
							if childNamespaceObj, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-%s", parentNamespaceLabels["edge-net.io/tenant"], updatedSubNamespace.GetName()), metav1.GetOptions{}); err == nil {
								childNamespaceObjLabels := childNamespaceObj.GetLabels()
								if childNamespaceObjLabels != nil && childNamespaceObjLabels["edge-net.io/generated"] == "true" && childNamespaceObjLabels["edge-net.io/owner"] == updatedSubNamespace.GetName() && childNamespaceObjLabels["edge-net.io/ownerNamespace"] == updatedSubNamespace.GetNamespace() {
									if parentResourceQuota, err := c.kubeclientset.CoreV1().ResourceQuotas(updatedSubNamespace.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-quota", parentNamespaceLabels["edge-net.io/kind"]), metav1.GetOptions{}); err == nil {
										cpuResource := resource.MustParse(updatedSubNamespace.Spec.Resources.CPU)
										cpuQuota := cpuResource.Value()
										memoryResource := resource.MustParse(updatedSubNamespace.Spec.Resources.Memory)
										memoryQuota := memoryResource.Value()

										parentQuotaCPU := parentResourceQuota.Spec.Hard.Cpu().Value()
										parentQuotaMemory := parentResourceQuota.Spec.Hard.Memory().Value()

										parentQuotaCPU += cpuQuota
										parentQuotaMemory += memoryQuota

										parentResourceQuota.Spec.Hard["cpu"] = *resource.NewQuantity(parentQuotaCPU, resource.DecimalSI)
										parentResourceQuota.Spec.Hard["memory"] = *resource.NewQuantity(parentQuotaMemory, resource.BinarySI)
										c.kubeclientset.CoreV1().ResourceQuotas(parentResourceQuota.GetNamespace()).Update(context.TODO(), parentResourceQuota, metav1.UpdateOptions{})

										c.kubeclientset.CoreV1().Namespaces().Delete(context.TODO(), childNamespaceObj.GetName(), metav1.DeleteOptions{})
									}
								}
							}
						}
						continue
					}

					if updatedSubNamespace.Spec.Expiry != nil {
						*newExpiry <- updatedSubNamespace.Spec.Expiry.Time
					}
				}
			}
		}
		go watchEvents(watchSubNamespace, &newExpiry)
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
			subNamespaceRaw, err := c.edgenetclientset.CoreV1alpha().SubNamespaces("").List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				// TO-DO: Provide more information on error
				log.Println(err)
			}
			for _, subNamespaceRow := range subNamespaceRaw.Items {
				if subNamespaceRow.Spec.Expiry != nil && subNamespaceRow.Spec.Expiry.Time.Sub(time.Now()) <= 0 {
					c.edgenetclientset.CoreV1alpha().SubNamespaces(subNamespaceRow.GetNamespace()).Delete(context.TODO(), subNamespaceRow.GetName(), metav1.DeleteOptions{})
				} else if subNamespaceRow.Spec.Expiry != nil && subNamespaceRow.Spec.Expiry.Time.Sub(time.Now()) > 0 {
					if closestExpiry.Sub(time.Now()) <= 0 || closestExpiry.Sub(subNamespaceRow.Spec.Expiry.Time) > 0 {
						closestExpiry = subNamespaceRow.Spec.Expiry.Time
						log.Printf("ExpiryController: Closest expiry date is %v after the expiration of a subsidiary namespace", closestExpiry)
					}
				}
			}

			if closestExpiry.Sub(time.Now()) <= 0 {
				closestExpiry = time.Now().AddDate(1, 0, 0)
				log.Printf("ExpiryController: Closest expiry date is %v after the expiration of a subsidiary namespace", closestExpiry)
			}
		case <-terminated:
			watchSubNamespace.Stop()
			break infiniteLoop
		}
	}
}
