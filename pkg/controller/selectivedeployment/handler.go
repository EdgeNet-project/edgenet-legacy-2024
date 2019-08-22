package selectivedeployment

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	selectivedeployment_v1 "headnode/pkg/apis/selectivedeployment/v1alpha"
	"headnode/pkg/authorization"
	"headnode/pkg/client/clientset/versioned"
	"headnode/pkg/node"

	log "github.com/Sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init() error
	ObjectCreated(obj interface{})
	ObjectUpdated(obj interface{}, delta string)
	ObjectDeleted(obj interface{}, delta string)
	ConfigureControllers()
}

// SDHandler is a implementation of Handler
type SDHandler struct {
	clientset     *kubernetes.Clientset
	sdClientset   *versioned.Clientset
	sdDet         sdDet
	wgHandler     map[string]*sync.WaitGroup
	wgRecovery    map[string]*sync.WaitGroup
	namespaceList []string
}

// The data defined by the user to be used for node selection
type desiredFilter struct {
	nodeSelectorTerms []corev1.NodeSelectorTerm
	nodeSelectorTerm  corev1.NodeSelectorTerm
	matchExpression   corev1.NodeSelectorRequirement
}

// The data of deleted/updated object to handle operations based on the deleted/updated object
type sdDet struct {
	name            string
	namespace       string
	sdType          string
	controllerDelta []string
}

// Definitions of the state of the selectivedeployment resource (failure, partial, success)
const failure = "Failure"
const partial = "Running Partially"
const success = "Running"

// Init handles any handler initialization
func (t *SDHandler) Init() error {
	log.Info("SDHandler.Init")
	t.sdDet = sdDet{}
	t.wgHandler = make(map[string]*sync.WaitGroup)
	t.wgRecovery = make(map[string]*sync.WaitGroup)
	var err error
	t.clientset, err = authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	t.sdClientset, err = authorization.CreateSelectiveDeploymentClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	return err
}

// namespaceInit does initialization of the namespace
func (t *SDHandler) namespaceInit(namespace string) {
	if t.wgHandler[namespace] == nil || t.wgRecovery[namespace] == nil {
		var wgHandler sync.WaitGroup
		var wgRecovery sync.WaitGroup
		t.wgHandler[namespace] = &wgHandler
		t.wgRecovery[namespace] = &wgRecovery
	}
	check := false
	for _, namespaceRow := range t.namespaceList {
		if namespace == namespaceRow {
			check = true
		}
	}
	if !check {
		t.namespaceList = append(t.namespaceList, namespace)
	}
}

// ObjectCreated is called when an object is created
func (t *SDHandler) ObjectCreated(obj interface{}) {
	log.Info("SDHandler.ObjectCreated")
	// Create a copy of the selectivedeployment object to make changes on it
	sdCopy := obj.(*selectivedeployment_v1.SelectiveDeployment).DeepCopy()
	t.namespaceInit(sdCopy.GetNamespace())
	t.wgHandler[sdCopy.GetNamespace()].Add(1)
	defer func() {
		// Sleep to prevent extra resource consumption by running ConfigureControllers
		time.Sleep(100 * time.Millisecond)
		t.wgHandler[sdCopy.GetNamespace()].Done()
	}()
	t.setControllerFilter(sdCopy, "", "create")
}

// ObjectUpdated is called when an object is updated
func (t *SDHandler) ObjectUpdated(obj interface{}, delta string) {
	log.Info("SDHandler.ObjectUpdated")
	// Create a copy of the selectivedeployment object to make changes on it
	sdCopy := obj.(*selectivedeployment_v1.SelectiveDeployment).DeepCopy()
	t.namespaceInit(sdCopy.GetNamespace())
	t.wgHandler[sdCopy.GetNamespace()].Add(1)
	defer func() {
		time.Sleep(100 * time.Millisecond)
		t.wgHandler[sdCopy.GetNamespace()].Done()
	}()
	t.setControllerFilter(sdCopy, delta, "update")
}

