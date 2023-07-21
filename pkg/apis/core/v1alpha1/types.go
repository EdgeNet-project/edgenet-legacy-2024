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

package v1alpha1

import (
	"fmt"
	"hash/adler32"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Values of Status.State
const (
	StatusFailed         = "Failure"
	StatusReconciliation = "Reconciliation"
	// Slice claim
	StatusPending   = "Pending"
	StatusRequested = "Requested"
	StatusEmployed  = "Employed"
	// Slice
	StatusBound       = "Bound" // Also used for slice claim
	StatusReserved    = "Reserved"
	StatusProvisioned = "Provisioned"
	// Subnamespace
	StatusPartitioned         = "Partitioned"
	StatusSubnamespaceCreated = "Created"
	StatusQuotaSet            = "Set"
	// Tenant
	StatusCoreNamespaceCreated = "Created"
	StatusEstablished          = "Established" // Also used for subnamespace
	// Tenant resource quota
	StatusQuotaCreated = "Created"
	StatusApplied      = "Applied"
	// Node contribution
	StatusAccessed = "Node Accessed"
	StatusReady    = "Ready"
)

// Values of string constants subject to repetitive use
const (
	DynamicStr                        = "Dynamic"
	TenantOwnerClusterRoleName        = "edgenet:tenant-owner"
	TenantAdminClusterRoleName        = "edgenet:tenant-admin"
	TenantCollaboratorClusterRoleName = "edgenet:tenant-collaborator"
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
	// Description provides additional information about the tenant.
	Description string `json:"description"`
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
	// Failed sets the backoff limit.
	Failed int `json:"failed"`
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

// MakeOwnerReference creates an owner reference for the given object.
func (t Tenant) MakeOwnerReference() metav1.OwnerReference {
	return *metav1.NewControllerRef(&t.ObjectMeta, SchemeGroupVersion.WithKind("Tenant"))
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
	SliceClaim *string `json:"sliceclaim"`
}

// Subtenant resource represents a tenant under another tenant.
type Subtenant struct {
	// Current allocation of certain resource types. Resource types are
	// kubernetes default resource types.
	ResourceAllocation map[corev1.ResourceName]resource.Quantity `json:"resourceallocation"`
	// Owner of the Subtenant.
	Owner Contact `json:"owner"`
	// SliceClaim is the name of a SliceClaim in the same namespace as the subtenant using this slice.
	SliceClaim *string `json:"sliceclaim"`
}

// SubNamespaceStatus is the status for a SubNamespace resource
type SubNamespaceStatus struct {
	// Denotes the state of the SubNamespace. This can be 'Failure', or 'Established'.
	State string `json:"state"`
	// Message contains additional information.
	Message string `json:"message"`
	// Failed sets the backoff limit.
	Failed int `json:"failed"`
	// Child is the name of the child namespace.
	Child *string `json:"child"`
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

// MakeOwnerReference creates an owner reference for the given object.
func (sn SubNamespace) MakeOwnerReference() metav1.OwnerReference {
	return *metav1.NewControllerRef(&sn.ObjectMeta, SchemeGroupVersion.WithKind("SubNamespace"))
}

// RetrieveQuantity gets quantity value from given resource name.
func (sn SubNamespace) RetrieveQuantity(key corev1.ResourceName) resource.Quantity {
	var quantity resource.Quantity
	if sn.Spec.Workspace != nil {
		if _, elementExists := sn.Spec.Workspace.ResourceAllocation[key]; elementExists {
			quantity = sn.Spec.Workspace.ResourceAllocation[key]
		}
	} else {
		if _, elementExists := sn.Spec.Subtenant.ResourceAllocation[key]; elementExists {
			quantity = sn.Spec.Subtenant.ResourceAllocation[key]
		}
	}
	return quantity
}

// GenerateChildName forms a name for child according to the mode, Workspace or Subtenant.
func (sn SubNamespace) GenerateChildName(clusterUID string) string {
	childName := sn.GetName()
	if sn.Spec.Workspace != nil && sn.Spec.Workspace.Scope == "federation" {
		childName = fmt.Sprintf("%s-%s", clusterUID, childName)
	}

	childNameHashed := hash(sn.GetNamespace(), childName)
	childName = strings.Join([]string{childName, childNameHashed}, "-")
	return childName
}

// GetMode return the mode as workspace or subtenant.
func (sn SubNamespace) GetMode() string {
	if sn.Spec.Workspace != nil {
		return "workspace"
	}
	return "subtenant"
}

// GetResourceAllocation return the allocated resources at workspace or subtenant.
func (sn SubNamespace) GetResourceAllocation() map[corev1.ResourceName]resource.Quantity {
	if sn.Spec.Workspace != nil {
		return sn.Spec.Workspace.DeepCopy().ResourceAllocation
	}
	return sn.Spec.Subtenant.DeepCopy().ResourceAllocation
}

// SetResourceAllocation set the allocated resources at workspace or subtenant.
func (sn SubNamespace) SetResourceAllocation(resource map[corev1.ResourceName]resource.Quantity) {
	if sn.Spec.Workspace != nil {
		sn.Spec.Workspace.ResourceAllocation = resource
	} else {
		sn.Spec.Subtenant.ResourceAllocation = resource
	}
}

// GetSliceClaim return the assigned slice claim at workspace or subtenant.
func (sn SubNamespace) GetSliceClaim() *string {
	if sn.Spec.Workspace != nil {
		return sn.Spec.Workspace.SliceClaim
	}
	return sn.Spec.Subtenant.SliceClaim
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
	Message string `json:"message"`
	// Failed sets the backoff limit.
	Failed int `json:"failed"`
	// UpdateTimestamp is the last time the status was updated.
	UpdateTimestamp *metav1.Time `json:"updateTimestamp"`
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

// MakeOwnerReference creates an owner reference for the given object.
func (nc NodeContribution) MakeOwnerReference() metav1.OwnerReference {
	return *metav1.NewControllerRef(&nc.ObjectMeta, SchemeGroupVersion.WithKind("NodeContribution"))
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
	ResourceList map[corev1.ResourceName]resource.Quantity `json:"resourcelist"`
	// Expiration date of the ResourceTuning. This can be nil if no expiration date is specified.
	Expiry *metav1.Time `json:"expiry"`
}

// TenantResourceQuotaStatus is the status for a tenant resouce quota resource
type TenantResourceQuotaStatus struct {
	// Denotes the state of the TenantResourceQuota. This can be 'Failure', or 'Success'.
	State string `json:"state"`
	// Message contains additional information.
	Message string `json:"message"`
	// Failed sets the backoff limit.
	Failed int `json:"failed"`
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

// Fetch as its name indicates, it fetches the net value of the resources. For example,
// 1Gb memory is claimed and 100 milliCPU are dropped. Then the function returns the net resources as '+1Gb', '-100m'.
func (t TenantResourceQuota) Fetch() map[corev1.ResourceName]resource.Quantity {
	assignedQuota := make(map[corev1.ResourceName]resource.Quantity)
	if len(t.Spec.Claim) > 0 {
		for _, claim := range t.Spec.Claim {
			if claim.Expiry == nil || (claim.Expiry != nil && time.Until(claim.Expiry.Time) >= 0) {
				for key, value := range claim.ResourceList {
					if assignedQuantity, elementExists := assignedQuota[key]; elementExists {
						assignedQuantity.Add(value)
						assignedQuota[key] = assignedQuantity
					} else {
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
					if assignedQuantity, elementExists := assignedQuota[key]; elementExists {
						assignedQuantity.Sub(value)
						assignedQuota[key] = assignedQuantity
					} else {
						value.Neg()
						assignedQuota[key] = value
					}
				}
			}
		}
	}
	return assignedQuota
}

// DropExpiredItems removes the resource tunings if they are expired.
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
	SliceClassName string `json:"sliceclassname"`
	// ClaimRef is part of a bi-directional binding between Slice and SliceClaim.
	// Expected to be non-nil when bound.
	ClaimRef *corev1.ObjectReference `json:"claimref"`
	// A selector for nodes to reserve.
	NodeSelector NodeSelector `json:"nodeselector"`
}

// NodeSelector is a selector for nodes to reserve.
type NodeSelector struct {
	// A label query over nodes to consider for choosing.
	Selector corev1.NodeSelector `json:"selector"`
	// Number of nodes to pick up for each match case
	Count int `json:"nodecount"`
	// Resources represents the minimum resources each selected node should have.
	Resources corev1.ResourceRequirements `json:"resources"`
}

// SliceStatus is the status for a slice resource
type SliceStatus struct {
	// Denotes the state of the Slice. This can be 'Failure', or 'Success'.
	State string `json:"state"`
	// Message contains additional information.
	Message string `json:"message"`
	// Failed sets the backoff limit.
	Failed int `json:"failed"`
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

// MakeOwnerReference creates an owner reference for the given object.
func (s Slice) MakeOwnerReference() metav1.OwnerReference {
	return *metav1.NewControllerRef(&s.ObjectMeta, SchemeGroupVersion.WithKind("Slice"))
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
	SliceClassName string `json:"sliceclassname"`
	// SliceName is the binding reference to the Slice backing this claim.
	SliceName string `json:"slicename"`
	// A selector for nodes to reserve.
	NodeSelector NodeSelector `json:"nodeselector"`
	// Expiration date of the slice.
	SliceExpiry *metav1.Time `json:"expiry"`
}

// SliceClaimStatus is the status for a slice claim resource
type SliceClaimStatus struct {
	// Denotes the state of the SliceClaim. This can be 'Failure', or 'Success'.
	State string `json:"state"`
	// Message contains additional information.
	Message string `json:"message"`
	// Failed sets the backoff limit.
	Failed int `json:"failed"`
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

// MakeObjectReference creates an object reference for the given object.
func (sc SliceClaim) MakeObjectReference() *corev1.ObjectReference {
	objectReference := corev1.ObjectReference{}
	groupVersionKind := SchemeGroupVersion.WithKind("SliceClaim")
	objectReference.APIVersion = groupVersionKind.GroupVersion().String()
	objectReference.Kind = groupVersionKind.Kind
	objectReference.Name = sc.GetName()
	objectReference.Namespace = sc.GetNamespace()
	objectReference.UID = sc.GetUID()
	return &objectReference
}

// MakeOwnerReference creates an owner reference for the given object.
func (sc SliceClaim) MakeOwnerReference() metav1.OwnerReference {
	return *metav1.NewControllerRef(&sc.ObjectMeta, SchemeGroupVersion.WithKind("SliceClaim"))
}

// hash returns a hash of the strings passed in.
func hash(strs ...string) string {
	str := strings.Join(strs, "-")
	adler32 := adler32.Checksum([]byte(str))
	return fmt.Sprintf("%x", adler32)
}
