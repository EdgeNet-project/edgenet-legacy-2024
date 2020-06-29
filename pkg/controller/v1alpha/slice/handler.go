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

package slice

import (
	"encoding/json"
	"fmt"
	"time"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/authorization"
	"edgenet/pkg/client/clientset/versioned"
	"edgenet/pkg/controller/v1alpha/totalresourcequota"
	"edgenet/pkg/mailer"
	"edgenet/pkg/registration"

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
	clientset         *kubernetes.Clientset
	edgenetClientset  *versioned.Clientset
	lowResourceQuota  *corev1.ResourceQuota
	medResourceQuota  *corev1.ResourceQuota
	highResourceQuota *corev1.ResourceQuota
}

// Init handles any handler initialization
func (t *Handler) Init() error {
	log.Info("SliceHandler.Init")
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
	t.lowResourceQuota = &corev1.ResourceQuota{}
	t.lowResourceQuota.Name = "slice-low-quota"
	t.lowResourceQuota.Spec = corev1.ResourceQuotaSpec{
		Hard: map[corev1.ResourceName]resource.Quantity{
			"cpu":              resource.MustParse("2000m"),
			"memory":           resource.MustParse("2048Mi"),
			"requests.storage": resource.MustParse("500Mi"),
		},
	}
	t.medResourceQuota = &corev1.ResourceQuota{}
	t.medResourceQuota.Name = "slice-medium-quota"
	t.medResourceQuota.Spec = corev1.ResourceQuotaSpec{
		Hard: map[corev1.ResourceName]resource.Quantity{
			"cpu":              resource.MustParse("4000m"),
			"memory":           resource.MustParse("4096Mi"),
			"requests.storage": resource.MustParse("2Gi"),
		},
	}
	t.highResourceQuota = &corev1.ResourceQuota{}
	t.highResourceQuota.Name = "slice-high-quota"
	t.highResourceQuota.Spec = corev1.ResourceQuotaSpec{
		Hard: map[corev1.ResourceName]resource.Quantity{
			"cpu":              resource.MustParse("8000m"),
			"memory":           resource.MustParse("8192Mi"),
			"requests.storage": resource.MustParse("8Gi"),
		},
	}
	return err
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("SliceHandler.ObjectCreated")
	// Create a copy of the slice object to make changes on it
	sliceCopy := obj.(*apps_v1alpha.Slice).DeepCopy()
	// Find the authority from the namespace in which the object is
	sliceOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(sliceCopy.GetNamespace(), metav1.GetOptions{})
	sliceOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(sliceOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	sliceChildNamespaceStr := fmt.Sprintf("%s-slice-%s", sliceCopy.GetNamespace(), sliceCopy.GetName())
	// The section below checks whether the slice belongs to a team or directly to a authority. After then, set the value as enabled
	// if the authority and the team (if it is an owner) enabled.
	var sliceOwnerEnabled bool
	if sliceOwnerNamespace.Labels["owner"] == "team" {
		sliceOwnerEnabled = sliceOwnerAuthority.Spec.Enabled
		if sliceOwnerEnabled {
			sliceOwnerTeam, _ := t.edgenetClientset.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", sliceOwnerNamespace.Labels["authority-name"])).
				Get(sliceOwnerNamespace.Labels["owner-name"], metav1.GetOptions{})
			sliceOwnerEnabled = sliceOwnerTeam.Spec.Enabled
		}
	} else {
		sliceOwnerEnabled = sliceOwnerAuthority.Spec.Enabled
	}
	// Check if the owner(s) is/are active
	if sliceOwnerEnabled {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		if sliceCopy.Status.Expires == nil {
			resourcesAvailability := t.checkResourcesAvailabilityForSlice(sliceCopy, sliceOwnerNamespace.Labels["authority-name"])
			if resourcesAvailability {
				// When a slice is deleted, the owner references feature allows the namespace to be automatically removed. Additionally,
				// when all users who participate in the slice are disabled, the slice is automatically removed because of the owner references.
				// Each namespace created by slices have an indicator as "slice" to provide singularity
				sliceChildNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: sliceChildNamespaceStr}}
				// Namespace labels indicate this namespace created by a slice, not by a authority or team
				namespaceLabels := map[string]string{"owner": "slice", "owner-name": sliceCopy.GetName(), "authority-name": sliceOwnerNamespace.Labels["authority-name"]}
				sliceChildNamespace.SetLabels(namespaceLabels)
				sliceChildNamespaceCreated, err := t.clientset.CoreV1().Namespaces().Create(sliceChildNamespace)
				if err == nil {
					// Create rolebindings according to the users who participate in the slice and are authority-admin and authorized users of the authority
					t.runUserInteractions(sliceCopy, sliceChildNamespaceCreated.GetName(), sliceOwnerNamespace.Labels["authority-name"],
						sliceOwnerNamespace.Labels["owner"], sliceOwnerNamespace.Labels["owner-name"], "slice-creation", true)
					// To set constraints in the slice namespace and to update the expiration date of slice
					sliceCopy = t.setConstrainsByProfile(sliceChildNamespaceCreated.GetName(), sliceCopy)
					sliceOwnerReferences := t.setOwnerReferences(sliceChildNamespaceCreated)
					sliceCopy.ObjectMeta.OwnerReferences = sliceOwnerReferences
					t.edgenetClientset.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Update(sliceCopy)
				} else {
					t.runUserInteractions(sliceCopy, sliceChildNamespaceCreated.GetName(), sliceOwnerNamespace.Labels["authority-name"],
						sliceOwnerNamespace.Labels["owner"], sliceOwnerNamespace.Labels["owner-name"], "slice-crash", true)
					t.edgenetClientset.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Delete(sliceCopy.GetName(), &metav1.DeleteOptions{})
					return
				}
			} else if !resourcesAvailability {
				log.Printf("Total resource quota exceeded for %s, %s couldn't be generated", sliceOwnerNamespace.Labels["authority-name"], sliceCopy.GetName())
				t.runUserInteractions(sliceCopy, sliceChildNamespaceStr, sliceOwnerNamespace.Labels["authority-name"], sliceOwnerNamespace.Labels["owner"], sliceOwnerNamespace.Labels["owner-name"], "slice-total-quota-exceeded", false)
				t.edgenetClientset.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Delete(sliceCopy.GetName(), &metav1.DeleteOptions{})
			}
		}
		// Run timeout goroutine
		go t.runTimeout(sliceCopy)
	} else {
		t.edgenetClientset.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Delete(sliceCopy.GetName(), &metav1.DeleteOptions{})
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj, updated interface{}) {
	log.Info("SliceHandler.ObjectUpdated")
	// Create a copy of the slice object to make changes on it
	sliceCopy := obj.(*apps_v1alpha.Slice).DeepCopy()
	// Find the authority from the namespace in which the object is
	sliceOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(sliceCopy.GetNamespace(), metav1.GetOptions{})
	sliceOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(sliceOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	sliceChildNamespaceStr := fmt.Sprintf("%s-slice-%s", sliceCopy.GetNamespace(), sliceCopy.GetName())
	fieldUpdated := updated.(fields)
	// The section below checks whether the slice belongs to a team or directly to a authority. After then, set the value as enabled
	// if the authority and the team (if it is an owner) enabled.
	var sliceOwnerEnabled bool
	if sliceOwnerNamespace.Labels["owner"] == "team" {
		sliceOwnerEnabled = sliceOwnerAuthority.Spec.Enabled
		if sliceOwnerEnabled {
			sliceOwnerTeam, _ := t.edgenetClientset.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", sliceOwnerNamespace.Labels["authority-name"])).
				Get(sliceOwnerNamespace.Labels["owner-name"], metav1.GetOptions{})
			sliceOwnerEnabled = sliceOwnerTeam.Spec.Enabled
		}
	} else {
		sliceOwnerEnabled = sliceOwnerAuthority.Spec.Enabled
	}
	// Check if the owner(s) is/are active
	if sliceOwnerEnabled {
		// If the users who participate in the slice have changed
		if fieldUpdated.users.status { // Delete all existing role bindings in the slice (child) namespace
			t.clientset.RbacV1().RoleBindings(sliceChildNamespaceStr).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
			// Create role bindings in the slice namespace from scratch
			t.runUserInteractions(sliceCopy, sliceChildNamespaceStr, sliceOwnerNamespace.Labels["authority-name"],
				sliceOwnerNamespace.Labels["owner"], sliceOwnerNamespace.Labels["owner-name"], "slice-creation", false)
			// Send emails to those who have been added to, or removed from the slice.
			var deletedUserList []apps_v1alpha.SliceUsers
			json.Unmarshal([]byte(fieldUpdated.users.deleted), &deletedUserList)
			var addedUserList []apps_v1alpha.SliceUsers
			json.Unmarshal([]byte(fieldUpdated.users.added), &addedUserList)
			if len(deletedUserList) > 0 {
				for _, deletedUser := range deletedUserList {
					t.sendEmail(deletedUser.Username, deletedUser.Authority, sliceOwnerNamespace.Labels["authority-name"], sliceCopy.GetNamespace(), sliceCopy.GetName(), sliceChildNamespaceStr, "slice-removal")
				}
			}
			if len(addedUserList) > 0 {
				for _, addedUser := range addedUserList {
					t.sendEmail(addedUser.Username, addedUser.Authority, sliceOwnerNamespace.Labels["authority-name"], sliceCopy.GetNamespace(), sliceCopy.GetName(), sliceChildNamespaceStr, "slice-creation")
				}
			}
		}
		// If the slice renewed or its profile updated
		if sliceCopy.Spec.Renew || fieldUpdated.profile.status {
			// Delete all existing resource quotas in the slice (child) namespace
			t.clientset.CoreV1().ResourceQuotas(sliceChildNamespaceStr).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
			if sliceCopy.Spec.Renew {
				sliceCopy.Spec.Renew = false
				sliceCopyUpdate, err := t.edgenetClientset.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Update(sliceCopy)
				if err == nil {
					sliceCopy = sliceCopyUpdate
				}
			}
			if fieldUpdated.profile.status {
				resourcesAvailability := t.checkResourcesAvailabilityForSlice(sliceCopy, sliceOwnerNamespace.Labels["authority-name"])
				if !resourcesAvailability {
					sliceCopy.Spec.Profile = fieldUpdated.profile.old
					sliceCopyUpdate, err := t.edgenetClientset.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Update(sliceCopy)
					if err == nil {
						sliceCopy = sliceCopyUpdate
						t.runUserInteractions(sliceCopy, sliceChildNamespaceStr, sliceOwnerNamespace.Labels["authority-name"], sliceOwnerNamespace.Labels["owner"], sliceOwnerNamespace.Labels["owner-name"], "slice-lack-of-quota", false)
					}
				}
			}
			t.setConstrainsByProfile(sliceChildNamespaceStr, sliceCopy)
		}
	} else {
		t.edgenetClientset.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Delete(sliceCopy.GetName(), &metav1.DeleteOptions{})
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("SliceHandler.ObjectDeleted")
	// Mail notification, TBD
}

func (t *Handler) checkResourcesAvailabilityForSlice(sliceCopy *apps_v1alpha.Slice, authorityName string) bool {
	TRQCopy, err := t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().Get(authorityName, metav1.GetOptions{})
	quotaExceeded := true
	if err == nil {
		TRQHandler := totalresourcequota.Handler{}
		err = TRQHandler.Init()
		if err == nil {
			switch sliceCopy.Spec.Profile {
			case "Low":
				_, quotaExceeded = TRQHandler.ResourceConsumptionControl(TRQCopy, t.lowResourceQuota.Spec.Hard.Cpu().Value(), t.lowResourceQuota.Spec.Hard.Memory().Value())
			case "Medium":
				_, quotaExceeded = TRQHandler.ResourceConsumptionControl(TRQCopy, t.medResourceQuota.Spec.Hard.Cpu().Value(), t.medResourceQuota.Spec.Hard.Memory().Value())
			case "High":
				_, quotaExceeded = TRQHandler.ResourceConsumptionControl(TRQCopy, t.highResourceQuota.Spec.Hard.Cpu().Value(), t.highResourceQuota.Spec.Hard.Memory().Value())
			}
		}
	}
	return !quotaExceeded
}

// setConstrainsByProfile allocates the resources corresponding to the slice profile and defines the expiration date
func (t *Handler) setConstrainsByProfile(childNamespace string, sliceCopy *apps_v1alpha.Slice) *apps_v1alpha.Slice {
	switch sliceCopy.Spec.Profile {
	case "Low":
		// Set the timeout which is 6 weeks for medium profile slices
		if sliceCopy.Spec.Renew || sliceCopy.Status.Expires == nil {
			sliceCopy.Status.Expires = &metav1.Time{
				Time: time.Now().Add(1344 * time.Hour),
			}
		} else {
			sliceCopy.Status.Expires = &metav1.Time{
				Time: sliceCopy.CreationTimestamp.Add(1344 * time.Hour),
			}
		}
		t.clientset.CoreV1().ResourceQuotas(childNamespace).Create(t.lowResourceQuota)
	case "Medium":
		// Set the timeout which is 4 weeks for medium profile slices
		if sliceCopy.Spec.Renew || sliceCopy.Status.Expires == nil {
			sliceCopy.Status.Expires = &metav1.Time{
				Time: time.Now().Add(672 * time.Hour),
			}
		} else {
			sliceCopy.Status.Expires = &metav1.Time{
				Time: sliceCopy.CreationTimestamp.Add(672 * time.Hour),
			}
		}
		t.clientset.CoreV1().ResourceQuotas(childNamespace).Create(t.medResourceQuota)
	case "High":
		// Set the timeout which is 2 weeks for high profile slices
		if sliceCopy.Spec.Renew || sliceCopy.Status.Expires == nil {
			sliceCopy.Status.Expires = &metav1.Time{
				Time: time.Now().Add(336 * time.Hour),
			}
		} else {
			sliceCopy.Status.Expires = &metav1.Time{
				Time: sliceCopy.CreationTimestamp.Add(336 * time.Hour),
			}
		}
		t.clientset.CoreV1().ResourceQuotas(childNamespace).Create(t.highResourceQuota)
	}
	sliceCopyUpdate, _ := t.edgenetClientset.AppsV1alpha().Slices(sliceCopy.GetNamespace()).UpdateStatus(sliceCopy)
	return sliceCopyUpdate
}

// runUserInteractions creates user role bindings according to the roles and send emails separately
func (t *Handler) runUserInteractions(sliceCopy *apps_v1alpha.Slice, sliceChildNamespaceStr, ownerAuthority, sliceOwner, sliceOwnerName, operation string, firstCreation bool) {
	// This part for the users who participate in the slice
	for _, sliceUser := range sliceCopy.Spec.Users {
		user, err := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("authority-%s", sliceUser.Authority)).Get(sliceUser.Username, metav1.GetOptions{})
		if err == nil && user.Spec.Active && user.Status.AUP {
			if operation == "slice-creation" {
				registration.EstablishRoleBindings(user.DeepCopy(), sliceChildNamespaceStr, "Slice")
			}
			if !(operation == "slice-creation" && !firstCreation) {
				t.sendEmail(sliceUser.Username, sliceUser.Authority, ownerAuthority, sliceCopy.GetNamespace(), sliceCopy.GetName(), sliceChildNamespaceStr, operation)
			}
		}
	}

	if !(sliceOwner == "team" && operation != "slice-creation") {
		// For those who are authority-admin and authorized users of the authority
		userRaw, err := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("authority-%s", ownerAuthority)).List(metav1.ListOptions{})
		if err == nil {
			for _, userRow := range userRaw.Items {
				if userRow.Spec.Active && userRow.Status.AUP && (userRow.Status.Type == "admin" ||
					authorization.CheckUserRole(t.clientset, sliceCopy.GetNamespace(), userRow.Spec.Email, "slices", sliceCopy.GetName())) {
					if operation == "slice-creation" {
						registration.EstablishRoleBindings(userRow.DeepCopy(), sliceChildNamespaceStr, "Slice")
						//mailSubject = "creation"
					}
					/*if !(operation == "slice-creation" && !firstCreation) && !(operation == "slice-creation" && sliceOwner == "team") {
						t.sendEmail(userRow.GetName(), ownerAuthority, ownerAuthority, sliceCopy.GetName(), sliceChildNamespaceStr, mailSubject)
					}*/
				}
			}
		}
	}
}

