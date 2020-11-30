/*
Copyright 2020 Sorbonne Université

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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SelectiveDeployment describes a SelectiveDeployment resource
type SelectiveDeployment struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the selectivedeployment resource spec
	Spec SelectiveDeploymentSpec `json:"spec"`
	// Status is the selectivedeployment resource status
	Status SelectiveDeploymentStatus `json:"status,omitempty"`
}

// SelectiveDeploymentSpec is the spec for a SelectiveDeployment resource
type SelectiveDeploymentSpec struct {
	// The controller indicates the name and type of controller desired to configure
	// Workloads: deployment, daemonset, and statefulsets
	// The type is for defining which kind of selectivedeployment it is, you could find the list of active types below.
	// Types of selector: city, state, country, continent, and polygon
	// The value represents the desired filter and it must be compatible with the type of selectivedeployment
	Workloads Workloads  `json:"workloads"`
	Selector  []Selector `json:"selector"`
}

// Workloads indicates deployments, daemonsets or statefulsets
type Workloads struct {
	Deployment  []appsv1.Deployment  `json:"deployment"`
	DaemonSet   []appsv1.DaemonSet   `json:"daemonset"`
	StatefulSet []appsv1.StatefulSet `json:"statefulset"`
}

// Selector to define desired node filtering parameters
type Selector struct {
	Name     string                      `json:"name"`
	Value    []string                    `json:"value"`
	Operator corev1.NodeSelectorOperator `json:"operator"`
	Quantity int                         `json:"quantity"`
}

// SelectiveDeploymentStatus is the status for a SelectiveDeployment resource
type SelectiveDeploymentStatus struct {
	Ready   string   `json:"ready"`
	State   string   `json:"state"`
	Message []string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SelectiveDeploymentList is a list of SelectiveDeployment resources
type SelectiveDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []SelectiveDeployment `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Authority describes a Authority resource
type Authority struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the authority resource spec
	Spec AuthoritySpec `json:"spec"`
	// Status is the authority resource status
	Status AuthorityStatus `json:"status,omitempty"`
}

// AuthoritySpec is the spec for a Authority resource
type AuthoritySpec struct {
	FullName  string  `json:"fullname"`
	ShortName string  `json:"shortname"`
	URL       string  `json:"url"`
	Address   Address `json:"address"`
	Contact   Contact `json:"contact"`
	Enabled   bool    `json:"enabled"`
}

// Contact
type Contact struct {
	Username  string `json:"username"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

// Address
type Address struct {
	Street  string `json:"street"`
	ZIP     string `json:"zip"`
	City    string `json:"city"`
	Region  string `json:"region"`
	Country string `json:"country"`
}

// AuthorityStatus is the status for a Authority resource
type AuthorityStatus struct {
	State   string   `json:"state"`
	Message []string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AuthorityList is a list of Authority resources
type AuthorityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Authority `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AuthorityRequest describes a AuthorityRequest resource
type AuthorityRequest struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the authorityrequest resource spec
	Spec AuthorityRequestSpec `json:"spec"`
	// Status is the authorityrequest resource status
	Status AuthorityRequestStatus `json:"status,omitempty"`
}

// AuthorityRequestSpec is the spec for a AuthorityRequest resource
type AuthorityRequestSpec struct {
	FullName  string  `json:"fullname"`
	ShortName string  `json:"shortname"`
	URL       string  `json:"url"`
	Address   Address `json:"address"`
	Contact   Contact `json:"contact"`
	Approved  bool    `json:"approved"`
}

// AuthorityRequestStatus is the status for a AuthorityRequest resource
type AuthorityRequestStatus struct {
	EmailVerified bool         `json:"emailverified"`
	Expires       *metav1.Time `json:"expires"`
	State         string       `json:"state"`
	Message       []string     `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AuthorityRequestList is a list of AuthorityRequest resources
type AuthorityRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AuthorityRequest `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Team describes a Team resource
type Team struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the team resource spec
	Spec TeamSpec `json:"spec"`
	// Status is the team resource status
	Status TeamStatus `json:"status,omitempty"`
}

// TeamSpec is the spec for a Team resource
type TeamSpec struct {
	Users       []TeamUsers `json:"users"`
	Description string      `json:"description"`
	Enabled     bool        `json:"enabled"`
}

type TeamUsers struct {
	Authority string `json:"authority"`
	Username  string `json:"username"`
}

// TeamStatus is the status for a Team resource
type TeamStatus struct {
	State   string   `json:"state"`
	Message []string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TeamList is a list of Team resources
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Team `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Slice describes a Slice resource
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

// SliceSpec is the spec for a Slice resource
type SliceSpec struct {
	Type        string       `json:"type"`
	Profile     string       `json:"profile"`
	Users       []SliceUsers `json:"users"`
	Description string       `json:"description"`
	Renew       bool         `json:"renew"`
}

type SliceUsers struct {
	Authority string `json:"authority"`
	Username  string `json:"username"`
}

// SliceStatus is the status for a Slice resource
type SliceStatus struct {
	Expires *metav1.Time `json:"expires"`
	State   string       `json:"state"`
	Message []string     `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SliceList is a list of Slice resources
type SliceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Slice `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// User describes a User resource
type User struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the user resource spec
	Spec UserSpec `json:"spec"`
	// Status is the user resource status
	Status UserStatus `json:"status,omitempty"`
}

// UserSpec is the spec for a User resource
type UserSpec struct {
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Email     string `json:"email"`
	URL       string `json:"url"`
	Bio       string `json:"bio"`
	Active    bool   `json:"active"`
}

// UserStatus is the status for a User resource
type UserStatus struct {
	Type    string   `json:"type"`
	AUP     bool     `json:"aup"`
	State   string   `json:"state"`
	Message []string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UserList is a list of User resources
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []User `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UserRegistrationRequest describes a UserRegistrationRequest resource
type UserRegistrationRequest struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the userregistrationrequest resource spec
	Spec UserRegistrationRequestSpec `json:"spec"`
	// Status is the userregistrationrequest resource status
	Status UserRegistrationRequestStatus `json:"status,omitempty"`
}

// UserRegistrationRequestSpec is the spec for a UserRegistrationRequest resource
type UserRegistrationRequestSpec struct {
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Email     string `json:"email"`
	URL       string `json:"url"`
	Bio       string `json:"bio"`
	Approved  bool   `json:"approved"`
}

// UserRegistrationRequestStatus is the status for a UserRegistrationRequest resource
type UserRegistrationRequestStatus struct {
	EmailVerified bool         `json:"emailverified"`
	Expires       *metav1.Time `json:"expires"`
	State         string       `json:"state"`
	Message       []string     `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UserRegistrationRequestList is a list of UserRegistrationRequest resources
type UserRegistrationRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []UserRegistrationRequest `json:"items"`
}

// +genclient
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
	Accepted bool `json:"accepted"`
	Renew    bool `json:"renew"`
}

// AcceptableUsePolicyStatus is the status for a AcceptableUsePolicy resource
type AcceptableUsePolicyStatus struct {
	Expires *metav1.Time `json:"expires"`
	State   string       `json:"state"`
	Message []string     `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AcceptableUsePolicyList is a list of AcceptableUsePolicy resources
type AcceptableUsePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AcceptableUsePolicy `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EmailVerification describes a EmailVerification resource
type EmailVerification struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the emailverification resource spec
	Spec EmailVerificationSpec `json:"spec"`
	// Status is the emailverification resource status
	Status EmailVerificationStatus `json:"status,omitempty"`
}

// EmailVerificationSpec is the spec for a EmailVerification resource
type EmailVerificationSpec struct {
	Kind       string `json:"kind"`
	Identifier string `json:"identifier"`
	Verified   bool   `json:"verified"`
}

// EmailVerificationStatus is the status for a EmailVerification resource
type EmailVerificationStatus struct {
	Expires *metav1.Time `json:"expires"`
	State   string       `json:"state"`
	Message []string     `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EmailVerificationList is a list of EmailVerification resources
type EmailVerificationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EmailVerification `json:"items"`
}

// +genclient
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
	Host        string        `json:"host"`
	Port        int           `json:"port"`
	User        string        `json:"user"`
	Password    string        `json:"password"`
	Enabled     bool          `json:"enabled"`
	Limitations []Limitations `json:"limitations"`
}

type Limitations struct {
	Authority string `json:"authority"`
	Team      string `json:"team"`
	Slice     string `json:"slice"`
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

// TotalResourceQuota describes a total resouce quota resource
type TotalResourceQuota struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the totalresourcequota resource spec
	Spec TotalResourceQuotaSpec `json:"spec"`
	// Status is the totalresourcequota resource status
	Status TotalResourceQuotaStatus `json:"status,omitempty"`
}

// TotalResourceQuotaSpec is the spec for a total resouce quota resource
type TotalResourceQuotaSpec struct {
	Claim   []TotalResourceDetails `json:"claim"`
	Drop    []TotalResourceDetails `json:"drop"`
	Enabled bool                   `json:"enabled"`
}

// TotalResourceDetails indicates resources to add or remove, and how long they will remain
type TotalResourceDetails struct {
	Name    string       `json:"name"`
	CPU     string       `json:"cpu"`
	Memory  string       `json:"memory"`
	Expires *metav1.Time `json:"expires"`
}

// TotalResourceQuotaStatus is the status for a total resouce quota resource
type TotalResourceQuotaStatus struct {
	Exceeded bool              `json:"exceeded"`
	Used     TotalResourceUsed `json:"used"`
	State    string            `json:"state"`
	Message  []string          `json:"message"`
}

// TotalResourceUsed presents the usage of total resource quota
type TotalResourceUsed struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TotalResourceQuotaList is a list of total resouce quota resources
type TotalResourceQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []TotalResourceQuota `json:"items"`
}
