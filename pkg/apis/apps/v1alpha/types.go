/*
Copyright 2019 Sorbonne Université

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
	Controller []Controller `json:"controller"`
	Type       string       `json:"type"`
	Selector   []Selector   `json:"selector"`
}

// Controller indicates deployment, daemonset or statefulsets and their names
type Controller struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// Selector to define desired node filtering parameters
type Selector struct {
	Value    string `json:"value"`
	Operator string `json:"operator"`
	Count    int    `json:"count"`
}

// SelectiveDeploymentStatus is the status for a SelectiveDeployment resource
type SelectiveDeploymentStatus struct {
	Ready   string  `json:"ready"`
	State   string  `json:"state"`
	Message string  `json:"message"`
	Crash   []Crash `json:"crash"`
}

// Crash is the list of controllers that the object cannot take them under control
type Crash struct {
	Controller Controller `json:"controller"`
	Reason     string     `json:"reason"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SelectiveDeploymentList is a list of SelectiveDeployment resources
type SelectiveDeploymentList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []SelectiveDeployment `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Site describes a Site resource
type Site struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	meta_v1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the site resource spec
	Spec SiteSpec `json:"spec"`
	// Status is the site resource status
	Status SiteStatus `json:"status,omitempty"`
}

// SiteSpec is the spec for a Site resource
type SiteSpec struct {
	FullName  string    `json:"fullname"`
	ShortName string    `json:"shortname"`
	URL       string    `json:"url"`
	Address   string    `json:"address"`
	Contact   []Contact `json:"contact"`
}

// Contact
type Contact struct {
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

// SiteStatus is the status for a Site resource
type SiteStatus struct {
	Enabled bool `json:"enabled"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SiteList is a list of Site resources
type SiteList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []Site `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SiteRegistrationRequest describes a SiteRegistrationRequest resource
type SiteRegistrationRequest struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	meta_v1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the siteregistrationrequest resource spec
	Spec SiteRegistrationRequestSpec `json:"spec"`
	// Status is the siteregistrationrequest resource status
	Status SiteRegistrationRequestStatus `json:"status,omitempty"`
}

// SiteRegistrationRequestSpec is the spec for a SiteRegistrationRequest resource
type SiteRegistrationRequestSpec struct {
	FullName  string    `json:"fullname"`
	ShortName string    `json:"shortname"`
	URL       string    `json:"url"`
	Address   string    `json:"address"`
	Contact   []Contact `json:"contact"`
}

// SiteRegistrationRequestStatus is the status for a SiteRegistrationRequest resource
type SiteRegistrationRequestStatus struct {
	EmailVerify bool          `json:"emailverify"`
	Approved    bool          `json:"approved"`
	Expires     *meta_v1.Time `json:"expires"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SiteRegistrationRequestList is a list of SiteRegistrationRequest resources
type SiteRegistrationRequestList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []SiteRegistrationRequest `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Project describes a Project resource
type Project struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	meta_v1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the project resource spec
	Spec ProjectSpec `json:"spec"`
	// Status is the project resource status
	Status ProjectStatus `json:"status,omitempty"`
}

// ProjectSpec is the spec for a Project resource
type ProjectSpec struct {
	Users       []ProjectUsers `json:"users"`
	Description string         `json:"description"`
}

type ProjectUsers struct {
	Site     string `json:"site"`
	Username string `json:"username"`
}

// ProjectStatus is the status for a Project resource
type ProjectStatus struct {
	Enabled bool `json:"enabled"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectList is a list of Project resources
type ProjectList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []Project `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Slice describes a Slice resource
type Slice struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	meta_v1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

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
}

type SliceUsers struct {
	Site     string `json:"site"`
	Username string `json:"username"`
}

// SliceStatus is the status for a Slice resource
type SliceStatus struct {
	Renew   bool          `json:"renew"`
	Expires *meta_v1.Time `json:"expires"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SliceList is a list of Slice resources
type SliceList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []Slice `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// User describes a User resource
type User struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	meta_v1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the user resource spec
	Spec UserSpec `json:"spec"`
	// Status is the user resource status
	Status UserStatus `json:"status,omitempty"`
}

// UserSpec is the spec for a User resource
type UserSpec struct {
	FirstName string   `json:"firstname"`
	LastName  string   `json:"lastname"`
	Email     string   `json:"email"`
	Password  string   `json:"password"`
	Roles     []string `json:"roles"`
	URL       string   `json:"url"`
	Bio       string   `json:"bio"`
}

// UserStatus is the status for a User resource
type UserStatus struct {
	Active  bool `json:"active"`
	AUP     bool `json:"aup"`
	WebAuth bool `json:"webauth"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UserList is a list of User resources
type UserList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []User `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UserRegistrationRequest describes a UserRegistrationRequest resource
type UserRegistrationRequest struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	meta_v1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the userregistrationrequest resource spec
	Spec UserRegistrationRequestSpec `json:"spec"`
	// Status is the userregistrationrequest resource status
	Status UserRegistrationRequestStatus `json:"status,omitempty"`
}

// UserRegistrationRequestSpec is the spec for a UserRegistrationRequest resource
type UserRegistrationRequestSpec struct {
	FirstName string   `json:"firstname"`
	LastName  string   `json:"lastname"`
	Email     string   `json:"email"`
	Password  string   `json:"password"`
	Roles     []string `json:"roles"`
	URL       string   `json:"url"`
	Bio       string   `json:"bio"`
}

// UserRegistrationRequestStatus is the status for a UserRegistrationRequest resource
type UserRegistrationRequestStatus struct {
	EmailVerify bool          `json:"emailverify"`
	Approved    bool          `json:"approved"`
	Expires     *meta_v1.Time `json:"expires"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UserRegistrationRequestList is a list of UserRegistrationRequest resources
type UserRegistrationRequestList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []UserRegistrationRequest `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AcceptableUsePolicy describes a AcceptableUsePolicy resource
type AcceptableUsePolicy struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	meta_v1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the acceptableusepolicy resource spec
	Spec AcceptableUsePolicySpec `json:"spec"`
	// Status is the acceptableusepolicy resource status
	Status AcceptableUsePolicyStatus `json:"status,omitempty"`
}

// AcceptableUsePolicySpec is the spec for a AcceptableUsePolicy resource
type AcceptableUsePolicySpec struct {
	Accepted bool `json:"accepted"`
}

// AcceptableUsePolicyStatus is the status for a AcceptableUsePolicy resource
type AcceptableUsePolicyStatus struct {
	Renew   bool          `json:"renew"`
	Expires *meta_v1.Time `json:"expires"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AcceptableUsePolicyList is a list of AcceptableUsePolicy resources
type AcceptableUsePolicyList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []AcceptableUsePolicy `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Login describes a Login resource
type Login struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	meta_v1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the login resource spec
	Spec LoginSpec `json:"spec"`
	// Status is the login resource status
	Status LoginStatus `json:"status,omitempty"`
}

// LoginSpec is the spec for a Login resource
type LoginSpec struct {
	Password string `json:"password"`
}

// LoginStatus is the status for a Login resource
type LoginStatus struct {
	Renew   bool          `json:"renew"`
	Expires *meta_v1.Time `json:"expires"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoginList is a list of Login resources
type LoginList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []Login `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EmailVerification describes a EmailVerification resource
type EmailVerification struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	meta_v1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the emailverification resource spec
	Spec EmailVerificationSpec `json:"spec"`
	// Status is the login resource status
	Status EmailVerificationStatus `json:"status,omitempty"`
}

// EmailVerificationSpec is the spec for a EmailVerification resource
type EmailVerificationSpec struct {
	Kind       string `json:"kind"`
	Identifier string `json:"identifier"`
	Verified   bool   `json:"verified"`
}

// EmailVerificationStatus is the status for a Login resource
type EmailVerificationStatus struct {
	Renew   bool          `json:"renew"`
	Expires *meta_v1.Time `json:"expires"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EmailVerificationList is a list of EmailVerification resources
type EmailVerificationList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []EmailVerification `json:"items"`
}