// ObjectDeleted is called when an object is deleted
func (t *SDHandler) ObjectDeleted(obj interface{}, delta string) {
	log.Info("SDHandler.ObjectDeleted")
	// Put the required data of the deleted object into variables
	objectDelta := strings.Split(delta, "-?delta?- ")
	t.sdDet = sdDet{
		name:            objectDelta[0],
		namespace:       objectDelta[1],
		sdType:          objectDelta[2],
		controllerDelta: strings.Split(objectDelta[3], "/?delta?/ "),
	}

	t.namespaceInit(t.sdDet.namespace)
	t.wgHandler[t.sdDet.namespace].Add(1)
	defer func() {
		time.Sleep(100 * time.Millisecond)
		t.wgHandler[t.sdDet.namespace].Done()
	}()
	// Detect and recover the selectivedeployment resource objects which are prevented by the this object from taking control of the controller(s)
	t.recoverSelectiveDeployments(t.sdDet)
}

// setControllerFilter used by ObjectCreated, ObjectUpdated, and recoverSelectiveDeployments functions
func (t *SDHandler) setControllerFilter(sdCopy *selectivedeployment_v1.SelectiveDeployment, delta string, eventType string) {
	// Flush the status
	sdCopy.Status = selectivedeployment_v1.SelectiveDeploymentStatus{}
	// Put the differences between the old and the new objects into variables
	t.sdDet = sdDet{
		name:      sdCopy.GetName(),
		namespace: sdCopy.GetNamespace(),
		sdType:    sdCopy.Spec.Type,
	}
	if delta != "" {
		t.sdDet.controllerDelta = strings.Split(delta, "/?delta?/ ")
	}

	if eventType != "recover" && eventType != "create" {
		defer t.recoverSelectiveDeployments(t.sdDet)
	} else if eventType == "recover" {
		t.wgRecovery[t.sdDet.namespace].Add(1)
		defer func() {
			time.Sleep(100 * time.Millisecond)
			t.wgRecovery[t.sdDet.namespace].Done()
		}()
	}
	defer t.sdClientset.EdgenetV1alpha().SelectiveDeployments(sdCopy.GetNamespace()).UpdateStatus(sdCopy)

	sdRaw, err := t.sdClientset.EdgenetV1alpha().SelectiveDeployments(sdCopy.GetNamespace()).List(metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	// Reveal conflicts by comparing selectivedeployment resource objects with the object in process
	sdCopy = setReasonListOfConflicts(sdCopy, sdRaw)
	counter := 0
	for _, controllerDet := range sdCopy.Spec.Controller {
		err = nil
		// Get the controller defined at the selectivedeployment object
		switch controllerDet[0] {
		case "deployment", "deployments":
			_, err = t.clientset.AppsV1().Deployments(sdCopy.GetNamespace()).Get(controllerDet[1], metav1.GetOptions{})
		case "daemonset", "daemonsets":
			_, err = t.clientset.AppsV1().DaemonSets(sdCopy.GetNamespace()).Get(controllerDet[1], metav1.GetOptions{})
		case "statefulset", "statefulsets":
			_, err = t.clientset.AppsV1().StatefulSets(sdCopy.GetNamespace()).Get(controllerDet[1], metav1.GetOptions{})
		default:
			err = nil
		}
		if err != nil {
			// In here, the errors caused by non-existent of the controller are added to reason list of the selectivedeployment object
			sdCopy = setReasonListOfNonExistents(sdCopy, controllerDet)
			counter++
		}
	}

	// Controller list without duplicate values
	controllerList := [][]string{}
	for _, reason := range sdCopy.Status.Reason {
		exists := false
		for _, controllerDet := range controllerList {
			if reason[0] == controllerDet[0] && reason[1] == controllerDet[1] {
				exists = true
			}
		}
		if !exists {
			controllerList = append(controllerList, []string{reason[0], reason[1]})
		}
	}

	// The problems and details of the desired new selectivedeployment object are described herein, and this step is the last of the error processing
	if len(controllerList) == len(sdCopy.Spec.Controller) {
		sdCopy.Status.State = failure
		sdCopy.Status.Message = "All controllers are already under the control of any different resource object(s) with the same type"
	} else if len(sdCopy.Status.Reason) == 0 {
		sdCopy.Status.State = success
		sdCopy.Status.Message = "SelectiveDeployment runs precisely to ensure that the actual state of the cluster matches the desired state"
	} else {
		sdCopy.Status.State = partial
		sdCopy.Status.Message = "Some controllers are already under the control of any different resource object(s) with the same type"
	}
	// Counter indicates the number of non-existent controller(s) already defined in the desired selectivedeployment object
	if counter != 0 {
		sdCopy.Status.Message = fmt.Sprintf("%s, %d controller(s) couldn't be found", sdCopy.Status.Message, counter)
	}
	// The number of controller(s) that the selectivedeployment resource successfully controls
	sdCopy.Status.Ready = fmt.Sprintf("%d/%d", len(sdCopy.Spec.Controller)-len(controllerList), len(sdCopy.Spec.Controller))
}

// recoverSelectiveDeployments compares the reason list with the controller list and the name of selectivedeployment to recover objects affected by the selectivedeployment
// object. The controller delta list contains the name of controllers removed from the selectivedeployment object by updating or deleting it
func (t *SDHandler) recoverSelectiveDeployments(sdDet sdDet) {
	sdRaw, err := t.sdClientset.EdgenetV1alpha().SelectiveDeployments(sdDet.namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	for _, sdRow := range sdRaw.Items {
		if sdRow.GetName() != sdDet.name && sdRow.Spec.Type == sdDet.sdType && sdRow.Status.State != "" {
			for _, controllerDetStr := range sdDet.controllerDelta {
				controllerDet := strings.Split(controllerDetStr, "?/delta/? ")
				if reasonMatch, _ := checkReasonList(sdRow.Status.Reason, controllerDet, sdDet.name, "all"); reasonMatch {
					selectivedeployment, err := t.sdClientset.EdgenetV1alpha().SelectiveDeployments(sdRow.GetNamespace()).Get(sdRow.GetName(), metav1.GetOptions{})
					if err == nil {
						t.setControllerFilter(selectivedeployment, "", "recover")
						t.wgRecovery[sdDet.namespace].Wait()
						time.Sleep(100 * time.Millisecond)
					}
				}
			}
		}
	}
}

// ConfigureControllers configures the controllers by selectivedeployments to match the desired state users supplied
func (t *SDHandler) ConfigureControllers() {
	log.Info("ConfigureControllers: start")

	configurationList := t.namespaceList
	t.namespaceList = []string{}
	for _, namespace := range configurationList {
		t.wgHandler[namespace].Wait()
		t.wgRecovery[namespace].Wait()
		time.Sleep(1200 * time.Millisecond)

		controllerSelector := desiredFilter{}

		sdRaw, err := t.sdClientset.EdgenetV1alpha().SelectiveDeployments(namespace).List(metav1.ListOptions{})
		if err != nil {
			log.Println(err.Error())
			panic(err.Error())
		}

		setFilterOfController := func(controllerName string, controllerType string, podSpec corev1.PodSpec) bool {
			// Clear the variables involved with node selection
			controllerSelector.nodeSelectorTerms = []corev1.NodeSelectorTerm{}
			for _, sdRow := range sdRaw.Items {
				if sdRow.Status.State == success || sdRow.Status.State == partial {
					controllerSelector.nodeSelectorTerm = corev1.NodeSelectorTerm{}
					controllerSelector.matchExpression.Operator = "In"
					controllerSelector.matchExpression = t.setFilter(sdRow.Spec.Type, sdRow.Spec.Value, controllerSelector.matchExpression, "addOrUpdate")
					if len(controllerSelector.matchExpression.Values) > 0 {
						for _, controllerDet := range sdRow.Spec.Controller {
							if reasonMatch, _ := checkReasonList(sdRow.Status.Reason, controllerDet, sdRow.GetNamespace(), "controller"); !reasonMatch && controllerType == controllerDet[0] && controllerName == controllerDet[1] {
								controllerSelector.nodeSelectorTerm.MatchExpressions = append(controllerSelector.nodeSelectorTerm.MatchExpressions, controllerSelector.matchExpression)
								controllerSelector.nodeSelectorTerms = append(controllerSelector.nodeSelectorTerms, controllerSelector.nodeSelectorTerm)
							}
						}
					}
				}
			}
			status := false
			if podSpec.Affinity != nil && podSpec.Affinity.NodeAffinity != nil && podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
				if !reflect.DeepEqual(podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, controllerSelector.nodeSelectorTerms) {
					status = true
				}
			} else if len(controllerSelector.nodeSelectorTerms) > 0 {
				status = true
			}
			return status
		}
		updateController := func(controllerRow interface{}) {
			// Set the new affinity configuration in the controller and update that
			nodeAffinity := &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: controllerSelector.nodeSelectorTerms,
					},
				},
			}
			if len(controllerSelector.nodeSelectorTerms) <= 0 {
				nodeAffinity.Reset()
			}
			switch controllerObj := controllerRow.(type) {
			case appsv1.Deployment:
				controllerCopy := controllerObj.DeepCopy()
				controllerCopy.Spec.Template.Spec.Affinity = nodeAffinity
				log.Printf("%s/Deployment/%s: %s", controllerCopy.GetNamespace(), controllerCopy.GetName(), nodeAffinity)
				t.clientset.AppsV1().Deployments(namespace).Update(controllerCopy)
			case appsv1.DaemonSet:
				controllerCopy := controllerObj.DeepCopy()
				controllerCopy.Spec.Template.Spec.Affinity = nodeAffinity
				log.Printf("%s/DaemonSet/%s: %s", controllerCopy.GetNamespace(), controllerCopy.GetName(), nodeAffinity)
				t.clientset.AppsV1().DaemonSets(namespace).Update(controllerCopy)
			case appsv1.StatefulSet:
				controllerCopy := controllerObj.DeepCopy()
				controllerCopy.Spec.Template.Spec.Affinity = nodeAffinity
				log.Printf("%s/StatefulSet/%s: %s", controllerCopy.GetNamespace(), controllerCopy.GetName(), nodeAffinity)
				t.clientset.AppsV1().StatefulSets(namespace).Update(controllerCopy)
			}
		}
		configureController := func(controllerList interface{}) {
			switch controllerRaw := controllerList.(type) {
			case *appsv1.DeploymentList:
				// Sync the desired filter fields according to the object
				controllerSelector = desiredFilter{}
				for _, controllerRow := range controllerRaw.Items {
					if changeStatus := setFilterOfController(controllerRow.GetName(), "deployment", controllerRow.Spec.Template.Spec); changeStatus {
						updateController(controllerRow)
					}
				}
			case *appsv1.DaemonSetList:
				// Sync the desired filter fields according to the object
				controllerSelector = desiredFilter{}
				for _, controllerRow := range controllerRaw.Items {
					if changeStatus := setFilterOfController(controllerRow.GetName(), "daemonset", controllerRow.Spec.Template.Spec); changeStatus {
						updateController(controllerRow)
					}
				}
			case *appsv1.StatefulSetList:
				// Sync the desired filter fields according to the object
				controllerSelector = desiredFilter{}
				for _, controllerRow := range controllerRaw.Items {
					if changeStatus := setFilterOfController(controllerRow.GetName(), "statefulset", controllerRow.Spec.Template.Spec); changeStatus {
						updateController(controllerRow)
					}
				}
			}
		}

		deploymentRaw, err := t.clientset.AppsV1().Deployments(namespace).List(metav1.ListOptions{})
		if err != nil {
			log.Println(err.Error())
			panic(err.Error())
		}
		configureController(deploymentRaw)
		time.Sleep(100 * time.Millisecond)
		daemonsetRaw, err := t.clientset.AppsV1().DaemonSets(namespace).List(metav1.ListOptions{})
		if err != nil {
			log.Println(err.Error())
			panic(err.Error())
		}
		configureController(daemonsetRaw)
		time.Sleep(100 * time.Millisecond)
		statefulsetRaw, err := t.clientset.AppsV1().StatefulSets(namespace).List(metav1.ListOptions{})
		if err != nil {
			log.Println(err.Error())
			panic(err.Error())
		}
		configureController(statefulsetRaw)
	}
}

