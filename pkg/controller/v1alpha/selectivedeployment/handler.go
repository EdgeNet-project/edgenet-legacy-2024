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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	"edgenet/pkg/node"
	"edgenet/pkg/util"

	log "github.com/Sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init(kubernetes kubernetes.Interface, edgenet versioned.Interface)
	ObjectCreated(obj interface{})
	ObjectUpdated(obj interface{}, delta string)
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
	// Create a copy of the selectivedeployment object to make changes on it
	sdCopy := obj.(*apps_v1alpha.SelectiveDeployment).DeepCopy()
	t.applyCriteria(sdCopy, "", "create")
}

// ObjectUpdated is called when an object is updated
func (t *SDHandler) ObjectUpdated(obj interface{}, delta string) {
	log.Info("SDHandler.ObjectUpdated")
	// Create a copy of the selectivedeployment object to make changes on it
	sdCopy := obj.(*apps_v1alpha.SelectiveDeployment).DeepCopy()
	t.applyCriteria(sdCopy, delta, "update")
}

// ObjectDeleted is called when an object is deleted
func (t *SDHandler) ObjectDeleted(obj interface{}) {
	log.Info("SDHandler.ObjectDeleted")
	// TBD
}

// getByNode generates selectivedeployment list from the owner references of controllers which contains the node that has an event (add/update/delete)
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
										ownerList = append(ownerList, ownerDet)
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
	deploymentRaw, err := t.clientset.AppsV1().Deployments("").List(metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	for _, deploymentRow := range deploymentRaw.Items {
		setList(deploymentRow.Spec.Template.Spec, deploymentRow.GetOwnerReferences(), deploymentRow.GetNamespace())
	}
	daemonsetRaw, err := t.clientset.AppsV1().DaemonSets("").List(metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	for _, daemonsetRow := range daemonsetRaw.Items {
		setList(daemonsetRow.Spec.Template.Spec, daemonsetRow.GetOwnerReferences(), daemonsetRow.GetNamespace())
	}
	statefulsetRaw, err := t.clientset.AppsV1().StatefulSets("").List(metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	for _, statefulsetRow := range statefulsetRaw.Items {
		setList(statefulsetRow.Spec.Template.Spec, statefulsetRow.GetOwnerReferences(), statefulsetRow.GetNamespace())
	}
	return ownerList, status
}

// applyCriteria used by ObjectCreated, ObjectUpdated, and recoverSelectiveDeployments functions
func (t *SDHandler) applyCriteria(sdCopy *apps_v1alpha.SelectiveDeployment, delta string, eventType string) {
	// Flush the status
	sdCopy.Status = apps_v1alpha.SelectiveDeploymentStatus{}
	defer t.edgenetClientset.AppsV1alpha().SelectiveDeployments(sdCopy.GetNamespace()).UpdateStatus(sdCopy)
	ownerReferences := SetAsOwnerReference(sdCopy)
	controllerCounter := 0
	failureCounter := 0
	if sdCopy.Spec.Controllers.Deployment != nil {
		controllerCounter += len(sdCopy.Spec.Controllers.Deployment)
		for _, sdDeployment := range sdCopy.Spec.Controllers.Deployment {
			deploymentObj, err := t.clientset.AppsV1().Deployments(sdCopy.GetNamespace()).Get(sdDeployment.GetName(), metav1.GetOptions{})
			if errors.IsNotFound(err) {
				configuredDeployment, failureCount := t.configureController(sdCopy, sdDeployment, ownerReferences)
				failureCounter += failureCount
				_, err = t.clientset.AppsV1().Deployments(sdCopy.GetNamespace()).Create(configuredDeployment.(*appsv1.Deployment))
				if err != nil {
					fmt.Println(err)
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf("Deployment %s could not be created", sdDeployment.GetName()))
					failureCounter++
				}
			} else {
				underControl := checkOwnerReferences(sdCopy, deploymentObj.GetOwnerReferences())
				if !underControl {
					// Configure the deployment according to the SD
					deploymentObj = sdDeployment.DeepCopy()
					configuredDeployment, failureCount := t.configureController(sdCopy, deploymentObj, ownerReferences)
					failureCounter += failureCount
					_, err = t.clientset.AppsV1().Deployments(sdCopy.GetNamespace()).Update(configuredDeployment.(*appsv1.Deployment))
					if err != nil {
						sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf("Deployment %s could not be updated", sdDeployment.GetName()))
						failureCounter++
					}
				} else {
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf("Deployment %s is already under the control of another selective deployment", sdDeployment.GetName()))
					failureCounter++
				}
			}
		}
	}
	if sdCopy.Spec.Controllers.DaemonSet != nil {
		controllerCounter += len(sdCopy.Spec.Controllers.DaemonSet)
		for _, sdDaemonset := range sdCopy.Spec.Controllers.DaemonSet {
			daemonsetObj, err := t.clientset.AppsV1().DaemonSets(sdCopy.GetNamespace()).Get(sdDaemonset.GetName(), metav1.GetOptions{})
			if errors.IsNotFound(err) {
				configuredDaemonSet, failureCount := t.configureController(sdCopy, sdDaemonset, ownerReferences)
				failureCounter += failureCount

				_, err = t.clientset.AppsV1().DaemonSets(sdCopy.GetNamespace()).Create(configuredDaemonSet.(*appsv1.DaemonSet))
				if err != nil {
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf("DaemonSet %s could not be created", sdDaemonset.GetName()))
					failureCounter++
				}
			} else {
				underControl := checkOwnerReferences(sdCopy, daemonsetObj.GetOwnerReferences())
				if !underControl {
					// Configure the daemonset according to the SD
					daemonsetObj = sdDaemonset.DeepCopy()
					configuredDaemonSet, failureCount := t.configureController(sdCopy, sdDaemonset, ownerReferences)
					failureCounter += failureCount

					_, err = t.clientset.AppsV1().DaemonSets(sdCopy.GetNamespace()).Update(configuredDaemonSet.(*appsv1.DaemonSet))
					if err != nil {
						sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf("DaemonSet %s could not be updated", sdDaemonset.GetName()))
						failureCounter++
					}
				} else {
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf("DaemonSet %s is already under the control of another selective deployment", sdDaemonset.GetName()))
					failureCounter++
				}
			}
		}
	}
	if sdCopy.Spec.Controllers.StatefulSet != nil {
		controllerCounter += len(sdCopy.Spec.Controllers.StatefulSet)
		for _, sdStatefulset := range sdCopy.Spec.Controllers.StatefulSet {
			statefulsetObj, err := t.clientset.AppsV1().StatefulSets(sdCopy.GetNamespace()).Get(sdStatefulset.GetName(), metav1.GetOptions{})
			if errors.IsNotFound(err) {
				configuredStatefulSet, failureCount := t.configureController(sdCopy, sdStatefulset, ownerReferences)
				failureCounter += failureCount
				_, err = t.clientset.AppsV1().StatefulSets(sdCopy.GetNamespace()).Create(configuredStatefulSet.(*appsv1.StatefulSet))
				if err != nil {
					sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf("StatefulSet %s could not be created", sdStatefulset.GetName()))
					failureCounter++
				} else {
					underControl := checkOwnerReferences(sdCopy, statefulsetObj.GetOwnerReferences())
					if !underControl {
						// Configure the statefulset according to the SD
						statefulsetObj = sdStatefulset.DeepCopy()
						configuredStatefulSet, failureCount := t.configureController(sdCopy, sdStatefulset, ownerReferences)
						failureCounter += failureCount

						_, err = t.clientset.AppsV1().StatefulSets(sdCopy.GetNamespace()).Update(configuredStatefulSet.(*appsv1.StatefulSet))
						if err != nil {
							sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf("StatefulSet %s could not be created", sdStatefulset.GetName()))
							failureCounter++
						}
					} else {
						sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf("StatefulSet %s is already under the control of another selective deployment", sdStatefulset.GetName()))
						failureCounter++
					}
				}
			}
		}
	}

	if failureCounter == 0 {
		sdCopy.Status.State = success
	} else if controllerCounter == failureCounter {
		sdCopy.Status.State = failure
	} else {
		sdCopy.Status.State = partial
	}
	sdCopy.Status.Ready = fmt.Sprintf("%d/%d", failureCounter, controllerCounter)
}

