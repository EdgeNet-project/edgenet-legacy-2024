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

package infrastructure

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/util"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"

	namecheap "github.com/billputer/go-namecheap"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"

	//nodebootstraptokenphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/bootstraptoken/node"

	kubeadmtypes "sigs.k8s.io/cluster-api/bootstrap/kubeadm/types/v1beta1"
)

// CreateToken creates the token to be used to add node
// and return the token
func CreateToken(clientset kubernetes.Interface, duration time.Duration, hostname string) (string, error) {
	tokenStr, err := bootstraputil.GenerateBootstrapToken()
	if err != nil {
		log.Printf("Error generating token to upload certs: %s", err)
		return "", err
	}
	token, err := kubeadmtypes.NewBootstrapTokenString(tokenStr)
	if err != nil {
		log.Printf("Error creating upload certs token: %s", err)
		return "", err
	}
	bootstrapToken := kubeadmtypes.BootstrapToken{}
	bootstrapToken.Description = fmt.Sprintf("EdgeNet token for adding node called %s", hostname)
	bootstrapToken.TTL = &metav1.Duration{
		Duration: duration,
	}
	bootstrapToken.Usages = []string{"authentication", "signing"}
	bootstrapToken.Groups = []string{"system:bootstrappers:kubeadm:default-node-token"}
	bootstrapToken.Token = token

	secret, err := clientset.CoreV1().Secrets(metav1.NamespaceSystem).Get(context.TODO(), token.ID, metav1.GetOptions{})
	if secret != nil && err == nil {
		return "", err
	}
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("bootstrap-token-%s", token.ID),
			Namespace: metav1.NamespaceSystem,
		},
		Type: corev1.SecretType(bootstrapapi.SecretTypeBootstrapToken),
		Data: encodeTokenSecretData(bootstrapToken.DeepCopy(), time.Now()),
	}

	if _, err := clientset.CoreV1().Secrets(secret.ObjectMeta.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return "", err
		}

		if _, err := clientset.CoreV1().Secrets(secret.ObjectMeta.Namespace).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
			return "", err
		}
	}

	// This reads server info of the current context from the config file
	server, err := util.GetServerOfCurrentContext()
	if err != nil {
		log.Println(err)
		return "", err
	}
	server = strings.Trim(server, "https://")
	server = strings.Trim(server, "http://")
	pathCA := "/etc/kubernetes/pki/ca.crt"
	if flag.Lookup("ca-path") != nil {
		pathCA = flag.Lookup("ca-path").Value.(flag.Getter).Get().(string)
	}
	// This reads CA cert to be hashed
	certs, err := cert.CertsFromFile(pathCA)
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

	joinCommand := fmt.Sprintf("kubeadm join %s --token %s --discovery-token-ca-cert-hash %s", server, tokenStr, CA)
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

// AddARecordRoute53 adds a new A record to the Route53
func AddARecordRoute53(input *route53.ChangeResourceRecordSetsInput) (bool, string) {
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

// encodeTokenSecretData takes the token discovery object and an optional duration and returns the .Data for the Secret
// now is passed in order to be able to used in unit testing
func encodeTokenSecretData(token *kubeadmtypes.BootstrapToken, now time.Time) map[string][]byte {
	data := map[string][]byte{
		bootstrapapi.BootstrapTokenIDKey:     []byte(token.Token.ID),
		bootstrapapi.BootstrapTokenSecretKey: []byte(token.Token.Secret),
	}

	if len(token.Description) > 0 {
		data[bootstrapapi.BootstrapTokenDescriptionKey] = []byte(token.Description)
	}

	// If for some strange reason both token.TTL and token.Expires would be set
	// (they are mutually exclusive in validation so this shouldn't be the case),
	// token.Expires has higher priority, as can be seen in the logic here.
	if token.Expires != nil {
		// Format the expiration date accordingly
		// TODO: This maybe should be a helper function in bootstraputil?
		expirationString := token.Expires.Time.Format(time.RFC3339)
		data[bootstrapapi.BootstrapTokenExpirationKey] = []byte(expirationString)

	} else if token.TTL != nil && token.TTL.Duration > 0 {
		// Only if .Expires is unset, TTL might have an effect
		// Get the current time, add the specified duration, and format it accordingly
		expirationString := now.Add(token.TTL.Duration).Format(time.RFC3339)
		data[bootstrapapi.BootstrapTokenExpirationKey] = []byte(expirationString)
	}

	for _, usage := range token.Usages {
		data[bootstrapapi.BootstrapTokenUsagePrefix+usage] = []byte("true")
	}

	if len(token.Groups) > 0 {
		data[bootstrapapi.BootstrapTokenExtraGroupsKey] = []byte(strings.Join(token.Groups, ","))
	}
	return data
}
