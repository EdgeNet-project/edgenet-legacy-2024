package node

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"strings"
	"time"

	"headnode/pkg/authorization"
	"headnode/pkg/node/infrastructure"

	namecheap "github.com/billputer/go-namecheap"
	geoip2 "github.com/oschwald/geoip2-golang"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// JSON structure of patch operation
type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// GeoFence function determines whether the point is inside a polygon by using the crossing number method.
// This method counts the number of times a ray starting at a point crosses a polygon boundary edge.
// The even numbers mean the point is outside and the odd ones mean the point is inside.
func GeoFence(boundbox []float64, polygon [][]float64, y float64, x float64) bool {
	vertices := len(polygon)
	lastIndex := vertices - 1
	oddNodes := false

	if boundbox[0] <= x && boundbox[1] >= x && boundbox[2] <= y && boundbox[3] >= y {
		for index := range polygon {
			if (polygon[index][0] < y && polygon[lastIndex][0] >= y || polygon[lastIndex][0] < y &&
				polygon[index][0] >= y) && (polygon[index][1] <= x || polygon[lastIndex][1] <= x) {
				if polygon[index][1]+(y-polygon[index][0])/(polygon[lastIndex][0]-polygon[index][0])*
					(polygon[lastIndex][1]-polygon[index][1]) < x {
					oddNodes = !oddNodes
				}
			}
			lastIndex = index
		}
	}

	return oddNodes
}

// Boundbox returns a rectangle which created according to the points of the polygon given
func Boundbox(points [][]float64) []float64 {
	var minX float64 = -math.MaxFloat64
	var maxX float64 = math.MaxFloat64
	var minY float64 = -math.MaxFloat64
	var maxY float64 = math.MaxFloat64

	for _, coordinates := range points {
		minX = math.Min(minX, coordinates[0])
		maxX = math.Max(maxX, coordinates[0])
		minY = math.Min(minY, coordinates[1])
		maxY = math.Max(maxY, coordinates[1])
	}

	bounding := []float64{minX, maxX, minY, maxY}
	return bounding
}

// setNodeLabels uses client-go to patch nodes by processing a labels map
func setNodeLabels(hostname string, labels map[string]string) bool {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	// Create a patch slice and initialize it to the label size
	nodePatchArr := make([]patchStringValue, len(labels))
	nodePatch := patchStringValue{}
	row := 0
	// Append the data existing in the label map to the slice
	for label, value := range labels {
		nodePatch.Op = "add"
		nodePatch.Path = fmt.Sprintf("/metadata/labels/%s", label)
		nodePatch.Value = value
		nodePatchArr[row] = nodePatch
		row++
	}
	nodesJSON, _ := json.Marshal(nodePatchArr)

	// Patch the nodes with the arguments:
	// hostname, patch type, and patch data
	_, err = clientset.CoreV1().Nodes().Patch(hostname, types.JSONPatchType, nodesJSON)
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	return true
}

// GetGeolocationByIP return geolabels by taking advantage of GeoLite database
func GetGeolocationByIP(hostname string, ipStr string) bool {
	// Parse IP address
	ip := net.ParseIP(ipStr)
	// Open GeoLite database
	db, err := geoip2.Open("../../assets/database/GeoLite2-City/GeoLite2-City.mmdb")
	if err != nil {
		log.Fatal(err)
		return false
	}
	// Close the database as a final job
	defer db.Close()
	// Get the geolocation information by IP
	record, err := db.City(ip)
	if err != nil {
		log.Fatal(err)
		return false
	}

	// Patch for being compatible with Kubernetes alphanumeric characters limitations
	continent := strings.Replace(record.Continent.Names["en"], " ", "_", -1)
	country := record.Country.IsoCode
	state := record.Country.IsoCode
	city := strings.Replace(record.City.Names["en"], " ", "_", -1)
	var lon string
	var lat string
	if record.Location.Longitude >= 0 {
		lon = fmt.Sprintf("e%.6f", record.Location.Longitude)
	} else {
		lon = fmt.Sprintf("w%.6f", record.Location.Longitude)
	}
	if record.Location.Latitude >= 0 {
		lat = fmt.Sprintf("n%.6f", record.Location.Latitude)
	} else {
		lat = fmt.Sprintf("s%.6f", record.Location.Latitude)
	}
	if len(record.Subdivisions) > 0 {
		state = record.Subdivisions[0].IsoCode
	}

	// Create label map to attach to the node
	geoLabels := map[string]string{
		"edge-net.io~1continent":   continent,
		"edge-net.io~1country-iso": country,
		"edge-net.io~1state-iso":   state,
		"edge-net.io~1city":        city,
		"edge-net.io~1lon":         lon,
		"edge-net.io~1lat":         lat,
	}

	// Attach geolabels to the node
	result := setNodeLabels(hostname, geoLabels)
	// If the result is different than the expected, return false
	// The expected result is having a different longitude and latitude than zero
	// Zero value typically means there isn't any result meaningful
	if record.Location.Longitude == 0 && record.Location.Latitude == 0 {
		return false
	}
	return result
}