// configureController manipulate the controller by selectivedeployments to match the desired state that users supplied
func (t *SDHandler) configureController(sdCopy *apps_v1alpha.SelectiveDeployment, controllerRow interface{}, ownerReferences []metav1.OwnerReference) (interface{}, int) {
	log.Info("configureController: start")
	nodeSelectorTermList, failureCount := t.setFilter(sdCopy, "addOrUpdate")
	// Set the new node affinity configuration in the controller and update that
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
	var controllerCopy interface{}
	switch controllerObj := controllerRow.(type) {
	case appsv1.Deployment:
		if len(nodeSelectorTermList) <= 0 && controllerObj.Spec.Template.Spec.Affinity != nil {
			controllerObj.Spec.Template.Spec.Affinity.Reset()
		} else if controllerObj.Spec.Template.Spec.Affinity != nil {
			controllerObj.Spec.Template.Spec.Affinity.NodeAffinity = nodeAffinity
		} else {
			controllerObj.Spec.Template.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: nodeAffinity,
			}
		}
		controllerObj.ObjectMeta.OwnerReferences = ownerReferences
		log.Printf("%s/Deployment/%s: %s", controllerObj.GetNamespace(), controllerObj.GetName(), nodeAffinity)
		controllerCopy = controllerObj.DeepCopy()
		//t.clientset.AppsV1().Deployments(sdCopy.GetNamespace()).Update(controllerCopy)
	case appsv1.DaemonSet:
		if len(nodeSelectorTermList) <= 0 && controllerObj.Spec.Template.Spec.Affinity != nil {
			controllerObj.Spec.Template.Spec.Affinity.Reset()
		} else if controllerObj.Spec.Template.Spec.Affinity != nil {
			controllerObj.Spec.Template.Spec.Affinity.NodeAffinity = nodeAffinity
		} else {
			controllerObj.Spec.Template.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: nodeAffinity,
			}
		}
		controllerObj.ObjectMeta.OwnerReferences = ownerReferences
		log.Printf("%s/DaemonSet/%s: %s", controllerObj.GetNamespace(), controllerObj.GetName(), nodeAffinity)
		controllerCopy = controllerObj.DeepCopy()
		//t.clientset.AppsV1().DaemonSets(sdCopy.GetNamespace()).Update(controllerCopy)
	case appsv1.StatefulSet:
		if len(nodeSelectorTermList) <= 0 && controllerObj.Spec.Template.Spec.Affinity != nil {
			controllerObj.Spec.Template.Spec.Affinity.Reset()
		} else if controllerObj.Spec.Template.Spec.Affinity != nil {
			controllerObj.Spec.Template.Spec.Affinity.NodeAffinity = nodeAffinity
		} else {
			controllerObj.Spec.Template.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: nodeAffinity,
			}
		}
		controllerObj.ObjectMeta.OwnerReferences = ownerReferences
		log.Printf("%s/StatefulSet/%s: %s", controllerObj.GetNamespace(), controllerObj.GetName(), nodeAffinity)
		controllerCopy = controllerObj.DeepCopy()
		//t.clientset.AppsV1().StatefulSets(sdCopy.GetNamespace()).Update(controllerCopy)
	}
	return controllerCopy, failureCount
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
				nodesRaw, err := t.clientset.CoreV1().Nodes().List(metav1.ListOptions{FieldSelector: "spec.unschedulable!=true"})
				if err != nil {
					log.Println(err.Error())
					panic(err.Error())
				}
				// This loop allows us to process each value defined at the object of selectivedeployment resource
				for _, selectorValue := range selectorRow.Value {
					counter := 0
					// The loop to process each node separately
				cityNodeLoop:
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
								break cityNodeLoop
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
						sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf("Fewer nodes issue, %d node(s) found instead of %d for %s%s", counter, selectorRow.Quantity, selectorRow.Value[0:strLen], strSuffix))
						failureCounter++
					}
				}
			}
		case "polygon":
			// If the event type is delete then we don't need to run the GeoFence functions
			if event != "delete" {
				// If the selectivedeployment key is polygon then certain calculations like geofence need to be done
				// for being had the list of nodes that the pods will be deployed on according to the desired state.
				// This gets the node list which includes the EdgeNet geolabels
				nodesRaw, err := t.clientset.CoreV1().Nodes().List(metav1.ListOptions{FieldSelector: "spec.unschedulable!=true"})
				if err != nil {
					log.Println(err.Error())
					panic(err.Error())
				}

				var polygon [][]float64
				// This loop allows us to process each polygon defined at the object of selectivedeployment resource
				for _, selectorValue := range selectorRow.Value {
					counter := 0
					err = json.Unmarshal([]byte(selectorValue), &polygon)
					if err != nil {
						strLen := 16
						strSuffix := "..."
						if len(selectorRow.Value) <= strLen {
							strLen = len(selectorRow.Value)
							strSuffix = ""
						}
						sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf("%s%s has a GeoJSON format error", selectorValue[0:strLen], strSuffix))
						failureCounter++
						continue
					}
					// The loop to process each node separately
				polyNodeLoop:
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
								break polyNodeLoop
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
						sdCopy.Status.Message = append(sdCopy.Status.Message, fmt.Sprintf("Fewer nodes issue, %d node(s) found instead of %d for %s%s", counter, selectorRow.Quantity, selectorRow.Value[0:strLen], strSuffix))
						failureCounter++
					}
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