// setFilter generates the values in the predefined form and puts those into the node selection fields of the selectivedeployment object
func (t *SDHandler) setFilter(sdType string, sdValue [][]string,
	matchExpression corev1.NodeSelectorRequirement, event string) corev1.NodeSelectorRequirement {
	matchExpression.Values = []string{}
	matchExpression.Key = "kubernetes.io/hostname"
	// Turn the key into the predefined form which is determined at the custom resource definition of selectivedeployment
	switch sdType {
	case "city", "state", "country", "continent":
		// If the event type is delete then we don't need to run the part below
		if event != "delete" {
			labelKeySuffix := ""
			if sdType == "state" || sdType == "country" {
				labelKeySuffix = "-iso"
			}
			labelKey := fmt.Sprintf("edge-net.io/%s%s", sdType, labelKeySuffix)
			// This gets the node list which includes the EdgeNet geolabels
			nodesRaw, err := t.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
			if err != nil {
				log.Println(err.Error())
				panic(err.Error())
			}
			// This loop allows us to process each value defined at the object of selectivedeployment resource
			for _, valueRow := range sdValue {
				count := 0
				limit, err := strconv.Atoi(valueRow[1])
				if err != nil {
					continue
				}

				// The loop to process each node separately
			cityNodeLoop:
				for _, nodeRow := range nodesRaw.Items {
					taintBlock := false
					for _, taint := range nodeRow.Spec.Taints {
						if (taint.Key == "node-role.kubernetes.io/master" && taint.Effect == "NoSchedule") ||
							(taint.Key == "node.kubernetes.io/unschedulable" && taint.Effect == "NoSchedule") {
							taintBlock = true
						}
					}
					if !nodeRow.Spec.Unschedulable && !taintBlock {
						if contains(matchExpression.Values, nodeRow.Labels["kubernetes.io/hostname"]) {
							continue
						}
						if valueRow[0] == nodeRow.Labels[labelKey] {
							matchExpression.Values = append(matchExpression.Values, nodeRow.Labels["kubernetes.io/hostname"])
							count++
						}
						if limit != 0 && limit == count {
							break cityNodeLoop
						}
					}
				}
			}
		}
	case "polygon":
		// If the event type is delete then we don't need to run the GeoFence functions
		if event != "delete" {
			// If the selectivedeployment key is polygon then certain calculations like geofence need to be done
			// for being had the list of nodes that the pods will be deployed on according to the desired state.
			// This gets the node list which includes the EdgeNet geolabels
			nodesRaw, err := t.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
			if err != nil {
				log.Println(err.Error())
				panic(err.Error())
			}

			var polygon [][]float64
			// This loop allows us to process each polygon defined at the object of selectivedeployment resource
			for _, valueRow := range sdValue {
				count := 0
				limit, err := strconv.Atoi(valueRow[1])
				if err != nil {
					continue
				}

				err = json.Unmarshal([]byte(valueRow[0]), &polygon)
				if err != nil {
					panic(err)
				}
				// The loop to process each node separately
			polyNodeLoop:
				for _, nodeRow := range nodesRaw.Items {
					taintBlock := false
					for _, taint := range nodeRow.Spec.Taints {
						if (taint.Key == "node-role.kubernetes.io/master" && taint.Effect == "NoSchedule") ||
							(taint.Key == "node.kubernetes.io/unschedulable" && taint.Effect == "NoSchedule") {
							taintBlock = true
						}
					}
					if !nodeRow.Spec.Unschedulable && !taintBlock {
						if nodeRow.Labels["edge-net.io/lon"] != "" && nodeRow.Labels["edge-net.io/lat"] != "" {
							if contains(matchExpression.Values, nodeRow.Labels["kubernetes.io/hostname"]) {
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
									if status {
										matchExpression.Values = append(matchExpression.Values, nodeRow.Labels["kubernetes.io/hostname"])
										count++
									}
								}
							}
						}
						if limit != 0 && limit == count {
							break polyNodeLoop
						}
					}
				}
			}
		}
	default:
		matchExpression.Key = ""
	}

	return matchExpression
}

