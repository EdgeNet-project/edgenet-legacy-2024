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
	"context"
	"fmt"
	"reflect"
	"strings"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/controller/v1alpha/totalresourcequota"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"
	ns "github.com/EdgeNet-project/edgenet/pkg/namespace"
	"github.com/EdgeNet-project/edgenet/pkg/permission"

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
	ObjectUpdated(obj interface{})
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
	log.Info("AuthorityHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
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
	permission.Clientset = t.clientset
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("AuthorityHandler.ObjectCreated")
	// Create a copy of the authority object to make changes on it
	authorityCopy := obj.(*apps_v1alpha.Authority).DeepCopy()
	// Check if the email address is already taken
	exists, message := t.checkDuplicateObject(authorityCopy)
	if exists {
		authorityCopy.Status.State = failure
		authorityCopy.Status.Message = []string{message}
		authorityCopy.Spec.Enabled = false
		t.edgenetClientset.AppsV1alpha().Authorities().UpdateStatus(context.TODO(), authorityCopy, metav1.UpdateOptions{})
		return
	}
	authorityCopy = t.authorityPreparation(authorityCopy)
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("AuthorityHandler.ObjectUpdated")
	// Create a copy of the authority object to make changes on it
	authorityCopy := obj.(*apps_v1alpha.Authority).DeepCopy()
	// Check if the email address is already taken
	exists, message := t.checkDuplicateObject(authorityCopy)
	if exists {
		authorityCopy.Status.State = failure
		authorityCopy.Status.Message = []string{message}
		authorityCopy.Spec.Enabled = false
		authorityCopyUpdated, err := t.edgenetClientset.AppsV1alpha().Authorities().UpdateStatus(context.TODO(), authorityCopy, metav1.UpdateOptions{})
		if err == nil {
			authorityCopy = authorityCopyUpdated
		}
	} else if !authorityCopy.Spec.Enabled && authorityCopy.Status.State == failure {
		authorityCopy = t.authorityPreparation(authorityCopy)
	}
	// Check whether the authority disabled
	if authorityCopy.Spec.Enabled == false {
		// Delete all RoleBindings, Teams, and Slices in the namespace of authority
		t.edgenetClientset.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", authorityCopy.GetName())).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{})
		t.edgenetClientset.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", authorityCopy.GetName())).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{})
		t.clientset.RbacV1().RoleBindings(fmt.Sprintf("authority-%s", authorityCopy.GetName())).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{})
		// List all authority users to deactivate and to remove their cluster role binding to get the authority
		usersRaw, _ := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("authority-%s", authorityCopy.GetName())).List(context.TODO(), metav1.ListOptions{})
		for _, user := range usersRaw.Items {
			userCopy := user.DeepCopy()
			userCopy.Spec.Active = false
			t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).Update(context.TODO(), userCopy, metav1.UpdateOptions{})
			t.clientset.RbacV1().ClusterRoleBindings().Delete(context.TODO(), fmt.Sprintf("%s-%s-for-authority", userCopy.GetNamespace(), userCopy.GetName()), metav1.DeleteOptions{})
		}
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("AuthorityHandler.ObjectDeleted")
	// Delete or disable nodes added by authority, TBD.
}

// Create function is for being used by other resources to create an authority
func (t *Handler) Create(obj interface{}) bool {
	failed := true
	switch obj.(type) {
	case *apps_v1alpha.AuthorityRequest:
		authorityRequestCopy := obj.(*apps_v1alpha.AuthorityRequest).DeepCopy()
		// Create a authority on the cluster
		authority := apps_v1alpha.Authority{}
		authority.SetName(authorityRequestCopy.GetName())
		authority.Spec.Address = authorityRequestCopy.Spec.Address
		authority.Spec.Contact = authorityRequestCopy.Spec.Contact
		authority.Spec.FullName = authorityRequestCopy.Spec.FullName
		authority.Spec.ShortName = authorityRequestCopy.Spec.ShortName
		authority.Spec.URL = authorityRequestCopy.Spec.URL
		authority.Spec.Enabled = true
		_, err := t.edgenetClientset.AppsV1alpha().Authorities().Create(context.TODO(), authority.DeepCopy(), metav1.CreateOptions{})
		if err == nil {
			failed = false
			t.edgenetClientset.AppsV1alpha().AuthorityRequests().Delete(context.TODO(), authorityRequestCopy.GetName(), metav1.DeleteOptions{})
		}
	}

	return failed
}

