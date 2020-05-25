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
	"time"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/authorization"
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
	Init() error
	ObjectCreated(obj interface{})
	ObjectUpdated(obj, updated interface{})
	ObjectDeleted(obj interface{})
}

// Handler implementation
type Handler struct {
	clientset        *kubernetes.Clientset
	edgenetClientset *versioned.Clientset
	resourceQuota    *corev1.ResourceQuota
}

// Init handles any handler initialization
func (t *Handler) Init() error {
	log.Info("TotalResourceQuotaHandler.Init")
	var err error
	t.clientset, err = authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	t.edgenetClientset, err = authorization.CreateEdgeNetClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	return err
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("TotalResourceQuotaHandler.ObjectCreated")
	// Create a copy of the TRQ object to make changes on it
	TRQCopy := obj.(*apps_v1alpha.TotalResourceQuota).DeepCopy()
	defer t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().UpdateStatus(TRQCopy)
	// Find the authority from the namespace in which the object is
	TRQNamespace, _ := t.clientset.CoreV1().Namespaces().Get(TRQCopy.GetNamespace(), metav1.GetOptions{})
	TRQAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(TRQNamespace.Labels["authority-name"], metav1.GetOptions{})
	// Check if the authority is active
	if TRQAuthority.Status.Enabled && TRQCopy.Spec.Enabled {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		TRQCopy = t.resourceConsumptionControl(TRQCopy)
		if TRQCopy.Status.Exceeded {
			TRQCopy = t.balanceResourceConsumption(TRQCopy)
		}
		exists := CheckExpiryDate(TRQCopy)
		if exists {
			go t.runTimeout(TRQCopy)
		}
	} else {
		t.prohibitResourceUsage(TRQCopy, TRQAuthority)
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj, updated interface{}) {
	log.Info("TotalResourceQuotaHandler.ObjectUpdated")
	// Create a copy of the TRQ object to make changes on it
	TRQCopy := obj.(*apps_v1alpha.TotalResourceQuota).DeepCopy()
	defer t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().UpdateStatus(TRQCopy)
	// Find the authority from the namespace in which the object is
	TRQNamespace, _ := t.clientset.CoreV1().Namespaces().Get(TRQCopy.GetNamespace(), metav1.GetOptions{})
	TRQAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(TRQNamespace.Labels["authority-name"], metav1.GetOptions{})
	fieldUpdated := updated.(fields)
	// Check if the authority is active
	if TRQAuthority.Status.Enabled && TRQCopy.Spec.Enabled {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		TRQCopy = t.resourceConsumptionControl(TRQCopy)
		if TRQCopy.Status.Exceeded {
			TRQCopy = t.balanceResourceConsumption(TRQCopy)
		}
		if fieldUpdated.expiry {
			go t.runTimeout(TRQCopy)
		}
	} else {
		t.prohibitResourceUsage(TRQCopy, TRQAuthority)
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("TotalResourceQuotaHandler.ObjectDeleted")
	// Delete or disable nodes added by TRQ, TBD.
}

// sendEmail to send notification to cluster admins
func (t *Handler) sendEmail(TRQCopy *apps_v1alpha.TotalResourceQuota, subject string) {
	// Set the HTML template variables
	contentData := mailer.CommonContentData{}
	contentData.CommonData.Authority = TRQCopy.GetName()
	mailer.Send(subject, contentData)
}

func (t *Handler) resourceConsumptionControl(TRQCopy *apps_v1alpha.TotalResourceQuota) *apps_v1alpha.TotalResourceQuota {
	TRQCopy, CPUQuota, MemoryQuota := calculateTotalQuota(TRQCopy)
	TRQCopyUpdated, err := t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().Update(TRQCopy)
	if err == nil {
		TRQCopy = TRQCopyUpdated
	}
	consumedCPU, consumedMemory := t.calculateConsumedResources(TRQCopy)
	TRQCopy = checkResourceBalance(TRQCopy, CPUQuota, MemoryQuota, consumedCPU, consumedMemory)
	TRQCopyUpdated, err = t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().UpdateStatus(TRQCopy)
	if err == nil {
		TRQCopy = TRQCopyUpdated
	}
	return TRQCopy
}

func (t *Handler) prohibitResourceUsage(TRQCopy *apps_v1alpha.TotalResourceQuota, TRQAuthority *apps_v1alpha.Authority) {
	TRQCopy.Status.State = failure
	if !TRQAuthority.Status.Enabled {
		TRQCopy.Status.Message = append(TRQCopy.Status.Message, "Authority disabled")
	}
	if !TRQAuthority.Status.Enabled {
		TRQCopy.Status.Message = append(TRQCopy.Status.Message, "Total resource quota disabled")
	}
	// Delete all slices of authority
	t.edgenetClientset.AppsV1alpha().Slices(TRQCopy.GetNamespace()).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
	teamsRaw, _ := t.edgenetClientset.AppsV1alpha().Teams(TRQCopy.GetNamespace()).List(metav1.ListOptions{})
	if len(teamsRaw.Items) != 0 {
		for _, teamRow := range teamsRaw.Items {
			teamChildNamespaceStr := fmt.Sprintf("%s-team-%s", teamRow.GetNamespace(), teamRow.GetName())
			t.edgenetClientset.AppsV1alpha().Slices(teamChildNamespaceStr).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
		}
	}
}

func (t *Handler) balanceResourceConsumption(TRQCopy *apps_v1alpha.TotalResourceQuota) *apps_v1alpha.TotalResourceQuota {
	var oldestDate metav1.Time
	var oldestSlice apps_v1alpha.Slice
	slicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(TRQCopy.GetNamespace()).List(metav1.ListOptions{})
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
	teamsRaw, _ := t.edgenetClientset.AppsV1alpha().Teams(TRQCopy.GetNamespace()).List(metav1.ListOptions{})
	if len(teamsRaw.Items) != 0 {
		for _, teamRow := range teamsRaw.Items {
			teamChildNamespaceStr := fmt.Sprintf("%s-team-%s", TRQCopy.GetNamespace(), teamRow.GetName())
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
	t.edgenetClientset.AppsV1alpha().Slices(oldestSlice.GetNamespace()).Delete(oldestSlice.GetName(), &metav1.DeleteOptions{})
	TRQCopy = t.resourceConsumptionControl(TRQCopy)
	if TRQCopy.Status.Exceeded {
		TRQCopy = t.balanceResourceConsumption(TRQCopy)
	}
	return TRQCopy
}

// calculateTotalQuota to
func calculateTotalQuota(TRQCopy *apps_v1alpha.TotalResourceQuota) (*apps_v1alpha.TotalResourceQuota, int64, int64) {
	var CPUQuota int64
	var memoryQuota int64
	claimSlice := TRQCopy.Spec.Claim
	dropSlice := TRQCopy.Spec.Drop
	j := 0
	for _, claim := range TRQCopy.Spec.Claim {
		if claim.Expires.Time.Sub(time.Now()) >= 0 {
			CPUResource := resource.MustParse(claim.CPU)
			CPUQuota += CPUResource.Value()
			memoryResource := resource.MustParse(claim.Memory)
			memoryQuota += memoryResource.Value()
		} else {
			claimSlice = append(claimSlice[:j], claimSlice[j+1:]...)
			j--
		}
		j++
	}
	TRQCopy.Spec.Claim = claimSlice
	j = 0
	for _, drop := range TRQCopy.Spec.Drop {
		if drop.Expires.Time.Sub(time.Now()) >= 0 {
			CPUResource := resource.MustParse(drop.CPU)
			CPUQuota -= CPUResource.Value()
			memoryResource := resource.MustParse(drop.Memory)
			memoryQuota -= memoryResource.Value()
		} else {
			dropSlice = append(dropSlice[:j], dropSlice[j+1:]...)
			j--
		}
		j++
	}
	TRQCopy.Spec.Drop = dropSlice
	return TRQCopy, CPUQuota, memoryQuota
}

// calculateConsumedResources to
func (t *Handler) calculateConsumedResources(TRQCopy *apps_v1alpha.TotalResourceQuota) (int64, int64) {
	var consumedCPU int64
	var consumedMemory int64
	slicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(TRQCopy.GetNamespace()).List(metav1.ListOptions{})
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
	return consumedCPU, consumedMemory
}

// checkResourceBalance to
func checkResourceBalance(TRQCopy *apps_v1alpha.TotalResourceQuota,
	CPUQuota, memoryQuota, consumedCPU, consumedMemory int64) *apps_v1alpha.TotalResourceQuota {
	if CPUQuota < consumedCPU || memoryQuota < consumedMemory {
		TRQCopy.Status.Exceeded = true
	}
	TRQCopy.Status.Used.CPU = fmt.Sprintf("%.2f", percentage(consumedCPU, CPUQuota))
	TRQCopy.Status.Used.Memory = fmt.Sprintf("%.2f", percentage(consumedMemory, memoryQuota))
	return TRQCopy
}

// CheckResourceAvailability to
func (t *Handler) CheckResourceAvailability(TRQCopy *apps_v1alpha.TotalResourceQuota, CPUDemand int64, memoryDemand int64) bool {
	TRQCopy, CPUQuota, MemoryQuota := calculateTotalQuota(TRQCopy)
	TRQCopyUpdated, err := t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().Update(TRQCopy)
	if err == nil {
		TRQCopy = TRQCopyUpdated
	}
	consumedCPU, consumedMemory := t.calculateConsumedResources(TRQCopy)
	consumedCPU += CPUDemand
	consumedMemory += memoryDemand
	TRQCopy = checkResourceBalance(TRQCopy, CPUQuota, MemoryQuota, consumedCPU, consumedMemory)
	if !TRQCopy.Status.Exceeded {
		t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().UpdateStatus(TRQCopy)
	}
	return !TRQCopy.Status.Exceeded
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
			TRQCopy = t.resourceConsumptionControl(TRQCopy)
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

func percentage(value1, value2 int64) float64 {
	var percentage float64
	percentage = float64(value1) / float64(value2)
	return percentage
}

// CheckExpiryDate to checker whether there is a item with expiry date
func CheckExpiryDate(TRQCopy *apps_v1alpha.TotalResourceQuota) bool {
	exists := false
	for _, claim := range TRQCopy.Spec.Claim {
		if claim.Expires.Time.Sub(time.Now()) >= 0 {
			exists = true
		}
	}
	for _, drop := range TRQCopy.Spec.Drop {
		if drop.Expires.Time.Sub(time.Now()) >= 0 {
			exists = true
		}
	}
	return exists
}

func getClosestExpiryDate(TRQCopy *apps_v1alpha.TotalResourceQuota) time.Time {
	var closestDate *metav1.Time
	for i, claim := range TRQCopy.Spec.Claim {
		if i == 0 {
			closestDate = claim.Expires
		} else if i != 0 {
			if closestDate.Sub(claim.Expires.Time) >= 0 {
				closestDate = claim.Expires
			}
		}
	}
	for _, drop := range TRQCopy.Spec.Drop {
		if closestDate.Sub(drop.Expires.Time) >= 0 {
			closestDate = drop.Expires
		}
	}
	return closestDate.Time
}