// setReasonListOfConflicts compares the controllers of the selectivedeployment resource objects with those of the object in the process
// to make a list of the conflicts which guides the user to understand its faults
func setReasonListOfConflicts(sdCopy *selectivedeployment_v1.SelectiveDeployment, sdRaw *selectivedeployment_v1.SelectiveDeploymentList) *selectivedeployment_v1.SelectiveDeployment {
	// The loop to process each selectivedeployment object separately
	for _, sdRow := range sdRaw.Items {
		if sdRow.GetName() != sdCopy.GetName() && sdRow.Spec.Type == sdCopy.Spec.Type && sdRow.Status.State != "" {
			for _, newController := range sdCopy.Spec.Controller {
				for _, otherObjController := range sdRow.Spec.Controller {
					if otherObjController[0] == newController[0] && otherObjController[1] == newController[1] {
						// Checks whether the reason list is empty and this reason exists in the reason list of the selectivedeployment object
						if reasonMatch, _ := checkReasonList(sdRow.Status.Reason, newController, sdCopy.GetName(), "all"); !reasonMatch {
							if reasonMatch, _ := checkReasonList(sdCopy.Status.Reason, otherObjController, sdRow.GetName(), "all"); !reasonMatch || len(sdCopy.Status.Reason) == 0 {
								conflictReason := []string{otherObjController[0], otherObjController[1], sdRow.GetName()}
								sdCopy.Status.Reason = append(sdCopy.Status.Reason, conflictReason)
							}
						}
					}
				}
			}
		}
	}
	return sdCopy
}

