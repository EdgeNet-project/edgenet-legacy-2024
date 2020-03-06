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

package site

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	apps_v1alpha "headnode/pkg/apis/apps/v1alpha"
	"headnode/pkg/authorization"
	"headnode/pkg/client/clientset/versioned"
	"headnode/pkg/mailer"

	log "github.com/Sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init() error
	ObjectCreated(obj interface{})
	ObjectUpdated(obj interface{})
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
	log.Info("SiteHandler.Init")
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
	t.resourceQuota.Name = "site-quota"
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
	log.Info("SiteHandler.ObjectCreated")
	// Create a copy of the site object to make changes on it
	siteCopy := obj.(*apps_v1alpha.Site).DeepCopy()
	// Check if the email address of user is already taken
	exist := t.checkDuplicateObject(siteCopy)
	if exist {
		// If it is already taken, remove the site object
		t.edgenetClientset.AppsV1alpha().Sites().Delete(siteCopy.GetName(), &metav1.DeleteOptions{})
		return
	}
	if siteCopy.GetGeneration() == 1 && !siteCopy.Status.Enabled {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		_, err := t.clientset.CoreV1().Namespaces().Get(fmt.Sprintf("site-%s", siteCopy.GetName()), metav1.GetOptions{})
		if err != nil {
			// Create a cluster role to be used by site users
			policyRule := []rbacv1.PolicyRule{{APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"sites"}, ResourceNames: []string{siteCopy.GetName()}, Verbs: []string{"get"}}}
			siteRole := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("site-%s", siteCopy.GetName())}, Rules: policyRule}
			_, err := t.clientset.RbacV1().ClusterRoles().Create(siteRole)
			if err != nil {
				log.Infof("Couldn't create site-%s role: %s", siteCopy.GetName(), err)
			}
			// Automatically create a namespace to host users, slices, and projects
			// When a site is deleted, the owner references feature allows the namespace to be automatically removed
			siteOwnerReferences := t.setOwnerReferences(siteCopy)
			// Every namespace of a site has the prefix as "site" to provide singularity
			siteChildNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("site-%s", siteCopy.GetName()), OwnerReferences: siteOwnerReferences}}
			// Namespace labels indicate this namespace created by a site, not by a project or slice
			namespaceLabels := map[string]string{"owner": "site", "owner-name": siteCopy.GetName(), "site-name": siteCopy.GetName()}
			siteChildNamespace.SetLabels(namespaceLabels)
			siteChildNamespaceCreated, _ := t.clientset.CoreV1().Namespaces().Create(siteChildNamespace)
			// Create the resource quota to ban users from using this namespace for their applications
			t.clientset.CoreV1().ResourceQuotas(siteChildNamespaceCreated.GetName()).Create(t.resourceQuota)

			childNamespaceOwnerReferences := t.setNamespaceOwnerReferences(siteChildNamespaceCreated)
			siteCopy.ObjectMeta.OwnerReferences = childNamespaceOwnerReferences
			siteCopyUpdated, err := t.edgenetClientset.AppsV1alpha().Sites().Update(siteCopy)
			if err == nil {
				// To manipulate the object later
				siteCopy = siteCopyUpdated
			}
			// Automatically enable site and update site status
			siteCopy.Status.Enabled = true
			enableSitePI := func() {
				t.edgenetClientset.AppsV1alpha().Sites().UpdateStatus(siteCopy)
				// Create a user as PI on site
				user := apps_v1alpha.User{}
				user.SetName(strings.ToLower(siteCopy.Spec.Contact.Username))
				user.Spec.Email = siteCopy.Spec.Contact.Email
				user.Spec.FirstName = siteCopy.Spec.Contact.FirstName
				user.Spec.LastName = siteCopy.Spec.Contact.LastName
				user.Spec.Password = generateRandomString(10)
				user.Spec.Roles = []string{"PI"}
				t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("site-%s", siteCopy.GetName())).Create(user.DeepCopy())
			}
			defer enableSitePI()

			// Set the HTML template variables
			contentData := mailer.CommonContentData{}
			contentData.CommonData.Site = siteCopy.GetName()
			contentData.CommonData.Username = siteCopy.Spec.Contact.Username
			contentData.CommonData.Name = fmt.Sprintf("%s %s", siteCopy.Spec.Contact.FirstName, siteCopy.Spec.Contact.LastName)
			contentData.CommonData.Email = []string{siteCopy.Spec.Contact.Email}
			mailer.Send("site-registration-successful", contentData)
		}
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("SiteHandler.ObjectUpdated")
	// Create a copy of the site object to make changes on it
	siteCopy := obj.(*apps_v1alpha.Site).DeepCopy()
	// Check whether the site disabled
	if siteCopy.Status.Enabled == false {
		// Delete all RoleBindings, Projects, and Slices in the namespace of site
		t.edgenetClientset.AppsV1alpha().Slices(fmt.Sprintf("site-%s", siteCopy.GetName())).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
		t.edgenetClientset.AppsV1alpha().Projects(fmt.Sprintf("site-%s", siteCopy.GetName())).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
		t.clientset.RbacV1().RoleBindings(fmt.Sprintf("site-%s", siteCopy.GetName())).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
		// List all site users to deactivate and to remove their cluster role binding to get the site
		usersRaw, _ := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("site-%s", siteCopy.GetName())).List(metav1.ListOptions{})
		for _, user := range usersRaw.Items {
			userCopy := user.DeepCopy()
			userCopy.Status.Active = false
			t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(userCopy)
			t.clientset.RbacV1().ClusterRoleBindings().Delete(fmt.Sprintf("%s-%s-for-site", userCopy.GetNamespace(), userCopy.GetName()), &metav1.DeleteOptions{})
		}
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("SiteHandler.ObjectDeleted")
	// Delete or disable nodes added by site, TBD.
}

