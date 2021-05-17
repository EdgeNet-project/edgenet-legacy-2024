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

package selectivedeployment

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/node"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init(kubernetes kubernetes.Interface, edgenet versioned.Interface)
	ObjectCreated(obj interface{})
	ObjectUpdated(obj interface{})
	ObjectDeleted(obj interface{})
}

// SDHandler is a implementation of Handler
type SDHandler struct {
	clientset        kubernetes.Interface
	edgenetClientset versioned.Interface
}

// Init handles any handler initialization
func (t *SDHandler) Init(kubernetes kubernetes.Interface, edgenet versioned.Interface) {
	log.Info("SDHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
}

// ObjectCreated is called when an object is created
func (t *SDHandler) ObjectCreated(obj interface{}) {
	log.Info("SDHandler.ObjectCreated")
	// Make a copy of the selectivedeployment object to make changes on it
	sdCopy := obj.(*apps_v1alpha.SelectiveDeployment).DeepCopy()
	t.applyCriteria(sdCopy, "create")
}

// ObjectUpdated is called when an object is updated
func (t *SDHandler) ObjectUpdated(obj interface{}) {
	log.Info("SDHandler.ObjectUpdated")
	// Make a copy of the selectivedeployment object to make changes on it
	sdCopy := obj.(*apps_v1alpha.SelectiveDeployment).DeepCopy()
	t.applyCriteria(sdCopy, "update")
}

// ObjectDeleted is called when an object is deleted
func (t *SDHandler) ObjectDeleted(obj interface{}) {
	log.Info("SDHandler.ObjectDeleted")
	// TBD
}

// getByNode generates selectivedeployment list from the owner references of workloads which contains the node that has an event (add/update/delete)
func (t *SDHandler) getByNode(nodeName string) ([][]string, bool) {
	ownerList := [][]string{}
	status := false

	setList := func(ctlPodSpec corev1.PodSpec, ownerReferences []metav1.OwnerReference, namespace string) {
		podSpec := ctlPodSpec
		if podSpec.Affinity != nil && podSpec.Affinity.NodeAffinity != nil && podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		nodeSelectorLoop:
			for _, nodeSelectorTerm := range podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
				for _, matchExpression := range nodeSelectorTerm.MatchExpressions {
					if matchExpression.Key == "kubernetes.io/hostname" {
						for _, expressionNodeName := range matchExpression.Values {
							if nodeName == expressionNodeName {
								for _, owner := range ownerReferences {
									if owner.Kind == "SelectiveDeployment" {
										ownerDet := []string{namespace, owner.Name}
										if !util.SliceContains(ownerList, ownerDet) {
											ownerList = append(ownerList, ownerDet)
										}
										status = true
									}
								}
								break nodeSelectorLoop
							}
						}
					}
				}
			}
		}
	}
	deploymentRaw, err := t.clientset.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	for _, deploymentRow := range deploymentRaw.Items {
		setList(deploymentRow.Spec.Template.Spec, deploymentRow.GetOwnerReferences(), deploymentRow.GetNamespace())
	}
	daemonsetRaw, err := t.clientset.AppsV1().DaemonSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	for _, daemonsetRow := range daemonsetRaw.Items {
		setList(daemonsetRow.Spec.Template.Spec, daemonsetRow.GetOwnerReferences(), daemonsetRow.GetNamespace())
	}
	statefulsetRaw, err := t.clientset.AppsV1().StatefulSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	for _, statefulsetRow := range statefulsetRaw.Items {
		setList(statefulsetRow.Spec.Template.Spec, statefulsetRow.GetOwnerReferences(), statefulsetRow.GetNamespace())
	}
	jobRaw, err := t.clientset.BatchV1().Jobs("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	for _, jobRow := range jobRaw.Items {
		setList(jobRow.Spec.Template.Spec, jobRow.GetOwnerReferences(), jobRow.GetNamespace())
	}
	cronjobRaw, err := t.clientset.BatchV1beta1().CronJobs("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	for _, cronjobRow := range cronjobRaw.Items {
		setList(cronjobRow.Spec.JobTemplate.Spec.Template.Spec, cronjobRow.GetOwnerReferences(), cronjobRow.GetNamespace())
	}
	return ownerList, status
}

