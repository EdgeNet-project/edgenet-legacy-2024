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
	"strings"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	//cmdconfig "k8s.io/kubernetes/pkg/kubectl/cmd/config"
)

// Clientset to be synced by the custom resources
var Clientset kubernetes.Interface
var EdgenetClientset clientset.Interface

// Create function is for being used by other resources to create a tenant
func CreateTenant(tenantRequest *registrationv1alpha.TenantRequest) bool {
	created := false
	// Create a tenant on the cluster
	tenant := new(corev1alpha.Tenant)
	tenant.SetName(tenantRequest.GetName())
	tenant.Spec.Address = tenantRequest.Spec.Address
	tenant.Spec.Contact = tenantRequest.Spec.Contact
	tenant.Spec.FullName = tenantRequest.Spec.FullName
	tenant.Spec.ShortName = tenantRequest.Spec.ShortName
	tenant.Spec.URL = tenantRequest.Spec.URL
	tenant.Spec.Enabled = true

	if _, err := EdgenetClientset.CoreV1alpha().Tenants().Create(context.TODO(), tenant, metav1.CreateOptions{}); err == nil {
		created = true
	} else {
		klog.V(4).Infof("Couldn't create tenant %s: %s", tenant.GetName(), err)
	}

	// TODO: Take tenant resource quota into account while error handling
	tenantResourceQuota := new(corev1alpha.TenantResourceQuota)
	tenantResourceQuota.SetName(tenantRequest.GetName())
	tenantResourceQuota.Spec.Claim = make(map[string]corev1alpha.ResourceTuning)
	claim := corev1alpha.ResourceTuning{
		ResourceList: tenantRequest.Spec.ResourceAllocation,
	}
	tenantResourceQuota.Spec.Claim["initial"] = claim

	if _, err := EdgenetClientset.CoreV1alpha().TenantResourceQuotas().Create(context.TODO(), tenantResourceQuota, metav1.CreateOptions{}); err != nil {
		klog.V(4).Infof("Couldn't create tenant resource quota %s: %s", tenantResourceQuota.GetName(), err)
	}

	return created
}

// Create to provide one-time code for verification
func CreateEmailVerification(obj interface{}, ownerReferences []metav1.OwnerReference) bool {
	// The section below is a part of the method which provides email verification
	// Email verification code is a security point for email verification. The user
	// registration object creates an email verification object with a name which is
	// this email verification code. Only who knows the email verification
	// code can manipulate that object by using a public token.
	created := false
	switch obj.(type) {
	case *registrationv1alpha.TenantRequest:
		tenantRequest := obj.(*registrationv1alpha.TenantRequest)

		code := "tr-" + util.GenerateRandomString(16)
		emailVerification := registrationv1alpha.EmailVerification{ObjectMeta: metav1.ObjectMeta{OwnerReferences: ownerReferences}}
		emailVerification.SetName(code)
		emailVerification.Spec.Email = tenantRequest.Spec.Contact.Email
		// labels: tenant, user, code - attach to the email verification and tenant
		labels := map[string]string{"edge-net.io/tenant": tenantRequest.GetName(), "edge-net.io/username": tenantRequest.Spec.Contact.Username, "edge-net.io/registration": "tenant"}
		emailVerification.SetLabels(labels)

		_, err := EdgenetClientset.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), emailVerification.DeepCopy(), metav1.CreateOptions{})
		if err == nil {
			created = true
			//SendEmailVerificationNotification("tenant-email-verification", tenantRequest.GetName(), tenantRequest.Spec.Contact.Username,
			//	fmt.Sprintf("%s %s", tenantRequest.Spec.Contact.FirstName, tenantRequest.Spec.Contact.LastName), tenantRequest.Spec.Contact.Email, code)
		} else {
			//SendEmailVerificationNotification("tenant-email-verification-malfunction", tenantRequest.GetName(), tenantRequest.Spec.Contact.Username,
			//	fmt.Sprintf("%s %s", tenantRequest.Spec.Contact.FirstName, tenantRequest.Spec.Contact.LastName), tenantRequest.Spec.Contact.Email, "")
		}
	case *registrationv1alpha.UserRequest:
		userRequest := obj.(*registrationv1alpha.UserRequest)
		code := "ur-" + util.GenerateRandomString(16)
		emailVerification := registrationv1alpha.EmailVerification{ObjectMeta: metav1.ObjectMeta{OwnerReferences: ownerReferences}}
		emailVerification.SetName(code)
		emailVerification.Spec.Email = userRequest.Spec.Email
		// labels: tenant, user, code - attach to the email verification and tenant
		labels := map[string]string{"edge-net.io/tenant": strings.ToLower(userRequest.Spec.Tenant), "edge-net.io/username": userRequest.GetName(), "edge-net.io/registration": "user"}
		emailVerification.SetLabels(labels)
		_, err := EdgenetClientset.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), emailVerification.DeepCopy(), metav1.CreateOptions{})
		if err == nil {
			created = true
			//SendEmailVerificationNotification("user-email-verification", strings.ToLower(userRequest.Spec.Tenant), userRequest.GetName(),
			//	fmt.Sprintf("%s %s", userRequest.Spec.FirstName, userRequest.Spec.LastName), userRequest.Spec.Email, code)
		} else {
			//SendEmailVerificationNotification("user-email-verification-malfunction", strings.ToLower(userRequest.Spec.Tenant), userRequest.GetName(),
			//	fmt.Sprintf("%s %s", userRequest.Spec.FirstName, userRequest.Spec.LastName), userRequest.Spec.Email, "")
		}
	}
	return created
}

