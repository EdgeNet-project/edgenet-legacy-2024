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

package v1alpha2

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Values of Status.State
const (
	StatusFailed         = "Failure"
	StatusReconciliation = "Reconciliation"
	// Selective Deployment
	StatusSuccessful = "Successful"
	StatusCreated    = "Created"
)

// Values of string constants subject to repetitive use
const (
	RemoteSelectiveDeploymentRole = "edgenet:federation:remoteselectivedeployment"
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
	// Workload can be Deployment, Deamonset, StatefulSet, Job, or CronJob
	Workloads Workloads `json:"workloads"`
	// ClusterAffinity is the cluster affinity for the specified workloads
	ClusterAffinity *metav1.LabelSelector `json:"clusterAffinity,omitempty"`
	// ClusterReplicas is the number of clusters per location
	ClusterReplicas int `json:"clusterReplicas,omitempty"`
}

// Workloads indicates deployments, daemonsets, statefulsets, jobs, or cronjobs
type Workloads struct {
	// Workload can have a list of Deployments
	Deployment []appsv1.Deployment `json:"deployment"`
	// Workload can have a list of DaemonSets
	DaemonSet []appsv1.DaemonSet `json:"daemonset"`
	// Workload can have a list of StatefulSets
	StatefulSet []appsv1.StatefulSet `json:"statefulset"`
	// Workload can have a list of Jobs
	Job []batchv1.Job `json:"job"`
	// Workload can have a list of CronJobs
	CronJob []batchv1.CronJob `json:"cronjob"`
}

// SelectiveDeploymentStatus is the status for a SelectiveDeployment resource
type SelectiveDeploymentStatus struct {
	// Represents state of the selective deployment
	State string `json:"state"`
	// Extended status message
	Message string `json:"message"`
	// Clusters is the list of clusters where the workloads are deployed
	Clusters map[string]WorkloadClusterStatus `json:"clusters"`
	// Failed sets the backoff limit.
	Failed int `json:"failed"`
}

type WorkloadClusterStatus struct {
	Server    string         `json:"server"`
	Location  string         `json:"location"`
	Workloads WorkloadStatus `json:"workloads"`
}

type WorkloadStatus struct {
	Deployment  map[string]string `json:"deployment"`
	DaemonSet   map[string]string `json:"daemonset"`
	StatefulSet map[string]string `json:"statefulset"`
	Job         map[string]string `json:"job"`
	CronJob     map[string]string `json:"cronjob"`
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

// MakeOwnerReference creates an owner reference for the given object.
func (s SelectiveDeployment) MakeOwnerReference() metav1.OwnerReference {
	return *metav1.NewControllerRef(&s.ObjectMeta, SchemeGroupVersion.WithKind("SelectiveDeployment"))
}
