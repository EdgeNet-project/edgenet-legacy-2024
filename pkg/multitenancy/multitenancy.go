/*
Copyright 2022 Contributors to the EdgeNet project.

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

package multitenancy

import (
	"context"
	"errors"
	"fmt"

	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	registrationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// Manager is the implementation to set up multitenancy.
type Manager struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface
}

// NewManager returns a new multitenancy manager
func NewManager(kubeclientset kubernetes.Interface,
	edgenetclientset clientset.Interface) *Manager {
	return &Manager{kubeclientset, edgenetclientset}
}

// CreateTenant function is for being used by other resources to create a tenant
func (m *Manager) CreateTenant(tenantRequest *registrationv1alpha1.TenantRequest) error {
	// Create a tenant on the cluster
	tenant := new(corev1alpha1.Tenant)
	tenant.SetName(tenantRequest.GetName())
	tenant.Spec.Address = tenantRequest.Spec.Address
	tenant.Spec.Contact = tenantRequest.Spec.Contact
	tenant.Spec.FullName = tenantRequest.Spec.FullName
	tenant.Spec.ShortName = tenantRequest.Spec.ShortName
	tenant.Spec.URL = tenantRequest.Spec.URL
	tenant.Spec.ClusterNetworkPolicy = tenantRequest.Spec.ClusterNetworkPolicy
	tenant.Spec.Description = tenantRequest.Spec.Description
	tenant.Spec.Enabled = true
	tenant.SetLabels(map[string]string{"edge-net.io/request-uid": string(tenantRequest.GetUID())})
	tenant.SetAnnotations(tenantRequest.GetAnnotations())
	if tenantRequest.GetOwnerReferences() != nil && len(tenantRequest.GetOwnerReferences()) > 0 {
		tenant.SetOwnerReferences(tenantRequest.GetOwnerReferences())
	}

	tenantCreated, err := m.edgenetclientset.CoreV1alpha1().Tenants().Create(context.TODO(), tenant, metav1.CreateOptions{})
	if err != nil {
		klog.Infof("Couldn't create tenant %s: %s", tenant.GetName(), err)
		return err
	}
	if tenantRequest.Spec.ResourceAllocation != nil {
		claim := corev1alpha1.ResourceTuning{
			ResourceList: tenantRequest.Spec.ResourceAllocation,
		}
		applied := make(chan error, 1)
		go m.ApplyTenantResourceQuota(tenant.GetName(), []metav1.OwnerReference{tenantCreated.MakeOwnerReference()}, claim, applied)
		return <-applied
	}
	return nil
}

// ApplyTenantResourceQuota generates a tenant resource quota with the name provided
func (m *Manager) ApplyTenantResourceQuota(name string, ownerReferences []metav1.OwnerReference, claim corev1alpha1.ResourceTuning, applied chan<- error) {
	created := make(chan bool, 1)
	go m.checkNamespaceCreation(name, created)
	if <-created {
		if tenantResourceQuota, err := m.edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), name, metav1.GetOptions{}); err == nil {
			tenantResourceQuota.Spec.Claim["initial"] = claim
			if _, err := m.edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuota.DeepCopy(), metav1.UpdateOptions{}); err != nil {
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
			if _, err := m.edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Create(context.TODO(), tenantResourceQuota.DeepCopy(), metav1.CreateOptions{}); err != nil {
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

func (m *Manager) checkNamespaceCreation(tenant string, created chan<- bool) {
	if coreNamespace, err := m.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), tenant, metav1.GetOptions{}); err == nil && coreNamespace.Status.Phase != "Terminating" {
		created <- true
		close(created)
		return
	}
	timeout := int64(300)
	watchNamespace, err := m.kubeclientset.CoreV1().Namespaces().Watch(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s", tenant), TimeoutSeconds: &timeout})
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
	created <- false
	close(created)
}
