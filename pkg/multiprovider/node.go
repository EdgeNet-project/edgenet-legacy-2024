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

package multiprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/savaki/geoip2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	namecheap "github.com/billputer/go-namecheap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

// SetOwnerReferences make the references owner of the node
func (m *Manager) SetOwnerReferences(hostname string, ownerReferences []metav1.OwnerReference) error {
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
	_, err := m.kubeclientset.CoreV1().Nodes().Patch(context.TODO(), hostname, types.JSONPatchType, nodePatchJSON, metav1.PatchOptions{})
	return err
}

// SetNodeScheduling syncs the node with the node contribution
func (m *Manager) SetNodeScheduling(hostname string, unschedulable bool) error {
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
	_, err := m.kubeclientset.CoreV1().Nodes().Patch(context.TODO(), hostname, types.JSONPatchType, nodePatchJSON, metav1.PatchOptions{})
	return err
}

// setNodeLabels uses client-go to patch nodes by processing a labels map
func (m *Manager) setNodeLabels(hostname string, labels map[string]string) bool {
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
	_, err := m.kubeclientset.CoreV1().Nodes().Patch(context.TODO(), hostname, types.JSONPatchType, nodesJSON, metav1.PatchOptions{})
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
func getMaxmindLocation(url string, accountID string, licenseKey string, address string) (*geoip2.Response, error) {
	req, err := http.NewRequest("GET", url+address, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(accountID, licenseKey)
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
func (m *Manager) GetGeolocationByIP(
	maxmindURL string,
	maxmindAccountID string,
	maxmindLicenseKey string,
	hostname string,
	address string,
) bool {
	// Fetch geolocation information
	record, err := getMaxmindLocation(maxmindURL, maxmindAccountID, maxmindLicenseKey, address)
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
	result := m.setNodeLabels(hostname, geoLabels)

	// If the result is different than the expected, return false
	// The expected result is having a different longitude and latitude than zero
	// Zero value typically means there isn't any result meaningful
	if record.Location.Longitude == 0 && record.Location.Latitude == 0 {
		return false
	}
	return result
}

func CompareAvailableResources(oldObj *corev1.Node, newObj *corev1.Node) bool {
	if oldObj.Status.Allocatable.Cpu().Cmp(*newObj.Status.Allocatable.Cpu()) != 0 {
		return true
	}
	if oldObj.Status.Allocatable.Memory().Cmp(*newObj.Status.Allocatable.Memory()) != 0 {
		return true
	}
	if oldObj.Status.Allocatable.Pods().Cmp(*newObj.Status.Allocatable.Pods()) != 0 {
		return true
	}
	if oldObj.Status.Allocatable.Storage().Cmp(*newObj.Status.Allocatable.Storage()) != 0 {
		return true
	}
	if oldObj.Status.Allocatable.StorageEphemeral().Cmp(*newObj.Status.Allocatable.StorageEphemeral()) != 0 {
		return true
	}
	return false
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

// SetHostnameNamecheap generates token to be used on adding a node onto the cluster
func SetHostnameNamecheap(hostRecord namecheap.DomainDNSHost) (bool, string) {
	client, err := bootstrap.CreateNamecheapClient()
	if err != nil {
		log.Println(err.Error())
		return false, "Unknown"
	}
	result, state := setHostnamesNamecheap(client, hostRecord)
	return result, state
}

// setHostnamesNamecheap allows comparing the current hosts with requested hostname by DNS check
func setHostnamesNamecheap(client *namecheap.Client, hostRecord namecheap.DomainDNSHost) (bool, string) {
	hostList := getHosts(client)
	exist := false
	for key, host := range hostList.Hosts {
		if host.Name == hostRecord.Name || host.Address == hostRecord.Address {
			// If the record exist then update it, overwrite it with new name and address
			hostList.Hosts[key] = hostRecord
			log.Printf("Update existing host: %s - %s \n Hostname  and ip address changed to: %s - %s", host.Name, host.Address, hostRecord.Name, hostRecord.Address)
			exist = true
			break
		}
	}
	// In case the record is new, it is appended to the list of existing records
	if !exist {
		hostList.Hosts = append(hostList.Hosts, hostRecord)
	}
	setResponse, err := client.DomainDNSSetHosts("edge-net", "io", hostList.Hosts)
	if err != nil {
		log.Println(err.Error())
		log.Printf("Set host failed: %s - %s", hostRecord.Name, hostRecord.Address)
		return false, "failed"
	} else if setResponse.IsSuccess == false {
		log.Printf("Set host unknown problem: %s - %s", hostRecord.Name, hostRecord.Address)
		return false, "unknown"
	}
	return true, ""
}

func getHosts(client *namecheap.Client) namecheap.DomainDNSGetHostsResult {
	hostsResponse, err := client.DomainsDNSGetHosts("edge-net", "io")
	if err != nil {
		log.Println(err.Error())
		return namecheap.DomainDNSGetHostsResult{}
	}
	responseJSON, err := json.Marshal(hostsResponse)
	if err != nil {
		log.Println(err.Error())
		return namecheap.DomainDNSGetHostsResult{}
	}
	hostList := namecheap.DomainDNSGetHostsResult{}
	json.Unmarshal([]byte(responseJSON), &hostList)
	return hostList
}

// SetHostnameRoute53 add a DNS record to a Route53 hosted zone
func SetHostnameRoute53(hostedZone, hostname, address, recordType string) (bool, string) {
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("CREATE"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(hostname),
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
	result, state := addARecordRoute53(input)
	return result, state
}

// addARecordRoute53 adds a new A record to the Route53
func addARecordRoute53(input *route53.ChangeResourceRecordSetsInput) (bool, string) {
	svc := route53.New(session.New())
	_, err := svc.ChangeResourceRecordSets(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case route53.ErrCodeNoSuchHostedZone:
				log.Println(route53.ErrCodeNoSuchHostedZone, aerr.Error())
			case route53.ErrCodeNoSuchHealthCheck:
				log.Println(route53.ErrCodeNoSuchHealthCheck, aerr.Error())
			case route53.ErrCodeInvalidChangeBatch:
				log.Println(route53.ErrCodeInvalidChangeBatch, aerr.Error())
			case route53.ErrCodeInvalidInput:
				log.Println(route53.ErrCodeInvalidInput, aerr.Error())
			case route53.ErrCodePriorRequestNotComplete:
				log.Println(route53.ErrCodePriorRequestNotComplete, aerr.Error())
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
		return false, "failed"
	}
	return true, ""
}

// CreateJoinToken generates token to be used on adding a node onto the cluster
func (m *Manager) CreateJoinToken(ttl string, hostname string) string {
	duration, _ := time.ParseDuration(ttl)
	token, err := m.createToken(duration, hostname)
	if err != nil {
		log.Println(err.Error())
		return "error"
	}
	return token
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
