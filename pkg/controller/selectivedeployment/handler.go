package selectivedeployment

import (
	"encoding/json"
	"strconv"
	"strings"

	geolocation_v1 "headnode/pkg/apis/geolocation/v1alpha"
	"headnode/pkg/authorization"
	"headnode/pkg/node"

	log "github.com/Sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init() error
	ObjectCreated(obj interface{})
	ObjectUpdated(obj interface{}, delta string)
	ObjectDeleted(obj interface{}, delta string)
}

// GeoHandler is a sample implementation of Handler
type GeoHandler struct{}

type desiredFilter struct {
	geolocation geolocationValues
	deployment  deploymentValues
}
type geolocationValues struct {
	namespace  string
	value      []string
	geoType    string
	deployment []string
}
type deploymentValues struct {
	nodeSelector     *corev1.NodeSelector
	nodeSelectorTerm corev1.NodeSelectorTerm
	matchExpression  corev1.NodeSelectorRequirement
	nodeAffinity     *corev1.NodeAffinity
	affinity         *corev1.Affinity
}

// Init handles any handler initialization
func (t *GeoHandler) Init() error {
	log.Info("GeoHandler.Init")
	return nil
}

// setDeploymentFilter used by ObjectCreated and ObjectUpdated functions
func setDeploymentFilter(obj interface{}, delta string) {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	// Sync the defined values of the resource object
	geolocationValues := geolocationValues{
		namespace:  obj.(*geolocation_v1.GeoLocation).GetNamespace(),
		value:      obj.(*geolocation_v1.GeoLocation).Spec.Value,
		geoType:    obj.(*geolocation_v1.GeoLocation).Spec.Type,
		deployment: obj.(*geolocation_v1.GeoLocation).Spec.Deployment,
	}
	deploymentValues := deploymentValues{
		nodeSelector: &corev1.NodeSelector{},
		nodeAffinity: &corev1.NodeAffinity{},
		affinity:     &corev1.Affinity{},
	}
	desiredFilter := desiredFilter{
		geolocation: geolocationValues,
		deployment:  deploymentValues,
	}
	var deltaValues = strings.Split(delta, "- ")
	desiredFilter.deployment.matchExpression.Operator = "In"
	desiredFilter.deployment.matchExpression.Values = desiredFilter.geolocation.value

	// Turn the key into the predefined form which is determined at the custom resource definition of geolocation
	switch desiredFilter.geolocation.geoType {
	case "city":
		desiredFilter.deployment.matchExpression.Key = "edge-net.io/city"
	case "country":
		desiredFilter.deployment.matchExpression.Key = "edge-net.io/country-iso"
	case "continent":
		desiredFilter.deployment.matchExpression.Key = "edge-net.io/continent"
	case "polygon":
		// If the geolocation key is polygon then certain calculations like geofence need to be done
		// for being had the list of nodes that the pods will be deployed on according to the desired state.
		// This gets the node list which includes the EdgeNet geolabels
		nodesRaw, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			log.Println(err.Error())
			panic(err.Error())
		}

		var polygon [][]float64
		// deltaPolygon, deltaRaw, and deltaValues use for finding out double values when the resource object updated
		var deltaPolygon [][]float64
		deltaRaw := deltaValues
		deltaValues = []string{}
		desiredFilter.deployment.matchExpression.Values = []string{}
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
					for _, polygonRow := range desiredFilter.geolocation.value {
						err = json.Unmarshal([]byte(polygonRow), &polygon)
						if err != nil {
							panic(err)
						}
						// boundbox is a rectangle which provides to check whether the point is inside polygon
						// without taking all point of the polygon into consideration
						boundbox := node.Boundbox(polygon)
						status := node.GeoFence(boundbox, polygon, lon, lat)
						if status {
							desiredFilter.deployment.matchExpression.Values = append(desiredFilter.deployment.matchExpression.Values, nodeRow.Labels["kubernetes.io/hostname"])
						}
					}
					// This part to store the current list of nodes concerned to the object if the event type is "update" and deltaRaw is not empty
					if len(deltaRaw) > 0 && deltaRaw[0] != "" {
						for _, deltaRow := range deltaRaw {
							err = json.Unmarshal([]byte(deltaRow), &deltaPolygon)
							if err != nil {
								panic(err)
							}
							// deltaBoundbox is a rectangle which provides to check whether the point is inside polygon
							// without taking all point of the polygon into consideration
							deltaBoundbox := node.Boundbox(deltaPolygon)
							deltaStatus := node.GeoFence(deltaBoundbox, deltaPolygon, lon, lat)
							if deltaStatus {
								deltaValues = append(deltaValues, nodeRow.Labels["kubernetes.io/hostname"])
							}
						}
					}
				}
			}
		}
		desiredFilter.deployment.matchExpression.Key = "kubernetes.io/hostname"
	default:
		desiredFilter.deployment.matchExpression.Key = ""
	}

	if desiredFilter.deployment.matchExpression.Key != "" {
		for _, deploymentName := range desiredFilter.geolocation.deployment {
			// Clear the variables involved with node selection
			desiredFilter.deployment.nodeSelector = &corev1.NodeSelector{}
			desiredFilter.deployment.nodeSelectorTerm = corev1.NodeSelectorTerm{}
			// Get the deployment defined at the geolocation object
			deployment, _ := clientset.AppsV1().Deployments(desiredFilter.geolocation.namespace).Get(deploymentName, metav1.GetOptions{})
			deployment.Spec.Template.Spec.NodeSelector = map[string]string{}

			// Check whether the node affinity feature already exists in the deployment, then can be handled smoothly
			deploymentExpression := desiredFilter.deployment.matchExpression
			if deployment.Spec.Template.Spec.Affinity != nil &&
				deployment.Spec.Template.Spec.Affinity.NodeAffinity != nil &&
				deployment.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
				currentState := deployment.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
				// Loop to check each match expression in the deployment separately
				for _, expression := range currentState.NodeSelectorTerms[0].MatchExpressions {
					// This part unifies the expressions if any of the current one is matched with the new desired one,
					// otherwise, it appends the expression directly.
					if expression.Key == desiredFilter.deployment.matchExpression.Key && expression.Operator == desiredFilter.deployment.matchExpression.Operator {
						deploymentExpression.Values = unique(desiredFilter.deployment.matchExpression.Values, expression.Values, deltaValues)
					} else {
						desiredFilter.deployment.nodeSelectorTerm.MatchExpressions = append(desiredFilter.deployment.nodeSelectorTerm.MatchExpressions, expression)
					}
				}
			}
			desiredFilter.deployment.nodeSelectorTerm.MatchExpressions = append(desiredFilter.deployment.nodeSelectorTerm.MatchExpressions, deploymentExpression)
			desiredFilter.deployment.nodeSelector.NodeSelectorTerms = append(desiredFilter.deployment.nodeSelector.NodeSelectorTerms, desiredFilter.deployment.nodeSelectorTerm)
			log.Info(desiredFilter.deployment.nodeSelector)
			// Set the new affinity configuration in the deployment and update the deployment
			desiredFilter.deployment.nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = desiredFilter.deployment.nodeSelector
			desiredFilter.deployment.affinity.NodeAffinity = desiredFilter.deployment.nodeAffinity
			deployment.Spec.Template.Spec.Affinity = desiredFilter.deployment.affinity
			clientset.AppsV1().Deployments(desiredFilter.geolocation.namespace).Update(deployment)
		}
	}
}

