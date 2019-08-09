package selectivedeployment

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	geolocation_v1 "headnode/pkg/apis/geolocation/v1alpha"
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

// GeoHandler is a implementation of Handler
type GeoHandler struct {
	clientset            *kubernetes.Clientset
	geolocationClientset *versioned.Clientset
	inProcess            bool
	desiredFilter        desiredFilter
	geolocationDet       geolocationDet
	wgHandler            map[string]*sync.WaitGroup
	wgRecovery           map[string]*sync.WaitGroup
	namespaceList        []string
}

// The data defined by the user to be used for node selection
type desiredFilter struct {
	nodeSelectorTerms []corev1.NodeSelectorTerm
	nodeSelectorTerm  corev1.NodeSelectorTerm
	matchExpression   corev1.NodeSelectorRequirement
}

// The data of deleted/updated object to handle operations based on the deleted/updated object
type geolocationDet struct {
	name            string
	namespace       string
	geoType         string
	deploymentDelta []string
}

// Definitions of the state of the geolocation resource (failure, partial, success)
const failure = "Failure"
const partial = "Running Partially"
const success = "Running"

// Init handles any handler initialization
func (t *GeoHandler) Init() error {
	log.Info("GeoHandler.Init")
	t.desiredFilter = desiredFilter{}
	t.geolocationDet = geolocationDet{}
	t.inProcess = false
	t.wgHandler = make(map[string]*sync.WaitGroup)
	t.wgRecovery = make(map[string]*sync.WaitGroup)
	var err error
	t.clientset, err = authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	t.geolocationClientset, err = authorization.CreateGeoLocationClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	return err
}

