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
	"fmt"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Tenant describes a tenant that consumes the cluster resources in an isolated environment
type Tenant struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the tenant resource spec
	Spec TenantSpec `json:"spec"`
	// Status is the tenant resource status
	Status TenantStatus `json:"status,omitempty"`
}

// TenantSpec is the spec for a Tenant resource
type TenantSpec struct {
	// Full name of the tenant.
	FullName string `json:"fullname"`
	// Shortened name of the tenant.
	ShortName string `json:"shortname"`
	// Website of the tenant.
	URL string `json:"url"`
	// Open address of the tenant, this includes country, city, and street information.
	Address Address `json:"address"`
	// Contact information of the tenant.
	Contact Contact `json:"contact"`
	// Whether cluster-level network policies will be applied to tenant namespaces
	// for security purposes.
	ClusterNetworkPolicy bool `json:"clusternetworkpolicy"`
	// If the tenant is active then this field is true.
	Enabled bool `json:"enabled"`
}

// Address describes postal address of tenant
type Address struct {
	// Street name.
	Street string `json:"street"`
	// ZIP code.
	ZIP string `json:"zip"`
	// City name.
	City string `json:"city"`
	// Region name.
	Region string `json:"region"`
	// County name.
	Country string `json:"country"`
}

// Contact contains handle, personal information, and role
type Contact struct {
	// First name.
	FirstName string `json:"firstname"`
	// Last name.
	LastName string `json:"lastname"`
	// Email address of the contact.
	Email string `json:"email"`
	// Phone number of the contact.
	Phone string `json:"phone"`
}

// TenantStatus is the status for a Tenant resource
type TenantStatus struct {
	// The state can be 'Established' or 'Failure'.
	State string `json:"state"`
	// Additional description can be located here.
	Message string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TenantList is a list of Tenant resources
type TenantList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// TenantList is a list of Tenant resources. This field contains Tenants.
	Items []Tenant `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SubNamespace describes a SubNamespace resource
type SubNamespace struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the subsidiary namespace resource spec
	Spec SubNamespaceSpec `json:"spec"`
	// Status is the subsidiary namespace resource status
	Status SubNamespaceStatus `json:"status,omitempty"`
}

// SubNamespaceSpec is the spec for a SubNamespace resource
type SubNamespaceSpec struct {
	// The mode of subnamespace, Workspace or Subtenant, cannot be changed after creation.
	// Workspace creates a child namespace within the namespace hierarchy, which fulfills
	// the organizational needs.
	Workspace *Workspace `json:"workspace"`
	// Subnamespace creates the subnamespace in form of subtenant, where all
	// information is hidden from it's parent.
	Subtenant *Subtenant `json:"subtenant"`
	// Expiration date of the subnamespace.
	Expiry *metav1.Time `json:"expiry"`
}

// Workspace contains possible resources such as cpu units or memory, which attributes to
// inherit, scope, and owner.
type Workspace struct {
	// Represents maximum resources to be used.
	ResourceAllocation map[corev1.ResourceName]resource.Quantity `json:"resourceallocation"`
	// Which services are going to be inherited from the parent namespace to the this workspace thus
	// subnamespace.
	// The supported resources are: RBAC, NetworkPolicies, Limit Ranges, Secrets, Config Maps, and
	// Service Accounts.
	Inheritance map[string]bool `json:"inheritance"`
	// Scope can be 'federated', or 'local'. It cannot be changed after creation.
	Scope string `json:"scope"`
	// Denote the workspace in sync with its parent.
	Sync bool `json:"sync"`
	// Owner of the workspace.
	Owner *Contact `json:"owner"`
	// SliceClaim is the name of a SliceClaim in the same namespace as the workspace using this slice.
	SliceClaim *string
}

// Subtenant resource represents a tenant under another tenant.
type Subtenant struct {
	// Current allocation of certain resource types. Resource types are
	// kubernetes default resource types.
	ResourceAllocation map[corev1.ResourceName]resource.Quantity `json:"resourceallocation"`
	// Owner of the Subtenant.
	Owner Contact `json:"owner"`
	// SliceClaim is the name of a SliceClaim in the same namespace as the subtenant using this slice.
	SliceClaim *string
}

// SubNamespaceStatus is the status for a SubNamespace resource
type SubNamespaceStatus struct {
	// Denotes the state of the SubNamespace. This can be 'Failure', or 'Established'.
	State string `json:"state"`
	// Message contains additional information.
	Message string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SubNamespaceList is a list of SubNamespace resources
type SubNamespaceList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// SubNamespaceList is a list of SubNamespace resources. This element contains
	// SubNamespace resources.
	Items []SubNamespace `json:"items"`
}

// Retrieves quantity value from given resource name.
func (s SubNamespace) RetrieveQuantityValue(key corev1.ResourceName) int64 {
	// TODO: Remove this function when using int64 is deprecated
	var value int64
	if s.Spec.Workspace != nil {
		if _, elementExists := s.Spec.Workspace.ResourceAllocation[key]; elementExists {
			quantity := s.Spec.Workspace.ResourceAllocation[key]
			value = quantity.Value()
		}
	} else {
		if _, elementExists := s.Spec.Subtenant.ResourceAllocation[key]; elementExists {
			quantity := s.Spec.Subtenant.ResourceAllocation[key]
			value = quantity.Value()
		}
	}

	return value
}

// GenerateChildName forms a name for child according to the mode, Workspace or Subtenant.
func (s SubNamespace) GenerateChildName(clusterUID string) (string, error) {
	childName := s.GetName()
	if s.Spec.Workspace != nil && s.Spec.Workspace.Scope != "local" {
		childName = fmt.Sprintf("%s-%s", clusterUID, childName)
	}

	childNameHashed, err := util.Hash(s.GetNamespace(), childName)
	return childNameHashed, err
}

// GetMode return the mode as workspace or subtenant.
func (s SubNamespace) GetMode() string {
	if s.Spec.Workspace != nil {
		return "workspace"
	} else {
		return "subtenant"
	}
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeContribution describes a NodeContribution resource
type NodeContribution struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the nodecontribution resource spec
	Spec NodeContributionSpec `json:"spec"`
	// Status is the nodecontribution resource status
	Status NodeContributionStatus `json:"status,omitempty"`
}

// NodeContributionSpec is the spec for a NodeContribution resource.
type NodeContributionSpec struct {
	// Tenant resource of the contributor. This is to award the tenant
	// who contributes to the cluster with the node.
	Tenant *string `json:"tenant"`
	// Name of the host.
	Host string `json:"host"`
	// SSH port.
	Port int `json:"port"`
	// SSH username.
	User string `json:"user"`
	// To enable/disable scheduling on the contributed node.
	Enabled bool `json:"enabled"`
	// Each contribution can have none or many limitations. This field denotese these
	// limitations.
	Limitations []Limitations `json:"limitations"`
}

// Limitations describes which tenants and namespaces can make use of node
type Limitations struct {
	// Kind of the limitation.
	Kind string `json:"kind"`
	// Identifier of the limitator.
	Indentifier string `json:"identifier"`
}

// NodeContributionStatus is the status for a node contribution
type NodeContributionStatus struct {
	// This can be 'InQueue', 'Failure', 'Success', 'Incomplete', or 'InProgress'.
	State string `json:"state"`
	// Message contains additional information.
	Message []string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeContributionList is a list of NodeContribution resources
type NodeContributionList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// NodeContributionList is a list of NodeContribution resources. This element contains
	// NodeContribution resources.
	Items []NodeContribution `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TenantResourceQuota describes a tenant resouce quota resource
type TenantResourceQuota struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the tenantresourcequota resource spec
	Spec TenantResourceQuotaSpec `json:"spec"`
	// Status is the tenantresourcequota resource status
	Status TenantResourceQuotaStatus `json:"status,omitempty"`
}