// CreateTenantResourceQuota generates a tenant resource quota with the name provided
func CreateTenantResourceQuota(name string, ownerReferences []metav1.OwnerReference) (string, string) {
	cpuQuota := "0m"
	memoryQuota := "0Mi"
	_, err := EdgenetClientset.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		// Set a tenant resource quota
		tenantResourceQuota := corev1alpha.TenantResourceQuota{}
		tenantResourceQuota.SetName(name)
		tenantResourceQuota.SetOwnerReferences(ownerReferences)
		claim := corev1alpha.ResourceTuning{}
		claim.ResourceList = make(map[corev1.ResourceName]resource.Quantity)
		cpuQuota := "8000m"
		memoryQuota := "8192Mi"
		if quantity, err := resource.ParseQuantity(cpuQuota); err == nil {
			claim.ResourceList["cpu"] = quantity
		}
		if quantity, err := resource.ParseQuantity(memoryQuota); err == nil {
			claim.ResourceList["memory"] = quantity
		}
		tenantResourceQuota.Spec.Claim = make(map[string]corev1alpha.ResourceTuning)
		tenantResourceQuota.Spec.Claim["initial"] = claim
		_, err = EdgenetClientset.CoreV1alpha().TenantResourceQuotas().Create(context.TODO(), tenantResourceQuota.DeepCopy(), metav1.CreateOptions{})
		if err != nil {
			klog.V(4).Infof(statusDict["TRQ-failed"], name, err)
		}
	}
	return cpuQuota, memoryQuota
}

// SendEmail to send notification to tenant admins and authorized users about email verification
/*func SendEmailVerificationNotification(subject, tenant, username, fullname, email, code string) {
	// Set the HTML template variables
	var contentData interface{}

	collective := mailer.CommonContentData{}
	collective.CommonData.Tenant = tenant
	collective.CommonData.Username = username
	collective.CommonData.Name = fullname
	collective.CommonData.Email = []string{}
	if subject == "tenant-email-verification" || subject == "user-email-verification-update" ||
		subject == "user-email-verification" {
		collective.CommonData.Email = []string{email}
		verifyContent := mailer.VerifyContentData{}
		verifyContent.Code = code
		verifyContent.CommonData = collective.CommonData
		contentData = verifyContent
	} else if subject == "user-email-verified-alert" {
		// Put the email addresses of the tenant admins and authorized users in the email to be sent list
		tenant, _ := EdgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), tenant, metav1.GetOptions{})

		if acceptableUsePolicyRaw, err := EdgenetClientset.CoreV1alpha().AcceptableUsePolicies().List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/generated=true,edge-net.io/tenant=%s,edge-net.io/identity=true", tenant.GetName())}); err == nil {
			for _, acceptableUsePolicyRow := range acceptableUsePolicyRaw.Items {
				aupLabels := acceptableUsePolicyRow.GetLabels()
				if aupLabels != nil && aupLabels["edge-net.io/username"] != "" && aupLabels["edge-net.io/firstname"] != "" && aupLabels["edge-net.io/lastname"] != "" {
					authorized := CheckAuthorization("", acceptableUsePolicyRow.Spec.Email, "userrequests", username, "cluster")
					if authorized {
						collective.CommonData.Email = append(collective.CommonData.Email, acceptableUsePolicyRow.Spec.Email)
					}
				}
			}
		}
		contentData = collective
	} else {
		collective.CommonData.Email = []string{email}
		contentData = collective
	}

	mailer.Send(subject, contentData)
}*/

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
