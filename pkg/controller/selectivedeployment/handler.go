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
	ConfigureDeployments()
}

// SDHandler is a implementation of Handler
type SDHandler struct {
	clientset     *kubernetes.Clientset
	sdClientset   *versioned.Clientset
	inProcess     bool
	desiredFilter desiredFilter
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
	deploymentDelta []string
}

// Definitions of the state of the selectivedeployment resource (failure, partial, success)
const failure = "Failure"
const partial = "Running Partially"
const success = "Running"

// Init handles any handler initialization
func (t *SDHandler) Init() error {
	log.Info("SDHandler.Init")
	t.desiredFilter = desiredFilter{}
	t.sdDet = sdDet{}
	t.inProcess = false
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
		// Sleep to prevent extra resource consumption by running ConfigureDeployments
		time.Sleep(50 * time.Millisecond)
		t.wgHandler[sdCopy.GetNamespace()].Done()
	}()
	t.setDeploymentFilter(sdCopy, "", "create")
}

// ObjectUpdated is called when an object is updated
func (t *SDHandler) ObjectUpdated(obj interface{}, delta string) {
	log.Info("SDHandler.ObjectUpdated")
	// Create a copy of the selectivedeployment object to make changes on it
	sdCopy := obj.(*selectivedeployment_v1.SelectiveDeployment).DeepCopy()
	t.namespaceInit(sdCopy.GetNamespace())
	t.wgHandler[sdCopy.GetNamespace()].Add(1)
	defer func() {
		time.Sleep(50 * time.Millisecond)
		t.wgHandler[sdCopy.GetNamespace()].Done()
	}()
	t.setDeploymentFilter(sdCopy, delta, "update")
}

// ObjectDeleted is called when an object is deleted
func (t *SDHandler) ObjectDeleted(obj interface{}, delta string) {
	log.Info("SDHandler.ObjectDeleted")
	// Put the required data of the deleted object into variables
	objectDelta := strings.Split(delta, "- ")
	t.sdDet = sdDet{
		name:            objectDelta[0],
		namespace:       objectDelta[1],
		sdType:          objectDelta[2],
		deploymentDelta: strings.Split(objectDelta[3], "/ "),
	}

	t.namespaceInit(t.sdDet.namespace)
	t.wgHandler[t.sdDet.namespace].Add(1)
	defer func() {
		time.Sleep(50 * time.Millisecond)
		t.wgHandler[t.sdDet.namespace].Done()
	}()
	// Detect and recover the selectivedeployment resource objects which are prevented by the this object from taking control of the deployments
	t.recoverSelectivedeployments(t.sdDet)
}

