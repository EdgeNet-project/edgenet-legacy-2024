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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Values of Status.State
const (
	StatusFailed         = "Failure"
	StatusReconciliation = "Reconciliation"
	// Cluster
	StatusCredsPrepared       = "Auth Credentials Prepared"
	StatusSubnamespaceCreated = "Subnamespace Created"
	StatusReady               = "Ready"
	// Selective Deployment Anchor
	StatusAssigned         = "A Federation Manager Assigned"
	StatusDelegated        = "Selective Deployment Delegated"
	StatusPendingScheduler = "Pending Scheduler"
	// Manager Cache
	StatusPending = "Pending Workload Cluster Creation"
	StatusUpdated = "Remote Manager Cache Updated"
)

// Values of string constants subject to repetitive use
const (
	RemoteClusterRole          = "edgenet:federation:remotecluster"
	FederationManagerNamespace = "federated-%s"
	AbundantResources          = "Abundance"
	NormalResources            = "Normal"
	LimitedResources           = "Limited"
	ScarceResources            = "Scarcity"
	FederationManagerRole      = "Manager"
	WorkloadRole               = "Workload"
	PeerRole                   = "Peer"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster describes a cluster that is part of the federation as a workload cluster or a federation manager
type Cluster struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the cluster resource spec
	Spec ClusterSpec `json:"spec"`
	// Status is the cluster resource status
	Status ClusterStatus `json:"status,omitempty"`
}

// ClusterSpec is the spec to define the desired state of the cluster resource
type ClusterSpec struct {
	// UID is the unique identifier of the cluster
	UID string `json:"uid"`
	// Role can be 'Workload', 'Manager', or 'Peer'
	Role string `json:"role"`
	// Server is the API server of the cluster
	Server string `json:"server"`
	// Preferences is to empower resource owners to set allowlist and denylist
	Preferences ClusterPreferences `json:"preferences"`
	// Visibility can be 'Public' or 'Private'
	Visibility string `json:"visibility"`
	// SecretName is the name of the secret that contains the token to access the cluster
	SecretName string `json:"secretName"`
	// Enabled is to open or close the cluster to the federation
	Enabled bool `json:"enabled"`
}

// ClusterPreferences is to set allowlist and denylist for federated objects
type ClusterPreferences struct {
	// Allowlist is the selector to target operators, clusters, tenants, and workloads that are allowed to be federated
	Allowlist *metav1.LabelSelector `json:"allowlist,omitempty"`
	// Denylist is the selector to target operators, clusters, tenants, and workloads that are allowed to be federated
	Denylist *metav1.LabelSelector `json:"denylist,omitempty"`
}

// ClusterSpec is the status that shows the actual state of the cluster resource
type ClusterStatus struct {
	// The state can be 'Established' or 'Failure'.
	State string `json:"state"`
	// Additional description can be located here.
	Message string `json:"message"`
	// RelativeResourceAvailability indicates the status of available resources in the cluster
	RelativeResourceAvailability string `json:"relativeResourceAvailability"`
	// AllocatableResources is the list of grouped allocatable resources in the cluster
	AllocatableResources []BundledAllocatableResources `json:"allocatableResources"`
	// Failed sets the backoff limit.
	Failed int `json:"failed"`
	// UpdateTimestamp is the last time the status was updated.
	UpdateTimestamp *metav1.Time `json:"updateTimestamp"`
}

// BundledAllocatableResources is the struct to bundle the allocatable resources
type BundledAllocatableResources struct {
	Count        int                 `json:"count"`
	ResourceList corev1.ResourceList `json:"resourceList"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList is a list of cluster resources
type ClusterList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// ClusterList is a list of cluster resources. This element contains
	// cluster resources.
	Items []Cluster `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SelectiveDeploymentAnchor is the resource to make scheduling decisions and object propagation at the federation manager level
type SelectiveDeploymentAnchor struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the selectivedeploymentanchor resource spec
	Spec SelectiveDeploymentAnchorSpec `json:"spec"`
	// Status is the selectivedeploymentanchor resource status
	Status SelectiveDeploymentAnchorStatus `json:"status,omitempty"`
}

// SelectiveDeploymentAnchorSpec is the spec to define the desired state of the selectivedeploymentanchor resource
type SelectiveDeploymentAnchorSpec struct {
	// OriginRef is the reference to the original selective deployment
	OriginRef OriginReference `json:"originRef"`
	// ClusterAffinity is the selector to target clusters that match the cluster affinity
	ClusterAffinity *metav1.LabelSelector `json:"clusterAffinity,omitempty"`
	// ClusterReplicas is to pick up defined number of clusters that match the cluster affinity
	ClusterReplicas int `json:"clusterReplicas,omitempty"`
	// WorkloadClusters is the list of workload clusters that match the cluster affinity
	WorkloadClusters []string `json:"workloadClusters,omitempty"`
	// FederationManager is the federation manager that is responsible for the selective deployment
	FederationManager *SelectedFederationManager `json:"federationManagers,omitempty"`
	// FederationUID is the unique identifier of the federation that the selected federation manager belongs to
	FederationUID *string `json:"federationUID"`
	// SecretName is the name of the secret that contains the token to access the original selective deployment
	SecretName string `json:"secretName"`
}

