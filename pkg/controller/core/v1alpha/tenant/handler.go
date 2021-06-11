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
	"strings"
	"sync"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha/tenantresourcequota"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"
	"github.com/EdgeNet-project/edgenet/pkg/permission"
	"github.com/EdgeNet-project/edgenet/pkg/registration"

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
	ObjectCreatedOrUpdated(obj interface{})
	ObjectDeleted(obj interface{})
}

// Handler implementation
type Handler struct {
	clientset        kubernetes.Interface
	edgenetClientset versioned.Interface
}

// Init handles any handler initialization
func (t *Handler) Init(kubernetes kubernetes.Interface, edgenet versioned.Interface) {
	log.Info("TenantHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet

	permission.Clientset = t.clientset
	registration.Clientset = t.clientset
}

// ObjectCreatedOrUpdated is called when an object is created
func (t *Handler) ObjectCreatedOrUpdated(obj interface{}) {
	log.Info("TenantHandler.ObjectCreatedOrUpdated")
	// Make a copy of the tenant object to make changes on it
	tenant := obj.(*corev1alpha.Tenant).DeepCopy()
	tenantStatus := corev1alpha.TenantStatus{}
	if tenant.Spec.Enabled == true {
		// When a tenant is deleted, the owner references feature allows the namespace to be automatically removed
		var wg sync.WaitGroup
		wg.Add(1)
		ownerReferences := SetAsOwnerReference(tenant)
		tenantStatus, err := t.createCoreNamespace(tenant, tenantStatus, ownerReferences)
		if err == nil || errors.IsAlreadyExists(err) {
			tenantStatus = t.applyQuota(tenant, tenantStatus)
			tenantStatus = t.configurePermissions(tenant, tenantStatus, ownerReferences, &wg)
		}

		if tenant.Status.State == "" && tenantStatus.State != failure {
			// TODO: Only at the first creation
			// Update tenant status
			tenantStatus.State = established
			tenantStatus.Message = []string{statusDict["tenant-ok"]}
			t.sendEmail(tenant, corev1alpha.User{}, "tenant-creation-successful")
		}
		tenant.Status = tenantStatus
		t.edgenetClientset.CoreV1alpha().Tenants().UpdateStatus(context.TODO(), tenant, metav1.UpdateOptions{})
		wg.Done()
	} else {
		// Delete all roles, role bindings, and subsidiary namespaces
		if err := t.clientset.RbacV1().ClusterRoles().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s", tenant.GetName())}); err != nil {
			// TODO: Provide err information at the status
		}
		if err := t.clientset.RbacV1().ClusterRoleBindings().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s", tenant.GetName())}); err != nil {
			// TODO: Provide err information at the status
		}
		if err := t.clientset.RbacV1().RoleBindings("").DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s", tenant.GetName())}); err != nil {
			// TODO: Provide err information at the status
		}
		if err := t.edgenetClientset.CoreV1alpha().SubNamespaces(tenant.GetName()).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s", tenant.GetName())}); err != nil {
			// TODO: Provide err information at the status
		}
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("TenantHandler.ObjectDeleted")
	// Delete or disable nodes added by tenant, TBD.
}

// Create function is for being used by other resources to create a tenant
func (t *Handler) Create(obj interface{}) bool {
	created := false
	switch obj.(type) {
	case *registrationv1alpha.TenantRequest:
		tenantRequest := obj.(*registrationv1alpha.TenantRequest).DeepCopy()
		// Create a tenant on the cluster
		tenant := corev1alpha.Tenant{}
		tenant.SetName(tenantRequest.GetName())
		tenant.Spec.Address = tenantRequest.Spec.Address
		tenant.Spec.Contact = tenantRequest.Spec.Contact
		tenant.Spec.FullName = tenantRequest.Spec.FullName
		tenant.Spec.ShortName = tenantRequest.Spec.ShortName
		tenant.Spec.URL = tenantRequest.Spec.URL
		tenant.Spec.Enabled = true

		user := corev1alpha.User{}
		user.Username = strings.ToLower(tenant.Spec.Contact.Username)
		user.Email = tenant.Spec.Contact.Email
		user.FirstName = tenant.Spec.Contact.FirstName
		user.LastName = tenant.Spec.Contact.LastName
		user.Role = "Owner"
		tenant.Spec.User = append(tenant.Spec.User, user)

		if _, err := t.edgenetClientset.CoreV1alpha().Tenants().Create(context.TODO(), tenant.DeepCopy(), metav1.CreateOptions{}); err == nil {
			created = true
			tenantRequest.Status.State = "Approved"
			tenantRequest.Status.Message = []string{statusDict["request-approved"]}
			t.edgenetClientset.RegistrationV1alpha().TenantRequests().UpdateStatus(context.TODO(), tenantRequest, metav1.UpdateOptions{})
		} else {
			log.Println(err)
		}
	}

	return created
}

