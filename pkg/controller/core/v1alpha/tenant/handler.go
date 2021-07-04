/*
Copyright 2021 Contributors to the EdgeNet project.

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
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha/tenantresourcequota"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"
	"github.com/EdgeNet-project/edgenet/pkg/permission"
	"github.com/EdgeNet-project/edgenet/pkg/registration"
	"github.com/EdgeNet-project/edgenet/pkg/util"

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
	if tenant.Spec.Enabled == true {
		defer func() {
			if !reflect.DeepEqual(obj.(*corev1alpha.Tenant).Status, tenant.Status) {
				if _, err := t.edgenetClientset.CoreV1alpha().Tenants().UpdateStatus(context.TODO(), tenant, metav1.UpdateOptions{}); err != nil {
					// TODO: Provide more information on error
					log.Println(err)
				}
			}
		}()
		// When a tenant is deleted, the owner references feature allows the namespace to be automatically removed
		ownerReferences := SetAsOwnerReference(tenant)
		err := t.createCoreNamespace(tenant, ownerReferences)
		if err == nil || errors.IsAlreadyExists(err) {
			t.applyQuota(tenant)
			// Create the cluster roles
			if err := permission.CreateObjectSpecificClusterRole(tenant.GetName(), "core.edgenet.io", "tenants", tenant.GetName(), "owner", []string{"get", "update", "patch"}, ownerReferences); err != nil && !errors.IsAlreadyExists(err) {
				log.Infof("Couldn't create owner cluster role %s: %s", tenant.GetName(), err)
				// TODO: Provide err information at the status
			}
			if err := permission.CreateObjectSpecificClusterRole(tenant.GetName(), "core.edgenet.io", "tenants", tenant.GetName(), "admin", []string{"get"}, ownerReferences); err != nil && !errors.IsAlreadyExists(err) {
				log.Infof("Couldn't create admin cluster role %s: %s", tenant.GetName(), err)
				// TODO: Provide err information at the status
			}
		}

		exists, _ := util.Contains(tenant.Status.Message, statusDict["tenant-established"])
		if !exists && len(tenant.Status.Message) == 0 {
			tenant.Status.State = established
			tenant.Status.Message = []string{statusDict["tenant-established"]}
			t.sendEmail(tenant, nil, "tenant-creation-successful")
		}
	} else {
		// Delete all subsidiary namespaces
		if namespaceRaw, err := t.clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s,edge-net.io/kind=sub", tenant.GetName())}); err == nil {
			for _, namespaceRow := range namespaceRaw.Items {
				t.clientset.CoreV1().Namespaces().Delete(context.TODO(), namespaceRow.GetName(), metav1.DeleteOptions{})
			}
		} else {
			// TODO: Provide err information at the status
		}
		// Delete all roles, role bindings, and subsidiary namespaces
		if err := t.clientset.RbacV1().ClusterRoles().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s", tenant.GetName())}); err != nil {
			// TODO: Provide err information at the status
		}
		if err := t.clientset.RbacV1().ClusterRoleBindings().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s", tenant.GetName())}); err != nil {
			// TODO: Provide err information at the status
		}
		if err := t.clientset.RbacV1().RoleBindings(tenant.GetName()).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
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

		if _, err := t.edgenetClientset.CoreV1alpha().Tenants().Create(context.TODO(), tenant.DeepCopy(), metav1.CreateOptions{}); err == nil {
			created = true
		} else {
			log.Infof("Couldn't create tenant %s: %s", tenant.GetName(), err)
		}
	}

	return created
}

func (t *Handler) createCoreNamespace(tenant *corev1alpha.Tenant, ownerReferences []metav1.OwnerReference) error {
	// Core namespace has the same name as the tenant
	tenantCoreNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenant.GetName(), OwnerReferences: ownerReferences}}
	// Namespace labels indicate this namespace created by a tenant, not by a team or slice
	namespaceLabels := map[string]string{"edge-net.io/kind": "core", "edge-net.io/tenant": tenant.GetName()}
	tenantCoreNamespace.SetLabels(namespaceLabels)
	exists, index := util.Contains(tenant.Status.Message, statusDict["namespace-failure"])
	_, err := t.clientset.CoreV1().Namespaces().Create(context.TODO(), tenantCoreNamespace, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		log.Infof("Couldn't create namespace for %s: %s", tenant.GetName(), err)
		if !exists {
			tenant.Status.State = failure
			tenant.Status.Message = append(tenant.Status.Message, statusDict["namespace-failure"])
		}
	} else if (err == nil || errors.IsAlreadyExists(err)) && exists {
		tenant.Status.Message = append(tenant.Status.Message[:index], tenant.Status.Message[index+1:]...)
	}
	return err
}

func (t *Handler) applyQuota(tenant *corev1alpha.Tenant) error {
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
	exists, index := util.Contains(tenant.Status.Message, statusDict["resource-quota-failure"])
	// Create the resource quota to prevent users from using this namespace for their applications
	_, err := t.clientset.CoreV1().ResourceQuotas(tenant.GetName()).Create(context.TODO(), resourceQuota.DeepCopy(), metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		log.Infof("Couldn't create resource quota in %s: %s", tenant.GetName(), err)
		if !exists {
			tenant.Status.State = failure
			tenant.Status.Message = append(tenant.Status.Message, statusDict["resource-quota-failure"])
		}
	} else if (err == nil || errors.IsAlreadyExists(err)) && exists {
		tenant.Status.Message = append(tenant.Status.Message[:index], tenant.Status.Message[index+1:]...)
	}
	return err
}

// ConfigurePermissions to generate rolebindings for owners, and users welcomed by owners
func (t *Handler) ConfigurePermissions(tenant *corev1alpha.Tenant, user *registrationv1alpha.UserRequest, ownerReferences []metav1.OwnerReference) {
	policyStatus := false
	exists, index := util.Contains(tenant.Status.Message, fmt.Sprintf(statusDict["user-failure"], user.Spec.Email))
	userLabels := user.GetLabels()
	if userLabels != nil && userLabels["edge-net.io/user-template-hash"] != "" {
		if exists {
			tenant.Status.Message = append(tenant.Status.Message[:index], tenant.Status.Message[index+1:]...)
		}

		var acceptableUsePolicyAccess = func(acceptableUsePolicy string) {
			if err := permission.CreateObjectSpecificClusterRole(tenant.GetName(), "core.edgenet.io", "acceptableusepolicies", acceptableUsePolicy, "owner", []string{"get", "update", "patch"}, ownerReferences); err != nil && !errors.IsAlreadyExists(err) {
				log.Infof("Couldn't create aup cluster role %s, %s: %s", tenant.GetName(), acceptableUsePolicy, err)
				// TODO: Provide err information at the status
			}
			clusterRoleName := fmt.Sprintf("edgenet:%s:acceptableusepolicies:%s-%s", tenant.GetName(), acceptableUsePolicy, "owner")
			roleBindLabels := map[string]string{"edge-net.io/tenant": tenant.GetName(), "edge-net.io/username": user.GetName(), "edge-net.io/user-template-hash": userLabels["edge-net.io/user-template-hash"]}
			exists, index := util.Contains(tenant.Status.Message, fmt.Sprintf(statusDict["aup-rolebinding-failure"], user.Spec.Email))
			if err := permission.CreateObjectSpecificClusterRoleBinding(tenant.GetName(), clusterRoleName, fmt.Sprintf("%s-%s", user.GetName(), userLabels["edge-net.io/user-template-hash"]), user.Spec.Email, roleBindLabels, ownerReferences); err != nil {
				log.Infof("Couldn't create aup cluster role binding %s, %s: %s", tenant.GetName(), acceptableUsePolicy, err)
				t.sendEmail(tenant, user, "user-creation-failure")
				if !exists {
					tenant.Status.State = failure
					tenant.Status.Message = append(tenant.Status.Message, fmt.Sprintf(statusDict["aup-rolebinding-failure"], user.Spec.Email))
				}
			} else if err == nil && exists {
				tenant.Status.Message = append(tenant.Status.Message[:index], tenant.Status.Message[index+1:]...)
			}

			if err := permission.CreateObjectSpecificClusterRole(tenant.GetName(), "core.edgenet.io", "acceptableusepolicies", acceptableUsePolicy, "administrator", []string{"get, delete"}, ownerReferences); err != nil && !errors.IsAlreadyExists(err) {
				log.Infof("Couldn't create aup cluster role %s for administrators, %s: %s", tenant.GetName(), acceptableUsePolicy, err)
				// TODO: Provide err information at the status
			}
			clusterRoleName = fmt.Sprintf("edgenet:%s:acceptableusepolicies:%s-%s", tenant.GetName(), acceptableUsePolicy, "administrator")
			// Give authorization to the administrators
			if acceptableUsePolicyRaw, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/generated=true,edge-net.io/tenant=%s,edge-net.io/identity=true", tenant.GetName())}); err == nil {
				for _, acceptableUsePolicyRow := range acceptableUsePolicyRaw.Items {
					aupLabels := acceptableUsePolicyRow.GetLabels()
					if aupLabels != nil && aupLabels["edge-net.io/username"] != "" && aupLabels["edge-net.io/role"] != "" {
						if user.GetName() != aupLabels["edge-net.io/username"] && (aupLabels["edge-net.io/role"] == "Owner" || aupLabels["edge-net.io/role"] == "Admin") {
							roleBindLabels := map[string]string{"edge-net.io/tenant": tenant.GetName(), "edge-net.io/username": aupLabels["edge-net.io/username"], "edge-net.io/user-template-hash": aupLabels["edge-net.io/user-template-hash"]}
							if err := permission.CreateObjectSpecificClusterRoleBinding(tenant.GetName(), clusterRoleName, fmt.Sprintf("%s-%s", aupLabels["edge-net.io/username"], aupLabels["edge-net.io/user-template-hash"]), acceptableUsePolicyRow.Spec.Email, roleBindLabels, ownerReferences); err != nil {
								log.Infof("Couldn't create aup cluster role binding %s, %s for %s: %s", tenant.GetName(), acceptableUsePolicy, fmt.Sprintf(aupLabels["edge-net.io/username"], aupLabels["edge-net.io/user-template-hash"]), err)
							}
						}
					}
				}
			}
		}
		// A hash code attached as suffix to allow people to roll in with the same username
		usernameHash := fmt.Sprintf("%s-%s", user.GetName(), userLabels["edge-net.io/user-template-hash"])
		acceptableUsePolicy, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), usernameHash, metav1.GetOptions{})
		if err == nil {
			policyStatus = acceptableUsePolicy.Spec.Accepted
			acceptableUsePolicyAccess(acceptableUsePolicy.GetName())
		} else if errors.IsNotFound(err) {
			// Generate an acceptable use policy object attached to user
			aupLabels := map[string]string{"edge-net.io/generated": "true", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/identity": "true", "edge-net.io/username": user.GetName(),
				"edge-net.io/user-template-hash": userLabels["edge-net.io/user-template-hash"], "edge-net.io/firstname": user.Spec.FirstName, "edge-net.io/lastname": user.Spec.LastName, "edge-net.io/role": user.Spec.Role}
			userAcceptableUsePolicy := &corev1alpha.AcceptableUsePolicy{TypeMeta: metav1.TypeMeta{Kind: "AcceptableUsePolicy", APIVersion: "apps.edgenet.io/v1alpha"},
				ObjectMeta: metav1.ObjectMeta{Name: usernameHash, OwnerReferences: ownerReferences}, Spec: corev1alpha.AcceptableUsePolicySpec{Email: user.Spec.Email, Accepted: false}}
			userAcceptableUsePolicy.SetLabels(aupLabels)
			if _, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Create(context.TODO(), userAcceptableUsePolicy, metav1.CreateOptions{}); err != nil {
				// TODO: Define the error precisely
			}
			acceptableUsePolicyAccess(userAcceptableUsePolicy.GetName())

			// Create the client certs for permanent use
			crt, key, err := registration.MakeUser(tenant.GetName(), usernameHash, user.Spec.Email)
			exists, index := util.Contains(tenant.Status.Message, fmt.Sprintf(statusDict["cert-failure"], user.Spec.Email))
			if err != nil {
				log.Infof("Couldn't generate client cert %s, %s: %s", tenant.GetName(), user.Spec.Email, err)
				t.sendEmail(tenant, user, "user-cert-failure")
				if !exists {
					tenant.Status.State = failure
					tenant.Status.Message = append(tenant.Status.Message, fmt.Sprintf(statusDict["cert-failure"], user.Spec.Email))
				}
			} else if err == nil && exists {
				tenant.Status.Message = append(tenant.Status.Message[:index], tenant.Status.Message[index+1:]...)
			}
			err = registration.MakeConfig(tenant.GetName(), usernameHash, user.Spec.Email, crt, key)
			exists, index = util.Contains(tenant.Status.Message, fmt.Sprintf(statusDict["kubeconfig-failure"], user.Spec.Email))
			if err != nil {
				log.Infof("Couldn't make kubeconfig file %s, %s: %s", tenant.GetName(), user.Spec.Email, err)
				t.sendEmail(tenant, user, "user-kubeconfig-failure")
				if !exists {
					tenant.Status.State = failure
					tenant.Status.Message = append(tenant.Status.Message, fmt.Sprintf(statusDict["kubeconfig-failure"], user.Spec.Email))
				}
			} else if err == nil && exists {
				tenant.Status.Message = append(tenant.Status.Message[:index], tenant.Status.Message[index+1:]...)
			}

			if aupFailure, _ := util.Contains(tenant.Status.Message, fmt.Sprintf(statusDict["aup-rolebinding-failure"], user.Spec.Email)); !aupFailure {
				if certFailure, _ := util.Contains(tenant.Status.Message, fmt.Sprintf(statusDict["cert-failure"], user.Spec.Email)); !certFailure {
					if kubeconfigFailure, _ := util.Contains(tenant.Status.Message, fmt.Sprintf(statusDict["kubeconfig-failure"], user.Spec.Email)); !kubeconfigFailure {
						t.sendEmail(nil, user, "user-registration-successful")
					}
				}
			}
		}

		if policyStatus {
			// Prepare role bindings
			// Create the role binding for essential permissions
			exists, index := util.Contains(tenant.Status.Message, fmt.Sprintf(statusDict["permission-rolebinding-failure"], user.Spec.Email))
			if err := permission.CreateObjectSpecificRoleBinding(tenant.GetName(), tenant.GetName(), fmt.Sprintf("edgenet:tenant-%s", strings.ToLower(user.Spec.Role)), user); err != nil {
				log.Infof("Couldn't create permission cluster role binding %s, %s: %s", tenant.GetName(), user.Spec.Email, err)
				t.sendEmail(tenant, user, "user-creation-failure")
				if !exists {
					tenant.Status.State = failure
					tenant.Status.Message = append(tenant.Status.Message, fmt.Sprintf(statusDict["permission-rolebinding-failure"], user.Spec.Email))
				}
			} else if err == nil && exists {
				tenant.Status.Message = append(tenant.Status.Message[:index], tenant.Status.Message[index+1:]...)
			}

			if strings.ToLower(user.Spec.Role) != "collaborator" {
				// Create the cluster role binding related to the tenant object
				roleBindLabels := map[string]string{"edge-net.io/tenant": tenant.GetName(), "edge-net.io/username": user.GetName(), "edge-net.io/user-template-hash": userLabels["edge-net.io/user-template-hash"]}
				exists, index := util.Contains(tenant.Status.Message, fmt.Sprintf(statusDict["administrator-rolebinding-failure"], user.Spec.Email))
				clusterRoleName := fmt.Sprintf("edgenet:%s:tenants:%s-%s", tenant.GetName(), tenant.GetName(), strings.ToLower(user.Spec.Role))
				if err := permission.CreateObjectSpecificClusterRoleBinding(tenant.GetName(), clusterRoleName, fmt.Sprintf("%s-%s", user.GetName(), userLabels["edge-net.io/user-template-hash"]), user.Spec.Email, roleBindLabels, ownerReferences); err != nil {
					log.Infof("Couldn't create administrator cluster role binding %s, %s: %s", tenant.GetName(), user.Spec.Email, err)
					t.sendEmail(tenant, user, "user-creation-failure")
					if !exists {
						tenant.Status.State = failure
						tenant.Status.Message = append(tenant.Status.Message, fmt.Sprintf(statusDict["administrator-rolebinding-failure"], user.Spec.Email))
					}
				} else if err == nil && exists {
					tenant.Status.Message = append(tenant.Status.Message[:index], tenant.Status.Message[index+1:]...)
				}
			}
		}
	} else if !exists {
		tenant.Status.State = failure
		tenant.Status.Message = append(tenant.Status.Message, fmt.Sprintf(statusDict["user-failure"], user.Spec.Email))
	}
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(tenant *corev1alpha.Tenant, user *registrationv1alpha.UserRequest, subject string) {
	// Set the HTML template variables
	contentData := mailer.CommonContentData{}
	if tenant == nil {
		userLabels := user.GetLabels()
		usernameHash := fmt.Sprintf("%s-%s", user.GetName(), userLabels["edge-net.io/user-template-hash"])
		contentData.CommonData.Tenant = user.Spec.Tenant
		contentData.CommonData.Username = usernameHash
		contentData.CommonData.Name = fmt.Sprintf("%s %s", user.Spec.FirstName, user.Spec.LastName)
		contentData.CommonData.Email = []string{user.Spec.Email}
	} else {
		contentData.CommonData.Tenant = tenant.GetName()
		if user == nil {
			if tenantRequest, err := t.edgenetClientset.RegistrationV1alpha().TenantRequests().Get(context.TODO(), tenant.GetName(), metav1.GetOptions{}); err == nil {
				contentData.CommonData.Username = tenantRequest.Spec.Contact.Username
			}
			contentData.CommonData.Name = fmt.Sprintf("%s %s", tenant.Spec.Contact.FirstName, tenant.Spec.Contact.LastName)
			contentData.CommonData.Email = []string{tenant.Spec.Contact.Email}
		} else {
			userLabels := user.GetLabels()
			usernameHash := fmt.Sprintf("%s-%s", user.GetName(), userLabels["edge-net.io/user-template-hash"])
			contentData.CommonData.Username = usernameHash
			contentData.CommonData.Name = fmt.Sprintf("%s %s", user.Spec.FirstName, user.Spec.LastName)
			if acceptableUsePolicyRaw, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/generated=true,edge-net.io/tenant=%s,edge-net.io/identity=true", tenant.GetName())}); err == nil {
				for _, acceptableUsePolicyRow := range acceptableUsePolicyRaw.Items {
					aupLabels := acceptableUsePolicyRow.GetLabels()
					if aupLabels != nil && aupLabels["edge-net.io/username"] != "" && aupLabels["edge-net.io/user-template-hash"] != "" {
						authorized := permission.CheckAuthorization("", acceptableUsePolicyRow.Spec.Email, "UserRequest", user.GetName(), "cluster")
						if authorized {
							contentData.CommonData.Email = append(contentData.CommonData.Email, acceptableUsePolicyRow.Spec.Email)
						}
					}
				}
			}
		}
	}
	mailer.Send(subject, contentData)
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
