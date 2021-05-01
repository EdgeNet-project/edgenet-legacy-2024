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

package tenant

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha/tenantresourcequota"
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
	log.Info("TenantHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
	t.resourceQuota = &corev1.ResourceQuota{}
	t.resourceQuota.Name = "tenant-quota"
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
	log.Info("TenantHandler.ObjectCreated")
	// Create a copy of the tenant object to make changes on it
	tenantCopy := obj.(*corev1alpha.Tenant).DeepCopy()
	// Check if the email address is already taken
	exists, message := t.checkDuplicateObject(tenantCopy)
	if exists {
		tenantCopy.Status.State = failure
		tenantCopy.Status.Message = []string{message}
		tenantCopy.Spec.Enabled = false
		t.edgenetClientset.CoreV1alpha().Tenants().UpdateStatus(context.TODO(), tenantCopy, metav1.UpdateOptions{})
		return
	}
	tenantCopy = t.tenantPreparation(tenantCopy)
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("TenantHandler.ObjectUpdated")
	// Create a copy of the tenant object to make changes on it
	tenantCopy := obj.(*corev1alpha.Tenant).DeepCopy()
	// Check if the email address is already taken
	exists, message := t.checkDuplicateObject(tenantCopy)
	if exists {
		tenantCopy.Status.State = failure
		tenantCopy.Status.Message = []string{message}
		tenantCopy.Spec.Enabled = false
		tenantCopyUpdated, err := t.edgenetClientset.CoreV1alpha().Tenants().UpdateStatus(context.TODO(), tenantCopy, metav1.UpdateOptions{})
		if err == nil {
			tenantCopy = tenantCopyUpdated
		}
	} else if !tenantCopy.Spec.Enabled && tenantCopy.Status.State == failure {
		tenantCopy = t.tenantPreparation(tenantCopy)
	}
	// Check whether the tenant disabled
	if tenantCopy.Spec.Enabled == false {
		// Delete all RoleBindings, Teams, and Slices in the namespace of tenant
		t.edgenetClientset.AppsV1alpha().Slices(fmt.Sprintf("tenant-%s", tenantCopy.GetName())).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{})
		t.edgenetClientset.CoreV1alpha().Teams(fmt.Sprintf("tenant-%s", tenantCopy.GetName())).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{})
		t.clientset.RbacV1().RoleBindings(fmt.Sprintf("tenant-%s", tenantCopy.GetName())).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{})
		// List all tenant users to deactivate and to remove their cluster role binding to get the tenant
		usersRaw, _ := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("tenant-%s", tenantCopy.GetName())).List(context.TODO(), metav1.ListOptions{})
		for _, user := range usersRaw.Items {
			userCopy := user.DeepCopy()
			userCopy.Spec.Active = false
			t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).Update(context.TODO(), userCopy, metav1.UpdateOptions{})
			t.clientset.RbacV1().ClusterRoleBindings().Delete(context.TODO(), fmt.Sprintf("%s-%s-for-tenant", userCopy.GetNamespace(), userCopy.GetName()), metav1.DeleteOptions{})
		}
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("TenantHandler.ObjectDeleted")
	// Delete or disable nodes added by tenant, TBD.
}

// Create function is for being used by other resources to create an tenant
func (t *Handler) Create(obj interface{}) bool {
	failed := true
	switch obj.(type) {
	case *corev1alpha.TenantRequest:
		tenantRequestCopy := obj.(*corev1alpha.TenantRequest).DeepCopy()
		// Create a tenant on the cluster
		tenant := corev1alpha.Tenant{}
		tenant.SetName(tenantRequestCopy.GetName())
		tenant.Spec.Address = tenantRequestCopy.Spec.Address
		tenant.Spec.Contact = tenantRequestCopy.Spec.Contact
		tenant.Spec.FullName = tenantRequestCopy.Spec.FullName
		tenant.Spec.ShortName = tenantRequestCopy.Spec.ShortName
		tenant.Spec.URL = tenantRequestCopy.Spec.URL
		tenant.Spec.Enabled = true
		_, err := t.edgenetClientset.CoreV1alpha().Tenants().Create(context.TODO(), tenant.DeepCopy(), metav1.CreateOptions{})
		if err == nil {
			failed = false
			t.edgenetClientset.CoreV1alpha().TenantRequests().Delete(context.TODO(), tenantRequestCopy.GetName(), metav1.DeleteOptions{})
		}
	}

	return failed
}