// TenantResourceQuotaSpec is the spec for a tenant resouce quota resource
type TenantResourceQuotaSpec struct {
	// To increase the overall quota.
	Claim map[string]ResourceTuning `json:"claim"`
	// To decrease the overall quota.
	Drop map[string]ResourceTuning `json:"drop"`
}

// ResourceTuning indicates resources to add or remove, and how long they will remain.
// The supported resources are: CPU, Memory, Local Storage, Ephemeral Storage, and
// Bandwidth.
type ResourceTuning struct {
	// This denotes which resources to be included.
	ResourceList map[corev1.ResourceName]resource.Quantity `json:"resourceList"`
	// Expiration date of the ResourceTuning. This can be nil if no expiration date is specified.
	Expiry *metav1.Time `json:"expiry"`
}

// TenantResourceQuotaStatus is the status for a tenant resouce quota resource
type TenantResourceQuotaStatus struct {
	// Denotes the state of the TenantResourceQuota. This can be 'Failure', or 'Success'.
	State string `json:"state"`
	// Message contains additional information.
	Message string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TenantResourceQuotaList is a list of tenant resouce quota resources
type TenantResourceQuotaList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// TenantResourceQuotaList is a list of TenantResourceQuota resources. This element contains
	// TenantResourceQuota resources.
	Items []TenantResourceQuota `json:"items"`
}

