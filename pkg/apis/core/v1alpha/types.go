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
	"time"

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
	FullName             string  `json:"fullname"`
	ShortName            string  `json:"shortname"`
	URL                  string  `json:"url"`
	Address              Address `json:"address"`
	Contact              Contact `json:"contact"`
	ClusterNetworkPolicy bool    `json:"clusternetworkpolicy"`
	Enabled              bool    `json:"enabled"`
}

// Address describes postal address of tenant
type Address struct {
	Street  string `json:"street"`
	ZIP     string `json:"zip"`
	City    string `json:"city"`
	Region  string `json:"region"`
	Country string `json:"country"`
}

// Contact contains handle, personal information, and role
type Contact struct {
	Handle    string `json:"handle"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

// TenantStatus is the status for a Tenant resource
type TenantStatus struct {
	PolicyAgreed map[string]bool `json:"policyagreed"`
	State        string          `json:"state"`
	Message      string          `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TenantList is a list of Tenant resources
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

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
// RBAC, NetworkPolicies, Limit Ranges, Secrets, Config Maps, Service Accounts
type SubNamespaceSpec struct {
	Workspace *Workspace   `json:"workspace"`
	Subtenant *Subtenant   `json:"subtenant"`
	Expiry    *metav1.Time `json:"expiry"`
}

type Workspace struct {
	ResourceAllocation map[corev1.ResourceName]resource.Quantity `json:"resourceallocation"`
	Inheritance        map[string]bool                           `json:"inheritance"`
	Scope              string                                    `json:"scope"`
	Sync               *bool                                     `json:"sync"`
	Owner              *Contact                                  `json:"owner"`
}

type Subtenant struct {
	ResourceAllocation map[corev1.ResourceName]resource.Quantity `json:"resourceallocation"`
	Owner              Contact                                   `json:"owner"`
}

// SubNamespaceStatus is the status for a SubNamespace resource
type SubNamespaceStatus struct {
	State   string `json:"state"`
	Message string `json:"message"`
	Child   *Child `json:"child"`
}

type Child struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SubNamespaceList is a list of SubNamespace resources
type SubNamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []SubNamespace `json:"items"`
}

// TODO: Remove this function when using int64 is deprecated
func (s SubNamespace) RetrieveQuantityValue(key corev1.ResourceName) int64 {
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

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AcceptableUsePolicy describes a AcceptableUsePolicy resource
type AcceptableUsePolicy struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the acceptableusepolicy resource spec
	Spec AcceptableUsePolicySpec `json:"spec"`
	// Status is the acceptableusepolicy resource status
	Status AcceptableUsePolicyStatus `json:"status,omitempty"`
}

// AcceptableUsePolicySpec is the spec for a AcceptableUsePolicy resource
type AcceptableUsePolicySpec struct {
	Email    string `json:"email"`
	Accepted bool   `json:"accepted"`
}

// AcceptableUsePolicyStatus is the status for a AcceptableUsePolicy resource
type AcceptableUsePolicyStatus struct {
	State   string `json:"state"`
	Message string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AcceptableUsePolicyList is a list of AcceptableUsePolicy resources
type AcceptableUsePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AcceptableUsePolicy `json:"items"`
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

// NodeContributionSpec is the spec for a NodeContribution resource
type NodeContributionSpec struct {
	Tenant      *string       `json:"tenant"`
	Host        string        `json:"host"`
	Port        int           `json:"port"`
	User        string        `json:"user"`
	Enabled     bool          `json:"enabled"`
	Limitations []Limitations `json:"limitations"`
}

// Limitations describes which tenants and namespaces can make use of node
type Limitations struct {
	Kind        string `json:"kind"`
	Indentifier string `json:"identifier"`
}

// NodeContributionStatus is the status for a node contribution
type NodeContributionStatus struct {
	State   string   `json:"state"`
	Message []string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeContributionList is a list of NodeContribution resources
type NodeContributionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

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
	Claim map[string]ResourceTuning `json:"claim"`
	Drop  map[string]ResourceTuning `json:"drop"`
}

// ResourceTuning indicates resources to add or remove, and how long they will remain
// CPU, Memory, Local Storage, Ephemeral Storage, and Bandwidth
type ResourceTuning struct {
	ResourceList map[corev1.ResourceName]resource.Quantity `json:"resourceList"`
	Expiry       *metav1.Time                              `json:"expiry"`
}

// TenantResourceQuotaStatus is the status for a tenant resouce quota resource
type TenantResourceQuotaStatus struct {
	State   string `json:"state"`
	Message string `json:"message"`
}

// Resources presents the usage of tenant resource quota
type Resources struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TenantResourceQuotaList is a list of tenant resouce quota resources
type TenantResourceQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []TenantResourceQuota `json:"items"`
}

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
