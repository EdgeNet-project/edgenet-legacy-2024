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

package v1alpha

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TenantRequest describes a TenantRequest resource
type TenantRequest struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the tenantrequest resource spec
	Spec TenantRequestSpec `json:"spec"`
	// Status is the tenantrequest resource status
	Status TenantRequestStatus `json:"status,omitempty"`
}

// TenantRequestSpec is the spec for a TenantRequest resource
type TenantRequestSpec struct {
	// Full name of the tenant.
	FullName string `json:"fullname"`
	// Shortened name of the tenant.
	ShortName string `json:"shortname"`
	// Website of the tenant.
	URL string `json:"url"`
	// Open address of the tenant, this includes country, city, and street information.
	Address corev1alpha.Address `json:"address"`
	// Contact information of the tenant.
	Contact corev1alpha.Contact `json:"contact"`

	ClusterNetworkPolicy bool `json:"clusternetworkpolicy"`
	// Requested allocation of certain resource types. Resource types are
	// kubernetes default resource types.
	ResourceAllocation map[corev1.ResourceName]resource.Quantity `json:"resourceallocation"`
	// If the tenant is approved or not by the EdgeNet administrators.
	Approved bool `json:"approved"`
}

// TenantRequestStatus is the status for a TenantRequest resource
type TenantRequestStatus struct {
	// True if the policy agreed false if not.
	PolicyAgreed *bool `json:"policyagreed"`
	// Expiration date of the policy.
	Expiry *metav1.Time `json:"expiry"`
	// Current state of the policy. This can be 'Failure', 'Pending', or 'Approved'.
	State string `json:"state"`
	// Description for additional information.
	Message string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TenantRequestList is a list of TenantRequest resources
type TenantRequestList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// Tenants can declare requests. This list contains their requests.
	Items []TenantRequest `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RoleRequest describes a RoleRequest resource
type RoleRequest struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the rolerequest resource spec
	Spec RoleRequestSpec `json:"spec"`
	// Status is the rolerequest resource status
	Status RoleRequestStatus `json:"status,omitempty"`
}

// RoleRequestSpec is the spec for a RoleRequest resource
type RoleRequestSpec struct {
	// First name of the person requesting the role.
	FirstName string `json:"firstname"`
	// Last name of the person requesting the role.
	LastName string `json:"lastname"`
	// Email of the person requesting the role.
	Email string `json:"email"`
	// RoleRefSpec indicates the requested Role or ClusterRole
	RoleRef RoleRefSpec `json:"roleref"`
	// True if this role request is approved false if not.
	Approved bool `json:"approved"`
}

// RoleRefSpec indicates the requested Role / ClusterRole
type RoleRefSpec struct {
	// The kind of the RoleRefSpec, this can be 'ClusterRole', or 'Role'.
	Kind string `json:"kind"`
	// Name of the owner of this request.
	Name string `json:"name"`
}

// RoleRequestStatus is the status for a RoleRequest resource
type RoleRequestStatus struct {
	// True if agreed to the policy false if not.
	PolicyAgreed *bool `json:"policyagreed"`
	// Expiration date of the policy.
	Expiry *metav1.Time `json:"expiry"`
	// Current state of the policy. This can be 'Failure', 'Pending', or 'Approved'.
	State string `json:"state"`
	// Description for additional information.
	Message string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RoleRequestList is a list of RoleRequest resources
type RoleRequestList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// RoleRequestList is a list of RoleRequests. This element contains
	// RoleRequest resources.
	Items []RoleRequest `json:"items"`
}