// tenantPreparation basically generates a namespace and creates tenant-admin
func (t *Handler) tenantPreparation(tenantCopy *corev1alpha.Tenant) *corev1alpha.Tenant {
	// If the service restarts, it creates all objects again
	// Because of that, this section covers a variety of possibilities
	_, err := t.clientset.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("tenant-%s", tenantCopy.GetName()), metav1.GetOptions{})
	if err != nil {
		permission.CreateClusterRoles(tenantCopy)
		// Automatically create a namespace to host users, slices, and teams
		// When a tenant is deleted, the owner references feature allows the namespace to be automatically removed
		ownerReferences := SetAsOwnerReference(tenantCopy)
		// Every namespace of a tenant has the prefix as "tenant" to provide singularity
		tenantChildNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("tenant-%s", tenantCopy.GetName()), OwnerReferences: ownerReferences}}
		// Namespace labels indicate this namespace created by a tenant, not by a team or slice
		namespaceLabels := map[string]string{"owner": "tenant", "owner-name": tenantCopy.GetName(), "tenant-name": tenantCopy.GetName()}
		tenantChildNamespace.SetLabels(namespaceLabels)
		tenantChildNamespaceCreated, err := t.clientset.CoreV1().Namespaces().Create(context.TODO(), tenantChildNamespace, metav1.CreateOptions{})
		if err != nil {
			log.Infof("Couldn't create namespace for %s: %s", tenantCopy.GetName(), err)
			tenantCopy.Status.State = failure
			tenantCopy.Status.Message = []string{statusDict["namespace-failure"]}
		}
		// Create the resource quota to ban users from using this namespace for their applications
		_, err = t.clientset.CoreV1().ResourceQuotas(tenantChildNamespaceCreated.GetName()).Create(context.TODO(), t.resourceQuota, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			log.Infof("Couldn't create resource quota in %s: %s", tenantCopy.GetName(), err)
		}
		childNamespaceOwnerReferences := ns.SetAsOwnerReference(tenantChildNamespaceCreated)
		tenantCopy.ObjectMeta.OwnerReferences = childNamespaceOwnerReferences
		tenantCopyUpdated, err := t.edgenetClientset.CoreV1alpha().Tenants().Update(context.TODO(), tenantCopy, metav1.UpdateOptions{})
		if err == nil {
			// To manipulate the object later
			tenantCopy = tenantCopyUpdated
		}
		TRQHandler := tenantresourcequota.Handler{}
		TRQHandler.Init(t.clientset, t.edgenetClientset)
		TRQHandler.Create(tenantCopy.GetName())
		enableTenantAdmin := func() {
			t.edgenetClientset.CoreV1alpha().Tenants().UpdateStatus(context.TODO(), tenantCopy, metav1.UpdateOptions{})
			// Create a user as admin on tenant
			user := corev1alpha.User{}
			user.SetName(strings.ToLower(tenantCopy.Spec.Contact.Username))
			user.Spec.Email = tenantCopy.Spec.Contact.Email
			user.Spec.FirstName = tenantCopy.Spec.Contact.FirstName
			user.Spec.LastName = tenantCopy.Spec.Contact.LastName
			user.Spec.Active = true
			_, err = t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("tenant-%s", tenantCopy.GetName())).Create(context.TODO(), user.DeepCopy(), metav1.CreateOptions{})
			if err != nil {
				t.sendEmail(tenantCopy, "user-creation-failure")
				tenantCopy.Status.State = failure
				tenantCopy.Status.Message = append(tenantCopy.Status.Message, []string{statusDict["user-failed"], err.Error()}...)
			}
		}
		defer enableTenantAdmin()
		if tenantCopy.Status.State != failure {
			// Update tenant status
			tenantCopy.Status.State = established
			tenantCopy.Status.Message = []string{statusDict["tenant-ok"]}
			t.sendEmail(tenantCopy, "tenant-creation-successful")
		}
	} else if err == nil {
		permission.CreateClusterRoles(tenantCopy)
		TRQHandler := tenantresourcequota.Handler{}
		TRQHandler.Init(t.clientset, t.edgenetClientset)
		TRQHandler.Create(tenantCopy.GetName())
	}
	return tenantCopy
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(tenantCopy *corev1alpha.Tenant, subject string) {
	// Set the HTML template variables
	contentData := mailer.CommonContentData{}
	contentData.CommonData.Tenant = tenantCopy.GetName()
	contentData.CommonData.Username = tenantCopy.Spec.Contact.Username
	contentData.CommonData.Name = fmt.Sprintf("%s %s", tenantCopy.Spec.Contact.FirstName, tenantCopy.Spec.Contact.LastName)
	contentData.CommonData.Email = []string{tenantCopy.Spec.Contact.Email}
	mailer.Send(subject, contentData)
}

// checkDuplicateObject checks whether a user exists with the same email address
func (t *Handler) checkDuplicateObject(tenantCopy *corev1alpha.Tenant) (bool, string) {
	exists := false
	var message string
	// To check email address
	userRaw, _ := t.edgenetClientset.AppsV1alpha().Users("").List(context.TODO(), metav1.ListOptions{})
	for _, userRow := range userRaw.Items {
		if userRow.Spec.Email == tenantCopy.Spec.Contact.Email {
			if userRow.GetNamespace() == fmt.Sprintf("tenant-%s", tenantCopy.GetName()) && userRow.GetName() == strings.ToLower(tenantCopy.Spec.Contact.Username) {
				continue
			}
			exists = true
			message = fmt.Sprintf(statusDict["email-exist"], tenantCopy.Spec.Contact.Email)
			break
		}
	}
	if !exists {
		// Update the tenant requests that have duplicate values, if any
		tenantRequestRaw, _ := t.edgenetClientset.CoreV1alpha().TenantRequests().List(context.TODO(), metav1.ListOptions{})
		for _, tenantRequestRow := range tenantRequestRaw.Items {
			if tenantRequestRow.Status.State == success {
				if tenantRequestRow.GetName() == tenantCopy.GetName() || tenantRequestRow.Spec.Contact.Email == tenantCopy.Spec.Contact.Email {
					t.edgenetClientset.CoreV1alpha().TenantRequests().Delete(context.TODO(), tenantRequestRow.GetName(), metav1.DeleteOptions{})
				}
			}
		}
	} else if exists && !reflect.DeepEqual(tenantCopy.Status.Message, message) {
		t.sendEmail(tenantCopy, "tenant-validation-failure-email")
	}
	return exists, message
}

// SetAsOwnerReference returns the tenant as owner
func SetAsOwnerReference(tenantCopy *corev1alpha.Tenant) []metav1.OwnerReference {
	// The following section makes tenant become the owner
	ownerReferences := []metav1.OwnerReference{}
	newTenantRef := *metav1.NewControllerRef(tenantCopy, corev1alpha.SchemeGroupVersion.WithKind("Tenant"))
	takeControl := false
	newTenantRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newTenantRef)
	return ownerReferences
}
