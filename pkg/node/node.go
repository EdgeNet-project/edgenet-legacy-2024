/*
Copyright 2019 Sorbonne Université

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

// This code includes GeoLite2 data created by MaxMind, available from
// https://www.maxmind.com.

package node

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"strings"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/node/infrastructure"

	namecheap "github.com/billputer/go-namecheap"
	geoip2 "github.com/oschwald/geoip2-golang"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// JSON structure of patch operation
type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}
type patchByBoolValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value bool   `json:"value"`
}
type patchByOwnerReferenceValue struct {
	Op    string                  `json:"op"`
	Path  string                  `json:"path"`
	Value []metav1.OwnerReference `json:"value"`
}

// Clientset to be synced by the custom resources
var Clientset kubernetes.Interface

// GeoFence function determines whether the point is inside a polygon by using the crossing number method.
// This method counts the number of times a ray starting at a point crosses a polygon boundary edge.
// The even numbers mean the point is outside and the odd ones mean the point is inside.
func GeoFence(boundbox []float64, polygon [][]float64, x float64, y float64) bool {
	vertices := len(polygon)
	lastIndex := vertices - 1
	oddNodes := false
	if boundbox[0] <= x && boundbox[1] >= x && boundbox[2] <= y && boundbox[3] >= y {
		for index := range polygon {
			if (polygon[index][1] < y && polygon[lastIndex][1] >= y || polygon[lastIndex][1] < y &&
				polygon[index][1] >= y) && (polygon[index][0] <= x || polygon[lastIndex][0] <= x) {
				if polygon[index][0]+(y-polygon[index][1])/(polygon[lastIndex][1]-polygon[index][1])*
					(polygon[lastIndex][0]-polygon[index][0]) < x {
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
	var minX float64 = math.MaxFloat64
	var maxX float64 = -math.MaxFloat64
	var minY float64 = math.MaxFloat64
	var maxY float64 = -math.MaxFloat64

	for _, coordinates := range points {
		minX = math.Min(minX, coordinates[0])
		maxX = math.Max(maxX, coordinates[0])
		minY = math.Min(minY, coordinates[1])
		maxY = math.Max(maxY, coordinates[1])
	}

	bounding := []float64{minX, maxX, minY, maxY}
	return bounding
}

// GetKubeletVersion looks at the head node to decide which version of Kubernetes to install
func GetKubeletVersion() string {
	nodeRaw, err := Clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/master"})
	if err != nil {
		log.Println(err.Error())
	}
	kubeletVersion := ""
	for _, nodeRow := range nodeRaw.Items {
		kubeletVersion = nodeRow.Status.NodeInfo.KubeletVersion
	}
	return kubeletVersion
}

// SetOwnerReferences make the references owner of the node
func SetOwnerReferences(nodeName string, ownerReferences []metav1.OwnerReference) error {
	// Create a patch slice and initialize it to the size of 1
	// Append the data existing in the label map to the slice
	nodePatchArr := make([]interface{}, 1)
	nodePatch := patchByOwnerReferenceValue{}
	nodePatch.Op = "add"
	nodePatch.Path = "/metadata/ownerReferences"
	nodePatch.Value = ownerReferences
	nodePatchArr[0] = nodePatch
	nodePatchJSON, _ := json.Marshal(nodePatchArr)
	// Patch the nodes with the arguments:
	// hostname, patch type, and patch data
	_, err := Clientset.CoreV1().Nodes().Patch(context.TODO(), nodeName, types.JSONPatchType, nodePatchJSON, metav1.PatchOptions{})
	return err
}

// SetNodeScheduling syncs the node with the node contribution
func SetNodeScheduling(nodeName string, unschedulable bool) error {
	// Create a patch slice and initialize it to the size of 1
	nodePatchArr := make([]interface{}, 1)
	nodePatch := patchByBoolValue{}
	nodePatch.Op = "replace"
	nodePatch.Path = "/spec/unschedulable"
	nodePatch.Value = unschedulable
	nodePatchArr[0] = nodePatch
	nodePatchJSON, _ := json.Marshal(nodePatchArr)
	// Patch the nodes with the arguments:
	// hostname, patch type, and patch data
	_, err := Clientset.CoreV1().Nodes().Patch(context.TODO(), nodeName, types.JSONPatchType, nodePatchJSON, metav1.PatchOptions{})
	return err
}

// setNodeLabels uses client-go to patch nodes by processing a labels map
func setNodeLabels(hostname string, labels map[string]string) bool {
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
	_, err := Clientset.CoreV1().Nodes().Patch(context.TODO(), hostname, types.JSONPatchType, nodesJSON, metav1.PatchOptions{})
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
	var pathDB string
	if flag.Lookup("geolite-path") != nil {
		pathDB = flag.Lookup("geolite-path").Value.(flag.Getter).Get().(string)
	}
	if pathDB == "" {
		pathDB = "../../assets/database/GeoLite2-City/GeoLite2-City.mmdb"
	}
	// Open GeoLite database
	db, err := geoip2.Open(pathDB)
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
func CompareIPAddresses(oldObj *corev1.Node, newObj *corev1.Node) bool {
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
func GetNodeIPAddresses(obj *corev1.Node) (string, string) {
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
	client, err := bootstrap.CreateNamecheapClient()
	if err != nil {
		log.Println(err.Error())
		return false, "Unknown"
	}
	result, state := infrastructure.SetHostname(client, hostRecord)
	return result, state
}

// CreateJoinToken generates token to be used on adding a node onto the cluster
func CreateJoinToken(ttl string, hostname string) string {
	duration, _ := time.ParseDuration(ttl)
	token, err := infrastructure.CreateToken(Clientset, duration, hostname)
	if err != nil {
		log.Println(err.Error())
		return "error"
	}
	return token
}

// GetList uses clientset to get node list of the cluster
func GetList() []string {
	nodesRaw, err := Clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
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

// GetConditionReadyStatus picks the ready status of node
func GetConditionReadyStatus(node *corev1.Node) string {
	for _, conditionRow := range node.Status.Conditions {
		if conditionType := conditionRow.Type; conditionType == "Ready" {
			return string(conditionRow.Status)
		}
	}
	return ""
}

// getNodeByHostname uses clientset to get namespace requested
func getNodeByHostname(hostname string) (string, error) {
	// Examples for error handling:
	// - Use helper functions like e.g. errors.IsNotFound()
	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	_, err := Clientset.CoreV1().Nodes().Get(context.TODO(), hostname, metav1.GetOptions{})
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

// unique function remove duplicate values from slice.
func unique(slice []string) []string {
	duplicateList := map[string]bool{}
	uniqueSlice := []string{}

	for _, ele := range slice {
		if _, exist := duplicateList[ele]; exist != true &&
			ele != "default" && ele != "kube-system" && ele != "kube-public" {
			duplicateList[ele] = true
			uniqueSlice = append(uniqueSlice, ele)
		}
	}
	return uniqueSlice
}