// checkDuplicateObject checks whether a user exists with the same email address
func (t *Handler) checkDuplicateObject(siteCopy *apps_v1alpha.Site) bool {
	exist := false
	// To check email address
	userRaw, _ := t.edgenetClientset.AppsV1alpha().Users("").List(metav1.ListOptions{})
	for _, userRow := range userRaw.Items {
		if userRow.Spec.Email == siteCopy.Spec.Contact.Email {
			if userRow.GetNamespace() == fmt.Sprintf("site-%s", siteCopy.GetName()) && userRow.GetName() == strings.ToLower(siteCopy.Spec.Contact.Username) {
				continue
			}
			exist = true
			break
		}
	}
	if !exist {
		// Delete the site registration requests which have duplicate values, if any
		SRRRaw, _ := t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().List(metav1.ListOptions{})
		for _, SRRRow := range SRRRaw.Items {
			if SRRRow.GetName() == siteCopy.GetName() || SRRRow.Spec.Contact.Email == siteCopy.Spec.Contact.Email ||
				SRRRow.Spec.Contact.Username == siteCopy.Spec.Contact.Username {
				t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().Delete(SRRRow.GetName(), &metav1.DeleteOptions{})
			}
		}
	}
	// Mail notification, TBD
	return exist
}

// setOwnerReferences returns the site as owner
func (t *Handler) setOwnerReferences(siteCopy *apps_v1alpha.Site) []metav1.OwnerReference {
	// The following section makes site become the namespace owner
	ownerReferences := []metav1.OwnerReference{}
	newSiteRef := *metav1.NewControllerRef(siteCopy, apps_v1alpha.SchemeGroupVersion.WithKind("Site"))
	takeControl := false
	newSiteRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newSiteRef)
	return ownerReferences
}

// setNamespaceOwnerReferences returns the namespace as owner
func (t *Handler) setNamespaceOwnerReferences(namespace *corev1.Namespace) []metav1.OwnerReference {
	// The section below makes namespace who created by the site become the site owner
	newNamespaceRef := *metav1.NewControllerRef(namespace, apps_v1alpha.SchemeGroupVersion.WithKind("Namespace"))
	takeControl := false
	newNamespaceRef.Controller = &takeControl
	namespaceOwnerReferences := []metav1.OwnerReference{newNamespaceRef}
	return namespaceOwnerReferences
}

// generateRandomString to have a unique string
func generateRandomString(n int) string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

	b := make([]rune, n)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}
