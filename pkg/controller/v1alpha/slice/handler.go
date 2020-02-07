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
	"fmt"
	"strings"
	"time"

	apps_v1alpha "headnode/pkg/apis/apps/v1alpha"
	"headnode/pkg/authorization"
	"headnode/pkg/client/clientset/versioned"
	"headnode/pkg/mailer"
	"headnode/pkg/registration"

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
			"cpu":              resource.MustParse("1000m"),
			"memory":           resource.MustParse("1024Mi"),
			"requests.storage": resource.MustParse("250Mi"),
		},
	}
	t.medResourceQuota = &corev1.ResourceQuota{}
	t.medResourceQuota.Name = "slice-medium-quota"
	t.medResourceQuota.Spec = corev1.ResourceQuotaSpec{
		Hard: map[corev1.ResourceName]resource.Quantity{
			"cpu":              resource.MustParse("2000m"),
			"memory":           resource.MustParse("2048Mi"),
			"requests.storage": resource.MustParse("500Mi"),
		},
	}
	t.highResourceQuota = &corev1.ResourceQuota{}
	t.highResourceQuota.Name = "slice-high-quota"
	t.highResourceQuota.Spec = corev1.ResourceQuotaSpec{
		Hard: map[corev1.ResourceName]resource.Quantity{
			"cpu":              resource.MustParse("4000m"),
			"memory":           resource.MustParse("4096Mi"),
			"requests.storage": resource.MustParse("1Gi"),
		},
	}
	return err
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("SliceHandler.ObjectCreated")
	// Create a copy of the slice object to make changes on it
	sliceCopy := obj.(*apps_v1alpha.Slice).DeepCopy()
	// Find the site from the namespace in which the object is
	sliceOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(sliceCopy.GetNamespace(), metav1.GetOptions{})
	sliceOwnerSite, _ := t.edgenetClientset.AppsV1alpha().Sites().Get(sliceOwnerNamespace.Labels["site-name"], metav1.GetOptions{})
	// The section below checks whether the slice belongs to a project or directly to a site. After then, set the value as enabled
	// if the site and the project (if it is an owner) enabled.
	var sliceOwnerEnabled bool
	if sliceOwnerNamespace.Labels["owner"] == "project" {
		sliceOwnerEnabled = sliceOwnerSite.Status.Enabled
		if sliceOwnerEnabled {
			sliceOwnerProject, _ := t.edgenetClientset.AppsV1alpha().Projects(fmt.Sprintf("site-%s", sliceOwnerNamespace.Labels["site-name"])).
				Get(sliceOwnerNamespace.Labels["owner-name"], metav1.GetOptions{})
			sliceOwnerEnabled = sliceOwnerProject.Status.Enabled
		}
	} else {
		sliceOwnerEnabled = sliceOwnerSite.Status.Enabled
	}
	// Check if the owner(s) is/are active
	if sliceOwnerEnabled {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		if sliceCopy.Status.Expires == nil {
			// When a slice is deleted, the owner references feature allows the namespace to be automatically removed. Additionally,
			// when all users who participate in the slice are disabled, the slice is automatically removed because of the owner references.
			sliceOwnerReferences, sliceChildNamespaceOwnerReferences := t.setOwnerReferences(sliceCopy)
			sliceCopy.ObjectMeta.OwnerReferences = sliceOwnerReferences
			sliceCopyUpdated, _ := t.edgenetClientset.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Update(sliceCopy)
			sliceCopy = sliceCopyUpdated
			// Each namespace created by slices have an indicator as "slice" to provide singularity
			sliceChildNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-slice-%s", sliceCopy.GetNamespace(), sliceCopy.GetName()), OwnerReferences: sliceChildNamespaceOwnerReferences}}
			// Namespace labels indicate this namespace created by a slice, not by a site or project
			namespaceLabels := map[string]string{"owner": "slice", "owner-name": sliceCopy.GetName(), "site-name": sliceOwnerNamespace.Labels["site-name"]}
			sliceChildNamespace.SetLabels(namespaceLabels)
			sliceChildNamespaceCreated, err := t.clientset.CoreV1().Namespaces().Create(sliceChildNamespace)
			if err == nil {
				// Create rolebindings according to the users who participate in the slice and are PI and managers of the site
				t.createRoleBindings(sliceChildNamespaceCreated.GetName(), sliceCopy, sliceOwnerNamespace.Labels["site-name"])
			} else {
				t.edgenetClientset.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Delete(sliceCopy.GetName(), &metav1.DeleteOptions{})
			}
			// To set constraints in the slice namespace and to update the expiration date of slice
			sliceCopy = t.setConstrainsByProfile(sliceChildNamespaceCreated.GetName(), sliceCopy)
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
	// Find the site from the namespace in which the object is
	sliceOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(sliceCopy.GetNamespace(), metav1.GetOptions{})
	sliceOwnerSite, _ := t.edgenetClientset.AppsV1alpha().Sites().Get(sliceOwnerNamespace.Labels["site-name"], metav1.GetOptions{})
	sliceChildNamespaceStr := fmt.Sprintf("%s-slice-%s", sliceCopy.GetNamespace(), sliceCopy.GetName())
	fieldUpdated := updated.(fields)
	// The section below checks whether the slice belongs to a project or directly to a site. After then, set the value as enabled
	// if the site and the project (if it is an owner) enabled.
	var sliceOwnerEnabled bool
	if sliceOwnerNamespace.Labels["owner"] == "project" {
		sliceOwnerEnabled = sliceOwnerSite.Status.Enabled
		if sliceOwnerEnabled {
			sliceOwnerProject, _ := t.edgenetClientset.AppsV1alpha().Projects(fmt.Sprintf("site-%s", sliceOwnerNamespace.Labels["site-name"])).
				Get(sliceOwnerNamespace.Labels["owner-name"], metav1.GetOptions{})
			sliceOwnerEnabled = sliceOwnerProject.Status.Enabled
		}
	} else {
		sliceOwnerEnabled = sliceOwnerSite.Status.Enabled
	}
	// Check if the owner(s) is/are active
	if sliceOwnerEnabled {
		// If the users who participate in the slice have changed
		if fieldUpdated.users {
			// Delete all existing role bindings in the slice (child) namespace
			t.clientset.RbacV1().RoleBindings(sliceChildNamespaceStr).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
			// Create role bindings in the slice namespace from scratch
			t.createRoleBindings(sliceChildNamespaceStr, sliceCopy, sliceOwnerNamespace.Labels["site-name"])
			// Update the owner references of the slice
			sliceOwnerReferences, _ := t.setOwnerReferences(sliceCopy)
			sliceCopy.ObjectMeta.OwnerReferences = sliceOwnerReferences
			sliceCopyUpdated, _ := t.edgenetClientset.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Update(sliceCopy)
			sliceCopy = sliceCopyUpdated
		}
		// If the slice renewed or its profile updated
		if sliceCopy.Status.Renew || fieldUpdated.profile {
			// Delete all existing resource quotas in the slice (child) namespace
			t.clientset.CoreV1().ResourceQuotas(sliceChildNamespaceStr).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
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

// setConstrainsByProfile allocates the resources corresponding to the slice profile and defines the expiration date
func (t *Handler) setConstrainsByProfile(childNamespace string, sliceCopy *apps_v1alpha.Slice) *apps_v1alpha.Slice {
	switch sliceCopy.Spec.Profile {
	case "Low":
		// Set the timeout which is 6 weeks for medium profile slices
		if sliceCopy.Status.Renew || sliceCopy.Status.Expires == nil {
			sliceCopy.Status.Expires = &metav1.Time{
				Time: time.Now().Add(1344 * time.Hour),
			}
		}
		t.clientset.CoreV1().ResourceQuotas(childNamespace).Create(t.lowResourceQuota)
	case "Medium":
		// Set the timeout which is 4 weeks for medium profile slices
		if sliceCopy.Status.Renew || sliceCopy.Status.Expires == nil {
			sliceCopy.Status.Expires = &metav1.Time{
				Time: time.Now().Add(672 * time.Hour),
			}
		}
		t.clientset.CoreV1().ResourceQuotas(childNamespace).Create(t.medResourceQuota)
	case "High":
		// Set the timeout which is 2 weeks for high profile slices
		if sliceCopy.Status.Renew || sliceCopy.Status.Expires == nil {
			sliceCopy.Status.Expires = &metav1.Time{
				Time: time.Now().Add(336 * time.Hour),
			}
		}
		t.clientset.CoreV1().ResourceQuotas(childNamespace).Create(t.highResourceQuota)
	}
	sliceCopy.Status.Renew = false
	sliceCopyUpdate, _ := t.edgenetClientset.AppsV1alpha().Slices(sliceCopy.GetNamespace()).UpdateStatus(sliceCopy)
	return sliceCopyUpdate
}

// createRoleBindings creates user role bindings according to the roles
func (t *Handler) createRoleBindings(sliceChildNamespaceStr string, sliceCopy *apps_v1alpha.Slice, ownerSite string) {
	// This part creates the rolebindings for the users who participate in the slice
	for _, sliceUser := range sliceCopy.Spec.Users {
		user, err := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("site-%s", sliceUser.Site)).Get(sliceUser.Username, metav1.GetOptions{})
		if err == nil && user.Status.Active && user.Status.AUP {
			registration.CreateRoleBindingsByRoles(user.DeepCopy(), sliceChildNamespaceStr, "Slice")
			contentData := mailer.ResourceAllocationData{}
			contentData.CommonData.Site = sliceUser.Site
			contentData.CommonData.Username = sliceUser.Username
			contentData.CommonData.Name = fmt.Sprintf("%s %s", user.Spec.FirstName, user.Spec.LastName)
			contentData.CommonData.Email = []string{user.Spec.Email}
			contentData.Site = ownerSite
			contentData.Name = sliceCopy.GetName()
			contentData.Namespace = sliceCopy.GetNamespace()
			mailer.Send("slice-creation", contentData)
		}
	}
	// To create the rolebindings for the users who are PI and managers of the site
	userRaw, err := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("site-%s", ownerSite)).List(metav1.ListOptions{})
	if err == nil {
		for _, userRow := range userRaw.Items {
			if userRow.Status.Active && userRow.Status.AUP && (containsRole(userRow.Spec.Roles, "pi") || containsRole(userRow.Spec.Roles, "manager")) {
				registration.CreateRoleBindingsByRoles(userRow.DeepCopy(), sliceChildNamespaceStr, "Slice")
				contentData := mailer.ResourceAllocationData{}
				contentData.CommonData.Site = ownerSite
				contentData.CommonData.Username = userRow.GetName()
				contentData.CommonData.Name = fmt.Sprintf("%s %s", userRow.Spec.FirstName, userRow.Spec.LastName)
				contentData.CommonData.Email = []string{userRow.Spec.Email}
				contentData.Site = ownerSite
				contentData.Name = sliceCopy.GetName()
				contentData.Namespace = sliceCopy.GetNamespace()
				mailer.Send("project-creation", contentData)
			}
		}
	}
}

// setOwnerReferences returns the users and the slice as owners
func (t *Handler) setOwnerReferences(sliceCopy *apps_v1alpha.Slice) ([]metav1.OwnerReference, []metav1.OwnerReference) {
	// The following section makes users who participate in that slice become the slice owners
	ownerReferences := []metav1.OwnerReference{}
	for _, sliceUser := range sliceCopy.Spec.Users {
		user, err := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("site-%s", sliceUser.Site)).Get(sliceUser.Username, metav1.GetOptions{})
		if err == nil && user.Status.Active && user.Status.AUP {
			newSliceRef := *metav1.NewControllerRef(user.DeepCopy(), apps_v1alpha.SchemeGroupVersion.WithKind("User"))
			takeControl := false
			newSliceRef.Controller = &takeControl
			ownerReferences = append(ownerReferences, newSliceRef)
		}
	}
	// The section below makes slice who created the child namespace become the namespace owner
	newNamespaceRef := *metav1.NewControllerRef(sliceCopy, apps_v1alpha.SchemeGroupVersion.WithKind("Slice"))
	takeControl := false
	newNamespaceRef.Controller = &takeControl
	namespaceOwnerReferences := []metav1.OwnerReference{newNamespaceRef}
	return ownerReferences, namespaceOwnerReferences
}

// runTimeout puts a procedure in place to remove slice after the timeout
func (t *Handler) runTimeout(sliceCopy *apps_v1alpha.Slice) {
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
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
				if status {
					if SliceEvent.Type == "DELETED" {
						terminated <- true
						continue
					}

					if updatedSlice.Status.Expires != nil {
						// Check whether expiration date updated
						if sliceCopy.Status.Expires != nil && timeout != nil {
							if sliceCopy.Status.Expires.Time == updatedSlice.Status.Expires.Time {
								sliceCopy = updatedSlice
								continue
							}
						}

						if updatedSlice.Status.Expires.Time.Sub(time.Now()) >= 0 {
							timeout = time.After(time.Until(updatedSlice.Status.Expires.Time))
							timeoutRenewed <- true
						}
					}
					sliceCopy = updatedSlice
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
		case <-timeout:
			watchSlice.Stop()
			t.edgenetClientset.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Delete(sliceCopy.GetName(), &metav1.DeleteOptions{})
			closeChannels()
			break timeoutLoop
		case <-terminated:
			watchSlice.Stop()
			closeChannels()
			break timeoutLoop
		}
	}
}

// To check whether user is holder of a role
func containsRole(roles []string, value string) bool {
	for _, ele := range roles {
		if strings.ToLower(value) == strings.ToLower(ele) {
			return true
		}
	}
	return false
}
