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

package team

import (
	"encoding/json"
	"fmt"
	"strings"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
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
	Init(kubernetes kubernetes.Interface, edgenet versioned.Interface) error
	ObjectCreated(obj interface{})
	ObjectUpdated(obj, updated interface{})
	ObjectDeleted(obj, deleted interface{})
}

// Handler implementation
type Handler struct {
	clientset        kubernetes.Interface
	edgenetClientset versioned.Interface
	resourceQuota    *corev1.ResourceQuota
}

// Init handles any handler initialization
func (t *Handler) Init(kubernetes kubernetes.Interface, edgenet versioned.Interface) error {
	log.Info("TeamHandler.Init")
	var err error
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
	t.resourceQuota = &corev1.ResourceQuota{}
	t.resourceQuota.Name = "team-quota"
	t.resourceQuota.Spec = corev1.ResourceQuotaSpec{
		Hard: map[corev1.ResourceName]resource.Quantity{
			"cpu":                           resource.MustParse("5m"),
			"memory":                        resource.MustParse("1Mi"),
			"requests.storage":              resource.MustParse("1Mi"),
			"pods":                          resource.Quantity{Format: "0"},
			"count/persistentvolumeclaims":  resource.Quantity{Format: "0"},
			"count/services":                resource.Quantity{Format: "0"},
			"count/configmaps":              resource.Quantity{Format: "0"},
			"count/replicationcontrollers":  resource.Quantity{Format: "0"},
			"count/deployments.apps":        resource.Quantity{Format: "0"},
			"count/deployments.extensions":  resource.Quantity{Format: "0"},
			"count/replicasets.apps":        resource.Quantity{Format: "0"},
			"count/replicasets.extensions":  resource.Quantity{Format: "0"},
			"count/statefulsets.apps":       resource.Quantity{Format: "0"},
			"count/statefulsets.extensions": resource.Quantity{Format: "0"},
			"count/jobs.batch":              resource.Quantity{Format: "0"},
			"count/cronjobs.batch":          resource.Quantity{Format: "0"},
		},
	}
	return err
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("TeamHandler.ObjectCreated")
	// Create a copy of the team object to make changes on it
	teamCopy := obj.(*apps_v1alpha.Team).DeepCopy()
	// Find the authority from the namespace in which the object is
	teamOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(teamCopy.GetNamespace(), metav1.GetOptions{})
	teamOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(teamOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	// Check if the authority is active
	if teamOwnerAuthority.Status.Enabled && !teamCopy.Status.Enabled {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		_, err := t.clientset.CoreV1().Namespaces().Get(fmt.Sprintf("%s-team-%s", teamCopy.GetNamespace(), teamCopy.GetName()), metav1.GetOptions{})
		if err != nil {
			// When a team is deleted, the owner references feature allows the namespace to be automatically removed. Additionally,
			// when all users who participate in the team are disabled, the team is automatically removed because of the owner references.
			// Enable the team
			teamCopy.Status.Enabled = true
			defer t.edgenetClientset.AppsV1alpha().Teams(teamCopy.GetNamespace()).UpdateStatus(teamCopy)
			// Each namespace created by teams have an indicator as "team" to provide singularity
			teamChildNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-team-%s", teamCopy.GetNamespace(), teamCopy.GetName())}}
			// Namespace labels indicate this namespace created by a team, not by a authority or slice
			namespaceLabels := map[string]string{"owner": "team", "owner-name": teamCopy.GetName(), "authority-name": teamOwnerNamespace.Labels["authority-name"]}
			teamChildNamespace.SetLabels(namespaceLabels)
			teamChildNamespaceCreated, err := t.clientset.CoreV1().Namespaces().Create(teamChildNamespace)
			if err != nil {
				t.runUserInteractions(teamCopy, teamChildNamespaceCreated.GetName(), teamOwnerNamespace.Labels["authority-name"],
					teamOwnerNamespace.Labels["owner"], teamOwnerNamespace.Labels["owner-name"], "team-crash", true)
				t.edgenetClientset.AppsV1alpha().Teams(teamCopy.GetNamespace()).Delete(teamCopy.GetName(), &metav1.DeleteOptions{})
				return
			}
		}
	} else if !teamOwnerAuthority.Status.Enabled {
		t.edgenetClientset.AppsV1alpha().Teams(teamCopy.GetNamespace()).Delete(teamCopy.GetName(), &metav1.DeleteOptions{})
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj, updated interface{}) {
	log.Info("TeamHandler.ObjectUpdated")
	// Create a copy of the team object to make changes on it
	teamCopy := obj.(*apps_v1alpha.Team).DeepCopy()
	// Find the authority from the namespace in which the object is
	teamOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(teamCopy.GetNamespace(), metav1.GetOptions{})
	teamOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(teamOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	teamChildNamespaceStr := fmt.Sprintf("%s-team-%s", teamCopy.GetNamespace(), teamCopy.GetName())
	fieldUpdated := updated.(fields)
	// Check if the authority and team are active
	if teamOwnerAuthority.Status.Enabled && teamCopy.Status.Enabled {
		if fieldUpdated.users.status || fieldUpdated.enabled {
			// Delete all existing role bindings in the team (child) namespace
			t.clientset.RbacV1().RoleBindings(teamChildNamespaceStr).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
			// Create rolebindings according to the users who participate in the team and are authority-admin and managers of the authority
			t.runUserInteractions(teamCopy, teamChildNamespaceStr, teamOwnerNamespace.Labels["authority-name"], teamOwnerNamespace.Labels["owner"], teamOwnerNamespace.Labels["owner-name"], "team-creation", fieldUpdated.enabled)
			// Send emails to those who have been added to, or removed from the slice.
			var deletedUserList []apps_v1alpha.TeamUsers
			json.Unmarshal([]byte(fieldUpdated.users.deleted), &deletedUserList)
			var addedUserList []apps_v1alpha.TeamUsers
			json.Unmarshal([]byte(fieldUpdated.users.added), &addedUserList)
			if len(deletedUserList) > 0 {
				for _, deletedUser := range deletedUserList {
					t.sendEmail(deletedUser.Username, deletedUser.Authority, teamOwnerNamespace.Labels["authority-name"], teamCopy.GetNamespace(), teamCopy.GetName(), teamChildNamespaceStr, "team-removal")
				}
			}
			if len(addedUserList) > 0 {
				for _, addedUser := range addedUserList {
					t.sendEmail(addedUser.Username, addedUser.Authority, teamOwnerNamespace.Labels["authority-name"], teamCopy.GetNamespace(), teamCopy.GetName(), teamChildNamespaceStr, "team-creation")
				}
			}
		}
	} else if teamOwnerAuthority.Status.Enabled && !teamCopy.Status.Enabled {
		t.edgenetClientset.AppsV1alpha().Slices(teamChildNamespaceStr).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
		t.clientset.RbacV1().RoleBindings(teamChildNamespaceStr).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
	} else if !teamOwnerAuthority.Status.Enabled {
		t.edgenetClientset.AppsV1alpha().Teams(teamChildNamespaceStr).Delete(teamCopy.GetName(), &metav1.DeleteOptions{})
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj, deleted interface{}) {
	log.Info("TeamHandler.ObjectDeleted")
	fieldDeleted := deleted.(fields)
	t.clientset.CoreV1().Namespaces().Delete(fieldDeleted.object.childNamespace, &metav1.DeleteOptions{})
	// If there are users who participate in the team and team is enabled
	if fieldDeleted.users.status && fieldDeleted.enabled {
		teamOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(fieldDeleted.object.ownerNamespace, metav1.GetOptions{})
		var deletedUserList []apps_v1alpha.SliceUsers
		json.Unmarshal([]byte(fieldDeleted.users.deleted), &deletedUserList)
		if len(deletedUserList) > 0 {
			for _, deletedUser := range deletedUserList {
				t.sendEmail(deletedUser.Username, deletedUser.Authority, teamOwnerNamespace.Labels["authority-name"], fieldDeleted.object.ownerNamespace, fieldDeleted.object.name, fieldDeleted.object.childNamespace, "team-deletion")
			}
		}
	}
}

// runUserInteractions creates user role bindings according to the roles
func (t *Handler) runUserInteractions(teamCopy *apps_v1alpha.Team, teamChildNamespaceStr, ownerAuthority, teamOwner, teamOwnerName, operation string, enabled bool) {
	// This part creates the rolebindings for the users who participate in the team
	for _, teamUser := range teamCopy.Spec.Users {
		user, err := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("authority-%s", teamUser.Authority)).Get(teamUser.Username, metav1.GetOptions{})
		if err == nil && user.Status.Active && user.Status.AUP {
			if operation == "team-creation" {
				registration.CreateRoleBindingsByRoles(user.DeepCopy(), teamChildNamespaceStr, "Team")
			}

			if !(operation == "team-creation" && !enabled) {
				t.sendEmail(teamUser.Username, teamUser.Authority, ownerAuthority, teamCopy.GetNamespace(), teamCopy.GetName(), teamChildNamespaceStr, operation)
			}
		}
	}
	// To create the rolebindings for the users who are authority-admin and managers of the authority
	userRaw, err := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("authority-%s", ownerAuthority)).List(metav1.ListOptions{})
	if err == nil {
		for _, userRow := range userRaw.Items {
			if userRow.Status.Active && userRow.Status.AUP && (containsRole(userRow.Spec.Roles, "admin") || containsRole(userRow.Spec.Roles, "manager")) {
				registration.CreateRoleBindingsByRoles(userRow.DeepCopy(), teamChildNamespaceStr, "Team")
			}
		}
	}
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(teamUsername, teamUserAuthority, teamAuthority, teamOwnerNamespace, teamName, teamChildNamespace, subject string) {
	user, err := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("authority-%s", teamUserAuthority)).Get(teamUsername, metav1.GetOptions{})
	if err == nil && user.Status.Active && user.Status.AUP {
		// Set the HTML template variables
		contentData := mailer.ResourceAllocationData{}
		contentData.CommonData.Authority = teamUserAuthority
		contentData.CommonData.Username = teamUsername
		contentData.CommonData.Name = fmt.Sprintf("%s %s", user.Spec.FirstName, user.Spec.LastName)
		contentData.CommonData.Email = []string{user.Spec.Email}
		contentData.Authority = teamAuthority
		contentData.Name = teamName
		contentData.OwnerNamespace = teamOwnerNamespace
		contentData.ChildNamespace = teamChildNamespace
		mailer.Send(subject, contentData)
	}
}

// setOwnerReferences returns the users and the team as owners
func (t *Handler) setOwnerReferences(teamCopy *apps_v1alpha.Team) ([]metav1.OwnerReference, []metav1.OwnerReference) {
	// The following section makes users who participate in that team become the team owners
	ownerReferences := []metav1.OwnerReference{}
	for _, teamUser := range teamCopy.Spec.Users {
		user, err := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("authority-%s", teamUser.Authority)).Get(teamUser.Username, metav1.GetOptions{})
		if err == nil && user.Status.Active && user.Status.AUP {
			newTeamRef := *metav1.NewControllerRef(user.DeepCopy(), apps_v1alpha.SchemeGroupVersion.WithKind("User"))
			takeControl := false
			newTeamRef.Controller = &takeControl
			ownerReferences = append(ownerReferences, newTeamRef)
		}
	}
	// The section below makes team who created the child namespace become the namespace owner
	newNamespaceRef := *metav1.NewControllerRef(teamCopy, apps_v1alpha.SchemeGroupVersion.WithKind("Team"))
	takeControl := false
	newNamespaceRef.Controller = &takeControl
	namespaceOwnerReferences := []metav1.OwnerReference{newNamespaceRef}
	return ownerReferences, namespaceOwnerReferences
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