type SelectedFederationManager struct {
	// Name is the UID of the federation manager
	Name string `json:"name"`
	// Path is the shortest path to the federation manager
	Path []string `json:"path"`
}

// OriginReference is the reference to the original selective deployment
type OriginReference struct {
	// UID is the unique identifier of the selective deployment
	UID string `json:"uid"`
	// Namespace is the namespace of the selective deployment
	Namespace string `json:"namespace"`
	// Name is the name of the selective deployment
	Name string `json:"name"`
}

// SelectiveDeploymentAnchorStatus is the status that shows the actual state of the selectivedeploymentanchor resource
type SelectiveDeploymentAnchorStatus struct {
	// The state can be 'Established' or 'Failure'.
	State string `json:"state"`
	// Additional description can be located here.
	Message string `json:"message"`
	// Failed sets the backoff limit.
	Failed int `json:"failed"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SelectiveDeploymentAnchorList is a list of selectivedeploymentanchor resources
type SelectiveDeploymentAnchorList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// SelectiveDeploymentAnchorList is a list of selectivedeploymentanchor resources.
	// This element contains selectivedeploymentanchor resources.
	Items []SelectiveDeploymentAnchor `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ManagerCache is to cache federation managers for scheduling decisions
type ManagerCache struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the managercache resource spec
	Spec ManagerCacheSpec `json:"spec"`
	// Status is the managercache resource status
	Status ManagerCacheStatus `json:"status,omitempty"`
}

// ManagerCacheSpec is the spec to define the desired state of the managercache resource
type ManagerCacheSpec struct {
	// FederationUID is the UID of the federation that the federation manager belongs to
	FederationUID string `json:"federationUID"`
	// Hierarchical information related to the federation manager
	Hierarchy *Hierarchy `json:"hierarchy"`
	// Clusters form a list of workload clusters that are managed by the federation manager
	Clusters map[string]ClusterCache `json:"clusters"`
	// LatestUpdateTimestamp is the last time the managercache resource was updated by its federation manager
	LatestUpdateTimestamp *metav1.Time `json:"latestUpdateTimestamp"`
	// Enabled indicates whether the federation manager is open to the federation or not
	Enabled bool `json:"enabled"`
}

// Hierarchy is to trace the federation manager's position in the hierarchy
type Hierarchy struct {
	// Level is the hierarchy level of the federation manager
	Level int `json:"level"`
	// Parent is the info of the federation manager's parent
	Parent *AssociatedManager `json:"parent"`
	// Children is the info of the federation manager's children
	Children []AssociatedManager `json:"children"`
}

// AssociatedManagers are the parent and children of the federation manager
type AssociatedManager struct {
	// Name is the UID of the federation manager
	Name string `json:"name"`
	// Enabled indicates whether the federation manager is open to the federation or not
	Enabled bool `json:"enabled"`
}

// ClusterCache is to cache workload cluster information for scheduling decisions
type ClusterCache struct {
	// Characteristics is the list of characteristics of the cluster such as GPU cluster, camera cluster, etc.
	Characteristics map[string]string `json:"characteristics"`
	// RelativeResourceAvailability indicates the status of available resources in the cluster
	RelativeResourceAvailability string `json:"relativeResourceAvailability"`
	// AllocatableResources is the list of grouped allocatable resources in the cluster
	AllocatableResources []BundledAllocatableResources `json:"allocatableResources"`
	// Enabled indicates whether the cluster is open to the federation or not
	Enabled bool `json:"enabled"`
}

// ManagerCacheStatus is the status that shows the actual state of the managercache resource
type ManagerCacheStatus struct {
	// The state can be 'Established' or 'Failure'.
	State string `json:"state"`
	// Additional description can be located here.
	Message string `json:"message"`
	// Failed sets the backoff limit.
	Failed int `json:"failed"`
	// UpdateTimestamp is the last time the status was updated.
	UpdateTimestamp *metav1.Time `json:"updateTimestamp"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ManagerCacheList is a list of managercache resources
type ManagerCacheList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// ManagerCacheList is a list of managercache resources.
	// This element contains managercache resources.
	Items []ManagerCache `json:"items"`
}