// applyCriteria used by ObjectCreated, ObjectUpdated, and recoverSelectiveDeployments functions
func (t *SDHandler) applyCriteria(sdCopy *apps_v1alpha.SelectiveDeployment, eventType string) {
	oldStatus := sdCopy.Status
	statusUpdate := func() {
		if !reflect.DeepEqual(oldStatus, sdCopy.Status) {
			t.edgenetClientset.AppsV1alpha().SelectiveDeployments(sdCopy.GetNamespace()).UpdateStatus(context.TODO(), sdCopy, metav1.UpdateOptions{})
		}
	}
	defer statusUpdate()
	// Flush the status
	sdCopy.Status = apps_v1alpha.SelectiveDeploymentStatus{}

	ownerReferences := SetAsOwnerReference(sdCopy)
	workloadCounter := 0
	failureCounter := 0
	if sdCopy.Spec.Workloads.Deployment != nil {
		workloadCounter += len(sdCopy.Spec.Workloads.Deployment)
		for _, sdDeployment := range sdCopy.Spec.Workloads.Deployment {
			deploymentObj, err := t.clientset.AppsV1().Deployments(sdCopy.GetNamespace()).Get(context.TODO(), sdDeployment.GetName(), metav1.GetOptions{})
			if errors.IsNotFound(err) {
				configuredDeployment, failureCount := t.configureWorkload(sdCopy, sdDeployment, ownerReferences)
				failureCounter += failureCount
				_, err = t.clientset.AppsV1().Deployments(sdCopy.GetNamespace()).Create(context.TODO(), configuredDeployment.(*appsv1.Deployment), metav1.CreateOptions{})
				if err != nil {
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["daemonset-creation-failure"], sdDeployment.GetName()))
					failureCounter++
				}
			} else {
				underControl := checkOwnerReferences(sdCopy, deploymentObj.GetOwnerReferences())
				if !underControl {
					// Configure the deployment according to the SD
					configuredDeployment, failureCount := t.configureWorkload(sdCopy, sdDeployment, ownerReferences)
					failureCounter += failureCount
					_, err = t.clientset.AppsV1().Deployments(sdCopy.GetNamespace()).Update(context.TODO(), configuredDeployment.(*appsv1.Deployment), metav1.UpdateOptions{})
					if err != nil {
						sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["daemonset-creation-failure"], sdDeployment.GetName()))
						failureCounter++
					}
				} else {
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["deployment-in-use"], sdDeployment.GetName()))
					failureCounter++
				}
			}
		}
	}
	if sdCopy.Spec.Workloads.DaemonSet != nil {
		workloadCounter += len(sdCopy.Spec.Workloads.DaemonSet)
		for _, sdDaemonset := range sdCopy.Spec.Workloads.DaemonSet {
			daemonsetObj, err := t.clientset.AppsV1().DaemonSets(sdCopy.GetNamespace()).Get(context.TODO(), sdDaemonset.GetName(), metav1.GetOptions{})
			if errors.IsNotFound(err) {
				configuredDaemonSet, failureCount := t.configureWorkload(sdCopy, sdDaemonset, ownerReferences)
				failureCounter += failureCount
				_, err = t.clientset.AppsV1().DaemonSets(sdCopy.GetNamespace()).Create(context.TODO(), configuredDaemonSet.(*appsv1.DaemonSet), metav1.CreateOptions{})
				if err != nil {
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["daemonset-creation-failure"], sdDaemonset.GetName()))
					failureCounter++
				}
			} else {
				underControl := checkOwnerReferences(sdCopy, daemonsetObj.GetOwnerReferences())
				if !underControl {
					// Configure the daemonset according to the SD
					configuredDaemonSet, failureCount := t.configureWorkload(sdCopy, sdDaemonset, ownerReferences)
					failureCounter += failureCount
					_, err = t.clientset.AppsV1().DaemonSets(sdCopy.GetNamespace()).Update(context.TODO(), configuredDaemonSet.(*appsv1.DaemonSet), metav1.UpdateOptions{})
					if err != nil {
						sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["daemonset-creation-failure"], sdDaemonset.GetName()))
						failureCounter++
					}
				} else {
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["daemonset-in-use"], sdDaemonset.GetName()))
					failureCounter++
				}
			}
		}
	}
	if sdCopy.Spec.Workloads.StatefulSet != nil {
		workloadCounter += len(sdCopy.Spec.Workloads.StatefulSet)
		for _, sdStatefulset := range sdCopy.Spec.Workloads.StatefulSet {
			statefulsetObj, err := t.clientset.AppsV1().StatefulSets(sdCopy.GetNamespace()).Get(context.TODO(), sdStatefulset.GetName(), metav1.GetOptions{})
			if errors.IsNotFound(err) {
				configuredStatefulSet, failureCount := t.configureWorkload(sdCopy, sdStatefulset, ownerReferences)
				failureCounter += failureCount
				_, err = t.clientset.AppsV1().StatefulSets(sdCopy.GetNamespace()).Create(context.TODO(), configuredStatefulSet.(*appsv1.StatefulSet), metav1.CreateOptions{})
				if err != nil {
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["statefulset-creation-failure"], sdStatefulset.GetName()))
					failureCounter++
				}
			} else {
				underControl := checkOwnerReferences(sdCopy, statefulsetObj.GetOwnerReferences())
				if !underControl {
					// Configure the statefulset according to the SD
					configuredStatefulSet, failureCount := t.configureWorkload(sdCopy, sdStatefulset, ownerReferences)
					failureCounter += failureCount
					_, err = t.clientset.AppsV1().StatefulSets(sdCopy.GetNamespace()).Update(context.TODO(), configuredStatefulSet.(*appsv1.StatefulSet), metav1.UpdateOptions{})
					if err != nil {
						sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["statefulset-creation-failure"], sdStatefulset.GetName()))
						failureCounter++
					}
				} else {
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["statefulset-in-use"], sdStatefulset.GetName()))
					failureCounter++
				}
			}
		}
	}
	if sdCopy.Spec.Workloads.Job != nil {
		workloadCounter += len(sdCopy.Spec.Workloads.Job)
		for _, sdJob := range sdCopy.Spec.Workloads.Job {
			jobObj, err := t.clientset.BatchV1().Jobs(sdCopy.GetNamespace()).Get(context.TODO(), sdJob.GetName(), metav1.GetOptions{})
			if errors.IsNotFound(err) {
				configuredJob, failureCount := t.configureWorkload(sdCopy, sdJob, ownerReferences)
				failureCounter += failureCount
				_, err = t.clientset.BatchV1().Jobs(sdCopy.GetNamespace()).Create(context.TODO(), configuredJob.(*batchv1.Job), metav1.CreateOptions{})
				if err != nil {
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["job-creation-failure"], sdJob.GetName()))
					failureCounter++
				}
			} else {
				underControl := checkOwnerReferences(sdCopy, jobObj.GetOwnerReferences())
				if !underControl {
					// Configure the job according to the SD
					configuredJob, failureCount := t.configureWorkload(sdCopy, sdJob, ownerReferences)
					failureCounter += failureCount
					_, err = t.clientset.BatchV1().Jobs(sdCopy.GetNamespace()).Update(context.TODO(), configuredJob.(*batchv1.Job), metav1.UpdateOptions{})
					if err != nil {
						sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["job-creation-failure"], sdJob.GetName()))
						failureCounter++
					}
				} else {
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["job-in-use"], sdJob.GetName()))
					failureCounter++
				}
			}
		}
	}
	if sdCopy.Spec.Workloads.CronJob != nil {
		workloadCounter += len(sdCopy.Spec.Workloads.CronJob)
		for _, sdCronJob := range sdCopy.Spec.Workloads.CronJob {
			cronjobObj, err := t.clientset.BatchV1beta1().CronJobs(sdCopy.GetNamespace()).Get(context.TODO(), sdCronJob.GetName(), metav1.GetOptions{})
			if errors.IsNotFound(err) {
				configuredCronJob, failureCount := t.configureWorkload(sdCopy, sdCronJob, ownerReferences)
				failureCounter += failureCount
				_, err = t.clientset.BatchV1beta1().CronJobs(sdCopy.GetNamespace()).Create(context.TODO(), configuredCronJob.(*batchv1beta.CronJob), metav1.CreateOptions{})
				if err != nil {
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["cronjob-creation-failure"], sdCronJob.GetName()))
					failureCounter++
				}
			} else {
				underControl := checkOwnerReferences(sdCopy, cronjobObj.GetOwnerReferences())
				if !underControl {
					// Configure the cronjob according to the SD
					configuredCronJob, failureCount := t.configureWorkload(sdCopy, sdCronJob, ownerReferences)
					failureCounter += failureCount
					_, err = t.clientset.BatchV1beta1().CronJobs(sdCopy.GetNamespace()).Update(context.TODO(), configuredCronJob.(*batchv1beta.CronJob), metav1.UpdateOptions{})
					if err != nil {
						sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["cronjob-creation-failure"], sdCronJob.GetName()))
						failureCounter++
					}
				} else {
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["cronjob-in-use"], sdCronJob.GetName()))
					failureCounter++
				}
			}
		}
	}

	if failureCounter == 0 && workloadCounter != 0 {
		sdCopy.Status.State = success
		sdCopy.Status.Message = []string{statusDict["sd-success"]}
	} else if workloadCounter == failureCounter {
		sdCopy.Status.State = failure
	} else {
		sdCopy.Status.State = partial
	}
	sdCopy.Status.Ready = fmt.Sprintf("%d/%d", (workloadCounter - failureCounter), workloadCounter)
}