// CompareIPAddresses makes a comparison between old and new objects of the node
// to return the information of the match
func CompareIPAddresses(oldObj *api_v1.Node, newObj *api_v1.Node) bool {
	updated := true
	oldInternalIP, oldExternalIP := GetNodeIPAddresses(oldObj)
	newInternalIP, newExternalIP := GetNodeIPAddresses(newObj)
	if oldInternalIP != "" && newInternalIP != "" {
		if oldExternalIP != "" && newExternalIP != "" {
			if oldInternalIP == newInternalIP && oldExternalIP == newExternalIP {
				updated = false
			}
		} else {
			if oldInternalIP == newInternalIP {
				updated = false
			}
		}
	} else {
		if oldExternalIP == newExternalIP {
			updated = false
		}
	}
	return updated
}

// GetNodeIPAddresses picks up the internal and external IP addresses of the Node
func GetNodeIPAddresses(obj *api_v1.Node) (string, string) {
	internalIP := ""
	externalIP := ""
	for _, addressesRow := range obj.Status.Addresses {
		if addressType := addressesRow.Type; addressType == "InternalIP" {
			internalIP = addressesRow.Address
		}
		if addressType := addressesRow.Type; addressType == "ExternalIP" {
			externalIP = addressesRow.Address
		}
	}
	return internalIP, externalIP
}

// SetHostname generates token to be used on adding a node onto the cluster
func SetHostname(hostRecord namecheap.DomainDNSHost) (bool, string) {
	client, err := authorization.CreateNamecheapClient()
	if err != nil {
		log.Println(err.Error())
		return false, "Unknown"
	}
	result, state := infrastructure.SetHostname(client, hostRecord)
	return result, state
}

// CreateJoinToken generates token to be used on adding a node onto the cluster
func CreateJoinToken(ttl string, hostname string) string {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	duration, _ := time.ParseDuration(ttl)
	token, err := infrastructure.CreateToken(clientset, duration, hostname)
	if err != nil {
		log.Println(err.Error())
		return "error"
	}
	return token
}

// GetList uses clientset to get node list of the cluster
func GetList() []string {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	nodesRaw, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	nodes := make([]string, len(nodesRaw.Items))
	for i, nodeRow := range nodesRaw.Items {
		nodes[i] = nodeRow.Name
	}

	return nodes
}

// GetStatusList uses clientset to get node list of the cluster that contains Ready State info
func GetStatusList() []byte {
	type nodeStatus struct {
		Node  string `json:"node"`
		Ready string `json:"ready"`
	}
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	nodesRaw, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	nodesArr := make([]nodeStatus, len(nodesRaw.Items))
	for i, nodeRow := range nodesRaw.Items {
		nodesArr[i].Node = nodeRow.Name
		for _, conditionRow := range nodeRow.Status.Conditions {
			if conditionType := conditionRow.Type; conditionType == "Ready" {
				nodesArr[i].Ready = string(conditionRow.Status)
			}
		}
	}
	nodesJSON, _ := json.Marshal(nodesArr)

	return nodesJSON
}

// getNodeByHostname uses clientset to get namespace requested
func getNodeByHostname(hostname string) (string, error) {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	// Examples for error handling:
	// - Use helper functions like e.g. errors.IsNotFound()
	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	_, err = clientset.CoreV1().Nodes().Get(hostname, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Printf("Node %s not found", hostname)
		return "false", err
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Printf("Error getting node %s: %v", hostname, statusError.ErrStatus)
		return "error", err
	} else if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	} else {
		return "true", nil
	}
}
