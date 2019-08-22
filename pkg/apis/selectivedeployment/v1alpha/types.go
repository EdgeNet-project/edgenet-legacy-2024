package v1alpha

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SelectiveDeployment describes a SelectiveDeployment resource
type SelectiveDeployment struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	meta_v1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the selectivedeployment resource spec
	Spec SelectiveDeploymentSpec `json:"spec"`
	// Status is the selectivedeployment resource status
	Status SelectiveDeploymentStatus `json:"status,omitempty"`
}

// SelectiveDeploymentSpec is the spec for a SelectiveDeployment resource
type SelectiveDeploymentSpec struct {
	// The controller indicates the name and type of controller desired to configure
	// Controllers: deployment, daemonset, and statefulsets
	// The type is for defining which kind of selectivedeployment it is, you could find the list of active types below.
	// Types: city, state, country, continent, and polygon
	// The value represents the desired filter and it must be compatible with the type of selectivedeployment
	Controller [][]string `json:"controller"`
	Type       string     `json:"type"`
	Value      [][]string `json:"value"`
}

// SelectiveDeploymentStatus is the status for a SelectiveDeployment resource
type SelectiveDeploymentStatus struct {
	Ready   string     `json:"ready"`
	State   string     `json:"state"`
	Message string     `json:"message"`
	Reason  [][]string `json:"reason"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SelectiveDeploymentList is a list of SelectiveDeployment resources
type SelectiveDeploymentList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []SelectiveDeployment `json:"items"`
}