// configureWorkload manipulate the workload by selectivedeployments to match the desired state that users supplied
func (t *SDHandler) configureWorkload(sdCopy *apps_v1alpha.SelectiveDeployment, workloadRow interface{}, ownerReferences []metav1.OwnerReference) (interface{}, int) {
	log.Info("configureWorkload: start")
	nodeSelectorTermList, failureCount := t.setFilter(sdCopy, "addOrUpdate")
	// Set the new node affinity configuration for the workload and update that
	nodeAffinity := &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: nodeSelectorTermList,
		},
	}
	if len(nodeSelectorTermList) <= 0 {
		affinity := &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: nodeSelectorTermList,
				},
			},
		}
		affinity.Reset()
	}
	var workloadCopy interface{}
	switch workloadObj := workloadRow.(type) {
	case appsv1.Deployment:
		if len(nodeSelectorTermList) <= 0 && workloadObj.Spec.Template.Spec.Affinity != nil {
			workloadObj.Spec.Template.Spec.Affinity.Reset()
		} else if workloadObj.Spec.Template.Spec.Affinity != nil {
			workloadObj.Spec.Template.Spec.Affinity.NodeAffinity = nodeAffinity
		} else {
			workloadObj.Spec.Template.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: nodeAffinity,
			}
		}
		workloadObj.ObjectMeta.OwnerReferences = ownerReferences
		//log.Printf("%s/Deployment/%s: %s", workloadObj.GetNamespace(), workloadObj.GetName(), nodeAffinity)
		workloadCopy = workloadObj.DeepCopy()
		//t.clientset.AppsV1().Deployments(sdCopy.GetNamespace()).Update(workloadCopy)
	case appsv1.DaemonSet:
		if len(nodeSelectorTermList) <= 0 && workloadObj.Spec.Template.Spec.Affinity != nil {
			workloadObj.Spec.Template.Spec.Affinity.Reset()
		} else if workloadObj.Spec.Template.Spec.Affinity != nil {
			workloadObj.Spec.Template.Spec.Affinity.NodeAffinity = nodeAffinity
		} else {
			workloadObj.Spec.Template.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: nodeAffinity,
			}
		}
		workloadObj.ObjectMeta.OwnerReferences = ownerReferences
		//log.Printf("%s/DaemonSet/%s: %s", workloadObj.GetNamespace(), workloadObj.GetName(), nodeAffinity)
		workloadCopy = workloadObj.DeepCopy()
		//t.clientset.AppsV1().DaemonSets(sdCopy.GetNamespace()).Update(workloadCopy)
	case appsv1.StatefulSet:
		if len(nodeSelectorTermList) <= 0 && workloadObj.Spec.Template.Spec.Affinity != nil {
			workloadObj.Spec.Template.Spec.Affinity.Reset()
		} else if workloadObj.Spec.Template.Spec.Affinity != nil {
			workloadObj.Spec.Template.Spec.Affinity.NodeAffinity = nodeAffinity
		} else {
			workloadObj.Spec.Template.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: nodeAffinity,
			}
		}
		workloadObj.ObjectMeta.OwnerReferences = ownerReferences
		//log.Printf("%s/StatefulSet/%s: %s", workloadObj.GetNamespace(), workloadObj.GetName(), nodeAffinity)
		workloadCopy = workloadObj.DeepCopy()
		//t.clientset.AppsV1().StatefulSets(sdCopy.GetNamespace()).Update(workloadCopy)
	case batchv1.Job:
		if len(nodeSelectorTermList) <= 0 && workloadObj.Spec.Template.Spec.Affinity != nil {
			workloadObj.Spec.Template.Spec.Affinity.Reset()
		} else if workloadObj.Spec.Template.Spec.Affinity != nil {
			workloadObj.Spec.Template.Spec.Affinity.NodeAffinity = nodeAffinity
		} else {
			workloadObj.Spec.Template.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: nodeAffinity,
			}
		}
		workloadObj.ObjectMeta.OwnerReferences = ownerReferences
		//log.Printf("%s/Job/%s: %s", workloadObj.GetNamespace(), workloadObj.GetName(), nodeAffinity)
		workloadCopy = workloadObj.DeepCopy()
		//t.clientset.BatchV1().Jobs(sdCopy.GetNamespace()).Update(workloadCopy)
	case batchv1beta.CronJob:
		if len(nodeSelectorTermList) <= 0 && workloadObj.Spec.JobTemplate.Spec.Template.Spec.Affinity != nil {
			workloadObj.Spec.JobTemplate.Spec.Template.Spec.Affinity.Reset()
		} else if workloadObj.Spec.JobTemplate.Spec.Template.Spec.Affinity != nil {
			workloadObj.Spec.JobTemplate.Spec.Template.Spec.Affinity.NodeAffinity = nodeAffinity
		} else {
			workloadObj.Spec.JobTemplate.Spec.Template.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: nodeAffinity,
			}
		}
		workloadObj.ObjectMeta.OwnerReferences = ownerReferences
		//log.Printf("%s/CronJob/%s: %s", workloadObj.GetNamespace(), workloadObj.GetName(), nodeAffinity)
		workloadCopy = workloadObj.DeepCopy()
		//t.clientset.BatchV1beta1().CronJob(sdCopy.GetNamespace()).Update(workloadCopy)
	}
	return workloadCopy, failureCount
}

