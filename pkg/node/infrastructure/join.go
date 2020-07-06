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

package infrastructure

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	s "strings"
	"time"

	custconfig "edgenet/pkg/config"

	namecheap "github.com/billputer/go-namecheap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	nodebootstraptokenphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/bootstraptoken/node"
)

// CreateToken creates the token to be used to add node
// and return the token
func CreateToken(clientset kubernetes.Interface, duration time.Duration, hostname string) (string, error) {
	tokenStr, err := bootstraputil.GenerateBootstrapToken()
	if err != nil {
		log.Printf("Error generating token to upload certs: %s", err)
		return "", err
	}
	token, err := kubeadmapi.NewBootstrapTokenString(tokenStr)
	if err != nil {
		log.Printf("Error creating upload certs token: %s", err)
		return "", err
	}
	tokens := []kubeadmapi.BootstrapToken{{
		Token:       token,
		Description: fmt.Sprintf("EdgeNet token for adding node called %s", hostname),
		TTL: &metav1.Duration{
			Duration: duration,
		},
		Usages: []string{"authentication", "signing"},
		Groups: []string{"system:bootstrappers:kubeadm:default-node-token"},
	}}

	if err := nodebootstraptokenphase.CreateNewTokens(clientset, tokens); err != nil {
		log.Printf("Error creating token: %s", err)
		return "", err
	}
	// This reads server info of the current context from the config file
	server, err := custconfig.GetServerOfCurrentContext()
	if err != nil {
		log.Println(err)
		return "", err
	}
	server = s.Trim(server, "https://")
	server = s.Trim(server, "http://")
	// This reads CA cert to be hashed
	certs, err := cert.CertsFromFile("/etc/kubernetes/pki/ca.crt")
	if err != nil {
		log.Println(err)
		return "", err
	}
	var CA string
	for i, cert := range certs {
		if i == 0 {
			hashedCA := sha256.Sum256([]byte(cert.RawSubjectPublicKeyInfo))
			CA = fmt.Sprintf("sha256:%x", hashedCA)
		}
	}

	joinCommand := fmt.Sprintf("kubeadm join %s --token %s --discovery-token-ca-cert-hash %s", server, tokens[0].Token.String(), CA)
	return joinCommand, nil
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

// SetHostname allows comparing the current hosts with requested hostname by DNS check
func SetHostname(client *namecheap.Client, hostRecord namecheap.DomainDNSHost) (bool, string) {
	hostList := getHosts(client)
	exist := false
	for _, host := range hostList.Hosts {
		if host.Name == hostRecord.Name || host.Address == hostRecord.Address {
			exist = true
			break
		}
	}

	if exist {
		log.Printf("Hostname or ip address already exists: %s - %s", hostRecord.Name, hostRecord.Address)
		return false, "exist"
	}

	hostList.Hosts = append(hostList.Hosts, hostRecord)
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
