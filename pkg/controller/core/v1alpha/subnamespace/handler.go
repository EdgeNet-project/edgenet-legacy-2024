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
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/namespace"
	"github.com/google/uuid"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init(kubernetes kubernetes.Interface, edgenet versioned.Interface)
	ObjectCreatedOrUpdated(obj interface{})
	ObjectDeleted(obj interface{})
	RunExpiryController()
}

// Handler implementation
type Handler struct {
	clientset        kubernetes.Interface
	edgenetClientset versioned.Interface
}

// Init handles any handler initialization
func (t *Handler) Init(kubernetes kubernetes.Interface, edgenet versioned.Interface) {
	log.Info("SubNamespaceHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
}

// ObjectCreatedOrUpdated is called when an object is created
func (t *Handler) ObjectCreatedOrUpdated(obj interface{}) {
	log.Info("SubNamespaceHandler.ObjectCreatedOrUpdated")
	// Make a copy of the subNamespace object to make changes on it
	subNamespace := obj.(*corev1alpha.SubNamespace).DeepCopy()

	tenantEnabled := false
	parentNamespace, _ := t.clientset.CoreV1().Namespaces().Get(context.TODO(), subNamespace.GetNamespace(), metav1.GetOptions{})
	labels := parentNamespace.GetLabels()

	if labels != nil && labels["edge-net.io/tenant"] != "" {
		if tenant, err := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), labels["edge-net.io/tenant"], metav1.GetOptions{}); err == nil {
			tenantEnabled = tenant.Spec.Enabled
		}
	} else {
		return
	}

	subNamespaceStr := fmt.Sprintf("%s-%s", labels["edge-net.io/tenant"], subNamespace.GetName())
	if tenantEnabled {
		defer func() {
			if !reflect.DeepEqual(obj.(*corev1alpha.SubNamespace).Status, subNamespace.Status) {
				if _, err := t.edgenetClientset.CoreV1alpha().SubNamespaces(subNamespace.GetNamespace()).UpdateStatus(context.TODO(), subNamespace, metav1.UpdateOptions{}); err != nil {
					// TO-DO: Provide more information on error
					log.Println(err)
				}
			}
		}()

		cpuResource := resource.MustParse(subNamespace.Spec.Resources.CPU)
		cpuDemand := cpuResource.Value()
		memoryResource := resource.MustParse(subNamespace.Spec.Resources.Memory)
		memoryDemand := memoryResource.Value()
		if parentResourceQuota, err := t.clientset.CoreV1().ResourceQuotas(subNamespace.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-quota", labels["edge-net.io/kind"]), metav1.GetOptions{}); err == nil {
			if parentResourceQuota.Spec.Hard.Cpu().Value() < cpuDemand && parentResourceQuota.Spec.Hard.Memory().Value() < memoryDemand {
				subNamespace.Status.State = failure
				subNamespace.Status.Message = []string{statusDict["quota-exceeded"]}
			} else {
				childNamespace, err := t.clientset.CoreV1().Namespaces().Get(context.TODO(), subNamespaceStr, metav1.GetOptions{})
				if err == nil {
					childNamespaceLabels := childNamespace.GetLabels()
					if childNamespaceLabels["edge-net.io/owner"] != subNamespace.GetName() && childNamespaceLabels["edge-net.io/ownerNamespace"] != subNamespace.GetNamespace() {
						// TODO: Error handling
						subNamespace.Status.State = failure
						subNamespace.Status.Message = []string{fmt.Sprintf(statusDict["namespace-exists"], subNamespaceStr)}
						return
					}
				} else {
					ownerReferences := namespace.SetAsOwnerReference(parentNamespace)
					childNamespaceObj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: subNamespaceStr, OwnerReferences: ownerReferences}}
					namespaceLabels := map[string]string{"edge-net.io/generated": "true", "edge-net.io/kind": "sub", "edge-net.io/tenant": labels["edge-net.io/tenant"], "edge-net.io/owner": subNamespace.GetName(), "edge-net.io/ownerNamespace": subNamespace.GetNamespace()}
					childNamespaceObj.SetLabels(namespaceLabels)

					childNamespace, err = t.clientset.CoreV1().Namespaces().Create(context.TODO(), childNamespaceObj, metav1.CreateOptions{})
					if err != nil {
						// TODO: Error handling
						subNamespace.Status.State = failure
						subNamespace.Status.Message = []string{statusDict["subnamespace-failure"]}
					}
				}

				remainingCPUQuota := cpuDemand
				remainingMemoryQuota := memoryDemand

				parentQuotaCPU := parentResourceQuota.Spec.Hard.Cpu().Value()
				parentQuotaMemory := parentResourceQuota.Spec.Hard.Memory().Value()
				parentQuotaCPU -= cpuDemand
				parentQuotaMemory -= memoryDemand
				if subResourceQuota, err := t.clientset.CoreV1().ResourceQuotas(childNamespace.GetName()).Get(context.TODO(), "sub-quota", metav1.GetOptions{}); err == nil {
					parentQuotaCPU += subResourceQuota.Spec.Hard.Cpu().Value()
					parentQuotaMemory += subResourceQuota.Spec.Hard.Memory().Value()

					var traverseSubnamespace = func() (int64, int64, *corev1alpha.SubNamespace) {
						subNamespaceRaw, _ := t.edgenetClientset.CoreV1alpha().SubNamespaces(childNamespace.GetName()).List(context.TODO(), metav1.ListOptions{})
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
								t.edgenetClientset.CoreV1alpha().SubNamespaces(lastInSubNamespace.GetNamespace()).Delete(context.TODO(), lastInSubNamespace.GetName(), metav1.DeleteOptions{})
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
					t.clientset.CoreV1().ResourceQuotas(childNamespace.GetName()).Update(context.TODO(), subResourceQuota, metav1.UpdateOptions{})
				} else {
					subResourceQuota := &corev1.ResourceQuota{}
					subResourceQuota.Name = "sub-quota"
					subResourceQuota.Spec = corev1.ResourceQuotaSpec{
						Hard: map[corev1.ResourceName]resource.Quantity{
							"cpu":    cpuResource,
							"memory": memoryResource,
						},
					}
					t.clientset.CoreV1().ResourceQuotas(childNamespace.GetName()).Create(context.TODO(), subResourceQuota, metav1.CreateOptions{})
				}

				parentResourceQuota.Spec.Hard["cpu"] = *resource.NewQuantity(parentQuotaCPU, resource.DecimalSI)
				parentResourceQuota.Spec.Hard["memory"] = *resource.NewQuantity(parentQuotaMemory, resource.BinarySI)
				t.clientset.CoreV1().ResourceQuotas(parentResourceQuota.GetNamespace()).Update(context.TODO(), parentResourceQuota, metav1.UpdateOptions{})

				if roleRaw, err := t.clientset.RbacV1().Roles(subNamespace.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subNamespace.Spec.Inheritance.RBAC {
					// TODO: Provide err information at the status
					for _, roleRow := range roleRaw.Items {
						role := roleRow.DeepCopy()
						role.SetNamespace(childNamespace.GetName())
						role.SetUID(types.UID(uuid.New().String()))
						t.clientset.RbacV1().Roles(childNamespace.GetName()).Create(context.TODO(), role, metav1.CreateOptions{})
					}
				}
				if roleBindingRaw, err := t.clientset.RbacV1().RoleBindings(subNamespace.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subNamespace.Spec.Inheritance.RBAC {
					// TODO: Provide err information at the status
					for _, roleBindingRow := range roleBindingRaw.Items {
						roleBinding := roleBindingRow.DeepCopy()
						roleBinding.SetNamespace(childNamespace.GetName())
						roleBinding.SetUID(types.UID(uuid.New().String()))
						t.clientset.RbacV1().RoleBindings(childNamespace.GetName()).Create(context.TODO(), roleBinding, metav1.CreateOptions{})
					}
				}
				if networkPolicyRaw, err := t.clientset.NetworkingV1().NetworkPolicies(subNamespace.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subNamespace.Spec.Inheritance.NetworkPolicy {
					// TODO: Provide err information at the status
					for _, networkPolicyRow := range networkPolicyRaw.Items {
						networkPolicy := networkPolicyRow.DeepCopy()
						networkPolicy.SetNamespace(childNamespace.GetName())
						networkPolicy.SetUID(types.UID(uuid.New().String()))
						t.clientset.NetworkingV1().NetworkPolicies(childNamespace.GetName()).Create(context.TODO(), networkPolicy, metav1.CreateOptions{})
					}
				}
				subNamespace.Status.State = established
				subNamespace.Status.Message = []string{statusDict["subnamespace-ok"]}
			}
		}
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("SubNamespaceHandler.ObjectDeleted")
	// Delete the namespace created by subsidiary namespace, TBD.
}

// RunExpiryController puts a procedure in place to turn accepted policies into not accepted
func (t *Handler) RunExpiryController() {
	var closestExpiry time.Time
	terminated := make(chan bool)
	newExpiry := make(chan time.Time)
	defer close(terminated)
	defer close(newExpiry)

	watchSubNamespace, err := t.edgenetClientset.CoreV1alpha().SubNamespaces("").Watch(context.TODO(), metav1.ListOptions{})
	if err == nil {
		watchEvents := func(watchSubNamespace watch.Interface, newExpiry *chan time.Time) {
			// Watch the events of subsidiary namespace object
			// Get events from watch interface
			for subNamespaceEvent := range watchSubNamespace.ResultChan() {
				// Get updated subsidiary namespace object
				updatedSubNamespace, status := subNamespaceEvent.Object.(*corev1alpha.SubNamespace)
				if status {
					if subNamespaceEvent.Type == "DELETED" {
						parentNamespace, _ := t.clientset.CoreV1().Namespaces().Get(context.TODO(), updatedSubNamespace.GetNamespace(), metav1.GetOptions{})
						parentNamespaceLabels := parentNamespace.GetLabels()
						if parentNamespaceLabels != nil && parentNamespaceLabels["edge-net.io/tenant"] != "" {
							if childNamespaceObj, err := t.clientset.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-%s", parentNamespaceLabels["edge-net.io/tenant"], updatedSubNamespace.GetName()), metav1.GetOptions{}); err == nil {
								childNamespaceObjLabels := childNamespaceObj.GetLabels()
								if childNamespaceObjLabels != nil && childNamespaceObjLabels["edge-net.io/generated"] == "true" && childNamespaceObjLabels["edge-net.io/owner"] == updatedSubNamespace.GetName() && childNamespaceObjLabels["edge-net.io/ownerNamespace"] == updatedSubNamespace.GetNamespace() {
									if parentResourceQuota, err := t.clientset.CoreV1().ResourceQuotas(updatedSubNamespace.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-quota", parentNamespaceLabels["edge-net.io/kind"]), metav1.GetOptions{}); err == nil {
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
										t.clientset.CoreV1().ResourceQuotas(parentResourceQuota.GetNamespace()).Update(context.TODO(), parentResourceQuota, metav1.UpdateOptions{})

										t.clientset.CoreV1().Namespaces().Delete(context.TODO(), childNamespaceObj.GetName(), metav1.DeleteOptions{})
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
		go t.RunExpiryController()
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
			subNamespaceRaw, err := t.edgenetClientset.CoreV1alpha().SubNamespaces("").List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				// TO-DO: Provide more information on error
				log.Println(err)
			}
			for _, subNamespaceRow := range subNamespaceRaw.Items {
				if subNamespaceRow.Spec.Expiry != nil && subNamespaceRow.Spec.Expiry.Time.Sub(time.Now()) <= 0 {
					t.edgenetClientset.CoreV1alpha().SubNamespaces(subNamespaceRow.GetNamespace()).Delete(context.TODO(), subNamespaceRow.GetName(), metav1.DeleteOptions{})
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
