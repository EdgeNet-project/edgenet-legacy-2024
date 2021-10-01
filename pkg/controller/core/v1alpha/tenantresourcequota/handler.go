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

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	log.Info("tenantResourceQuotaHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
}

// ObjectCreatedOrUpdated is called when an object is created or updated
func (t *Handler) ObjectCreatedOrUpdated(obj interface{}) {
	log.Info("tenantResourceQuotaHandler.ObjectCreated")
	// Make a copy of the tenant resource quota object to make changes on it
	tenantResourceQuota := obj.(*corev1alpha.TenantResourceQuota).DeepCopy()
	tenant, err := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	if errors.IsNotFound(err) {
		t.edgenetClientset.CoreV1alpha().TenantResourceQuotas().Delete(context.TODO(), tenant.GetName(), metav1.DeleteOptions{})
	} else {
		if tenant.Spec.Enabled {
			if tenantResourceQuota.Status.State != success {
				defer func() {
					tenantResourceQuota.Status.State = success
					tenantResourceQuota.Status.Message = []string{statusDict["TRQ-created"]}
					if _, err := t.edgenetClientset.CoreV1alpha().TenantResourceQuotas().UpdateStatus(context.TODO(), tenantResourceQuota, metav1.UpdateOptions{}); err != nil {
						// TODO: Provide more information on error
						if err != nil {
							log.Println(err)
						}
					}
				}()
			}
			t.tuneResourceQuota(tenant.GetName(), tenantResourceQuota)
		}
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("tenantResourceQuotaHandler.ObjectDeleted")
	// Delete or disable subnamespaces added by tenant, TBD.
}

// Create generates a tenant resource quota with the name provided
func (t *Handler) Create(name string, ownerReferences []metav1.OwnerReference) (string, string) {
	cpuQuota := "0m"
	memoryQuota := "0Mi"
	_, err := t.edgenetClientset.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		// Set a tenant resource quota
		tenantResourceQuota := corev1alpha.TenantResourceQuota{}
		tenantResourceQuota.SetName(name)
		tenantResourceQuota.SetOwnerReferences(ownerReferences)
		claim := corev1alpha.TenantResourceDetails{}
		claim.Name = "Default"
		claim.CPU = "8000m"
		claim.Memory = "8192Mi"
		tenantResourceQuota.Spec.Claim = append(tenantResourceQuota.Spec.Claim, claim)
		_, err = t.edgenetClientset.CoreV1alpha().TenantResourceQuotas().Create(context.TODO(), tenantResourceQuota.DeepCopy(), metav1.CreateOptions{})
		if err != nil {
			log.Infof(statusDict["TRQ-failed"], name, err)
		}

		cpuQuota = claim.CPU
		memoryQuota = claim.Memory
	}
	return cpuQuota, memoryQuota
}

func (t *Handler) NamespaceTraversal(coreNamespace string) (int64, int64, *corev1alpha.SubNamespace) {
	// Get the total consumption that all namespaces do in tenant
	var aggregatedCPU, aggregatedMemory int64 = 0, 0
	var lastInDate metav1.Time
	var lastInSubNamespace *corev1alpha.SubNamespace
	t.traverse(coreNamespace, coreNamespace, &aggregatedCPU, &aggregatedMemory, lastInSubNamespace, &lastInDate)
	return aggregatedCPU, aggregatedMemory, lastInSubNamespace
}

func (t *Handler) traverse(coreNamespace, namespace string, aggregatedCPU *int64, aggregatedMemory *int64, lastInSubNamespace *corev1alpha.SubNamespace, lastInDate *metav1.Time) {
	t.aggregateQuota(coreNamespace, aggregatedCPU, aggregatedMemory)

	subNamespaceRaw, _ := t.edgenetClientset.CoreV1alpha().SubNamespaces(namespace).List(context.TODO(), metav1.ListOptions{})
	if len(subNamespaceRaw.Items) != 0 {
		for _, subNamespaceRow := range subNamespaceRaw.Items {
			if lastInDate.IsZero() || lastInDate.Sub(subNamespaceRow.GetCreationTimestamp().Time) >= 0 {
				*lastInSubNamespace = subNamespaceRow
				*lastInDate = subNamespaceRow.GetCreationTimestamp()
			}
			subNamespaceStr := fmt.Sprintf("%s-%s", coreNamespace, subNamespaceRow.GetName())
			t.traverse(coreNamespace, subNamespaceStr, aggregatedCPU, aggregatedMemory, lastInSubNamespace, lastInDate)
		}
	}
}

