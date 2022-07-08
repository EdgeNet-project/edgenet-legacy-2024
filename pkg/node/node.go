/*
Copyright 2021 Contributors to the EdgeNet project.

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
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/node/infrastructure"
	"github.com/savaki/geoip2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	namecheap "github.com/billputer/go-namecheap"
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
	nodeRaw, err := Clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/control-plane"})
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

// sanitizeNodeLabel converts arbitrary strings to valid k8s labels.
// > a valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.',
// > and must start and end with an alphanumeric character (e.g. 'MyValue',  or 'my_value',  or '12345',
// > regex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?')
func sanitizeNodeLabel(s string) string {
	r, _ := regexp.Compile("[^\\w-_.]")
	s = r.ReplaceAllString(s, "_")
	s = strings.TrimLeft(s, "-_.")
	s = strings.TrimRight(s, "-_.")
	return s
}

// getMaxmindLocation is similar to geoip2.fetch(...) but allows to specify a custom URL for testing.
func getMaxmindLocation(url string, accountId string, licenseKey string, address string) (*geoip2.Response, error) {
	req, err := http.NewRequest("GET", url+address, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(accountId, licenseKey)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		v := geoip2.Error{}
		err := json.NewDecoder(res.Body).Decode(&v)
		if err != nil {
			return nil, err
		}
		return nil, v
	}
	response := &geoip2.Response{}
	err = json.NewDecoder(res.Body).Decode(response)
	return response, err
}

// GetGeolocationByIP returns geolabels from the MaxMind GeoIP2 precision service
func GetGeolocationByIP(
	maxmindUrl string,
	maxmindAccountId string,
	maxmindLicenseKey string,
	hostname string,
	address string,
) bool {
	// Fetch geolocation information
	record, err := getMaxmindLocation(maxmindUrl, maxmindAccountId, maxmindLicenseKey, address)
	if err != nil {
		log.Println(err)
		return false
	}

	continent := sanitizeNodeLabel(record.Continent.Names["en"])
	country := record.Country.IsoCode
	state := record.Country.IsoCode
	city := sanitizeNodeLabel(record.City.Names["en"])
	isp := sanitizeNodeLabel(record.Traits.Isp)
	as := sanitizeNodeLabel(record.Traits.AutonomousSystemOrganization)
	asn := strconv.Itoa(record.Traits.AutonomousSystemNumber)
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
		"edge-net.io~1isp":         isp,
		"edge-net.io~1as":          as,
		"edge-net.io~1asn":         asn,
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

func SetHostnameRoute53(hostedZone, nodeName, address, recordType string) (bool, string) {
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("CREATE"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(nodeName),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(address),
							},
						},
						TTL:  aws.Int64(60),
						Type: aws.String(recordType),
					},
				},
			},
			Comment: aws.String("Node contribution for EdgeNet"),
		},
		HostedZoneId: aws.String(hostedZone),
	}
	result, state := infrastructure.AddARecordRoute53(input)
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
		} else if conditionType := conditionRow.Type; conditionType == "NotReady" {
			return "False"
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