// setOwnerReferences returns the namespace as owner
func (t *Handler) setOwnerReferences(childNamespace *corev1.Namespace) []metav1.OwnerReference {
	// The section below makes the child namespace become the slice owner
	newSliceRef := *metav1.NewControllerRef(childNamespace, corev1.SchemeGroupVersion.WithKind("Namespace"))
	takeControl := false
	newSliceRef.Controller = &takeControl
	ownerReferences := []metav1.OwnerReference{newSliceRef}
	return ownerReferences
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(sliceUsername, sliceUserAuthority, sliceAuthority, sliceOwnerNamespace, sliceName, sliceNamespace, subject string) {
	user, err := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("authority-%s", sliceUserAuthority)).Get(sliceUsername, metav1.GetOptions{})
	if err == nil && user.Spec.Active && user.Status.AUP {
		// Set the HTML template variables
		contentData := mailer.ResourceAllocationData{}
		contentData.CommonData.Authority = sliceUserAuthority
		contentData.CommonData.Username = sliceUsername
		contentData.CommonData.Name = fmt.Sprintf("%s %s", user.Spec.FirstName, user.Spec.LastName)
		contentData.CommonData.Email = []string{user.Spec.Email}
		contentData.Authority = sliceAuthority
		contentData.Name = sliceName
		contentData.OwnerNamespace = sliceOwnerNamespace
		contentData.ChildNamespace = sliceNamespace
		mailer.Send(subject, contentData)
	}
}