// authorityPreparation basically generates a namespace and creates authority-admin
func (t *Handler) authorityPreparation(authorityCopy *apps_v1alpha.Authority) *apps_v1alpha.Authority {
	// If the service restarts, it creates all objects again
	// Because of that, this section covers a variety of possibilities
	_, err := t.clientset.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("authority-%s", authorityCopy.GetName()), metav1.GetOptions{})
	if err != nil {
		permission.CreateClusterRoles(authorityCopy)
		// Automatically create a namespace to host users, slices, and teams
		// When a authority is deleted, the owner references feature allows the namespace to be automatically removed
		ownerReferences := SetAsOwnerReference(authorityCopy)
		// Every namespace of a authority has the prefix as "authority" to provide singularity
		authorityChildNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("authority-%s", authorityCopy.GetName()), OwnerReferences: ownerReferences}}
		// Namespace labels indicate this namespace created by a authority, not by a team or slice
		namespaceLabels := map[string]string{"owner": "authority", "owner-name": authorityCopy.GetName(), "authority-name": authorityCopy.GetName()}
		authorityChildNamespace.SetLabels(namespaceLabels)
		authorityChildNamespaceCreated, err := t.clientset.CoreV1().Namespaces().Create(context.TODO(), authorityChildNamespace, metav1.CreateOptions{})
		if err != nil {
			log.Infof("Couldn't create namespace for %s: %s", authorityCopy.GetName(), err)
			authorityCopy.Status.State = failure
			authorityCopy.Status.Message = []string{statusDict["namespace-failure"]}
		}
		// Create the resource quota to ban users from using this namespace for their applications
		_, err = t.clientset.CoreV1().ResourceQuotas(authorityChildNamespaceCreated.GetName()).Create(context.TODO(), t.resourceQuota, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			log.Infof("Couldn't create resource quota in %s: %s", authorityCopy.GetName(), err)
		}
		childNamespaceOwnerReferences := ns.SetAsOwnerReference(authorityChildNamespaceCreated)
		authorityCopy.ObjectMeta.OwnerReferences = childNamespaceOwnerReferences
		authorityCopyUpdated, err := t.edgenetClientset.AppsV1alpha().Authorities().Update(context.TODO(), authorityCopy, metav1.UpdateOptions{})
		if err == nil {
			// To manipulate the object later
			authorityCopy = authorityCopyUpdated
		}
		TRQHandler := totalresourcequota.Handler{}
		TRQHandler.Init(t.clientset, t.edgenetClientset)
		TRQHandler.Create(authorityCopy.GetName())
		enableAuthorityAdmin := func() {
			t.edgenetClientset.AppsV1alpha().Authorities().UpdateStatus(context.TODO(), authorityCopy, metav1.UpdateOptions{})
			// Create a user as admin on authority
			user := apps_v1alpha.User{}
			user.SetName(strings.ToLower(authorityCopy.Spec.Contact.Username))
			user.Spec.Email = authorityCopy.Spec.Contact.Email
			user.Spec.FirstName = authorityCopy.Spec.Contact.FirstName
			user.Spec.LastName = authorityCopy.Spec.Contact.LastName
			user.Spec.Active = true
			_, err = t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("authority-%s", authorityCopy.GetName())).Create(context.TODO(), user.DeepCopy(), metav1.CreateOptions{})
			if err != nil {
				t.sendEmail(authorityCopy, "user-creation-failure")
				authorityCopy.Status.State = failure
				authorityCopy.Status.Message = append(authorityCopy.Status.Message, []string{statusDict["user-failed"], err.Error()}...)
			}
		}
		defer enableAuthorityAdmin()
		if authorityCopy.Status.State != failure {
			// Update authority status
			authorityCopy.Status.State = established
			authorityCopy.Status.Message = []string{statusDict["authority-ok"]}
			t.sendEmail(authorityCopy, "authority-creation-successful")
		}
	} else if err == nil {
		permission.CreateClusterRoles(authorityCopy)
		TRQHandler := totalresourcequota.Handler{}
		TRQHandler.Init(t.clientset, t.edgenetClientset)
		TRQHandler.Create(authorityCopy.GetName())
	}
	return authorityCopy
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(authorityCopy *apps_v1alpha.Authority, subject string) {
	// Set the HTML template variables
	contentData := mailer.CommonContentData{}
	contentData.CommonData.Authority = authorityCopy.GetName()
	contentData.CommonData.Username = authorityCopy.Spec.Contact.Username
	contentData.CommonData.Name = fmt.Sprintf("%s %s", authorityCopy.Spec.Contact.FirstName, authorityCopy.Spec.Contact.LastName)
	contentData.CommonData.Email = []string{authorityCopy.Spec.Contact.Email}
	mailer.Send(subject, contentData)
}

// checkDuplicateObject checks whether a user exists with the same email address
func (t *Handler) checkDuplicateObject(authorityCopy *apps_v1alpha.Authority) (bool, string) {
	exists := false
	var message string
	// To check email address
	userRaw, _ := t.edgenetClientset.AppsV1alpha().Users("").List(context.TODO(), metav1.ListOptions{})
	for _, userRow := range userRaw.Items {
		if userRow.Spec.Email == authorityCopy.Spec.Contact.Email {
			if userRow.GetNamespace() == fmt.Sprintf("authority-%s", authorityCopy.GetName()) && userRow.GetName() == strings.ToLower(authorityCopy.Spec.Contact.Username) {
				continue
			}
			exists = true
			message = fmt.Sprintf(statusDict["email-exist"], authorityCopy.Spec.Contact.Email)
			break
		}
	}
	if !exists {
		// Update the authority requests that have duplicate values, if any
		authorityRequestRaw, _ := t.edgenetClientset.AppsV1alpha().AuthorityRequests().List(context.TODO(), metav1.ListOptions{})
		for _, authorityRequestRow := range authorityRequestRaw.Items {
			if authorityRequestRow.Status.State == success {
				if authorityRequestRow.GetName() == authorityCopy.GetName() || authorityRequestRow.Spec.Contact.Email == authorityCopy.Spec.Contact.Email {
					t.edgenetClientset.AppsV1alpha().AuthorityRequests().Delete(context.TODO(), authorityRequestRow.GetName(), metav1.DeleteOptions{})
				}
			}
		}
	} else if exists && !reflect.DeepEqual(authorityCopy.Status.Message, message) {
		t.sendEmail(authorityCopy, "authority-validation-failure-email")
	}
	return exists, message
}

// SetAsOwnerReference returns the authority as owner
func SetAsOwnerReference(authorityCopy *apps_v1alpha.Authority) []metav1.OwnerReference {
	// The following section makes authority become the owner
	ownerReferences := []metav1.OwnerReference{}
	newAuthorityRef := *metav1.NewControllerRef(authorityCopy, apps_v1alpha.SchemeGroupVersion.WithKind("Authority"))
	takeControl := false
	newAuthorityRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newAuthorityRef)
	return ownerReferences
}
