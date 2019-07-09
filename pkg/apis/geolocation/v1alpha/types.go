package v1alpha

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GeoLocation describes a GeoLocation resource
type GeoLocation struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	meta_v1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the geolocation resource spec
	Spec GeoLocationSpec `json:"spec"`
}

// GeoLocationSpec is the spec for a GeoLocation resource
type GeoLocationSpec struct {
	// The deployment indicates the names of deployments desired to configure
	// The type is for defining which kind of geolocation it is, you could find the list of active types below:
	// city | country | continent | polygon
	// The value represents the desired filter and it must be compatible with the type of geolocation
	Deployment []string `json:"deployment"`
	Type       string   `json:"type"`
	Value      []string `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GeoLocationList is a list of GeoLocation resources
type GeoLocationList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []GeoLocation `json:"items"`
}
