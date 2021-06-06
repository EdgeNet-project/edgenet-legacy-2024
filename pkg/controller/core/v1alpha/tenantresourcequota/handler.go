/*
Copyright 2020 Sorbonne Université

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

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init(kubernetes kubernetes.Interface, edgenet versioned.Interface)
	ObjectCreated(obj interface{})
	ObjectUpdated(obj, updated interface{})
	ObjectDeleted(obj interface{})
}

// Handler implementation
type Handler struct {
	clientset        kubernetes.Interface
	edgenetClientset versioned.Interface
	resourceQuota    *corev1.ResourceQuota
}

// Init handles any handler initialization
func (t *Handler) Init(kubernetes kubernetes.Interface, edgenet versioned.Interface) {
	log.Info("tenantResourceQuotaHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("tenantResourceQuotaHandler.ObjectCreated")
	// Make a copy of the tenant resource quota object to make changes on it
	tenantResourceQuota := obj.(*corev1alpha.TenantResourceQuota).DeepCopy()
	// Find the tenant from the namespace in which the object is
	tenant, err := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	if errors.IsNotFound(err) {
		t.edgenetClientset.CoreV1alpha().TenantResourceQuotas().Delete(context.TODO(), tenant.GetName(), metav1.DeleteOptions{})
	} else {
		// Check if the tenant is active
		if tenant.Spec.Enabled {
			// If the service restarts, it creates all objects again
			// Because of that, this section covers a variety of possibilities
			tenantResourceQuota.Status.State = success
			tenantResourceQuota.Status.Message = []string{statusDict["TRQ-created"]}
			tenantResourceQuotaUpdated, err := t.edgenetClientset.CoreV1alpha().TenantResourceQuotas().UpdateStatus(context.TODO(), tenantResourceQuota, metav1.UpdateOptions{})
			if err == nil {
				tenantResourceQuota = tenantResourceQuotaUpdated
			} else {
				log.Infof("Couldn't update the status of tenant resource quota in %s: %s", tenantResourceQuota.GetName(), err)
			}
			// Check the total resource consumption in tenant
			tenantResourceQuota, quotaExceeded, cpuDecline, memoryDecline := t.ResourceConsumptionControl(tenantResourceQuota, 0, 0)
			// If they reached the limit, remove some subnamespaces randomly
			if quotaExceeded {
				t.balanceResourceConsumption(tenantResourceQuota.GetName(), cpuDecline, memoryDecline)
			}
			// Run timeout function if there is a claim or drop with an expiry date
			exists := CheckExpiryDate(tenantResourceQuota)
			if exists {
				go t.runTimeout(tenantResourceQuota)
			}
		} else {
			// Block the tenant to prevent using the cluster resources
			t.prohibitResourceConsumption(tenantResourceQuota, tenant)
		}
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj, updated interface{}) {
	log.Info("tenantResourceQuotaHandler.ObjectUpdated")
	// Make a copy of the tenant resource quota object to make changes on it
	tenantResourceQuota := obj.(*corev1alpha.TenantResourceQuota).DeepCopy()
	// Find the tenant from the namespace in which the object is
	tenant, err := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	if errors.IsNotFound(err) {
		t.edgenetClientset.CoreV1alpha().TenantResourceQuotas().Delete(context.TODO(), tenant.GetName(), metav1.DeleteOptions{})
	} else {
		fieldUpdated := updated.(fields)
		// Check if the tenant is active
		if tenant.Spec.Enabled {
			// Start procedures if the spec changes
			if fieldUpdated.spec {
				tenantResourceQuota, quotaExceeded, cpuDecline, memoryDecline := t.ResourceConsumptionControl(tenantResourceQuota, 0, 0)
				if quotaExceeded {
					t.balanceResourceConsumption(tenantResourceQuota.GetName(), cpuDecline, memoryDecline)
				}
				if fieldUpdated.expiry {
					exists := CheckExpiryDate(tenantResourceQuota)
					if exists {
						go t.runTimeout(tenantResourceQuota)
					}
				}
			}
		} else {
			t.prohibitResourceConsumption(tenantResourceQuota, tenant)
		}
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("tenantResourceQuotaHandler.ObjectDeleted")
	// Delete or disable subnamespaces added by tenant, TBD.
}

// Create generates a tenant resource quota with the name provided
func (t *Handler) Create(name string) {
	_, err := t.edgenetClientset.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		// Set a tenant resource quota
		tenantResourceQuota := corev1alpha.TenantResourceQuota{}
		tenantResourceQuota.SetName(name)
		claim := corev1alpha.TenantResourceDetails{}
		claim.Name = "Default"
		claim.CPU = "12000m"
		claim.Memory = "12Gi"
		tenantResourceQuota.Spec.Claim = append(tenantResourceQuota.Spec.Claim, claim)
		_, err = t.edgenetClientset.CoreV1alpha().TenantResourceQuotas().Create(context.TODO(), tenantResourceQuota.DeepCopy(), metav1.CreateOptions{})
		if err != nil {
			log.Infof(statusDict["TRQ-failed"], name, err)
		}
	}
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(tenant, subnamespaceOwner, subnamespace, subnamespaceName, subject string) {
	// Set the HTML template variables
	contentData := mailer.ResourceAllocationData{}
	contentData.CommonData.Tenant = tenant
	contentData.Tenant = tenant
	contentData.Name = subnamespace
	contentData.OwnerNamespace = subnamespaceOwner
	contentData.ChildNamespace = subnamespaceName
	mailer.Send(subject, contentData)
}

// prohibitResourceConsumption deletes all subnamespaces in tenant
func (t *Handler) prohibitResourceConsumption(tenantResourceQuota *corev1alpha.TenantResourceQuota, tenant *corev1alpha.Tenant) {
	// Delete all subnamespaces of tenant
	err := t.edgenetClientset.CoreV1alpha().SubNamespaces(tenant.GetName()).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{})
	if err != nil {
		log.Printf("Subnamespace deletion failed in tenant %s", tenantResourceQuota.GetName())
		t.sendEmail(tenantResourceQuota.GetName(), "", "", "", "subnamespace-collection-deletion-failed")
	}
}

// CheckExpiryDate to checker whether there is an item with expiry date
func CheckExpiryDate(tenantResourceQuota *corev1alpha.TenantResourceQuota) bool {
	exists := false
	for _, claim := range tenantResourceQuota.Spec.Claim {
		if claim.Expiry != nil && claim.Expiry.Time.Sub(time.Now()) >= 0 {
			exists = true
		}
	}
	for _, drop := range tenantResourceQuota.Spec.Drop {
		if drop.Expiry != nil && drop.Expiry.Time.Sub(time.Now()) >= 0 {
			exists = true
		}
	}
	return exists
}

// ResourceConsumptionControl both calculates the tenant resource quota and the total consumption in the tenant.
// Additionally, when a Slice created it comes along with a resource consumption demand. This function also allows us
// to compare free resources with demands as well.
func (t *Handler) ResourceConsumptionControl(tenantResourceQuota *corev1alpha.TenantResourceQuota, cpuDemand int64, memoryDemand int64) (*corev1alpha.TenantResourceQuota, bool, int64, int64) {
	// Find out the tenant resource quota by taking claims and drops into account
	tenantResourceQuota, cpuQuota, memoryQuota := t.calculateTotalQuota(tenantResourceQuota)
	// Get the total consumption that all namespaces do in tenant
	var aggregatedCPU, aggregatedMemory int64 = 0, 0
	t.aggregateConsumedResources(tenantResourceQuota.GetName(), &aggregatedCPU, &aggregatedMemory)
	aggregatedCPU += cpuDemand
	aggregatedMemory += memoryDemand
	demand := false
	if cpuDemand != 0 || memoryDemand != 0 {
		demand = true
	}
	quotaExceeded := false
	if (aggregatedCPU == 0 && aggregatedMemory == 0 && demand) || (aggregatedCPU > 0 || aggregatedMemory > 0) {
		if cpuQuota < aggregatedCPU || memoryQuota < aggregatedMemory {
			quotaExceeded = true
		}
	}
	return tenantResourceQuota, quotaExceeded, (aggregatedCPU - cpuQuota), (aggregatedMemory - memoryQuota)
}

// calculateTotalQuota adds the resources defined in claims, and subtracts those in drops to calculate the tenant resource quota.
// Moreover, the function checkes whether any claim or drop has an expiry date and updates the object if exists.
func (t *Handler) calculateTotalQuota(tenantResourceQuota *corev1alpha.TenantResourceQuota) (*corev1alpha.TenantResourceQuota, int64, int64) {
	var cpuQuota int64
	var memoryQuota int64
	// To make comparison
	oldTenantResourceQuota := tenantResourceQuota.DeepCopy()
	// claimSlice to be manipulated
	claimSlice := tenantResourceQuota.Spec.Claim
	// dropSlice to be manipulated
	dropSlice := tenantResourceQuota.Spec.Drop
	if len(tenantResourceQuota.Spec.Claim) > 0 {
		j := 0
		for _, claim := range tenantResourceQuota.Spec.Claim {
			if claim.Expiry == nil || (claim.Expiry != nil && claim.Expiry.Time.Sub(time.Now()) >= 0) {
				cpuResource := resource.MustParse(claim.CPU)
				cpuQuota += cpuResource.Value()
				memoryResource := resource.MustParse(claim.Memory)
				memoryQuota += memoryResource.Value()
			} else {
				// Remove the item from claims if the expiry date has run out
				claimSlice = append(claimSlice[:j], claimSlice[j+1:]...)
				j--
			}
			j++
		}
		// Sync the claims
		tenantResourceQuota.Spec.Claim = claimSlice
	}
	if len(tenantResourceQuota.Spec.Drop) > 0 {
		j := 0
		for _, drop := range tenantResourceQuota.Spec.Drop {
			if drop.Expiry == nil || (drop.Expiry != nil && drop.Expiry.Time.Sub(time.Now()) >= 0) {
				cpuResource := resource.MustParse(drop.CPU)
				cpuQuota -= cpuResource.Value()
				memoryResource := resource.MustParse(drop.Memory)
				memoryQuota -= memoryResource.Value()
			} else {
				// Remove the item from drops if the expiry date has run out
				dropSlice = append(dropSlice[:j], dropSlice[j+1:]...)
				j--
			}
			j++
		}
		// Sync the drops
		tenantResourceQuota.Spec.Drop = dropSlice
	}
	// Check if there is an update
	if !reflect.DeepEqual(oldTenantResourceQuota, tenantResourceQuota) {
		tenantResourceQuotaUpdated, err := t.edgenetClientset.CoreV1alpha().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuota, metav1.UpdateOptions{})
		if err == nil {
			tenantResourceQuota = tenantResourceQuotaUpdated
			tenantResourceQuota.Status.State = success
			tenantResourceQuota.Status.Message = []string{statusDict["TRQ-applied"]}
		} else {
			log.Infof("Couldn't update tenant resource quota in %s: %s", tenantResourceQuota.GetName(), err)
			tenantResourceQuota.Status.State = failure
			tenantResourceQuota.Status.Message = []string{statusDict["TRQ-appliedFail"]}
		}
	}
	return tenantResourceQuota, cpuQuota, memoryQuota
}

// aggregateConsumedResources looks out for namespaces in tenant and teams to determine the total consumption
func (t *Handler) aggregateConsumedResources(namespace string, aggregatedCPU *int64, aggregatedMemory *int64) {
	// Check out the resource quotas in the namespace rather than in its profile
	resourceQuotasRaw, _ := t.clientset.CoreV1().ResourceQuotas(namespace).List(context.TODO(), metav1.ListOptions{})
	if len(resourceQuotasRaw.Items) != 0 {
		for _, resourceQuotasRow := range resourceQuotasRaw.Items {
			if resourceQuotasRow.GetName() != "core-quota" {
				*aggregatedCPU += resourceQuotasRow.Spec.Hard.Cpu().Value()
				*aggregatedMemory += resourceQuotasRow.Spec.Hard.Memory().Value()
			}
		}
	}

	subNamespaceRaw, _ := t.edgenetClientset.CoreV1alpha().SubNamespaces(namespace).List(context.TODO(), metav1.ListOptions{})
	if len(subNamespaceRaw.Items) != 0 {
		for _, subNamespaceRow := range subNamespaceRaw.Items {
			subNamespaceStr := fmt.Sprintf("%s-%s", subNamespaceRow.GetNamespace(), subNamespaceRow.GetName())
			t.aggregateConsumedResources(subNamespaceStr, aggregatedCPU, aggregatedMemory)
		}
	}
}

// eliminateSubNamespace determines the subnamespaces to be removed by LIFO (Last In First Out)
func (t *Handler) eliminateSubNamespace(tenant, namespace string, lastInDate *metav1.Time, lastInSubNamespace *corev1alpha.SubNamespace) {
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()
	defer wg.Done()
	subNamespaceRaw, _ := t.edgenetClientset.CoreV1alpha().SubNamespaces(namespace).List(context.TODO(), metav1.ListOptions{})
	if len(subNamespaceRaw.Items) != 0 {
		for i, subNamespaceRow := range subNamespaceRaw.Items {
			if i == 0 && lastInDate.IsZero() {
				*lastInDate = subNamespaceRow.GetCreationTimestamp()
				*lastInSubNamespace = subNamespaceRow
			} else {
				if lastInDate.Sub(subNamespaceRow.GetCreationTimestamp().Time) >= 0 {
					*lastInDate = subNamespaceRow.GetCreationTimestamp()
					*lastInSubNamespace = subNamespaceRow
				}
			}
			subNamespaceStr := fmt.Sprintf("%s-%s", tenant, subNamespaceRow.GetName())
			wg.Add(1)
			go t.eliminateSubNamespace(tenant, subNamespaceStr, lastInDate, lastInSubNamespace)
			wg.Done()
		}
	}
}

// balanceResourceConsumption determines the slice to be removed by LIFO
func (t *Handler) balanceResourceConsumption(tenant string, cpuDecline, memoryDecline int64) {
	var lastInDate metav1.Time
	var lastInSubNamespace corev1alpha.SubNamespace

	coreResourceQuota, _ := t.clientset.CoreV1().ResourceQuotas(tenant).Get(context.TODO(), "core-quota", metav1.GetOptions{})
	freeCPUQuota := coreResourceQuota.Spec.Hard.Cpu().Value()
	freeMemoryQuota := coreResourceQuota.Spec.Hard.Memory().Value()
	if freeCPUQuota > cpuDecline && freeMemoryQuota > memoryDecline {
		coreResourceQuota.Spec.Hard.Cpu().Set(freeCPUQuota - cpuDecline)
		coreResourceQuota.Spec.Hard.Memory().Set(freeMemoryQuota - memoryDecline)
		t.clientset.CoreV1().ResourceQuotas(tenant).Update(context.TODO(), coreResourceQuota, metav1.UpdateOptions{})
	} else {
		t.eliminateSubNamespace(tenant, tenant, &lastInDate, &lastInSubNamespace)
	}

	subNamespaceStr := fmt.Sprintf("%s-%s", tenant, lastInSubNamespace.GetName())
	subNamespaceResourceQuota, _ := t.clientset.CoreV1().ResourceQuotas(subNamespaceStr).Get(context.TODO(), "core-quota", metav1.GetOptions{})
	if subNamespaceResourceQuota.Spec.Hard.Cpu().Value() >= cpuDecline && subNamespaceResourceQuota.Spec.Hard.Memory().Value() >= memoryDecline {
		// Delete the subnamespace and send a notification email
		err := t.edgenetClientset.CoreV1alpha().SubNamespaces(lastInSubNamespace.GetNamespace()).Delete(context.TODO(), lastInSubNamespace.GetName(), metav1.DeleteOptions{})
		if err == nil {
			t.sendEmail(tenant, lastInSubNamespace.GetNamespace(), lastInSubNamespace.GetName(), subNamespaceStr, "subnamespace-tenant-quota-exceeded")
		} else {
			log.Printf("SubNamespace %s deletion failed in %s", lastInSubNamespace.GetName(), lastInSubNamespace.GetNamespace())
			t.sendEmail(tenant, lastInSubNamespace.GetNamespace(), lastInSubNamespace.GetName(), subNamespaceStr, "subnamespace-deletion-failed")
		}
	} else {
		t.balanceResourceConsumption(tenant, (cpuDecline - subNamespaceResourceQuota.Spec.Hard.Cpu().Value()), (memoryDecline - subNamespaceResourceQuota.Spec.Hard.Memory().Value()))
	}
}

// runTimeout puts a procedure in place to remove claims and drops after the timeout
func (t *Handler) runTimeout(tenantResourceQuota *corev1alpha.TenantResourceQuota) {
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	timeout = time.After(time.Until(getClosestExpiryDate(tenantResourceQuota)))
	closeChannels := func() {
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of tenant resource quota object
	watchTenantResourceQuota, err := t.edgenetClientset.CoreV1alpha().TenantResourceQuotas().Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", tenantResourceQuota.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for tenantResourceQuotaEvent := range watchTenantResourceQuota.ResultChan() {
				// Get updated slice object
				updatedTenantResourceQuota, status := tenantResourceQuotaEvent.Object.(*corev1alpha.TenantResourceQuota)
				if tenantResourceQuota.GetUID() == updatedTenantResourceQuota.GetUID() {
					if status {
						if tenantResourceQuotaEvent.Type == "DELETED" {
							terminated <- true
							continue
						}
						tenantResourceQuota = updatedTenantResourceQuota
						exists := CheckExpiryDate(tenantResourceQuota)
						if exists {
							timeout = time.After(time.Until(getClosestExpiryDate(tenantResourceQuota)))
							timeoutRenewed <- true
						} else {
							select {
							case <-terminated:
								watchTenantResourceQuota.Stop()
							default:
								terminated <- true
							}
						}
					}
				}
			}
		}()
	} else {
		// In case of any malfunction of watching tenant resource quota objects,
		// there is a timeout at 72 hours
		timeout = time.After(72 * time.Hour)
	}

	// Infinite loop
timeoutLoop:
	for {
		// Wait on multiple channel operations
	timeoutOptions:
		select {
		case <-timeoutRenewed:
			break timeoutOptions
		case <-timeout:
			tenantResourceQuota, quotaExceeded, cpuDecline, memoryDecline := t.ResourceConsumptionControl(tenantResourceQuota, 0, 0)
			if quotaExceeded {
				t.balanceResourceConsumption(tenantResourceQuota.GetName(), cpuDecline, memoryDecline)
			}
			exists := CheckExpiryDate(tenantResourceQuota)
			if !exists {
				terminated <- true
			}
			break timeoutOptions
		case <-terminated:
			watchTenantResourceQuota.Stop()
			closeChannels()
			break timeoutLoop
		}
	}
}

// getClosestExpiryDate determines the item, a claim or a drop, having the closest expiry date
func getClosestExpiryDate(tenantResourceQuota *corev1alpha.TenantResourceQuota) time.Time {
	var closestDate *metav1.Time
	for i, claim := range tenantResourceQuota.Spec.Claim {
		if i == 0 {
			if claim.Expiry != nil {
				closestDate = claim.Expiry
			}
		} else if i != 0 {
			if claim.Expiry != nil {
				if closestDate.Sub(claim.Expiry.Time) >= 0 {
					closestDate = claim.Expiry
				}
			}
		}
	}
	for j, drop := range tenantResourceQuota.Spec.Drop {
		if j == 0 {
			if drop.Expiry != nil {
				closestDate = drop.Expiry
			}
		} else if j != 0 {
			if drop.Expiry != nil {
				if closestDate.Sub(drop.Expiry.Time) >= 0 {
					closestDate = drop.Expiry
				}
			}
		}
	}
	return closestDate.Time
}

// percentage to give a overview of resource consumption
func percentage(value1, value2 int64) float64 {
	var percentage float64
	percentage = float64(value1) / float64(value2) * 100
	return percentage
}
