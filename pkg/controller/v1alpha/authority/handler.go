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

package authority

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
	log.Info("AuthorityHandler.Init")
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
	t.resourceQuota.Name = "authority-quota"
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
	log.Info("AuthorityHandler.ObjectCreated")
	// Create a copy of the authority object to make changes on it
	authorityCopy := obj.(*apps_v1alpha.Authority).DeepCopy()
	// Check if the email address of user is already taken
	exist := t.checkDuplicateObject(authorityCopy)
	if exist {
		// If it is already taken, remove the authority object
		t.edgenetClientset.AppsV1alpha().Authorities().Delete(authorityCopy.GetName(), &metav1.DeleteOptions{})
		return
	}
	if authorityCopy.GetGeneration() == 1 && !authorityCopy.Status.Enabled {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		_, err := t.clientset.CoreV1().Namespaces().Get(fmt.Sprintf("authority-%s", authorityCopy.GetName()), metav1.GetOptions{})
		if err != nil {
			// Create a cluster role to be used by authority users
			policyRule := []rbacv1.PolicyRule{{APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"authorities"}, ResourceNames: []string{authorityCopy.GetName()}, Verbs: []string{"get"}}}
			authorityRole := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("authority-%s", authorityCopy.GetName())}, Rules: policyRule}
			_, err := t.clientset.RbacV1().ClusterRoles().Create(authorityRole)
			if err != nil {
				log.Infof("Couldn't create authority-%s role: %s", authorityCopy.GetName(), err)
			}
			// Automatically create a namespace to host users, slices, and teams
			// When a authority is deleted, the owner references feature allows the namespace to be automatically removed
			authorityOwnerReferences := t.setOwnerReferences(authorityCopy)
			// Every namespace of a authority has the prefix as "authority" to provide singularity
			authorityChildNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("authority-%s", authorityCopy.GetName()), OwnerReferences: authorityOwnerReferences}}
			// Namespace labels indicate this namespace created by a authority, not by a team or slice
			namespaceLabels := map[string]string{"owner": "authority", "owner-name": authorityCopy.GetName(), "authority-name": authorityCopy.GetName()}
			authorityChildNamespace.SetLabels(namespaceLabels)
			authorityChildNamespaceCreated, _ := t.clientset.CoreV1().Namespaces().Create(authorityChildNamespace)
			// Create the resource quota to ban users from using this namespace for their applications
			t.clientset.CoreV1().ResourceQuotas(authorityChildNamespaceCreated.GetName()).Create(t.resourceQuota)

			childNamespaceOwnerReferences := t.setNamespaceOwnerReferences(authorityChildNamespaceCreated)
			authorityCopy.ObjectMeta.OwnerReferences = childNamespaceOwnerReferences
			authorityCopyUpdated, err := t.edgenetClientset.AppsV1alpha().Authorities().Update(authorityCopy)
			if err == nil {
				// To manipulate the object later
				authorityCopy = authorityCopyUpdated
			}
			// Automatically enable authority and update authority status
			authorityCopy.Status.Enabled = true
			enableAuthorityPI := func() {
				t.edgenetClientset.AppsV1alpha().Authorities().UpdateStatus(authorityCopy)
				// Create a user as PI on authority
				user := apps_v1alpha.User{}
				user.SetName(strings.ToLower(authorityCopy.Spec.Contact.Username))
				user.Spec.Email = authorityCopy.Spec.Contact.Email
				user.Spec.FirstName = authorityCopy.Spec.Contact.FirstName
				user.Spec.LastName = authorityCopy.Spec.Contact.LastName
				user.Spec.Password = generateRandomString(10)
				user.Spec.Roles = []string{"PI"}
				t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("authority-%s", authorityCopy.GetName())).Create(user.DeepCopy())
			}
			defer enableAuthorityPI()

			// Set the HTML template variables
			contentData := mailer.CommonContentData{}
			contentData.CommonData.Authority = authorityCopy.GetName()
			contentData.CommonData.Username = authorityCopy.Spec.Contact.Username
			contentData.CommonData.Name = fmt.Sprintf("%s %s", authorityCopy.Spec.Contact.FirstName, authorityCopy.Spec.Contact.LastName)
			contentData.CommonData.Email = []string{authorityCopy.Spec.Contact.Email}
			mailer.Send("authority-creation-successful", contentData)
		}
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("AuthorityHandler.ObjectUpdated")
	// Create a copy of the authority object to make changes on it
	authorityCopy := obj.(*apps_v1alpha.Authority).DeepCopy()
	// Check whether the authority disabled
	if authorityCopy.Status.Enabled == false {
		// Delete all RoleBindings, Teams, and Slices in the namespace of authority
		t.edgenetClientset.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", authorityCopy.GetName())).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
		t.edgenetClientset.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", authorityCopy.GetName())).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
		t.clientset.RbacV1().RoleBindings(fmt.Sprintf("authority-%s", authorityCopy.GetName())).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
		// List all authority users to deactivate and to remove their cluster role binding to get the authority
		usersRaw, _ := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("authority-%s", authorityCopy.GetName())).List(metav1.ListOptions{})
		for _, user := range usersRaw.Items {
			userCopy := user.DeepCopy()
			userCopy.Status.Active = false
			t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(userCopy)
			t.clientset.RbacV1().ClusterRoleBindings().Delete(fmt.Sprintf("%s-%s-for-authority", userCopy.GetNamespace(), userCopy.GetName()), &metav1.DeleteOptions{})
		}
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("AuthorityHandler.ObjectDeleted")
	// Delete or disable nodes added by authority, TBD.
}