// setDeploymentFilter used by ObjectCreated, ObjectUpdated, and recoverSelectivedeployments functions
func (t *SDHandler) setDeploymentFilter(sdCopy *selectivedeployment_v1.SelectiveDeployment, delta string, eventType string) {
	// Flush the status
	sdCopy.Status = selectivedeployment_v1.SelectiveDeploymentStatus{}
	// Put the differences between the old and the new objects into variables
	t.sdDet = sdDet{
		name:      sdCopy.GetName(),
		namespace: sdCopy.GetNamespace(),
		sdType:    sdCopy.Spec.Type,
	}
	if delta != "" {
		t.sdDet.deploymentDelta = strings.Split(delta, "/ ")
	}

	if eventType != "recover" && eventType != "create" {
		defer t.recoverSelectivedeployments(t.sdDet)
	} else if eventType == "recover" {
		t.wgRecovery[t.sdDet.namespace].Add(1)
		defer func() {
			time.Sleep(50 * time.Millisecond)
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
	for _, deploymentName := range sdCopy.Spec.Deployment {
		// Get the deployment defined at the selectivedeployment object
		_, err := t.clientset.AppsV1().Deployments(sdCopy.GetNamespace()).Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			// In here, the errors caused by non-existent of the deployment are added to reason list of the selectivedeployment object
			sdCopy = setReasonListOfNonExistents(sdCopy, deploymentName)
			counter++
		}
	}

	// Deployment list without duplicate values
	deploymentList := []string{}
	for _, reason := range sdCopy.Status.Reason {
		exists := false
		for _, deployment := range deploymentList {
			if reason[0] == deployment {
				exists = true
			}
		}
		if !exists {
			deploymentList = append(deploymentList, reason[0])
		}
	}

	// The problems and details of the desired new selectivedeployment object are described herein, and this step is the last of the error processing
	if len(deploymentList) == len(sdCopy.Spec.Deployment) {
		sdCopy.Status.State = failure
		sdCopy.Status.Message = "All deployments are already under the control of any different resource object(s) with the same type"
	} else if len(sdCopy.Status.Reason) == 0 {
		sdCopy.Status.State = success
		sdCopy.Status.Message = "SelectiveDeployment runs precisely to ensure that the actual state of the cluster matches the desired state"
	} else {
		sdCopy.Status.State = partial
		sdCopy.Status.Message = "Some deployments are already under the control of any different resource object(s) with the same type"
	}
	// Counter indicates the number of non-existent deployment already defined in the desired selectivedeployment object
	if counter != 0 {
		sdCopy.Status.Message = fmt.Sprintf("%s, %d deployment(s) couldn't be found", sdCopy.Status.Message, counter)
	}
	// The number of deployments that the selectivedeployment resource successfully controls
	sdCopy.Status.Ready = fmt.Sprintf("%d/%d", len(sdCopy.Spec.Deployment)-len(deploymentList), len(sdCopy.Spec.Deployment))
}

// recoverSelectivedeployments compares the reason list with the deployment list and the name of selectivedeployment to recover objects affected by the selectivedeployment
// object. The deployment delta list contains the name of deployments removed from the selectivedeployment object by updating or deleting it
func (t *SDHandler) recoverSelectivedeployments(sdDet sdDet) {
	sdRaw, err := t.sdClientset.EdgenetV1alpha().SelectiveDeployments(sdDet.namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	for _, sdRow := range sdRaw.Items {
		if sdRow.GetName() != sdDet.name && sdRow.Spec.Type == sdDet.sdType && sdRow.Status.State != "" {
			for _, deployment := range sdDet.deploymentDelta {
				if reasonMatch, _ := checkReasonList(sdRow.Status.Reason, deployment, sdDet.name, "all"); reasonMatch {
					selectivedeployment, err := t.sdClientset.EdgenetV1alpha().SelectiveDeployments(sdRow.GetNamespace()).Get(sdRow.GetName(), metav1.GetOptions{})
					if err == nil {
						t.setDeploymentFilter(selectivedeployment, "", "recover")
						t.wgRecovery[sdDet.namespace].Wait()
						time.Sleep(50 * time.Millisecond)
					}
				}
			}
		}
	}
}

// ConfigureDeployments configures the deployments by selectivedeployments to match the desired state users supplied
func (t *SDHandler) ConfigureDeployments() {
	log.Info("ConfigureDeployments: start")
	configurationList := t.namespaceList
	t.namespaceList = []string{}
	for _, namespace := range configurationList {
		t.wgHandler[namespace].Wait()
		t.wgRecovery[namespace].Wait()
		time.Sleep(1200 * time.Millisecond)

		sdRaw, err := t.sdClientset.EdgenetV1alpha().SelectiveDeployments(namespace).List(metav1.ListOptions{})
		if err != nil {
			log.Println(err.Error())
			panic(err.Error())
		}
		deploymentRaw, err := t.clientset.AppsV1().Deployments(namespace).List(metav1.ListOptions{})
		if err != nil {
			log.Println(err.Error())
			panic(err.Error())
		}

		// Sync the desired filter fields according to the object
		t.desiredFilter = desiredFilter{}
		for _, deploymentRow := range deploymentRaw.Items {
			// Clear the variables involved with node selection
			t.desiredFilter.nodeSelectorTerms = []corev1.NodeSelectorTerm{}
			for _, sdRow := range sdRaw.Items {
				if sdRow.Status.State == success || sdRow.Status.State == partial {
					t.desiredFilter.nodeSelectorTerm = corev1.NodeSelectorTerm{}
					t.desiredFilter.matchExpression.Operator = "In"
					t.desiredFilter.matchExpression = t.setFilter(sdRow.Spec.Type, sdRow.Spec.Value, t.desiredFilter.matchExpression, "addOrUpdate")
					for _, deployment := range sdRow.Spec.Deployment {
						if reasonMatch, _ := checkReasonList(sdRow.Status.Reason, deployment, sdRow.GetNamespace(), "deployment"); !reasonMatch && deploymentRow.GetName() == deployment {
							t.desiredFilter.nodeSelectorTerm.MatchExpressions = append(t.desiredFilter.nodeSelectorTerm.MatchExpressions, t.desiredFilter.matchExpression)
							t.desiredFilter.nodeSelectorTerms = append(t.desiredFilter.nodeSelectorTerms, t.desiredFilter.nodeSelectorTerm)
						}
					}
				}
			}

			updateDeployment := func(deploymentRow appsv1.Deployment) {
				deploymentCopy := deploymentRow.DeepCopy()
				if len(t.desiredFilter.nodeSelectorTerms) > 0 {
					// Set the new affinity configuration in the deployment and update the deployment
					deploymentCopy.Spec.Template.Spec.Affinity = &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: t.desiredFilter.nodeSelectorTerms,
							},
						},
					}
					log.Printf("%s/%s: %s", deploymentCopy.GetNamespace(), deploymentCopy.GetName(), deploymentCopy.Spec.Template.Spec.Affinity.NodeAffinity.
						RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)
				} else {
					log.Printf("%s/%s: No Expressions", deploymentCopy.GetNamespace(), deploymentCopy.GetName())
					deploymentCopy.Spec.Template.Spec.Affinity.Reset()
				}
				t.clientset.AppsV1().Deployments(namespace).Update(deploymentCopy)
			}

			if deploymentRow.Spec.Template.Spec.Affinity != nil &&
				deploymentRow.Spec.Template.Spec.Affinity.NodeAffinity != nil &&
				deploymentRow.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
				if !reflect.DeepEqual(deploymentRow.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, t.desiredFilter.nodeSelectorTerms) {
					updateDeployment(deploymentRow)
				}
			} else if len(t.desiredFilter.nodeSelectorTerms) > 0 {
				updateDeployment(deploymentRow)
			}
		}
	}
}

