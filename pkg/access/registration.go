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
	"errors"
	"fmt"
	"os"

	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	registrationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"
	slack "github.com/EdgeNet-project/edgenet/pkg/slack"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	//cmdconfig "k8s.io/kubernetes/pkg/kubectl/cmd/config"
)

// Clientset to be synced by the custom resources
var Clientset kubernetes.Interface
var EdgenetClientset clientset.Interface

// Create function is for being used by other resources to create a tenant
func CreateTenant(tenantRequest *registrationv1alpha1.TenantRequest) error {
	// Create a tenant on the cluster
	tenant := new(corev1alpha1.Tenant)
	tenant.SetName(tenantRequest.GetName())
	tenant.Spec.Address = tenantRequest.Spec.Address
	tenant.Spec.Contact = tenantRequest.Spec.Contact
	tenant.Spec.FullName = tenantRequest.Spec.FullName
	tenant.Spec.ShortName = tenantRequest.Spec.ShortName
	tenant.Spec.URL = tenantRequest.Spec.URL
	tenant.Spec.ClusterNetworkPolicy = tenantRequest.Spec.ClusterNetworkPolicy
	tenant.Spec.Enabled = true
	tenant.SetLabels(map[string]string{"edge-net.io/request-uid": string(tenantRequest.GetUID())})
	tenant.SetAnnotations(tenantRequest.GetAnnotations())
	if tenantRequest.GetOwnerReferences() != nil && len(tenantRequest.GetOwnerReferences()) > 0 {
		tenant.SetOwnerReferences(tenantRequest.GetOwnerReferences())
	}

	if tenantCreated, err := EdgenetClientset.CoreV1alpha1().Tenants().Create(context.TODO(), tenant, metav1.CreateOptions{}); err != nil {
		klog.Infof("Couldn't create tenant %s: %s", tenant.GetName(), err)
		return err
	} else {
		if tenantRequest.Spec.ResourceAllocation != nil {
			claim := corev1alpha1.ResourceTuning{
				ResourceList: tenantRequest.Spec.ResourceAllocation,
			}
			applied := make(chan error, 1)
			go ApplyTenantResourceQuota(tenant.GetName(), []metav1.OwnerReference{tenantCreated.MakeOwnerReference()}, claim, applied)
			return <-applied
		}
		return nil
	}

}

// ApplyTenantResourceQuota generates a tenant resource quota with the name provided
func ApplyTenantResourceQuota(name string, ownerReferences []metav1.OwnerReference, claim corev1alpha1.ResourceTuning, applied chan<- error) {
	created := make(chan bool, 1)
	go checkNamespaceCreation(name, created)
	if <-created {
		if tenantResourceQuota, err := EdgenetClientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), name, metav1.GetOptions{}); err == nil {
			tenantResourceQuota.Spec.Claim["initial"] = claim
			if _, err := EdgenetClientset.CoreV1alpha1().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuota.DeepCopy(), metav1.UpdateOptions{}); err != nil {
				klog.Infof("Couldn't update tenant resource quota %s: %s", name, err)
				applied <- err
			}
		} else {
			tenantResourceQuota := new(corev1alpha1.TenantResourceQuota)
			tenantResourceQuota.SetName(name)
			if ownerReferences != nil {
				tenantResourceQuota.SetOwnerReferences(ownerReferences)
			}
			tenantResourceQuota.Spec.Claim = make(map[string]corev1alpha1.ResourceTuning)
			tenantResourceQuota.Spec.Claim["initial"] = claim
			if _, err := EdgenetClientset.CoreV1alpha1().TenantResourceQuotas().Create(context.TODO(), tenantResourceQuota.DeepCopy(), metav1.CreateOptions{}); err != nil {
				klog.Infof("Couldn't create tenant resource quota %s: %s", name, err)
				applied <- err
			}
		}
		close(applied)
		return
	}
	applied <- errors.New("tenant namespace could not be created in 5 minutes")
	close(applied)
}

