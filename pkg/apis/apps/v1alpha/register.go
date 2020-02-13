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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"headnode/pkg/apis/apps"
)

// SchemeGroupVersion is the identifier for the API which includes
// the name of the group and the version of the API
var SchemeGroupVersion = schema.GroupVersion{
	Group:   apps.GroupName,
	Version: "v1alpha",
}

// Create a SchemeBuilder which uses functions to add types to the scheme
var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Resource handles adding types to the schemes
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// addKnownTypes adds our types to the API scheme by registering
// SelectiveDeployment and SelectiveDeploymentList
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&SelectiveDeployment{},
		&SelectiveDeploymentList{},
		&Site{},
		&SiteList{},
		&SiteRegistrationRequest{},
		&SiteRegistrationRequestList{},
		&User{},
		&UserList{},
		&UserRegistrationRequest{},
		&UserRegistrationRequestList{},
		&AcceptableUsePolicy{},
		&AcceptableUsePolicyList{},
		&Login{},
		&LoginList{},
		&EmailVerification{},
		&EmailVerificationList{},
		&Slice{},
		&SliceList{},
		&Project{},
		&ProjectList{},
	)

	// Register the type in the scheme
	meta_v1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