func (t *Handler) createCoreNamespace(tenant *corev1alpha.Tenant, tenantStatus corev1alpha.TenantStatus, ownerReferences []metav1.OwnerReference) (corev1alpha.TenantStatus, error) {
	// Core namespace has the same name as the tenant
	tenantCoreNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenant.GetName(), OwnerReferences: ownerReferences}}
	// Namespace labels indicate this namespace created by a tenant, not by a team or slice
	namespaceLabels := map[string]string{"edge-net.io/kind": "core", "edge-net.io/tenant": tenant.GetName()}
	tenantCoreNamespace.SetLabels(namespaceLabels)
	_, err := t.clientset.CoreV1().Namespaces().Create(context.TODO(), tenantCoreNamespace, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		log.Infof("Couldn't create namespace for %s: %s", tenant.GetName(), err)
		tenantStatus.State = failure
		tenantStatus.Message = []string{statusDict["namespace-failure"]}
	}
	return tenantStatus, err
}

func (t *Handler) applyQuota(tenant *corev1alpha.Tenant, tenantStatus corev1alpha.TenantStatus) corev1alpha.TenantStatus {
	trqHandler := tenantresourcequota.Handler{}
	trqHandler.Init(t.clientset, t.edgenetClientset)
	cpuQuota, memoryQuota := trqHandler.Create(tenant.GetName())

	resourceQuota := corev1.ResourceQuota{}
	resourceQuota.Name = "core-quota"
	resourceQuota.Spec = corev1.ResourceQuotaSpec{
		Hard: map[corev1.ResourceName]resource.Quantity{
			"cpu":              resource.MustParse(cpuQuota),
			"memory":           resource.MustParse(memoryQuota),
			"requests.storage": resource.MustParse("8Gi"),
		},
	}
	// Create the resource quota to prevent users from using this namespace for their applications
	if _, err := t.clientset.CoreV1().ResourceQuotas(tenant.GetName()).Create(context.TODO(), resourceQuota.DeepCopy(), metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		log.Infof("Couldn't create resource quota in %s: %s", tenant.GetName(), err)
		// TODO: Provide err information at the status
	}
	return tenantStatus
}