// namespaceInit does initialization of the namespace
func (t *GeoHandler) namespaceInit(namespace string) {
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
func (t *GeoHandler) ObjectCreated(obj interface{}) {
	log.Info("GeoHandler.ObjectCreated")
	// Create a copy of the geolocation object to make changes on it
	geolocationCopy := obj.(*geolocation_v1.GeoLocation).DeepCopy()
	t.namespaceInit(geolocationCopy.GetNamespace())
	t.wgHandler[geolocationCopy.GetNamespace()].Add(1)
	defer func() {
		// Sleep to prevent extra resource consumption by running ConfigureDeployments
		time.Sleep(50 * time.Millisecond)
		t.wgHandler[geolocationCopy.GetNamespace()].Done()
	}()
	t.setDeploymentFilter(geolocationCopy, "", "create")
}

// ObjectUpdated is called when an object is updated
func (t *GeoHandler) ObjectUpdated(obj interface{}, delta string) {
	log.Info("GeoHandler.ObjectUpdated")
	// Create a copy of the geolocation object to make changes on it
	geolocationCopy := obj.(*geolocation_v1.GeoLocation).DeepCopy()
	t.namespaceInit(geolocationCopy.GetNamespace())
	t.wgHandler[geolocationCopy.GetNamespace()].Add(1)
	defer func() {
		time.Sleep(50 * time.Millisecond)
		t.wgHandler[geolocationCopy.GetNamespace()].Done()
	}()
	t.setDeploymentFilter(geolocationCopy, delta, "update")
}

// ObjectDeleted is called when an object is deleted
func (t *GeoHandler) ObjectDeleted(obj interface{}, delta string) {
	log.Info("GeoHandler.ObjectDeleted")
	// Put the required data of the deleted object into variables
	objectDelta := strings.Split(delta, "- ")
	t.geolocationDet = geolocationDet{
		name:            objectDelta[0],
		namespace:       objectDelta[1],
		geoType:         objectDelta[2],
		deploymentDelta: strings.Split(objectDelta[3], "/ "),
	}

	t.namespaceInit(t.geolocationDet.namespace)
	t.wgHandler[t.geolocationDet.namespace].Add(1)
	defer func() {
		time.Sleep(50 * time.Millisecond)
		t.wgHandler[t.geolocationDet.namespace].Done()
	}()
	// Detect and recover the geolocation resource objects which are prevented by the this object from taking control of the deployments
	t.recoverGeolocations(t.geolocationDet)
}

// setDeploymentFilter used by ObjectCreated, ObjectUpdated, and recoverGeolocations functions
func (t *GeoHandler) setDeploymentFilter(geolocationCopy *geolocation_v1.GeoLocation, delta string, eventType string) {
	// Flush the status
	geolocationCopy.Status = geolocation_v1.GeoLocationStatus{}
	// Put the differences between the old and the new objects into variables
	t.geolocationDet = geolocationDet{
		name:      geolocationCopy.GetName(),
		namespace: geolocationCopy.GetNamespace(),
		geoType:   geolocationCopy.Spec.Type,
	}
	if delta != "" {
		t.geolocationDet.deploymentDelta = strings.Split(delta, "/ ")
	}

	if eventType != "recover" && eventType != "create" {
		defer t.recoverGeolocations(t.geolocationDet)
	} else if eventType == "recover" {
		t.wgRecovery[t.geolocationDet.namespace].Add(1)
		defer func() {
			time.Sleep(50 * time.Millisecond)
			t.wgRecovery[t.geolocationDet.namespace].Done()
		}()
	}
	defer t.geolocationClientset.EdgenetV1alpha().GeoLocations(geolocationCopy.GetNamespace()).UpdateStatus(geolocationCopy)

	geolocationsRaw, err := t.geolocationClientset.EdgenetV1alpha().GeoLocations(geolocationCopy.GetNamespace()).List(metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	// Reveal conflicts by comparing geolocation resource objects with the object in process
	geolocationCopy = setReasonListOfConflicts(geolocationCopy, geolocationsRaw)
	counter := 0
	for _, deploymentName := range geolocationCopy.Spec.Deployment {
		// Get the deployment defined at the geolocation object
		_, err := t.clientset.AppsV1().Deployments(geolocationCopy.GetNamespace()).Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			// In here, the errors caused by non-existent of the deployment are added to reason list of the geolocation object
			geolocationCopy = setReasonListOfNonExistents(geolocationCopy, deploymentName)
			counter++
		}
	}

	// Deployment list without duplicate values
	deploymentList := []string{}
	for _, reason := range geolocationCopy.Status.Reason {
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

	// The problems and details of the desired new geolocation object are described herein, and this step is the last of the error processing
	if len(deploymentList) == len(geolocationCopy.Spec.Deployment) {
		geolocationCopy.Status.State = failure
		geolocationCopy.Status.Message = "All deployments are already under the control of any different resource object(s) with the same type"
	} else if len(geolocationCopy.Status.Reason) == 0 {
		geolocationCopy.Status.State = success
		geolocationCopy.Status.Message = "GeoLocation runs precisely to ensure that the actual state of the cluster matches the desired state"
	} else {
		geolocationCopy.Status.State = partial
		geolocationCopy.Status.Message = "Some deployments are already under the control of any different resource object(s) with the same type"
	}
	// Counter indicates the number of non-existent deployment already defined in the desired geolocation object
	if counter != 0 {
		geolocationCopy.Status.Message = fmt.Sprintf("%s, %d deployment(s) couldn't be found", geolocationCopy.Status.Message, counter)
	}
	// The number of deployments that the geolocation resource successfully controls
	geolocationCopy.Status.Ready = fmt.Sprintf("%d/%d", len(geolocationCopy.Spec.Deployment)-len(deploymentList), len(geolocationCopy.Spec.Deployment))
}

// recoverGeolocations compares the reason list with the deployment list and the name of geolocation to recover objects affected by the geolocation
// object. The deployment delta list contains the name of deployments removed from the geolocation object by updating or deleting it
func (t *GeoHandler) recoverGeolocations(geolocationDet geolocationDet) {
	geolocationsRaw, err := t.geolocationClientset.EdgenetV1alpha().GeoLocations(geolocationDet.namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	for _, geolocationRow := range geolocationsRaw.Items {
		if geolocationRow.GetName() != geolocationDet.name && geolocationRow.Spec.Type == geolocationDet.geoType && geolocationRow.Status.State != "" {
			for _, deployment := range geolocationDet.deploymentDelta {
				if reasonMatch, _ := checkReasonList(geolocationRow.Status.Reason, deployment, geolocationDet.name, "all"); reasonMatch {
					geolocation, err := t.geolocationClientset.EdgenetV1alpha().GeoLocations(geolocationRow.GetNamespace()).Get(geolocationRow.GetName(), metav1.GetOptions{})
					if err == nil {
						t.setDeploymentFilter(geolocation, "", "recover")
						t.wgRecovery[geolocationDet.namespace].Wait()
						time.Sleep(50 * time.Millisecond)
					}
				}
			}
		}
	}
}

// ConfigureDeployments configures the deployments by geolocations to match the desired state users supplied
func (t *GeoHandler) ConfigureDeployments() {
	log.Info("ConfigureDeployments: start")
	configurationList := t.namespaceList
	t.namespaceList = []string{}
	for _, namespace := range configurationList {
		t.wgHandler[namespace].Wait()
		t.wgRecovery[namespace].Wait()
		time.Sleep(1200 * time.Millisecond)

		geolocationsRaw, err := t.geolocationClientset.EdgenetV1alpha().GeoLocations(namespace).List(metav1.ListOptions{})
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
			for _, geolocationRow := range geolocationsRaw.Items {
				if geolocationRow.Status.State == success || geolocationRow.Status.State == partial {
					t.desiredFilter.nodeSelectorTerm = corev1.NodeSelectorTerm{}
					t.desiredFilter.matchExpression.Operator = "In"
					t.desiredFilter.matchExpression = t.setFilter(geolocationRow.Spec.Type, geolocationRow.Spec.Value, t.desiredFilter.matchExpression, "addOrUpdate")
					for _, deployment := range geolocationRow.Spec.Deployment {
						if reasonMatch, _ := checkReasonList(geolocationRow.Status.Reason, deployment, geolocationRow.GetNamespace(), "deployment"); !reasonMatch && deploymentRow.GetName() == deployment {
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

// setFilter generates the values in the predefined form and puts those into the node selection fields of the geolocation object
func (t *GeoHandler) setFilter(geoType string, geoValue []string,
	matchExpression corev1.NodeSelectorRequirement, event string) corev1.NodeSelectorRequirement {
	// Turn the key into the predefined form which is determined at the custom resource definition of geolocation
	matchExpression.Values = geoValue
	switch geoType {
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
			// If the geolocation key is polygon then certain calculations like geofence need to be done
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
						// This loop allows us to process each polygon defined at the object of geolocation resource
						for _, polygonRow := range geoValue {
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

// setReasonListOfConflicts compares the deployments of the geolocation resource objects with those of the object in the process
// to make a list of the conflicts which guides the user to understand its faults
func setReasonListOfConflicts(geolocationCopy *geolocation_v1.GeoLocation, geolocationsRaw *geolocation_v1.GeoLocationList) *geolocation_v1.GeoLocation {
	// The loop to process each geolocation object separately
	for _, geolocationRow := range geolocationsRaw.Items {
		if geolocationRow.GetName() != geolocationCopy.GetName() && geolocationRow.Spec.Type == geolocationCopy.Spec.Type && geolocationRow.Status.State != "" {
			for _, newDeployment := range geolocationCopy.Spec.Deployment {
				for _, otherObjDeployment := range geolocationRow.Spec.Deployment {
					if otherObjDeployment == newDeployment {
						// Checks whether the reason list is empty and this reason exists in the reason list of the geolocation object
						if reasonMatch, _ := checkReasonList(geolocationRow.Status.Reason, newDeployment, geolocationCopy.GetName(), "all"); !reasonMatch {
							if reasonMatch, _ := checkReasonList(geolocationCopy.Status.Reason, otherObjDeployment, geolocationRow.GetName(), "all"); !reasonMatch || len(geolocationCopy.Status.Reason) == 0 {
								conflictReason := []string{otherObjDeployment, geolocationRow.GetName()}
								geolocationCopy.Status.Reason = append(geolocationCopy.Status.Reason, conflictReason)
							}
						}
					}
				}
			}
		}
	}
	return geolocationCopy
}

// setReasonListOfNonExistents checks whether the deployment exists to put it into the list and it will be listed in case of non-existent
func setReasonListOfNonExistents(geolocationCopy *geolocation_v1.GeoLocation, deploymentName string) *geolocation_v1.GeoLocation {
	if reasonMatch, _ := checkReasonList(geolocationCopy.Status.Reason, deploymentName, "nonexistent", "all"); !reasonMatch {
		conflictReason := []string{deploymentName, "nonexistent"}
		geolocationCopy.Status.Reason = append(geolocationCopy.Status.Reason, conflictReason)
	}
	return geolocationCopy
}

// checkReasonList compares the reason list with the given names of deployment and geolocation
func checkReasonList(reasonList [][]string, deployment string, geolocation string, compareType string) (bool, int) {
	exists := false
	index := -1
	for i, reason := range reasonList {
		reasonDeployment := reason[0]
		reasonGeolocation := reason[1]
		if compareType == "deployment" {
			reasonGeolocation = geolocation
		} else if compareType == "geolocation" {
			reasonDeployment = deployment
		}
		if deployment == reasonDeployment && geolocation == reasonGeolocation {
			exists = true
			index = i
		}
	}
	return exists, index
}