// ObjectCreated is called when an object is created
func (t *GeoHandler) ObjectCreated(obj interface{}) {
	log.Info("GeoHandler.ObjectCreated")
	setDeploymentFilter(obj, "")
}

// ObjectUpdated is called when an object is updated
func (t *GeoHandler) ObjectUpdated(obj interface{}, delta string) {
	log.Info("GeoHandler.ObjectUpdated")
	setDeploymentFilter(obj, delta)
}

// ObjectDeleted is called when an object is deleted
func (t *GeoHandler) ObjectDeleted(obj interface{}, delta string) {
	log.Info("GeoHandler.ObjectDeleted")
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	// Sync the defined values of the resource object
	var deltaValues = strings.Split(delta, "- ")
	geolocationValues := geolocationValues{
		namespace:  deltaValues[0],
		geoType:    deltaValues[1],
		deployment: strings.Split(deltaValues[2], "/ "),
	}
	deploymentValues := deploymentValues{
		nodeSelector: &corev1.NodeSelector{},
		nodeAffinity: &corev1.NodeAffinity{},
		affinity:     &corev1.Affinity{},
	}
	desiredFilter := desiredFilter{
		geolocation: geolocationValues,
		deployment:  deploymentValues,
	}
	// Turn the key into the predefined form which is determined at the custom resource definition of geolocation
	desiredFilter.deployment.matchExpression.Values = []string{}
	switch desiredFilter.geolocation.geoType {
	case "city":
		desiredFilter.deployment.matchExpression.Key = "edge-net.io/city"
	case "country":
		desiredFilter.deployment.matchExpression.Key = "edge-net.io/country-iso"
	case "continent":
		desiredFilter.deployment.matchExpression.Key = "edge-net.io/continent"
	case "polygon":
		desiredFilter.deployment.matchExpression.Key = "kubernetes.io/hostname"
	default:
		desiredFilter.deployment.matchExpression.Key = ""
	}

	if desiredFilter.deployment.matchExpression.Key != "" {
		for _, deploymentName := range desiredFilter.geolocation.deployment {
			// Clear the variables involved with node selection
			desiredFilter.deployment.nodeSelector = &corev1.NodeSelector{}
			desiredFilter.deployment.nodeSelectorTerm = corev1.NodeSelectorTerm{}
			// Get the deployment defined at the geolocation object
			deployment, _ := clientset.AppsV1().Deployments(desiredFilter.geolocation.namespace).Get(deploymentName, metav1.GetOptions{})

			// Check whether the node affinity feature already exists in the deployment, then can be handled smoothly
			if deployment.Spec.Template.Spec.Affinity != nil &&
				deployment.Spec.Template.Spec.Affinity.NodeAffinity != nil &&
				deployment.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
				currentState := deployment.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
				for _, expression := range currentState.NodeSelectorTerms[0].MatchExpressions {
					// This part appends directly the current expression which is not matched with any of the expressions of the removed object
					if expression.Key != desiredFilter.deployment.matchExpression.Key {
						desiredFilter.deployment.nodeSelectorTerm.MatchExpressions = append(desiredFilter.deployment.nodeSelectorTerm.MatchExpressions, expression)
					}
				}
			}
			desiredFilter.deployment.nodeSelector.NodeSelectorTerms = append(desiredFilter.deployment.nodeSelector.NodeSelectorTerms, desiredFilter.deployment.nodeSelectorTerm)

			// Set the new affinity configuration in the deployment and update the deployment
			desiredFilter.deployment.nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = desiredFilter.deployment.nodeSelector
			desiredFilter.deployment.affinity.NodeAffinity = desiredFilter.deployment.nodeAffinity
			deployment.Spec.Template.Spec.Affinity = desiredFilter.deployment.affinity
			clientset.AppsV1().Deployments(desiredFilter.geolocation.namespace).Update(deployment)
		}
	}
}

// unique function joins the slices and then merges duplicate values
func unique(mainSlice []string, appendSlice []string, deltaSlice []string) []string {
	uniqueSlice := mainSlice
	for _, appendValue := range appendSlice {
		exists := false
		for _, mainValue := range mainSlice {
			if appendValue == mainValue {
				exists = true
			}
		}
		for _, deltaValue := range deltaSlice {
			if appendValue == deltaValue {
				exists = true
			}
		}
		if !exists {
			uniqueSlice = append(uniqueSlice, appendValue)
		}
	}
	return uniqueSlice
}