// setReasonListOfNonExistents checks whether the controller exists to put it into the list and it will be listed in case of non-existent
func setReasonListOfNonExistents(sdCopy *selectivedeployment_v1.SelectiveDeployment, controllerDet []string) *selectivedeployment_v1.SelectiveDeployment {
	if reasonMatch, _ := checkReasonList(sdCopy.Status.Reason, controllerDet, "nonexistent", "all"); !reasonMatch {
		conflictReason := []string{controllerDet[0], controllerDet[1], "nonexistent"}
		sdCopy.Status.Reason = append(sdCopy.Status.Reason, conflictReason)
	}
	return sdCopy
}

// checkReasonList compares the reason list with the given names of controller and selectivedeployment
func checkReasonList(reasonList [][]string, controllerDet []string, sdName string, compareType string) (bool, int) {
	exists := false
	index := -1
	for i, reason := range reasonList {
		reasonControllerType := reason[0]
		reasonControllerName := reason[1]
		reasonsdName := reason[2]
		if compareType == "controller" {
			reasonsdName = sdName
		} else if compareType == "selectivedeployment" {
			reasonControllerType = controllerDet[0]
			reasonControllerName = controllerDet[1]
		}
		if controllerDet[0] == reasonControllerType && controllerDet[1] == reasonControllerName && sdName == reasonsdName {
			exists = true
			index = i
		}
	}
	return exists, index
}

// Return whether slice contains value
func contains(slice []string, value string) bool {
	for _, ele := range slice {
		if value == ele {
			return true
		}
	}
	return false
}