// Fetches the net value of the resources. For example, 1Gb memory is claimed and 100 milliCPU
// are dropped. Then the function returns the net resources as '+1Gb', '-100m'.
func (t TenantResourceQuota) Fetch() (map[corev1.ResourceName]int64, map[corev1.ResourceName]resource.Quantity) {
	// TODO: Remove the assignedQuotaValue map
	assignedQuotaValue := make(map[corev1.ResourceName]int64)
	assignedQuota := make(map[corev1.ResourceName]resource.Quantity)

	if len(t.Spec.Claim) > 0 {
		for _, claim := range t.Spec.Claim {
			if claim.Expiry == nil || (claim.Expiry != nil && time.Until(claim.Expiry.Time) >= 0) {
				for key, value := range claim.ResourceList {
					if _, elementExists := assignedQuotaValue[key]; elementExists {
						assignedQuotaValue[key] += value.Value()
						quantity := assignedQuota[key]
						quantity.Add(value)
						assignedQuota[key] = quantity
					} else {
						assignedQuotaValue[key] = value.Value()
						assignedQuota[key] = value
					}
				}
			}
		}
	}
	if len(t.Spec.Drop) > 0 {
		for _, drop := range t.Spec.Drop {
			if drop.Expiry == nil || (drop.Expiry != nil && time.Until(drop.Expiry.Time) >= 0) {
				for key, value := range drop.ResourceList {
					if _, elementExists := assignedQuotaValue[key]; elementExists {
						assignedQuotaValue[key] -= value.Value()
						quantity := assignedQuota[key]
						quantity.Sub(value)
						assignedQuota[key] = quantity
					} else {
						assignedQuotaValue[key] = -value.Value()
						value.Neg()
						assignedQuota[key] = value
					}
				}
			}
		}
	}
	return assignedQuotaValue, assignedQuota
}

// Removes the resource tunings if they are expired.
func (t TenantResourceQuota) DropExpiredItems() bool {
	remove := func(objects ...map[string]ResourceTuning) bool {
		expired := false
		for _, obj := range objects {
			for key, value := range obj {
				if value.Expiry != nil && time.Until(value.Expiry.Time) <= 0 {
					expired = true
					delete(obj, key)
				}
			}
		}
		return expired
	}
	return remove(t.Spec.Claim, t.Spec.Drop)
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Slice describes a slice resource
type Slice struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the slice resource spec
	Spec SliceSpec `json:"spec"`
	// Status is the slice resource status
	Status SliceStatus `json:"status,omitempty"`
}

// SliceSpec is the spec for a slice resource
type SliceSpec struct {
	// Name of the SliceClass required by the claim. This can be 'Node', or 'Resource'.
	SliceClassName string `json:"sliceClassName"`
	// ClaimRef is part of a bi-directional binding between Slice and SliceClaim.
	// Expected to be non-nil when bound.
	ClaimRef *corev1.ObjectReference `json:"claimRef"`
	// A selector for nodes to reserve.
	NodeSelector NodeSelector `json:"nodeSelector"`
}

type NodeSelector struct {
	// A label query over nodes to consider for choosing.
	Selector corev1.NodeSelector `json:"selector"`
	// Number of nodes to pick up for each match case
	Count int `json:"nodeCount"`
	// Resources represents the minimum resources each selected node should have.
	Resources corev1.ResourceRequirements `json:"resources"`
}

// SliceStatus is the status for a slice resource
type SliceStatus struct {
	// Denotes the state of the Slice. This can be 'Failure', or 'Success'.
	State string `json:"state"`
	// Message contains additional information.
	Message string `json:"message"`
	// Expiration date of the slice.
	Expiry *metav1.Time `json:"expiry"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SliceList is a list of slice resources
type SliceList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// SliceList is a list of Slice resources. This element contains
	// Slice resources.
	Items []Slice `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SliceClaim describes a slice claim resource
type SliceClaim struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the slice claim resource spec
	Spec SliceClaimSpec `json:"spec"`
	// Status is the slice claim resource status
	Status SliceClaimStatus `json:"status,omitempty"`
}

// SliceClaimSpec is the spec for a slice claim resource
type SliceClaimSpec struct {
	// Name of the SliceClass required by the claim. This can be 'Node', or 'Resource'.
	SliceClassName string `json:"sliceClassName"`
	// SliceName is the binding reference to the Slice backing this claim.
	SliceName string `json:"sliceName"`
	// A selector for nodes to reserve.
	NodeSelector NodeSelector `json:"nodeSelector"`
	// Expiration date of the slice.
	SliceExpiry *metav1.Time `json:"expiry"`
}

// SliceClaimStatus is the status for a slice claim resource
type SliceClaimStatus struct {
	// Denotes the state of the SliceClaim. This can be 'Failure', or 'Success'.
	State string `json:"state"`
	// Message contains additional information.
	Message string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SliceClaimList is a list of slice claim resources
type SliceClaimList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// SliceClaimList is a list of SliceClaim resources. This element contains
	// SliceClaim resources.
	Items []SliceClaim `json:"items"`
}

func (sc SliceClaim) GetObjectReference() *corev1.ObjectReference {
	objectReference := corev1.ObjectReference{}
	objectReference.APIVersion = sc.APIVersion
	objectReference.Kind = sc.Kind
	objectReference.Name = sc.GetName()
	objectReference.Namespace = sc.GetNamespace()
	objectReference.UID = sc.GetUID()
	return &objectReference
}
