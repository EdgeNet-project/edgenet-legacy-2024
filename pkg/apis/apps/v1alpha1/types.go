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
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta "k8s.io/api/batch/v1beta1"
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

// SelectiveDeploymentSpec is the spec for a SelectiveDeployment resource.
// Selectors filter the nodes to be used for specified workloads.
type SelectiveDeploymentSpec struct {
	// Workload can be Deployment, Deamonset, StatefulSet, Job, or CronJob.
	Workloads Workloads `json:"workloads"`
	// List of Selector resources. Each selector filters the nodes with the
	// requested method.
	Selector []Selector `json:"selector"`
	// If true, selective deployment tries to find another suitable
	// node to run the workload in case of a node goes down.
	Recovery bool `json:"recovery"`
}

// Workloads indicates deployments, daemonsets, statefulsets, jobs, or cronjobs.
type Workloads struct {
	// Workload can have a list of Deployments.
	Deployment []appsv1.Deployment `json:"deployment"`
	// Workload can have a list of DaemonSets.
	DaemonSet []appsv1.DaemonSet `json:"daemonset"`
	// Workload can have a list of StatefulSets.
	StatefulSet []appsv1.StatefulSet `json:"statefulset"`
	// Workload can have a list of Jobs.
	Job []batchv1.Job `json:"job"`
	// Workload can have a list of CronJobs.
	CronJob []batchv1beta.CronJob `json:"cronjob"`
}

// Selector to define desired node filtering parameters
type Selector struct {
	// Name of the selector. This can be City, State, Country, Continent, or Polygon
	Name string `json:"name"`
	// Value of the selector. For example; if the name of the selector is 'City'
	// then the value can be the city name. For example; if the name of
	// the selector is 'Polygon' then the value can be the GeoJSON representation of the polygon.
	Value []string `json:"value"`
	// Operator means basic mathematical operators such as 'In', 'NotIn', 'Exists', 'NotExsists' etc...
	Operator corev1.NodeSelectorOperator `json:"operator"`
	// Quantity represents number of nodes on which the workloads will be running.
	Quantity int `json:"quantity"`
}

// SelectiveDeploymentStatus is the status for a SelectiveDeployment resource
type SelectiveDeploymentStatus struct {
	// Ready string denotes number of workloads running filtered by the SelectiveDeployments.
	// The string is 'x/y' if x instances are running and y instances are requested.
	Ready string `json:"ready"`
	// Represents state of the selective deployment. This can be 'Failure' if none
	// of the workloads are deployed. 'Partial' if some of the the workloads
	// are deployed. 'Success' if all of the workloads are deployed.
	State string `json:"state"`
	// There can be multiple display messages for state description.
	Message string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SelectiveDeploymentList is a list of SelectiveDeployment resources
type SelectiveDeploymentList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// SelectiveDeploymentList is a list of SelectiveDeployment resources thus,
	// SelectiveDeployments are contained here.
	Items []SelectiveDeployment `json:"items"`
}
