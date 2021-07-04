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
	FullName  string              `json:"fullname"`
	ShortName string              `json:"shortname"`
	URL       string              `json:"url"`
	Address   corev1alpha.Address `json:"address"`
	Contact   corev1alpha.Contact `json:"contact"`
	Approved  bool                `json:"approved"`
}

// TenantRequestStatus is the status for a TenantRequest resource
type TenantRequestStatus struct {
	EmailVerified bool         `json:"emailverified"`
	Expiry        *metav1.Time `json:"expiry"`
	State         string       `json:"state"`
	Message       []string     `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TenantRequestList is a list of TenantRequest resources
type TenantRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []TenantRequest `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UserRequest describes a UserRequest resource
type UserRequest struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the userrequest resource spec
	Spec UserRequestSpec `json:"spec"`
	// Status is the userrequest resource status
	Status UserRequestStatus `json:"status,omitempty"`
}

// UserRequestSpec is the spec for a UserRequest resource
type UserRequestSpec struct {
	Tenant    string `json:"tenant"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Approved  bool   `json:"approved"`
}

// UserRequestStatus is the status for a UserRequest resource
type UserRequestStatus struct {
	EmailVerified bool         `json:"emailverified"`
	Expiry        *metav1.Time `json:"expiry"`
	State         string       `json:"state"`
	Message       []string     `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UserRequestList is a list of UserRequest resources
type UserRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []UserRequest `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
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
	Email    string `json:"email"`
	Verified bool   `json:"verified"`
}

// EmailVerificationStatus is the status for a EmailVerification resource
type EmailVerificationStatus struct {
	Expiry  *metav1.Time `json:"expiry"`
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