func (t *Handler) configurePermissions(tenant *corev1alpha.Tenant, tenantStatus corev1alpha.TenantStatus, ownerReferences []metav1.OwnerReference, wg *sync.WaitGroup) corev1alpha.TenantStatus {
	// Create the cluster roles
	if err := permission.CreateObjectSpecificClusterRole(tenant.GetName(), "core.edgenet.io", "tenants", tenant.GetName(), "owner", []string{"get", "update", "patch"}, ownerReferences); err != nil && !errors.IsAlreadyExists(err) {
		log.Infof("Couldn't create owner cluster role %s: %s", tenant.GetName(), err)
		// TODO: Provide err information at the status
	}
	if err := permission.CreateObjectSpecificClusterRole(tenant.GetName(), "core.edgenet.io", "tenants", tenant.GetName(), "admin", []string{"get"}, ownerReferences); err != nil && !errors.IsAlreadyExists(err) {
		log.Infof("Couldn't create admin cluster role %s: %s", tenant.GetName(), err)
		// TODO: Provide err information at the status
	}

	if err := t.clientset.RbacV1().ClusterRoleBindings().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/generated=true,edge-net.io/tenant=%s", tenant.GetName())}); err != nil {
		// TODO: Provide err information at the status
	}
	if err := t.clientset.RbacV1().RoleBindings(tenant.GetName()).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/generated=true,edge-net.io/tenant=%s", tenant.GetName())}); err != nil {
		// TODO: Provide err information at the status
	}

	var acceptableUsePolicyAccess = func(user corev1alpha.User, acceptableUsePolicy string) {
		if err := permission.CreateObjectSpecificClusterRole(tenant.GetName(), "core.edgenet.io", "acceptableusepolicies", acceptableUsePolicy, "owner", []string{"get", "update", "patch"}, ownerReferences); err != nil && !errors.IsAlreadyExists(err) {
			log.Infof("Couldn't create aup cluster role %s, %s: %s", tenant.GetName(), acceptableUsePolicy, err)
			// TODO: Provide err information at the status
		}
		clusterRoleName := fmt.Sprintf("edgenet:%s:acceptableusepolicies:%s-%s", tenant.GetName(), acceptableUsePolicy, "owner")
		if err := permission.CreateObjectSpecificClusterRoleBinding(tenant.GetName(), clusterRoleName, user, ownerReferences); err != nil {
			t.sendEmail(tenant, corev1alpha.User{}, "user-creation-failure")
			// TODO: Define the error precisely
			tenantStatus.State = failure
			tenantStatus.Message = append(tenant.Status.Message, []string{statusDict["user-failed"], err.Error()}...)
		}
	}

	for _, userRow := range tenant.Spec.User {
		policyStatus := false
		acceptableUsePolicy, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), userRow.GetName(), metav1.GetOptions{})
		if err == nil {
			policyStatus = acceptableUsePolicy.Spec.Accepted
			acceptableUsePolicyAccess(userRow, acceptableUsePolicy.GetName())
		} else if errors.IsNotFound(err) {
			// Create the client certs for permanent use
			userRow.Tenant = tenant.GetName()
			crt, key, err := registration.MakeUser(tenant.GetName(), userRow.GetName(), userRow.Email)
			if err != nil {
				tenantStatus.State = failure
				tenantStatus.Message = append(tenant.Status.Message, fmt.Sprintf(statusDict["cert-fail"], userRow.GetName()))
				t.sendEmail(tenant, userRow, "user-cert-failure")
			}
			err = registration.MakeConfig(tenant.GetName(), userRow.GetName(), userRow.Email, crt, key)
			if err != nil {
				tenantStatus.State = failure
				tenantStatus.Message = append(tenant.Status.Message, fmt.Sprintf(statusDict["kubeconfig-fail"], userRow.GetName()))
				t.sendEmail(tenant, userRow, "user-kubeconfig-failure")
			}

			if tenantStatus.State != failure {
				go func() {
					wg.Wait()
					// Generate an acceptable use policy object attached to user
					aupLabels := map[string]string{"edge-net.io/generated": "true", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/user": userRow.GetName()}
					userAcceptableUsePolicy := &corev1alpha.AcceptableUsePolicy{TypeMeta: metav1.TypeMeta{Kind: "AcceptableUsePolicy", APIVersion: "apps.edgenet.io/v1alpha"},
						ObjectMeta: metav1.ObjectMeta{Name: userRow.GetName(), OwnerReferences: ownerReferences}, Spec: corev1alpha.AcceptableUsePolicySpec{Accepted: false}}
					userAcceptableUsePolicy.SetLabels(aupLabels)
					if _, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Create(context.TODO(), userAcceptableUsePolicy, metav1.CreateOptions{}); err != nil {
						// TODO: Define the error precisely
					}
					acceptableUsePolicyAccess(userRow, userAcceptableUsePolicy.GetName())
					t.sendEmail(tenant, userRow, "user-registration-successful")
				}()
			}
		}

		if policyStatus {
			// Prepare role bindings
			// Create the owner role binding
			if err := permission.CreateObjectSpecificRoleBinding(tenant.GetName(), tenant.GetName(), fmt.Sprintf("edgenet:tenant-%s", strings.ToLower(userRow.Role)), userRow); err != nil {
				t.sendEmail(tenant, corev1alpha.User{}, "user-creation-failure")
				// TODO: Define the error precisely
				tenantStatus.State = failure
				tenantStatus.Message = append(tenant.Status.Message, []string{statusDict["user-failed"], err.Error()}...)
			}

			if strings.ToLower(userRow.Role) != "collaborator" {
				// Create the cluster role binding related to the tenant object
				clusterRoleName := fmt.Sprintf("edgenet:%s:tenants:%s-%s", tenant.GetName(), tenant.GetName(), strings.ToLower(userRow.Role))
				if err := permission.CreateObjectSpecificClusterRoleBinding(tenant.GetName(), clusterRoleName, userRow, ownerReferences); err != nil {
					t.sendEmail(tenant, corev1alpha.User{}, "user-creation-failure")
					// TODO: Define the error precisely
					tenantStatus.State = failure
					tenantStatus.Message = append(tenant.Status.Message, []string{statusDict["user-failed"], err.Error()}...)
				}
			}
		}
	}

	return tenantStatus
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(tenant *corev1alpha.Tenant, user corev1alpha.User, subject string) {
	// Set the HTML template variables
	contentData := mailer.CommonContentData{}
	if (user != corev1alpha.User{}) {
		contentData.CommonData.Tenant = user.GetTenant()
		contentData.CommonData.Username = user.GetName()
		contentData.CommonData.Name = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		contentData.CommonData.Email = []string{user.Email}
		mailer.Send(subject, contentData)
	} else {
		contentData.CommonData.Tenant = tenant.GetName()
		for _, user := range tenant.Spec.User {
			if user.Role == "Owner" || user.Role == "Admin" {
				contentData.CommonData.Username = user.Username
				contentData.CommonData.Name = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
				contentData.CommonData.Email = []string{user.Email}
				mailer.Send(subject, contentData)
			}
		}
	}
}

// SetAsOwnerReference returns the tenant as owner
func SetAsOwnerReference(tenant *corev1alpha.Tenant) []metav1.OwnerReference {
	// The following section makes tenant become the owner
	ownerReferences := []metav1.OwnerReference{}
	newTenantRef := *metav1.NewControllerRef(tenant, corev1alpha.SchemeGroupVersion.WithKind("Tenant"))
	takeControl := false
	newTenantRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newTenantRef)
	return ownerReferences
}