// checkDuplicateObject checks whether a user exists with the same email address
func (t *Handler) checkDuplicateObject(authorityCopy *apps_v1alpha.Authority) bool {
	exist := false
	// To check email address
	userRaw, _ := t.edgenetClientset.AppsV1alpha().Users("").List(metav1.ListOptions{})
	for _, userRow := range userRaw.Items {
		if userRow.Spec.Email == authorityCopy.Spec.Contact.Email {
			if userRow.GetNamespace() == fmt.Sprintf("authority-%s", authorityCopy.GetName()) && userRow.GetName() == strings.ToLower(authorityCopy.Spec.Contact.Username) {
				continue
			}
			exist = true
			break
		}
	}
	if !exist {
		// Delete the authority requests which have duplicate values, if any
		authorityRequestRaw, _ := t.edgenetClientset.AppsV1alpha().AuthorityRequests().List(metav1.ListOptions{})
		for _, authorityRequestRow := range authorityRequestRaw.Items {
			if authorityRequestRow.GetName() == authorityCopy.GetName() || authorityRequestRow.Spec.Contact.Email == authorityCopy.Spec.Contact.Email ||
				authorityRequestRow.Spec.Contact.Username == authorityCopy.Spec.Contact.Username {
				t.edgenetClientset.AppsV1alpha().AuthorityRequests().Delete(authorityRequestRow.GetName(), &metav1.DeleteOptions{})
			}
		}
	}
	// Mail notification, TBD
	return exist
}

// setOwnerReferences returns the authority as owner
func (t *Handler) setOwnerReferences(authorityCopy *apps_v1alpha.Authority) []metav1.OwnerReference {
	// The following section makes authority become the namespace owner
	ownerReferences := []metav1.OwnerReference{}
	newAuthorityRef := *metav1.NewControllerRef(authorityCopy, apps_v1alpha.SchemeGroupVersion.WithKind("Authority"))
	takeControl := false
	newAuthorityRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newAuthorityRef)
	return ownerReferences
}

// setNamespaceOwnerReferences returns the namespace as owner
func (t *Handler) setNamespaceOwnerReferences(namespace *corev1.Namespace) []metav1.OwnerReference {
	// The section below makes namespace who created by the authority become the authority owner
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