// setFilter generates the values in the predefined form and puts those into the node selection fields of the selectivedeployment object
func (t *SDHandler) setFilter(sdCopy *apps_v1alpha.SelectiveDeployment, event string) ([]corev1.NodeSelectorTerm, int) {
	var nodeSelectorTermList []corev1.NodeSelectorTerm
	failureCounter := 0
	for _, selectorRow := range sdCopy.Spec.Selector {
		var matchExpression corev1.NodeSelectorRequirement
		matchExpression.Values = []string{}
		matchExpression.Operator = selectorRow.Operator
		matchExpression.Key = "kubernetes.io/hostname"
		selectorName := strings.ToLower(selectorRow.Name)
		// Turn the key into the predefined form which is determined at the custom resource definition of selectivedeployment
		switch selectorName {
		case "city", "state", "country", "continent":
			// If the event type is delete then we don't need to run the part below
			if event != "delete" {
				labelKeySuffix := ""
				if selectorName == "state" || selectorName == "country" {
					labelKeySuffix = "-iso"
				}
				labelKey := strings.ToLower(fmt.Sprintf("edge-net.io/%s%s", selectorName, labelKeySuffix))
				// This gets the node list which includes the EdgeNet geolabels
				nodesRaw, err := t.clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{FieldSelector: "spec.unschedulable!=true"})
				if err != nil {
					log.Println(err.Error())
					panic(err.Error())
				}
				counter := 0
				// This loop allows us to process each value defined at the object of selectivedeployment resource
			valueLoop:
				for _, selectorValue := range selectorRow.Value {
					// The loop to process each node separately
					for _, nodeRow := range nodesRaw.Items {
						taintBlock := false
						for _, taint := range nodeRow.Spec.Taints {
							if (taint.Key == "node-role.kubernetes.io/master" && taint.Effect == noSchedule) ||
								(taint.Key == "node.kubernetes.io/unschedulable" && taint.Effect == noSchedule) {
								taintBlock = true
							}
						}
						conditionBlock := false
						if node.GetConditionReadyStatus(nodeRow.DeepCopy()) != trueStr {
							conditionBlock = true
						}

						if !conditionBlock && !taintBlock {
							if util.Contains(matchExpression.Values, nodeRow.Labels["kubernetes.io/hostname"]) {
								continue
							}
							if selectorValue == nodeRow.Labels[labelKey] && selectorRow.Operator == "In" {
								matchExpression.Values = append(matchExpression.Values, nodeRow.Labels["kubernetes.io/hostname"])
								counter++
							} else if selectorValue != nodeRow.Labels[labelKey] && selectorRow.Operator == "NotIn" {
								matchExpression.Values = append(matchExpression.Values, nodeRow.Labels["kubernetes.io/hostname"])
								counter++
							}
							if selectorRow.Quantity != 0 && selectorRow.Quantity == counter {
								break valueLoop
							}
						}
					}
				}
				if selectorRow.Quantity != 0 && selectorRow.Quantity > counter {
					strLen := 16
					strSuffix := "..."
					if len(selectorRow.Value) <= strLen {
						strLen = len(selectorRow.Value)
						strSuffix = ""
					}
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["nodes-fewer"], counter, selectorRow.Quantity, selectorRow.Value[0:strLen], strSuffix))
					failureCounter++
				}
			}
		case "polygon":
			// If the event type is delete then we don't need to run the GeoFence functions
			if event != "delete" {
				// If the selectivedeployment key is polygon then certain calculations like geofence need to be done
				// for being had the list of nodes that the pods will be deployed on according to the desired state.
				// This gets the node list which includes the EdgeNet geolabels
				nodesRaw, err := t.clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{FieldSelector: "spec.unschedulable!=true"})
				if err != nil {
					log.Println(err.Error())
					panic(err.Error())
				}

				var polygon [][]float64
				// This loop allows us to process each polygon defined at the object of selectivedeployment resource
				counter := 0
			polyValueLoop:
				for _, selectorValue := range selectorRow.Value {
					err = json.Unmarshal([]byte(selectorValue), &polygon)
					if err != nil {
						strLen := 16
						strSuffix := "..."
						if len(selectorRow.Value) <= strLen {
							strLen = len(selectorRow.Value)
							strSuffix = ""
						}
						sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["GeoJSON-err"], selectorValue[0:strLen], strSuffix))
						failureCounter++
						continue
					}
					// The loop to process each node separately
					for _, nodeRow := range nodesRaw.Items {
						taintBlock := false
						for _, taint := range nodeRow.Spec.Taints {
							if (taint.Key == "node-role.kubernetes.io/master" && taint.Effect == noSchedule) ||
								(taint.Key == "node.kubernetes.io/unschedulable" && taint.Effect == noSchedule) {
								taintBlock = true
							}
						}
						conditionBlock := false
						for _, conditionRow := range nodeRow.Status.Conditions {
							if conditionType := conditionRow.Type; conditionType == "Ready" {
								if conditionRow.Status != trueStr {
									conditionBlock = true
								}
							}
						}
						if !conditionBlock && !taintBlock {
							if nodeRow.Labels["edge-net.io/lon"] != "" && nodeRow.Labels["edge-net.io/lat"] != "" {
								if util.Contains(matchExpression.Values, nodeRow.Labels["kubernetes.io/hostname"]) {
									continue
								}
								// Because of alphanumeric limitations of Kubernetes on the labels we use "w", "e", "n", and "s" prefixes
								// at the labels of latitude and longitude. Here is the place those prefixes are dropped away.
								lonStr := nodeRow.Labels["edge-net.io/lon"]
								lonStr = string(lonStr[1:])
								latStr := nodeRow.Labels["edge-net.io/lat"]
								latStr = string(latStr[1:])
								if lon, err := strconv.ParseFloat(lonStr, 64); err == nil {
									if lat, err := strconv.ParseFloat(latStr, 64); err == nil {
										// boundbox is a rectangle which provides to check whether the point is inside polygon
										// without taking all point of the polygon into consideration
										boundbox := node.Boundbox(polygon)
										status := node.GeoFence(boundbox, polygon, lon, lat)
										if status && selectorRow.Operator == "In" {
											matchExpression.Values = append(matchExpression.Values, nodeRow.Labels["kubernetes.io/hostname"])
											counter++
										} else if !status && selectorRow.Operator == "NotIn" {
											matchExpression.Values = append(matchExpression.Values, nodeRow.Labels["kubernetes.io/hostname"])
											counter++
										}
									}
								}
							}
							if selectorRow.Quantity != 0 && selectorRow.Quantity == counter {
								break polyValueLoop
							}
						}
					}
				}
				if selectorRow.Quantity != 0 && selectorRow.Quantity > counter {
					strLen := 16
					strSuffix := "..."
					if len(selectorRow.Value) <= strLen {
						strLen = len(selectorRow.Value)
						strSuffix = ""
					}
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf(statusDict["nodes-fewer"], counter, selectorRow.Quantity, selectorRow.Value[0:strLen], strSuffix))
					failureCounter++
				}
			}
		default:
			matchExpression.Key = ""
		}

		var nodeSelectorTerm corev1.NodeSelectorTerm
		nodeSelectorTerm.MatchExpressions = append(nodeSelectorTerm.MatchExpressions, matchExpression)
		nodeSelectorTermList = append(nodeSelectorTermList, nodeSelectorTerm)
	}
	return nodeSelectorTermList, failureCounter
}

// SetAsOwnerReference returns the tenant as owner
func SetAsOwnerReference(sdCopy *apps_v1alpha.SelectiveDeployment) []metav1.OwnerReference {
	// The following section makes tenant become the owner
	ownerReferences := []metav1.OwnerReference{}
	newSDRef := *metav1.NewControllerRef(sdCopy, apps_v1alpha.SchemeGroupVersion.WithKind("SelectiveDeployment"))
	takeControl := false
	newSDRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newSDRef)
	return ownerReferences
}

func checkOwnerReferences(sdCopy *apps_v1alpha.SelectiveDeployment, ownerReferences []metav1.OwnerReference) bool {
	underControl := false
	for _, reference := range ownerReferences {
		if reference.Kind == "SelectiveDeployment" && reference.UID != sdCopy.GetUID() {
			underControl = true
		}
	}
	return underControl
}
