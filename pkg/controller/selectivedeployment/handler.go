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
const noSchedule = "NoSchedule"

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
	sdCopy = setCrashListByConflicts(sdCopy, sdRaw)
	nonExistentCounter := 0
	for _, controllerDet := range sdCopy.Spec.Controller {
		err = nil
		// Get the controller defined at the selectivedeployment object
		switch strings.ToLower(controllerDet.Type) {
		case "deployment":
			_, err = t.clientset.AppsV1().Deployments(sdCopy.GetNamespace()).Get(controllerDet.Name, metav1.GetOptions{})
		case "daemonset":
			_, err = t.clientset.AppsV1().DaemonSets(sdCopy.GetNamespace()).Get(controllerDet.Name, metav1.GetOptions{})
		case "statefulset":
			_, err = t.clientset.AppsV1().StatefulSets(sdCopy.GetNamespace()).Get(controllerDet.Name, metav1.GetOptions{})
		default:
			err = nil
		}
		if err != nil {
			// In here, the errors caused by non-existent of the controller are added to crash list of the selectivedeployment object
			sdCopy = setCrashListByNonExistents(sdCopy, controllerDet)
			nonExistentCounter++
		}
	}

	// uniqueCrashList is a list without duplicate values
	uniqueCrashList := []selectivedeployment_v1.Controller{}
	for _, crash := range sdCopy.Status.Crash {
		exists := false
		for _, controllerDet := range uniqueCrashList {
			if crash.Controller.Type == controllerDet.Type && crash.Controller.Name == controllerDet.Name {
				exists = true
			}
		}
		if !exists {
			uniqueCrashList = append(uniqueCrashList, crash.Controller)
		}
	}

	// The problems and details of the desired new selectivedeployment object are described herein, and this step is the last of the error processing
	if len(uniqueCrashList) == len(sdCopy.Spec.Controller) {
		sdCopy.Status.State = failure
		// nonExistentCounter indicates the number of non-existent controller(s) already defined in the desired selectivedeployment object
		if nonExistentCounter != 0 && len(sdCopy.Status.Crash) != nonExistentCounter {
			sdCopy.Status.Message = fmt.Sprintf("%d controller(s) are already under the control of any different resource object(s) with the same type, %d controller(s) couldn't be found", (len(sdCopy.Status.Crash) - nonExistentCounter), nonExistentCounter)
		} else if nonExistentCounter != 0 && len(sdCopy.Status.Crash) == nonExistentCounter {
			sdCopy.Status.Message = "All controllers couldn't be found"
		} else {
			sdCopy.Status.Message = "All controllers are already under the control of any different resource object(s) with the same type"
		}
	} else if len(sdCopy.Status.Crash) == 0 {
		sdCopy.Status.State = success
		sdCopy.Status.Message = "SelectiveDeployment runs precisely to ensure that the actual state of the cluster matches the desired state"
	} else {
		sdCopy.Status.State = partial
		if len(sdCopy.Status.Crash) != nonExistentCounter {
			sdCopy.Status.Message = fmt.Sprintf("%d controller(s) are already under the control of any different resource object(s) with the same type", (len(sdCopy.Status.Crash) - nonExistentCounter))
		}
		if nonExistentCounter != 0 {
			sdCopy.Status.Message = fmt.Sprintf("%d controller(s) couldn't be found", nonExistentCounter)
		}
	}

	// The number of controller(s) that the selectivedeployment resource successfully controls
	sdCopy.Status.Ready = fmt.Sprintf("%d/%d", len(sdCopy.Spec.Controller)-len(uniqueCrashList), len(sdCopy.Spec.Controller))
}

