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

package access

import (
	"context"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	//cmdconfig "k8s.io/kubernetes/pkg/kubectl/cmd/config"
)

// Clientset to be synced by the custom resources
var Clientset kubernetes.Interface
var EdgenetClientset clientset.Interface

// Create function is for being used by other resources to create a tenant
func CreateTenant(tenantRequest *registrationv1alpha.TenantRequest) error {
	// Create a tenant on the cluster
	tenant := new(corev1alpha.Tenant)
	tenant.SetName(tenantRequest.GetName())
	tenant.Spec.Address = tenantRequest.Spec.Address
	tenant.Spec.Contact = tenantRequest.Spec.Contact
	tenant.Spec.FullName = tenantRequest.Spec.FullName
	tenant.Spec.ShortName = tenantRequest.Spec.ShortName
	tenant.Spec.URL = tenantRequest.Spec.URL
	tenant.Spec.Enabled = true
	if tenantRequest.GetOwnerReferences() != nil {
		tenant.SetOwnerReferences(tenantRequest.GetOwnerReferences())
	}

	if _, err := EdgenetClientset.CoreV1alpha().Tenants().Create(context.TODO(), tenant, metav1.CreateOptions{}); err != nil {
		klog.V(4).Infof("Couldn't create tenant %s: %s", tenant.GetName(), err)
		return err
	}

	if tenantRequest.Spec.ResourceAllocation != nil {
		// TODO: Take tenant resource quota into account while error handling
		claim := corev1alpha.ResourceTuning{
			ResourceList: tenantRequest.Spec.ResourceAllocation,
		}
		err := ApplyTenantResourceQuota(tenantRequest.GetName(), nil, claim)
		if err != nil {
			klog.V(4).Infof("Couldn't create tenant resource quota %s: %s", tenantRequest.GetName(), err)
		}
	}

	return nil
}

// ApplyTenantResourceQuota generates a tenant resource quota with the name provided
func ApplyTenantResourceQuota(name string, ownerReferences []metav1.OwnerReference, claim corev1alpha.ResourceTuning) error {
	// Set a tenant resource quota
	if tenantResourceQuota, err := EdgenetClientset.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), name, metav1.GetOptions{}); err == nil {
		tenantResourceQuota.Spec.Claim["initial"] = claim
		if _, err := EdgenetClientset.CoreV1alpha().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuota.DeepCopy(), metav1.UpdateOptions{}); err != nil {
			return err
		}
	} else {
		tenantResourceQuota := new(corev1alpha.TenantResourceQuota)
		tenantResourceQuota.SetName(name)
		if ownerReferences != nil {
			tenantResourceQuota.SetOwnerReferences(ownerReferences)
		}
		tenantResourceQuota.Spec.Claim = make(map[string]corev1alpha.ResourceTuning)
		tenantResourceQuota.Spec.Claim["initial"] = claim
		if _, err := EdgenetClientset.CoreV1alpha().TenantResourceQuotas().Create(context.TODO(), tenantResourceQuota.DeepCopy(), metav1.CreateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func SendEmailForRoleRequest(roleRequestCopy *registrationv1alpha.RoleRequest, purpose, subject, clusterUID string, recipient []string) {
	email := new(mailer.Content)
	email.Cluster = clusterUID
	email.User = roleRequestCopy.Spec.Email
	email.FirstName = roleRequestCopy.Spec.FirstName
	email.LastName = roleRequestCopy.Spec.LastName
	email.Subject = subject
	email.Recipient = recipient
	email.RoleRequest = new(mailer.RoleRequest)
	email.RoleRequest.Name = roleRequestCopy.GetName()
	email.RoleRequest.Namespace = roleRequestCopy.GetNamespace()
	email.Send(purpose)
}

func SendEmailForTenantRequest(tenantRequestCopy *registrationv1alpha.TenantRequest, purpose, subject, clusterUID string, recipient []string) {
	email := new(mailer.Content)
	email.Cluster = clusterUID
	email.User = tenantRequestCopy.Spec.Contact.Email
	email.FirstName = tenantRequestCopy.Spec.Contact.FirstName
	email.LastName = tenantRequestCopy.Spec.Contact.LastName
	email.Subject = subject
	email.Recipient = recipient
	email.TenantRequest = new(mailer.TenantRequest)
	email.TenantRequest.Tenant = tenantRequestCopy.GetName()
	email.Send(purpose)
}
