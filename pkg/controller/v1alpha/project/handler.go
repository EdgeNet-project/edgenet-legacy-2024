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

package project

import (
	"fmt"
	"strings"

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
	clientset        *kubernetes.Clientset
	edgenetClientset *versioned.Clientset
	resourceQuota    *corev1.ResourceQuota
}

// Init handles any handler initialization
func (t *Handler) Init() error {
	log.Info("ProjectHandler.Init")
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
	t.resourceQuota = &corev1.ResourceQuota{}
	t.resourceQuota.Name = "project-quota"
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
	log.Info("ProjectHandler.ObjectCreated")
	// Create a copy of the project object to make changes on it
	projectCopy := obj.(*apps_v1alpha.Project).DeepCopy()
	// Find the site from the namespace in which the object is
	projectOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(projectCopy.GetNamespace(), metav1.GetOptions{})
	projectOwnerSite, _ := t.edgenetClientset.AppsV1alpha().Sites().Get(projectOwnerNamespace.Labels["site-name"], metav1.GetOptions{})
	// Check if the site is active
	if projectOwnerSite.Status.Enabled && projectCopy.GetGeneration() == 1 {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		_, err := t.clientset.CoreV1().Namespaces().Get(fmt.Sprintf("project-%s", projectCopy.GetName()), metav1.GetOptions{})
		if err != nil {
			// When a project is deleted, the owner references feature allows the namespace to be automatically removed. Additionally,
			// when all users who participate in the project are disabled, the project is automatically removed because of the owner references.
			projectOwnerReferences, projectChildNamespaceOwnerReferences := t.setOwnerReferences(projectCopy)
			projectCopy.ObjectMeta.OwnerReferences = projectOwnerReferences
			projectCopyUpdated, _ := t.edgenetClientset.AppsV1alpha().Projects(projectCopy.GetNamespace()).Update(projectCopy)
			projectCopy = projectCopyUpdated
			// Enable the project
			projectCopy.Status.Enabled = true
			defer t.edgenetClientset.AppsV1alpha().Projects(projectCopy.GetNamespace()).UpdateStatus(projectCopy)
			// Each namespace created by projects have an indicator as "project" to provide singularity
			projectChildNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-project-%s", projectCopy.GetNamespace(), projectCopy.GetName()), OwnerReferences: projectChildNamespaceOwnerReferences}}
			// Namespace labels indicate this namespace created by a project, not by a site or slice
			namespaceLabels := map[string]string{"owner": "project", "owner-name": projectCopy.GetName(), "site-name": projectOwnerNamespace.Labels["site-name"]}
			projectChildNamespace.SetLabels(namespaceLabels)
			t.clientset.CoreV1().Namespaces().Create(projectChildNamespace)
		}
	} else if !projectOwnerSite.Status.Enabled {
		t.edgenetClientset.AppsV1alpha().Projects(projectCopy.GetNamespace()).Delete(projectCopy.GetName(), &metav1.DeleteOptions{})
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj, updated interface{}) {
	log.Info("ProjectHandler.ObjectUpdated")
	// Create a copy of the project object to make changes on it
	projectCopy := obj.(*apps_v1alpha.Project).DeepCopy()
	// Find the site from the namespace in which the object is
	projectOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(projectCopy.GetNamespace(), metav1.GetOptions{})
	projectOwnerSite, _ := t.edgenetClientset.AppsV1alpha().Sites().Get(projectOwnerNamespace.Labels["site-name"], metav1.GetOptions{})
	projectChildNamespaceStr := fmt.Sprintf("%s-project-%s", projectCopy.GetNamespace(), projectCopy.GetName())
	fieldUpdated := updated.(fields)
	// Check if the site and project are active
	if projectOwnerSite.Status.Enabled && projectCopy.Status.Enabled {
		if fieldUpdated.users || fieldUpdated.enabled {
			// Delete all existing role bindings in the project (child) namespace
			t.clientset.RbacV1().RoleBindings(projectChildNamespaceStr).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
			// Create rolebindings according to the users who participate in the project and are PI and managers of the site
			t.createRoleBindings(projectChildNamespaceStr, projectCopy, projectOwnerNamespace.Labels["site-name"])
			// Update the owner references of the project
			projectOwnerReferences, _ := t.setOwnerReferences(projectCopy)
			projectCopy.ObjectMeta.OwnerReferences = projectOwnerReferences
			t.edgenetClientset.AppsV1alpha().Projects(projectCopy.GetNamespace()).Update(projectCopy)
		}
	} else if projectOwnerSite.Status.Enabled && !projectCopy.Status.Enabled {
		t.edgenetClientset.AppsV1alpha().Slices(projectChildNamespaceStr).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
		t.clientset.RbacV1().RoleBindings(projectChildNamespaceStr).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
	} else if !projectOwnerSite.Status.Enabled {
		t.edgenetClientset.AppsV1alpha().Projects(projectChildNamespaceStr).Delete(projectCopy.GetName(), &metav1.DeleteOptions{})
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("ProjectHandler.ObjectDeleted")
	// Mail notification, TBD
}

// createRoleBindings creates user role bindings according to the roles
func (t *Handler) createRoleBindings(projectChildNamespaceStr string, projectCopy *apps_v1alpha.Project, ownerSite string) {
	// This part creates the rolebindings for the users who participate in the project
	for _, projectUser := range projectCopy.Spec.Users {
		user, err := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("site-%s", projectUser.Site)).Get(projectUser.Username, metav1.GetOptions{})
		if err == nil && user.Status.Active && user.Status.AUP {
			registration.CreateRoleBindingsByRoles(user.DeepCopy(), projectChildNamespaceStr, "Project")
			contentData := mailer.ResourceAllocationData{}
			contentData.CommonData.Site = projectUser.Site
			contentData.CommonData.Username = projectUser.Username
			contentData.CommonData.Name = fmt.Sprintf("%s %s", user.Spec.FirstName, user.Spec.LastName)
			contentData.CommonData.Email = []string{user.Spec.Email}
			contentData.Site = ownerSite
			contentData.Name = projectCopy.GetName()
			contentData.Namespace = projectChildNamespaceStr
			mailer.Send("project-invitation", contentData)
		}
	}
	// To create the rolebindings for the users who are PI and managers of the site
	userRaw, err := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("site-%s", ownerSite)).List(metav1.ListOptions{})
	if err == nil {
		for _, userRow := range userRaw.Items {
			if userRow.Status.Active && userRow.Status.AUP && (containsRole(userRow.Spec.Roles, "pi") || containsRole(userRow.Spec.Roles, "manager")) {
				registration.CreateRoleBindingsByRoles(userRow.DeepCopy(), projectChildNamespaceStr, "Project")
				contentData := mailer.ResourceAllocationData{}
				contentData.CommonData.Site = ownerSite
				contentData.CommonData.Username = userRow.GetName()
				contentData.CommonData.Name = fmt.Sprintf("%s %s", userRow.Spec.FirstName, userRow.Spec.LastName)
				contentData.CommonData.Email = []string{userRow.Spec.Email}
				contentData.Site = ownerSite
				contentData.Name = projectCopy.GetName()
				contentData.Namespace = projectChildNamespaceStr
				mailer.Send("project-invitation", contentData)
			}
		}
	}
}

// setOwnerReferences returns the users and the project as owners
func (t *Handler) setOwnerReferences(projectCopy *apps_v1alpha.Project) ([]metav1.OwnerReference, []metav1.OwnerReference) {
	// The following section makes users who participate in that project become the project owners
	ownerReferences := []metav1.OwnerReference{}
	for _, projectUser := range projectCopy.Spec.Users {
		user, err := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("site-%s", projectUser.Site)).Get(projectUser.Username, metav1.GetOptions{})
		if err == nil && user.Status.Active && user.Status.AUP {
			newProjectRef := *metav1.NewControllerRef(user.DeepCopy(), apps_v1alpha.SchemeGroupVersion.WithKind("User"))
			takeControl := false
			newProjectRef.Controller = &takeControl
			ownerReferences = append(ownerReferences, newProjectRef)
		}
	}
	// The section below makes project who created the child namespace become the namespace owner
	newNamespaceRef := *metav1.NewControllerRef(projectCopy, apps_v1alpha.SchemeGroupVersion.WithKind("Project"))
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