// recoverSelectiveDeployments compares the crash list with the controller list and the name of selectivedeployment to recover objects affected by the selectivedeployment
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
				controllerDetStrArr := strings.Split(controllerDetStr, "?/delta/? ")
				controllerDet := selectivedeployment_v1.Controller{}
				controllerDet.Type = controllerDetStrArr[0]
				controllerDet.Name = controllerDetStrArr[1]
				if crashMatch, _ := checkCrashList(sdRow.Status.Crash, controllerDet, sdDet.name, "all"); crashMatch {
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
					controllerSelector.matchExpression = t.setFilter(sdRow.Spec.Type, sdRow.Spec.Selector, controllerSelector.matchExpression, "addOrUpdate")
					if len(controllerSelector.matchExpression.Values) > 0 {
						for _, controllerDet := range sdRow.Spec.Controller {
							if crashMatch, _ := checkCrashList(sdRow.Status.Crash, controllerDet, sdRow.GetNamespace(), "controller"); !crashMatch && controllerType == strings.ToLower(controllerDet.Type) && controllerName == controllerDet.Name {
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
func (t *SDHandler) setFilter(sdType string, sdValue []selectivedeployment_v1.Selector,
	matchExpression corev1.NodeSelectorRequirement, event string) corev1.NodeSelectorRequirement {
	matchExpression.Values = []string{}
	matchExpression.Key = "kubernetes.io/hostname"
	sdType = strings.ToLower(sdType)
	// Turn the key into the predefined form which is determined at the custom resource definition of selectivedeployment
	switch sdType {
	case "city", "state", "country", "continent":
		// If the event type is delete then we don't need to run the part below
		if event != "delete" {
			labelKeySuffix := ""
			if sdType == "state" || sdType == "country" {
				labelKeySuffix = "-iso"
			}
			labelKey := strings.ToLower(fmt.Sprintf("edge-net.io/%s%s", sdType, labelKeySuffix))
			// This gets the node list which includes the EdgeNet geolabels
			nodesRaw, err := t.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
			if err != nil {
				log.Println(err.Error())
				panic(err.Error())
			}
			// This loop allows us to process each value defined at the object of selectivedeployment resource
			for _, valueRow := range sdValue {
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
					if !nodeRow.Spec.Unschedulable && !taintBlock {
						if contains(matchExpression.Values, nodeRow.Labels["kubernetes.io/hostname"]) {
							continue
						}
						if valueRow.Value == nodeRow.Labels[labelKey] {
							matchExpression.Values = append(matchExpression.Values, nodeRow.Labels["kubernetes.io/hostname"])
							counter++
						}
						if valueRow.Count != 0 && valueRow.Count == counter {
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
				counter := 0
				err = json.Unmarshal([]byte(valueRow.Value), &polygon)
				if err != nil {
					// Will be developed further
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
										counter++
									}
								}
							}
						}
						if valueRow.Count != 0 && valueRow.Count == counter {
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

// setCrashListByConflicts compares the controllers of the selectivedeployment resource objects with those of the object in the process
// to make a list of the conflicts which guides the user to understand its faults
func setCrashListByConflicts(sdCopy *selectivedeployment_v1.SelectiveDeployment, sdRaw *selectivedeployment_v1.SelectiveDeploymentList) *selectivedeployment_v1.SelectiveDeployment {
	// The loop to process each selectivedeployment object separately
	for _, sdRow := range sdRaw.Items {
		if sdRow.GetName() != sdCopy.GetName() && sdRow.Spec.Type == sdCopy.Spec.Type && sdRow.Status.State != "" {
			for _, newController := range sdCopy.Spec.Controller {
				for _, otherObjController := range sdRow.Spec.Controller {
					if otherObjController.Type == newController.Type && otherObjController.Name == newController.Name {
						// Checks whether the crash list is empty and this crash exists in the crash list of the selectivedeployment object
						if crashMatch, _ := checkCrashList(sdRow.Status.Crash, newController, sdCopy.GetName(), "all"); !crashMatch {
							if crashMatch, _ := checkCrashList(sdCopy.Status.Crash, otherObjController, sdRow.GetName(), "all"); !crashMatch || len(sdCopy.Status.Crash) == 0 {
								crash := selectivedeployment_v1.Crash{}
								crash.Controller.Type = otherObjController.Type
								crash.Controller.Name = otherObjController.Name
								crash.Reason = sdRow.GetName()
								sdCopy.Status.Crash = append(sdCopy.Status.Crash, crash)
							}
						}
					}
				}
			}
		}
	}
	return sdCopy
}

// setCrashListByNonExistents checks whether the controller exists to put it into the list and it will be listed in case of non-existent
func setCrashListByNonExistents(sdCopy *selectivedeployment_v1.SelectiveDeployment, controllerDet selectivedeployment_v1.Controller) *selectivedeployment_v1.SelectiveDeployment {
	if crashMatch, _ := checkCrashList(sdCopy.Status.Crash, controllerDet, "nonexistent", "all"); !crashMatch {
		crash := selectivedeployment_v1.Crash{}
		crash.Controller.Type = controllerDet.Type
		crash.Controller.Name = controllerDet.Name
		crash.Reason = "nonexistent"
		sdCopy.Status.Crash = append(sdCopy.Status.Crash, crash)
	}
	return sdCopy
}

// checkCrashList compares the crash list with the given names of controller and selectivedeployment
func checkCrashList(crashList []selectivedeployment_v1.Crash, controllerDet selectivedeployment_v1.Controller, sdName string, compareType string) (bool, int) {
	exists := false
	index := -1
	for i, crash := range crashList {
		crashControllerType := crash.Controller.Type
		crashControllerName := crash.Controller.Name
		crashsdName := crash.Reason
		if compareType == "controller" {
			crashsdName = sdName
		} else if compareType == "selectivedeployment" {
			crashControllerType = controllerDet.Type
			crashControllerName = controllerDet.Name
		}
		if controllerDet.Type == crashControllerType && controllerDet.Name == crashControllerName && sdName == crashsdName {
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