func checkNamespaceCreation(tenant string, created chan<- bool) {
	if coreNamespace, err := Clientset.CoreV1().Namespaces().Get(context.TODO(), tenant, metav1.GetOptions{}); err == nil && coreNamespace.Status.Phase != "Terminating" {
		created <- true
		close(created)
		return
	} else {
		timeout := int64(300)
		watchNamespace, err := Clientset.CoreV1().Namespaces().Watch(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s", tenant), TimeoutSeconds: &timeout})
		if err == nil {
			// Get events from watch interface
			for namespaceEvent := range watchNamespace.ResultChan() {
				namespace, status := namespaceEvent.Object.(*corev1.Namespace)
				if status {
					if namespace.Status.Phase != "Terminating" {
						created <- true
						close(created)
						watchNamespace.Stop()
						return
					}
				}

			}
		}
	}
	created <- false
	close(created)
}

func SendEmailForRoleRequest(roleRequestCopy *registrationv1alpha1.RoleRequest, purpose, subject, clusterUID string, recipient []string) {
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

func SendEmailForTenantRequest(tenantRequestCopy *registrationv1alpha1.TenantRequest, purpose, subject, clusterUID string, recipient []string) {
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

func SendEmailForClusterRoleRequest(clusterRoleRequestCopy *registrationv1alpha1.ClusterRoleRequest, purpose, subject, clusterUID string, recipient []string) {
	email := new(mailer.Content)
	email.Cluster = clusterUID
	email.User = clusterRoleRequestCopy.Spec.Email
	email.FirstName = clusterRoleRequestCopy.Spec.FirstName
	email.LastName = clusterRoleRequestCopy.Spec.LastName
	email.Subject = subject
	email.Recipient = recipient
	email.ClusterRoleRequest = new(mailer.ClusterRoleRequest)
	email.ClusterRoleRequest.Name = clusterRoleRequestCopy.GetName()
	email.Send(purpose)
}

// Send a slack notification for role request
func SendSlackNotificationForRoleRequest(roleRequestCopy *registrationv1alpha1.RoleRequest, purpose, subject, clusterUID string) {
	slackNotification := new(slack.Content)
	slackNotification.Cluster = clusterUID
	slackNotification.User = roleRequestCopy.Spec.Email
	slackNotification.FirstName = roleRequestCopy.Spec.FirstName
	slackNotification.LastName = roleRequestCopy.Spec.LastName
	slackNotification.Subject = subject
	slackNotification.AuthToken = os.Getenv(slack.AUTH_TOKEN_IDENTIFIER)
	slackNotification.ChannelId = os.Getenv(slack.CHANNEL_ID_IDENTIFIER)
	slackNotification.RoleRequest = new(slack.RoleRequest)
	slackNotification.RoleRequest.Name = roleRequestCopy.GetName()
	slackNotification.RoleRequest.Namespace = roleRequestCopy.GetNamespace()
	slackNotification.Send(purpose)
}

// Send a slack notification for tenant request
func SendSlackNotificationForTenantRequest(tenantRequestCopy *registrationv1alpha1.TenantRequest, purpose, subject, clusterUID string) {
	slackNotification := new(slack.Content)
	slackNotification.Cluster = clusterUID
	slackNotification.User = tenantRequestCopy.Spec.Contact.Email
	slackNotification.FirstName = tenantRequestCopy.Spec.Contact.FirstName
	slackNotification.LastName = tenantRequestCopy.Spec.Contact.LastName
	slackNotification.Subject = subject
	slackNotification.AuthToken = os.Getenv(slack.AUTH_TOKEN_IDENTIFIER)
	slackNotification.ChannelId = os.Getenv(slack.CHANNEL_ID_IDENTIFIER)
	slackNotification.TenantRequest = new(slack.TenantRequest)
	slackNotification.TenantRequest.Tenant = tenantRequestCopy.GetName()
	slackNotification.Send(purpose)
}

// Send a slack notification for cluster role request
func SendSlackNotificationForClusterRoleRequest(clusterRoleRequestCopy *registrationv1alpha1.ClusterRoleRequest, purpose, subject, clusterUID string) {
	slackNotification := new(slack.Content)
	slackNotification.Cluster = clusterUID
	slackNotification.User = clusterRoleRequestCopy.Spec.Email
	slackNotification.FirstName = clusterRoleRequestCopy.Spec.FirstName
	slackNotification.LastName = clusterRoleRequestCopy.Spec.LastName
	slackNotification.Subject = subject
	slackNotification.AuthToken = os.Getenv(slack.AUTH_TOKEN_IDENTIFIER)
	slackNotification.ChannelId = os.Getenv(slack.CHANNEL_ID_IDENTIFIER)
	slackNotification.ClusterRolerequest = new(slack.ClusterRoleRequest)
	slackNotification.ClusterRolerequest.Name = clusterRoleRequestCopy.GetName()
	slackNotification.Send(purpose)
}
