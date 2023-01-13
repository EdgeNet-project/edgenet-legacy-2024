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

package multiprovider

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"

	//nodebootstraptokenphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/bootstraptoken/node"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	kubeadmtypes "sigs.k8s.io/cluster-api/bootstrap/kubeadm/types/v1beta1"
)

// A part of the general structure of a kubeconfig file
type clusterDetails struct {
	CA     []byte `json:"certificate-authority-data"`
	Server string `json:"server"`
}
type clusters struct {
	Cluster clusterDetails `json:"cluster"`
	Name    string         `json:"name"`
}
type contextDetails struct {
	Cluster string `json:"cluster"`
	User    string `json:"user"`
}
type contexts struct {
	Context contextDetails `json:"context"`
	Name    string         `json:"name"`
}
type configView struct {
	Clusters       []clusters `json:"clusters"`
	Contexts       []contexts `json:"contexts"`
	CurrentContext string     `json:"current-context"`
}

// This reads the kubeconfig file by admin context and returns it in json format.
func getConfigView() (api.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	// if you want to change the loading rules (which files in which order), you can do so here

	configOverrides := &clientcmd.ConfigOverrides{}
	// if you want to change override values or bind them to flags, there are methods to help you

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		// Do something
		return rawConfig, err
	}
	return rawConfig, nil
}

// GetServerOfCurrentContext provides the server info of the current context
func getServerOfCurrentContext() (string, error) {
	rawConfig, err := getConfigView()
	if err != nil {
		log.Printf("unexpected error executing command: %v", err)
		return "", err
	}
	var server string = rawConfig.Clusters["kubernetes"].Server
	return server, nil
}

// CreateToken creates the token to be used to add node
// and return the token
func (m *Manager) createToken(duration time.Duration, hostname string) (string, error) {
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

	secret, err := m.kubeclientset.CoreV1().Secrets(metav1.NamespaceSystem).Get(context.TODO(), token.ID, metav1.GetOptions{})
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

	if _, err := m.kubeclientset.CoreV1().Secrets(secret.ObjectMeta.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return "", err
		}

		if _, err := m.kubeclientset.CoreV1().Secrets(secret.ObjectMeta.Namespace).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
			return "", err
		}
	}

	// This is to get server info
	kubeconfigPath := "/edgenet/.kube/config"
	if flag.Lookup("kubeconfig-path") != nil {
		kubeconfigPath = flag.Lookup("kubeconfig-path").Value.(flag.Getter).Get().(string)
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Println(err.Error())
		return "", err
	}
	server := config.Host
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