func (t *Handler) aggregateQuota(namespace string, aggregatedCPU *int64, aggregatedMemory *int64) {
	resourceQuotasRaw, _ := t.clientset.CoreV1().ResourceQuotas(namespace).List(context.TODO(), metav1.ListOptions{})
	if len(resourceQuotasRaw.Items) != 0 {
		for _, resourceQuotasRow := range resourceQuotasRaw.Items {
			*aggregatedCPU += resourceQuotasRow.Spec.Hard.Cpu().Value()
			*aggregatedMemory += resourceQuotasRow.Spec.Hard.Memory().Value()
		}
	}
}

// calculateTenantQuota adds the resources defined in claims, and subtracts those in drops to calculate the tenant resource quota.
func (t *Handler) calculateTenantQuota(tenantResourceQuota *corev1alpha.TenantResourceQuota) (int64, int64) {
	var cpuQuota int64
	var memoryQuota int64
	if len(tenantResourceQuota.Spec.Claim) > 0 {
		for _, claim := range tenantResourceQuota.Spec.Claim {
			if claim.Expiry == nil || (claim.Expiry != nil && claim.Expiry.Time.Sub(time.Now()) >= 0) {
				cpuResource := resource.MustParse(claim.CPU)
				cpuQuota += cpuResource.Value()
				memoryResource := resource.MustParse(claim.Memory)
				memoryQuota += memoryResource.Value()
			}
		}
	}
	if len(tenantResourceQuota.Spec.Drop) > 0 {
		for _, drop := range tenantResourceQuota.Spec.Drop {
			if drop.Expiry == nil || (drop.Expiry != nil && drop.Expiry.Time.Sub(time.Now()) >= 0) {
				cpuResource := resource.MustParse(drop.CPU)
				cpuQuota -= cpuResource.Value()
				memoryResource := resource.MustParse(drop.Memory)
				memoryQuota -= memoryResource.Value()
			}
		}
	}
	return cpuQuota, memoryQuota
}

func (t *Handler) tuneResourceQuota(coreNamespace string, tenantResourceQuota *corev1alpha.TenantResourceQuota) {
	aggregatedCPU, aggregatedMemory, lastInSubNamespace := t.NamespaceTraversal(coreNamespace)
	cpuQuota, memoryQuota := t.calculateTenantQuota(tenantResourceQuota)
	if cpuQuota < aggregatedCPU || memoryQuota < aggregatedMemory {
		cpuShortage := aggregatedCPU - cpuQuota
		memoryShortage := aggregatedMemory - memoryQuota
		coreResourceQuota, err := t.clientset.CoreV1().ResourceQuotas(coreNamespace).Get(context.TODO(), "core-quota", metav1.GetOptions{})
		if err == nil {
			coreCPUQuota := coreResourceQuota.Spec.Hard.Cpu().DeepCopy()
			coreMemoryQuota := coreResourceQuota.Spec.Hard.Memory().DeepCopy()
			if coreCPUQuota.Value() >= cpuShortage && coreMemoryQuota.Value() >= memoryShortage {
				coreCPUQuota.Set(coreCPUQuota.Value() - cpuShortage)
				coreResourceQuota.Spec.Hard["cpu"] = coreCPUQuota
				coreMemoryQuota.Set(coreMemoryQuota.Value() - memoryShortage)
				coreResourceQuota.Spec.Hard["memory"] = coreMemoryQuota
				t.clientset.CoreV1().ResourceQuotas(coreNamespace).Update(context.TODO(), coreResourceQuota, metav1.UpdateOptions{})
			} else {
				if lastInSubNamespace != nil {
					t.edgenetClientset.CoreV1alpha().SubNamespaces(lastInSubNamespace.GetNamespace()).Delete(context.TODO(), lastInSubNamespace.GetName(), metav1.DeleteOptions{})
					time.Sleep(200 * time.Millisecond)
					defer t.tuneResourceQuota(coreNamespace, tenantResourceQuota)
				}
			}
		}
	} else if cpuQuota > aggregatedCPU || memoryQuota > aggregatedMemory {
		cpuLacune := cpuQuota - aggregatedCPU
		memoryLacune := memoryQuota - aggregatedMemory
		coreResourceQuota, err := t.clientset.CoreV1().ResourceQuotas(coreNamespace).Get(context.TODO(), "core-quota", metav1.GetOptions{})
		if err == nil {
			coreCPUQuota := coreResourceQuota.Spec.Hard.Cpu().DeepCopy()
			coreMemoryQuota := coreResourceQuota.Spec.Hard.Memory().DeepCopy()
			coreCPUQuota.Set(coreCPUQuota.Value() + cpuLacune)
			coreResourceQuota.Spec.Hard["cpu"] = coreCPUQuota
			coreMemoryQuota.Set(coreMemoryQuota.Value() + memoryLacune)
			coreResourceQuota.Spec.Hard["memory"] = coreMemoryQuota
			t.clientset.CoreV1().ResourceQuotas(coreNamespace).Update(context.TODO(), coreResourceQuota, metav1.UpdateOptions{})
		}
	}
}