// setFilter generates the values in the predefined form and puts those into the node selection fields of the selectivedeployment object
func (t *SDHandler) setFilter(sdType string, sdValue []string,
	matchExpression corev1.NodeSelectorRequirement, event string) corev1.NodeSelectorRequirement {
	// Turn the key into the predefined form which is determined at the custom resource definition of selectivedeployment
	matchExpression.Values = sdValue
	switch sdType {
	case "city":
		matchExpression.Key = "edge-net.io/city"
	case "state":
		matchExpression.Key = "edge-net.io/state-iso"
	case "country":
		matchExpression.Key = "edge-net.io/country-iso"
	case "continent":
		matchExpression.Key = "edge-net.io/continent"
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
			matchExpression.Values = []string{}
			// The loop to process each node separately
			for _, nodeRow := range nodesRaw.Items {
				// Because of alphanumeric limitations of Kubernetes on the labels we use "w", "e", "n", and "s" prefixes
				// at the labels of latitude and longitude. Here is the place those prefixes are dropped away.
				lonStr := nodeRow.Labels["edge-net.io/lon"]
				lonStr = string(lonStr[1:])
				latStr := nodeRow.Labels["edge-net.io/lat"]
				latStr = string(latStr[1:])
				if lon, err := strconv.ParseFloat(lonStr, 64); err == nil {
					if lat, err := strconv.ParseFloat(latStr, 64); err == nil {
						// This loop allows us to process each polygon defined at the object of selectivedeployment resource
						for _, polygonRow := range sdValue {
							err = json.Unmarshal([]byte(polygonRow), &polygon)
							if err != nil {
								panic(err)
							}
							// boundbox is a rectangle which provides to check whether the point is inside polygon
							// without taking all point of the polygon into consideration
							boundbox := node.Boundbox(polygon)
							status := node.GeoFence(boundbox, polygon, lon, lat)
							if status {
								matchExpression.Values = append(matchExpression.Values, nodeRow.Labels["kubernetes.io/hostname"])
							}
						}
					}
				}
			}
		}
		matchExpression.Key = "kubernetes.io/hostname"
	default:
		matchExpression.Key = ""
	}

	return matchExpression
}

// setReasonListOfConflicts compares the deployments of the selectivedeployment resource objects with those of the object in the process
// to make a list of the conflicts which guides the user to understand its faults
func setReasonListOfConflicts(sdCopy *selectivedeployment_v1.SelectiveDeployment, sdRaw *selectivedeployment_v1.SelectiveDeploymentList) *selectivedeployment_v1.SelectiveDeployment {
	// The loop to process each selectivedeployment object separately
	for _, sdRow := range sdRaw.Items {
		if sdRow.GetName() != sdCopy.GetName() && sdRow.Spec.Type == sdCopy.Spec.Type && sdRow.Status.State != "" {
			for _, newDeployment := range sdCopy.Spec.Deployment {
				for _, otherObjDeployment := range sdRow.Spec.Deployment {
					if otherObjDeployment == newDeployment {
						// Checks whether the reason list is empty and this reason exists in the reason list of the selectivedeployment object
						if reasonMatch, _ := checkReasonList(sdRow.Status.Reason, newDeployment, sdCopy.GetName(), "all"); !reasonMatch {
							if reasonMatch, _ := checkReasonList(sdCopy.Status.Reason, otherObjDeployment, sdRow.GetName(), "all"); !reasonMatch || len(sdCopy.Status.Reason) == 0 {
								conflictReason := []string{otherObjDeployment, sdRow.GetName()}
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

// setReasonListOfNonExistents checks whether the deployment exists to put it into the list and it will be listed in case of non-existent
func setReasonListOfNonExistents(sdCopy *selectivedeployment_v1.SelectiveDeployment, deploymentName string) *selectivedeployment_v1.SelectiveDeployment {
	if reasonMatch, _ := checkReasonList(sdCopy.Status.Reason, deploymentName, "nonexistent", "all"); !reasonMatch {
		conflictReason := []string{deploymentName, "nonexistent"}
		sdCopy.Status.Reason = append(sdCopy.Status.Reason, conflictReason)
	}
	return sdCopy
}

// checkReasonList compares the reason list with the given names of deployment and selectivedeployment
func checkReasonList(reasonList [][]string, deployment string, selectivedeployment string, compareType string) (bool, int) {
	exists := false
	index := -1
	for i, reason := range reasonList {
		reasonDeployment := reason[0]
		reasonSelectivedeployment := reason[1]
		if compareType == "deployment" {
			reasonSelectivedeployment = selectivedeployment
		} else if compareType == "selectivedeployment" {
			reasonDeployment = deployment
		}
		if deployment == reasonDeployment && selectivedeployment == reasonSelectivedeployment {
			exists = true
			index = i
		}
	}
	return exists, index
}
