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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
)

// Values of Status.State
const (
	StatusFailed = "Failed"
	// Tenant request
	StatusPending  = "Pending"  // Also used for role request and cluster role request
	StatusApproved = "Approved" // Also used for role request and cluster role request
	StatusCreated  = "Created"
	// Role request
	StatusBound = "Bound" // Also used for cluster role request
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
	Address corev1alpha1.Address `json:"address"`
	// Contact information of the tenant.
	Contact corev1alpha1.Contact `json:"contact"`
	// Whether cluster-level network policies will be applied to tenant namespaces
	// for security purposes.
	ClusterNetworkPolicy bool `json:"clusternetworkpolicy"`
	// Requested allocation of certain resource types. Resource types are
	// kubernetes default resource types.
	ResourceAllocation map[corev1.ResourceName]resource.Quantity `json:"resourceallocation"`
	// If the tenant is approved or not by the administrators.
	Approved bool `json:"approved"`
	// Description provides additional information about the tenant.
	Description string `json:"description"`
}

// TenantRequestStatus is the status for a TenantRequest resource
type TenantRequestStatus struct {
	// Expiration date of the request.
	Expiry *metav1.Time `json:"expiry"`
	// Current state of the policy. This can be 'Failure', 'Pending', or 'Approved'.
	State string `json:"state"`
	// Description for additional information.
	Message string `json:"message"`
	// True if the notification send out
	Notified bool `json:"notified"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TenantRequestList is a list of TenantRequest resources
type TenantRequestList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// TenantRequestList is a list of TenantRequest resources. This element contains
	// TenantRequest resources.
	Items []TenantRequest `json:"items"`
}

func (tr TenantRequest) MakeOwnerReference() metav1.OwnerReference {
	return *metav1.NewControllerRef(&tr.ObjectMeta, SchemeGroupVersion.WithKind("TenantRequest"))
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterRoleRequest describes a RoleRequest resource
type ClusterRoleRequest struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the clusterrolerequest resource spec
	Spec ClusterRoleRequestSpec `json:"spec"`
	// Status is the clusterrolerequest resource status
	Status ClusterRoleRequestStatus `json:"status,omitempty"`
}

// ClusterRoleRequestSpec is the spec for a ClusterRoleRequest resource
type ClusterRoleRequestSpec struct {
	// First name of the person requesting the cluster role.
	FirstName string `json:"firstname"`
	// Last name of the person requesting the cluster role.
	LastName string `json:"lastname"`
	// Email of the person requesting the cluster role.
	Email string `json:"email"`
	// Name of the cluster role to bind
	RoleName string `json:"rolename"`
	// True if this role request is approved false if not.
	Approved bool `json:"approved"`
}

// ClusterRoleRequestStatus is the status for a ClusterRoleRequest resource
type ClusterRoleRequestStatus struct {
	// Expiration date of the request.
	Expiry *metav1.Time `json:"expiry"`
	// Current state of the policy. This can be 'Failure', 'Pending', or 'Approved'.
	State string `json:"state"`
	// Description for additional information.
	Message string `json:"message"`
	// True if the notification send out
	Notified bool `json:"notified"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterRoleRequestList is a list of ClusterRoleRequest resources
type ClusterRoleRequestList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// ClusterRoleRequestList is a list of ClusterRoleRequest resources. This element contains
	// ClusterRoleRequest resources.
	Items []ClusterRoleRequest `json:"items"`
}

func (crr ClusterRoleRequest) MakeOwnerReference() metav1.OwnerReference {
	return *metav1.NewControllerRef(&crr.ObjectMeta, SchemeGroupVersion.WithKind("ClusterRoleRequest"))
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
	// Name of the role.
	Name string `json:"name"`
}

// RoleRequestStatus is the status for a RoleRequest resource
type RoleRequestStatus struct {
	// Expiration date of the request.
	Expiry *metav1.Time `json:"expiry"`
	// Current state of the policy. This can be 'Failure', 'Pending', or 'Approved'.
	State string `json:"state"`
	// Description for additional information.
	Message string `json:"message"`
	// True if the notification send out
	Notified bool `json:"notified"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RoleRequestList is a list of RoleRequest resources
type RoleRequestList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// RoleRequestList is a list of RoleRequest resources. This element contains
	// RoleRequest resources.
	Items []RoleRequest `json:"items"`
}

func (rr RoleRequest) MakeOwnerReference() metav1.OwnerReference {
	return *metav1.NewControllerRef(&rr.ObjectMeta, SchemeGroupVersion.WithKind("RoleRequest"))
}