// RunExpiryController puts a procedure in place to remove claims and drops after the timeout
func (t *Handler) RunExpiryController() {
	var closestExpiry time.Time = time.Now().AddDate(1, 0, 0)
	terminated := make(chan bool)
	newExpiry := make(chan time.Time)
	defer close(terminated)
	defer close(newExpiry)

	watchTenantResourceQuota, err := t.edgenetClientset.CoreV1alpha().TenantResourceQuotas().Watch(context.TODO(), metav1.ListOptions{})
	if err == nil {
		watchEvents := func(watchTenantResourceQuota watch.Interface, newExpiry *chan time.Time) {
			// Watch the events
			// Get events from watch interface
			for tenantResourceQuotaEvent := range watchTenantResourceQuota.ResultChan() {
				updatedTenantResourceQuota, status := tenantResourceQuotaEvent.Object.(*corev1alpha.TenantResourceQuota)
				if status {
					expiry, exists := t.getClosestExpiryDate(updatedTenantResourceQuota)
					if exists {
						if closestExpiry.Sub(expiry) > 0 {
							*newExpiry <- expiry
						}
					}
				}
			}
		}
		go watchEvents(watchTenantResourceQuota, &newExpiry)
	} else {
		go t.RunExpiryController()
		terminated <- true
	}

infiniteLoop:
	for {
		// Wait on multiple channel operations
		select {
		case timeout := <-newExpiry:
			closestExpiry = timeout
			log.Printf("ExpiryController: Sooner expiry date is %v", closestExpiry)
		case <-time.After(time.Until(closestExpiry)):
			tenantResourceQuotaRaw, err := t.edgenetClientset.CoreV1alpha().TenantResourceQuotas().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				// TODO: Provide more information on error
				log.Println(err)
			}
			soonerDate := closestExpiry
			for _, tenantResourceQuotaRow := range tenantResourceQuotaRaw.Items {
				if expiry, exists := t.getClosestExpiryDate(&tenantResourceQuotaRow); exists {
					if soonerDate.Sub(expiry) > 0 {
						soonerDate = expiry
					}
				}
			}

			if soonerDate.Sub(closestExpiry) > 0 {
				newExpiry <- soonerDate
			} else {
				newExpiry <- time.Now().AddDate(1, 0, 0)
			}
		case <-terminated:
			watchTenantResourceQuota.Stop()
			break infiniteLoop
		}
	}
}

// getClosestExpiryDate determines the item, a claim or a drop, having the closest expiry date
func (t *Handler) getClosestExpiryDate(tenantResourceQuota *corev1alpha.TenantResourceQuota) (time.Time, bool) {
	// To make comparison
	oldTenantResourceQuota := tenantResourceQuota.DeepCopy()
	// claimSlice to be manipulated
	claimSlice := tenantResourceQuota.Spec.DeepCopy().Claim
	// dropSlice to be manipulated
	dropSlice := tenantResourceQuota.Spec.DeepCopy().Drop

	soonerDate := time.Now().AddDate(1, 0, 0)
	exists := false
	i := 0
	for _, claim := range tenantResourceQuota.Spec.Claim {
		if claim.Expiry != nil {
			if claim.Expiry.Time.Sub(time.Now().Add(1*time.Second)) > 0 {
				if soonerDate.Sub(claim.Expiry.Time) >= 0 {
					soonerDate = claim.Expiry.Time
				}
				exists = true
			} else {
				// Remove the item from claims if the expiry date has run out
				claimSlice = append(claimSlice[:i], claimSlice[i+1:]...)
				i--
			}
		}
		i++
	}
	tenantResourceQuota.Spec.Claim = claimSlice
	j := 0
	for _, dropRow := range tenantResourceQuota.Spec.Drop {
		if dropRow.Expiry != nil {
			if dropRow.Expiry.Time.Sub(time.Now().Add(1*time.Second)) > 0 {
				if soonerDate.Sub(dropRow.Expiry.Time) >= 0 {
					soonerDate = dropRow.Expiry.Time
					exists = true
				}
			} else {
				// Remove the item from drops if the expiry date has run out
				dropSlice = append(dropSlice[:j], dropSlice[j+1:]...)
				j--
			}
		}
		j++
	}
	tenantResourceQuota.Spec.Drop = dropSlice

	if !reflect.DeepEqual(oldTenantResourceQuota, tenantResourceQuota) {
		t.edgenetClientset.CoreV1alpha().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuota, metav1.UpdateOptions{})
	}

	return soonerDate, exists
}
