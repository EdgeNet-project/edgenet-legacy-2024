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

package totalresourcequota

import (
	"fmt"
	"reflect"
	"time"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	"edgenet/pkg/mailer"

	log "github.com/Sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
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
	log.Info("TotalResourceQuotaHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("TotalResourceQuotaHandler.ObjectCreated")
	// Create a copy of the TRQ object to make changes on it
	TRQCopy := obj.(*apps_v1alpha.TotalResourceQuota).DeepCopy()
	// Find the authority from the namespace in which the object is
	TRQAuthority, err := t.edgenetClientset.AppsV1alpha().Authorities().Get(TRQCopy.GetName(), metav1.GetOptions{})
	if err == nil {
		// Check if the authority is active
		if TRQAuthority.Spec.Enabled && TRQCopy.Spec.Enabled {
			// If the service restarts, it creates all objects again
			// Because of that, this section covers a variety of possibilities
			TRQCopy.Status.State = success
			TRQCopy.Status.Message = []string{statusDict["TRQ-created"]}
			// Check the total resource consumption in authority
			TRQCopy, _ = t.ResourceConsumptionControl(TRQCopy, 0, 0)
			// If they reached the limit, remove some slices randomly
			if TRQCopy.Status.Exceeded {
				TRQCopy = t.balanceResourceConsumption(TRQCopy)
			}
			// Run timeout function if there is a claim or drop with an expiry date
			exists := CheckExpiryDate(TRQCopy)
			if exists {
				go t.runTimeout(TRQCopy)
			}
		} else {
			// Block the authority to prevent using the cluster resources
			t.prohibitResourceUsage(TRQCopy, TRQAuthority)
		}
	} else {
		t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().Delete(TRQAuthority.GetName(), &metav1.DeleteOptions{})
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj, updated interface{}) {
	log.Info("TotalResourceQuotaHandler.ObjectUpdated")
	// Create a copy of the TRQ object to make changes on it
	TRQCopy := obj.(*apps_v1alpha.TotalResourceQuota).DeepCopy()
	// Find the authority from the namespace in which the object is
	TRQAuthority, err := t.edgenetClientset.AppsV1alpha().Authorities().Get(TRQCopy.GetName(), metav1.GetOptions{})
	if err == nil {
		fieldUpdated := updated.(fields)
		// Check if the authority is active
		if TRQAuthority.Spec.Enabled && TRQCopy.Spec.Enabled {
			// Start procedures if the spec changes
			if fieldUpdated.spec {
				TRQCopy, _ = t.ResourceConsumptionControl(TRQCopy, 0, 0)
				if TRQCopy.Status.Exceeded {
					TRQCopy = t.balanceResourceConsumption(TRQCopy)
				}
				if fieldUpdated.expiry {
					go t.runTimeout(TRQCopy)
				}
			}
		} else {
			t.prohibitResourceUsage(TRQCopy, TRQAuthority)
		}
	} else {
		t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().Delete(TRQAuthority.GetName(), &metav1.DeleteOptions{})
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("TotalResourceQuotaHandler.ObjectDeleted")
	// Delete or disable slices added by authority, TBD.
}

// Create generates a total resource quota with the name provided
func (t *Handler) Create(name string) {
	_, err := t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().Get(name, metav1.GetOptions{})
	if err != nil {
		// Set a total resource quota
		TRQ := apps_v1alpha.TotalResourceQuota{}
		TRQ.SetName(name)
		claim := apps_v1alpha.TotalResourceDetails{}
		claim.Name = "Default"
		claim.CPU = "12000m"
		claim.Memory = "12Gi"
		TRQ.Spec.Claim = append(TRQ.Spec.Claim, claim)
		TRQ.Spec.Enabled = true
		_, err = t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().Create(TRQ.DeepCopy())
		if err != nil {
			log.Infof(statusDict["TRQ-failed"], name, err)
		}
	}
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(username, name, email, userAuthority, sliceAuthority, sliceOwnerNamespace, sliceName, sliceNamespace, subject string) {
	// Set the HTML template variables
	contentData := mailer.ResourceAllocationData{}
	contentData.CommonData.Authority = userAuthority
	contentData.CommonData.Username = username
	contentData.CommonData.Name = name
	contentData.CommonData.Email = []string{email}
	contentData.Authority = sliceAuthority
	contentData.Name = sliceName
	contentData.OwnerNamespace = sliceOwnerNamespace
	contentData.ChildNamespace = sliceNamespace
	mailer.Send(subject, contentData)
}

// prohibitResourceUsage deletes all slices in authority and sets a status message
func (t *Handler) prohibitResourceUsage(TRQCopy *apps_v1alpha.TotalResourceQuota, TRQAuthority *apps_v1alpha.Authority) {
	defer t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().UpdateStatus(TRQCopy)
	if TRQCopy.Status.State != failure {
		TRQCopy.Status.State = failure
		TRQCopy.Status.Message = []string{}
	}
	if !TRQAuthority.Spec.Enabled {
		TRQCopy.Status.Message = append(TRQCopy.Status.Message, statusDict["authority-disable"])
	}
	if !TRQAuthority.Spec.Enabled {
		TRQCopy.Status.Message = append(TRQCopy.Status.Message, statusDict["TRQ-disabled"])
	}
	// Delete all slices of authority
	err := t.edgenetClientset.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", TRQCopy.GetName())).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
	if err != nil {
		log.Printf("Slice deletion failed in authority %s", TRQCopy.GetName())
		t.sendEmail("", "", "", "", TRQCopy.GetName(), "", "", "", "slice-collection-deletion-failed")
	}
	teamsRaw, _ := t.edgenetClientset.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", TRQCopy.GetName())).List(metav1.ListOptions{})
	if len(teamsRaw.Items) != 0 {
		for _, teamRow := range teamsRaw.Items {
			teamChildNamespaceStr := fmt.Sprintf("%s-team-%s", teamRow.GetNamespace(), teamRow.GetName())
			err = t.edgenetClientset.AppsV1alpha().Slices(teamChildNamespaceStr).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
			if err != nil {
				log.Printf("Slice deletion failed in %s", teamChildNamespaceStr)
				t.sendEmail("", "", "", "", TRQCopy.GetName(), teamChildNamespaceStr, "", "", "slice-collection-deletion-failed")
			}
		}
	}
}

// CheckExpiryDate to checker whether there is an item with expiry date
func CheckExpiryDate(TRQCopy *apps_v1alpha.TotalResourceQuota) bool {
	exists := false
	for _, claim := range TRQCopy.Spec.Claim {
		if claim.Expires != nil && claim.Expires.Time.Sub(time.Now()) >= 0 {
			exists = true
		}
	}
	for _, drop := range TRQCopy.Spec.Drop {
		if drop.Expires != nil && drop.Expires.Time.Sub(time.Now()) >= 0 {
			exists = true
		}
	}
	return exists
}

// ResourceConsumptionControl both calculates the total resource quota and the total consumption in the authority.
// Additionally, when a Slice created it comes along with a resource consumption demand. This function also allows us
// to compare free resources with demands as well.
func (t *Handler) ResourceConsumptionControl(TRQCopy *apps_v1alpha.TotalResourceQuota, CPUDemand int64, memoryDemand int64) (*apps_v1alpha.TotalResourceQuota, bool) {
	// Find out the total resource quota by taking claims and drops into account
	TRQCopy, CPUQuota, MemoryQuota := t.calculateTotalQuota(TRQCopy)
	// Get the total consumption that all Slices do in authority
	consumedCPU, consumedMemory := t.calculateConsumedResources(TRQCopy)
	consumedCPU += CPUDemand
	consumedMemory += memoryDemand
	demand := false
	if CPUDemand != 0 || memoryDemand != 0 {
		demand = true
	}
	// Compare the consumption with the total resource quota
	TRQCopy, quotaExceeded := t.checkResourceBalance(TRQCopy, CPUQuota, MemoryQuota, consumedCPU, consumedMemory, demand)
	return TRQCopy, quotaExceeded
}

// calculateTotalQuota adds the resources defined in claims, and subtracts those in drops to calculate the total resource quota.
// Moreover, the function checkes whether any claim or drop has an expiry date and updates the object if exists.
func (t *Handler) calculateTotalQuota(TRQCopy *apps_v1alpha.TotalResourceQuota) (*apps_v1alpha.TotalResourceQuota, int64, int64) {
	var CPUQuota int64
	var memoryQuota int64
	// To make comparison
	oldTRQCopy := TRQCopy.DeepCopy()
	// claimSlice to be manipulated
	claimSlice := TRQCopy.Spec.Claim
	// dropSlice to be manipulated
	dropSlice := TRQCopy.Spec.Drop
	if len(TRQCopy.Spec.Claim) > 0 {
		j := 0
		for _, claim := range TRQCopy.Spec.Claim {
			if claim.Expires == nil || (claim.Expires != nil && claim.Expires.Time.Sub(time.Now()) >= 0) {
				CPUResource := resource.MustParse(claim.CPU)
				CPUQuota += CPUResource.Value()
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
		TRQCopy.Spec.Claim = claimSlice
	}
	if len(TRQCopy.Spec.Drop) > 0 {
		j := 0
		for _, drop := range TRQCopy.Spec.Drop {
			if drop.Expires == nil || (drop.Expires != nil && drop.Expires.Time.Sub(time.Now()) >= 0) {
				CPUResource := resource.MustParse(drop.CPU)
				CPUQuota -= CPUResource.Value()
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
		TRQCopy.Spec.Drop = dropSlice
	}
	// Check if there is an update
	if !reflect.DeepEqual(oldTRQCopy, TRQCopy) {
		TRQCopyUpdated, err := t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().Update(TRQCopy)
		if err == nil {
			TRQCopy = TRQCopyUpdated
			TRQCopy.Status.State = success
			TRQCopy.Status.Message = []string{statusDict["TRQ-applied"]}
		} else {
			log.Infof("Couldn't update total resource quota in %s: %s", TRQCopy.GetName(), err)
			TRQCopy.Status.State = failure
			TRQCopy.Status.Message = []string{statusDict["TRQ-appliedFail"]}
		}
	}
	return TRQCopy, CPUQuota, memoryQuota
}

// calculateConsumedResources looks out for slices in authority and teams to determine the total consumption
func (t *Handler) calculateConsumedResources(TRQCopy *apps_v1alpha.TotalResourceQuota) (int64, int64) {
	var consumedCPU int64
	var consumedMemory int64
	slicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", TRQCopy.GetName())).List(metav1.ListOptions{})
	if len(slicesRaw.Items) != 0 {
		for _, slicesRow := range slicesRaw.Items {
			sliceChildNamespaceStr := fmt.Sprintf("%s-slice-%s", slicesRow.GetNamespace(), slicesRow.GetName())
			// Check out the resource quotas in the slice namespace rather than the slice profile
			resourceQuotasRaw, _ := t.clientset.CoreV1().ResourceQuotas(sliceChildNamespaceStr).List(metav1.ListOptions{})
			if len(resourceQuotasRaw.Items) != 0 {
				for _, resourceQuotasRow := range resourceQuotasRaw.Items {
					consumedCPU += resourceQuotasRow.Spec.Hard.Cpu().Value()
					consumedMemory += resourceQuotasRow.Spec.Hard.Memory().Value()
				}
			}
		}
	}
	teamsRaw, _ := t.edgenetClientset.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", TRQCopy.GetName())).List(metav1.ListOptions{})
	if len(teamsRaw.Items) != 0 {
		for _, teamRow := range teamsRaw.Items {
			teamChildNamespaceStr := fmt.Sprintf("%s-team-%s", teamRow.GetNamespace(), teamRow.GetName())
			slicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(teamChildNamespaceStr).List(metav1.ListOptions{})
			if len(slicesRaw.Items) != 0 {
				for _, slicesRow := range slicesRaw.Items {
					sliceChildNamespaceStr := fmt.Sprintf("%s-slice-%s", slicesRow.GetNamespace(), slicesRow.GetName())
					resourceQuotasRaw, _ := t.clientset.CoreV1().ResourceQuotas(sliceChildNamespaceStr).List(metav1.ListOptions{})
					if len(resourceQuotasRaw.Items) != 0 {
						for _, resourceQuotasRow := range resourceQuotasRaw.Items {
							consumedCPU += resourceQuotasRow.Spec.Hard.Cpu().Value()
							consumedMemory += resourceQuotasRow.Spec.Hard.Memory().Value()
						}
					}
				}
			}
		}
	}
	return consumedCPU, consumedMemory
}

// checkResourceBalance compares the total resource quota with the total consumption to detect if there is an overusing of resources
func (t *Handler) checkResourceBalance(TRQCopy *apps_v1alpha.TotalResourceQuota,
	CPUQuota, memoryQuota, consumedCPU, consumedMemory int64, resourceDemand bool) (*apps_v1alpha.TotalResourceQuota, bool) {
	log.Println("checkResourceBalance")
	log.Printf("Quota = CPU: %d, Memory: %d - Consumed = CPU: %d, Memory: %d", CPUQuota, memoryQuota, consumedCPU, consumedMemory)
	// To be compared
	oldTRQCopy := TRQCopy.DeepCopy()
	// Check CPU and memory usage separately
	quotaExceeded := false
	if CPUQuota < consumedCPU || memoryQuota < consumedMemory {
		quotaExceeded = true
	}
	// Set the status
	TRQCopy.Status.Exceeded = quotaExceeded
	TRQCopy.Status.Used.CPU = percentage(consumedCPU, CPUQuota)
	TRQCopy.Status.Used.Memory = percentage(consumedMemory, memoryQuota)
	// Check if there is an update
	if !reflect.DeepEqual(oldTRQCopy, TRQCopy) {
		// If there is a resource request causing the quota to be exceeded, skip this section.
		// The slice demanding resource will be removed.
		if (!TRQCopy.Status.Exceeded && resourceDemand) || !resourceDemand {
			TRQCopyUpdated, err := t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().UpdateStatus(TRQCopy)
			log.Println(err)
			if err == nil {
				TRQCopy = TRQCopyUpdated
			} else {
				log.Infof("Couldn't update the status of total resource quota in %s: %s", TRQCopy.GetName(), err)
			}
		} else {
			// Replace with the old version since the process is canceled
			TRQCopy.Status = oldTRQCopy.Status
		}
	}
	return TRQCopy, quotaExceeded
}

// balanceResourceConsumption determines the slice to be removed by picking the oldest one
func (t *Handler) balanceResourceConsumption(TRQCopy *apps_v1alpha.TotalResourceQuota) *apps_v1alpha.TotalResourceQuota {
	var oldestDate metav1.Time
	var oldestSlice apps_v1alpha.Slice
	log.Println("balanceResourceConsumption")
	// Get the oldest slice in the authority namespace
	slicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", TRQCopy.GetName())).List(metav1.ListOptions{})
	if len(slicesRaw.Items) != 0 {
		for i, sliceRow := range slicesRaw.Items {
			if i == 0 {
				oldestDate = sliceRow.GetCreationTimestamp()
				oldestSlice = sliceRow
			} else if i != 0 {
				if oldestDate.Sub(sliceRow.GetCreationTimestamp().Time) <= 0 {
					oldestDate = sliceRow.GetCreationTimestamp()
					oldestSlice = sliceRow
				}
			}
		}
	}
	// Get the oldest slice in the team namespaces
	teamsRaw, _ := t.edgenetClientset.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", TRQCopy.GetName())).List(metav1.ListOptions{})
	if len(teamsRaw.Items) != 0 {
		for _, teamRow := range teamsRaw.Items {
			teamChildNamespaceStr := fmt.Sprintf("authority-%s-team-%s", TRQCopy.GetName(), teamRow.GetName())
			slicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(teamChildNamespaceStr).List(metav1.ListOptions{})
			if len(slicesRaw.Items) != 0 {
				for _, sliceRow := range slicesRaw.Items {
					if oldestDate.Sub(sliceRow.GetCreationTimestamp().Time) <= 0 {
						oldestDate = sliceRow.GetCreationTimestamp()
						oldestSlice = sliceRow
					}
				}
			}
		}
	}
	// Delete the oldest slice and send a notification email
	err := t.edgenetClientset.AppsV1alpha().Slices(oldestSlice.GetNamespace()).Delete(oldestSlice.GetName(), &metav1.DeleteOptions{})
	sliceChildNamespaceStr := fmt.Sprintf("%s-slice-%s", oldestSlice.GetNamespace(), oldestSlice.GetName())
	if err == nil {
		for _, sliceUser := range oldestSlice.Spec.Users {
			user, err := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("authority-%s", sliceUser.Authority)).Get(sliceUser.Username, metav1.GetOptions{})
			if err == nil && user.Spec.Active && user.Status.AUP {
				t.sendEmail(sliceUser.Username, fmt.Sprintf("%s %s", user.Spec.FirstName, user.Spec.LastName), user.Spec.Email, sliceUser.Authority,
					TRQCopy.GetName(), oldestSlice.GetNamespace(), oldestSlice.GetName(), sliceChildNamespaceStr, "slice-total-quota-exceeded")
			}
		}
	} else {
		log.Printf("Slice %s deletion failed in %s", oldestSlice.GetName(), oldestSlice.GetNamespace())
		t.sendEmail("", "", "", "", TRQCopy.GetName(), oldestSlice.GetNamespace(), oldestSlice.GetName(), sliceChildNamespaceStr, "slice-deletion-failed")
	}
	// Check out the balance again
	TRQCopy, _ = t.ResourceConsumptionControl(TRQCopy, 0, 0)
	// Run the procedure again if the consumption still reaches the quota limit
	if TRQCopy.Status.Exceeded {
		TRQCopy = t.balanceResourceConsumption(TRQCopy)
	}
	return TRQCopy
}

// runTimeout puts a procedure in place to remove claims and drops after the timeout
func (t *Handler) runTimeout(TRQCopy *apps_v1alpha.TotalResourceQuota) {
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	timeout = time.After(time.Until(getClosestExpiryDate(TRQCopy)))
	closeChannels := func() {
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of total resource quota object
	watchTRQ, err := t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().Watch(metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", TRQCopy.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for TRQEvent := range watchTRQ.ResultChan() {
				// Get updated slice object
				updatedTRQ, status := TRQEvent.Object.(*apps_v1alpha.TotalResourceQuota)
				if TRQCopy.GetUID() == updatedTRQ.GetUID() {
					if status {
						if TRQEvent.Type == "DELETED" {
							terminated <- true
							continue
						}
						TRQCopy = updatedTRQ
						exists := CheckExpiryDate(TRQCopy)
						if exists {
							timeout = time.After(time.Until(getClosestExpiryDate(TRQCopy)))
							timeoutRenewed <- true
						} else {
							terminated <- true
						}
					}
				}
			}
		}()
	} else {
		// In case of any malfunction of watching total resource quota objects,
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
			TRQCopy, _ = t.ResourceConsumptionControl(TRQCopy, 0, 0)
			if TRQCopy.Status.Exceeded {
				TRQCopy = t.balanceResourceConsumption(TRQCopy)
			}
			exists := CheckExpiryDate(TRQCopy)
			if !exists {
				terminated <- true
			}
			break timeoutOptions
		case <-terminated:
			watchTRQ.Stop()
			closeChannels()
			break timeoutLoop
		}
	}
}

// getClosestExpiryDate determines the item, a claim or a drop, having the closest expiry date
func getClosestExpiryDate(TRQCopy *apps_v1alpha.TotalResourceQuota) time.Time {
	var closestDate *metav1.Time
	for i, claim := range TRQCopy.Spec.Claim {
		if i == 0 {
			if claim.Expires != nil {
				closestDate = claim.Expires
			}
		} else if i != 0 {
			if claim.Expires != nil {
				if closestDate.Sub(claim.Expires.Time) >= 0 {
					closestDate = claim.Expires
				}
			}
		}
	}
	for _, drop := range TRQCopy.Spec.Drop {
		if drop.Expires != nil {
			if closestDate.Sub(drop.Expires.Time) >= 0 {
				closestDate = drop.Expires
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