// SetAsOwnerReference returns the authority as owner
func SetAsOwnerReference(sdCopy *apps_v1alpha.SelectiveDeployment) []metav1.OwnerReference {
	// The following section makes authority become the owner
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

func (t *SDHandler) checkController(controllerName, controllerKind, namespace string) (apps_v1alpha.SelectiveDeployment, bool) {
	exists := false
	var ownerSD apps_v1alpha.SelectiveDeployment
	SDRaw, err := t.edgenetClientset.AppsV1alpha().SelectiveDeployments(namespace).List(metav1.ListOptions{})
	if err != nil {
		return ownerSD, exists
	}
	for _, SDRow := range SDRaw.Items {
		if controllerKind == "Deployment" {
			for _, deployment := range SDRow.Spec.Controllers.Deployment {
				if controllerName == deployment.GetName() {
					exists = true
					ownerSD = SDRow
				}
			}
		} else if controllerKind == "DaemonSet" {
			for _, daemonset := range SDRow.Spec.Controllers.DaemonSet {
				if controllerName == daemonset.GetName() {
					exists = true
					ownerSD = SDRow
				}
			}
		} else if controllerKind == "StatefulSet" {
			for _, statefulset := range SDRow.Spec.Controllers.StatefulSet {
				if controllerName == statefulset.GetName() {
					exists = true
					ownerSD = SDRow
				}
			}
		}
	}
	return ownerSD, exists
}

// dry function remove the same values of the old and new objects from the old object to have
// the slice of deleted values.
/*func dry(oldSlice []apps_v1alpha.Controllers, newSlice []apps_v1alpha.Controllers) []string {
	var uniqueSlice []string
	for _, oldValue := range oldSlice {
		exists := false
		for _, newValue := range newSlice {
			if oldValue.Type == newValue.Type && oldValue.Name == newValue.Name {
				exists = true
			}
		}
		if !exists {
			uniqueSlice = append(uniqueSlice, fmt.Sprintf("%s?/delta/? %s", oldValue.Type, oldValue.Name))
		}
	}
	return uniqueSlice
}*/