// runTimeout puts a procedure in place to remove slice after the timeout
func (t *Handler) runTimeout(sliceCopy *apps_v1alpha.Slice) {
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	var reminder <-chan time.Time
	if sliceCopy.Status.Expires != nil {
		timeout = time.After(time.Until(sliceCopy.Status.Expires.Time))
		reminder = time.After(time.Until(sliceCopy.Status.Expires.Time.Add(time.Hour * -72)))
	}
	closeChannels := func() {
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of slice object
	watchSlice, err := t.edgenetClientset.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Watch(metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", sliceCopy.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for SliceEvent := range watchSlice.ResultChan() {
				// Get updated slice object
				updatedSlice, status := SliceEvent.Object.(*apps_v1alpha.Slice)
				// FieldSelector doesn't work properly, and will be checked in for next releases.
				if sliceCopy.GetUID() == updatedSlice.GetUID() {
					if status {
						if SliceEvent.Type == "DELETED" {
							terminated <- true
							continue
						}

						if updatedSlice.Status.Expires != nil {
							// Check whether expiration date updated - TBD
							/*if sliceCopy.Status.Expires != nil && timeout != nil {
								if sliceCopy.Status.Expires.Time == updatedSlice.Status.Expires.Time {
									sliceCopy = updatedSlice
									continue
								}
							}*/

							if updatedSlice.Status.Expires.Time.Sub(time.Now()) >= 0 {
								timeout = time.After(time.Until(updatedSlice.Status.Expires.Time))
								reminder = time.After(time.Until(updatedSlice.Status.Expires.Time.Add(time.Hour * -72)))
								timeoutRenewed <- true
							} else {
								terminated <- true
							}
						}
						sliceCopy = updatedSlice
					}
				}
			}
		}()
	} else {
		// In case of any malfunction of watching slice resources,
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
		case <-reminder:
			sliceOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(sliceCopy.GetNamespace(), metav1.GetOptions{})
			sliceChildNamespaceStr := fmt.Sprintf("%s-slice-%s", sliceCopy.GetNamespace(), sliceCopy.GetName())
			t.runUserInteractions(sliceCopy, sliceChildNamespaceStr, sliceOwnerNamespace.Labels["authority-name"], sliceOwnerNamespace.Labels["owner"], sliceOwnerNamespace.Labels["owner-name"], "slice-reminder", false)
			break timeoutOptions
		case <-timeout:
			t.edgenetClientset.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Delete(sliceCopy.GetName(), &metav1.DeleteOptions{})
			break timeoutOptions
		case <-terminated:
			watchSlice.Stop()
			sliceOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(sliceCopy.GetNamespace(), metav1.GetOptions{})
			sliceChildNamespaceStr := fmt.Sprintf("%s-slice-%s", sliceCopy.GetNamespace(), sliceCopy.GetName())
			t.runUserInteractions(sliceCopy, sliceChildNamespaceStr, sliceOwnerNamespace.Labels["authority-name"], sliceOwnerNamespace.Labels["owner"], sliceOwnerNamespace.Labels["owner-name"], "slice-deletion", false)
			t.clientset.CoreV1().Namespaces().Delete(sliceChildNamespaceStr, &metav1.DeleteOptions{})
			TRQCopy, err := t.edgenetClientset.AppsV1alpha().TotalResourceQuotas().Get(sliceOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
			if err == nil {
				TRQHandler := totalresourcequota.Handler{}
				err = TRQHandler.Init()
				if err == nil {
					TRQHandler.ResourceConsumptionControl(TRQCopy, 0, 0)
				}
			}
			closeChannels()
			break timeoutLoop
		}
	}
}
