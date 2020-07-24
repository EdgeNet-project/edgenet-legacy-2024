/*
Copyright 2020 Sorbonne Université

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

package registration

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"time"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/util"

	yaml "gopkg.in/yaml.v2"
	"k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/cert"
	kubeconfigutil "k8s.io/kubernetes/cmd/kubeadm/app/util/kubeconfig"
	cmdconfig "k8s.io/kubernetes/pkg/kubectl/cmd/config"
)

// headnode implementation
type headnode struct {
	DNS string `yaml:"dns"`
	IP  string `yaml:"ip"`
}

// Clientset to be synced by the custom resources
var Clientset kubernetes.Interface

// MakeUser generates key and certificate and then set user credentials into the config file.
func MakeUser(authority, username, email string) ([]byte, []byte, error) {

	var headnodePath string

	flag.Parse()
	args := flag.Args()
	if len(args) == 1 {
		commandLine := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		commandLine.StringVar(&headnodePath, "headnode-path", "", "headnode-path")
		commandLine.Parse(os.Args[0:2])
	}

	path := fmt.Sprintf("../../assets/certs/%s", email)
	reader := rand.Reader
	bitSize := 4096

	key, _ := rsa.GenerateKey(reader, bitSize)
	pemdata := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)

	subject := pkix.Name{
		CommonName:   email,
		Organization: []string{authority},
	}

	file, err := os.Open("../../configs/headnode.yaml")
	if err != nil {
		log.Printf("Registration: unexpected error executing command: %v", err)
	}
	if headnodePath != "" {
		file, err = os.Open(headnodePath)
		if err != nil {
			log.Printf("Registration: unexpected error executing command: %v", err)
			return nil, nil, err
		}
	}
	decoder := yaml.NewDecoder(file)
	var headnode headnode
	err = decoder.Decode(&headnode)
	if err != nil {
		log.Printf("Registration: unexpected error executing command: %v", err)
		return nil, nil, err
	}
	dnsSANs := []string{headnode.DNS}
	ipSANs := []net.IP{net.ParseIP(headnode.IP)}

	csr, _ := cert.MakeCSR(key, &subject, dnsSANs, ipSANs)

	var CSRCopy *v1beta1.CertificateSigningRequest
	CSRObject := v1beta1.CertificateSigningRequest{}
	CSRObject.Name = fmt.Sprintf("%s-%s", authority, username)
	CSRObject.Spec.Groups = []string{"system:authenticated"}
	CSRObject.Spec.Usages = []v1beta1.KeyUsage{"digital signature", "key encipherment", "server auth", "client auth"}
	CSRObject.Spec.Request = csr
	CSRCopyCreated, err := Clientset.CertificatesV1beta1().CertificateSigningRequests().Create(&CSRObject)
	if err != nil {
		return nil, nil, err
	}
	CSRCopy = CSRCopyCreated
	CSRCopy.Status.Conditions = append(CSRCopy.Status.Conditions, v1beta1.CertificateSigningRequestCondition{
		Type:           v1beta1.CertificateApproved,
		Reason:         "User creation is completed",
		Message:        "This CSR was approved automatically by EdgeNet",
		LastUpdateTime: metav1.Now(),
	})
	_, err = Clientset.CertificatesV1beta1().CertificateSigningRequests().UpdateApproval(CSRCopy)
	if err != nil {
		return nil, nil, err
	}
	timeout := time.After(15 * time.Minute)
	ticker := time.Tick(15 * time.Second)
	if headnodePath != "" {
		timeout = time.After(3 * time.Second)
		ticker = time.Tick(1 * time.Second)
	}
check:
	for {
		select {
		case <-timeout:
			return nil, nil, err
		case <-ticker:
			CSRCopy, err = Clientset.CertificatesV1beta1().CertificateSigningRequests().Get(CSRCopy.GetName(), metav1.GetOptions{})
			if err != nil {
				return nil, nil, err
			}
			if len(CSRCopy.Status.Certificate) != 0 {
				break check
			}
		}
	}
	err = ioutil.WriteFile(fmt.Sprintf("%s.crt", path), CSRCopy.Status.Certificate, 0700)
	if err != nil {
		return nil, nil, err
	}
	err = ioutil.WriteFile(fmt.Sprintf("%s.key", path), pemdata, 0700)
	if err != nil {
		return nil, nil, err
	}
	pathOptions := clientcmd.NewDefaultPathOptions()
	buf := bytes.NewBuffer([]byte{})
	kcmd := cmdconfig.NewCmdConfigSetAuthInfo(buf, pathOptions)
	kcmd.SetArgs([]string{email})
	kcmd.Flags().Parse([]string{
		fmt.Sprintf("--client-certificate=/var/www/edgenet/assets/certs/%s.crt", email),
		fmt.Sprintf("--client-key=/var/www/edgenet/assets/certs/%s.key", email),
	})

	if err := kcmd.Execute(); err != nil {
		log.Printf("Couldn't set auth info on the kubeconfig file: %s", username)
		return nil, nil, err
	}
	return CSRCopy.Status.Certificate, pemdata, nil
}

// MakeConfig reads cluster, server, and CA info of the current context from the config file
// to use them on the creation of kubeconfig. Then generates kubeconfig by certs.
func MakeConfig(authority, username, email string, clientCert, clientKey []byte) error {
	// Define the cluster and server by taking advantage of the current config file
	cluster, server, CA, err := util.GetClusterServerOfCurrentContext()
	if err != nil {
		log.Println(err)
		return err
	}
	// Put the collected data into new kubeconfig file
	newKubeConfig := kubeconfigutil.CreateWithCerts(server, cluster, email, CA, clientKey, clientCert)
	newKubeConfig.Contexts[newKubeConfig.CurrentContext].Namespace = fmt.Sprintf("authority-%s", authority)
	kubeconfigutil.WriteToDisk(fmt.Sprintf("../../assets/kubeconfigs/%s-%s.cfg", authority, username), newKubeConfig)
	// Check if the creation process is completed
	_, err = ioutil.ReadFile(fmt.Sprintf("../../assets/kubeconfigs/%s-%s.cfg", authority, username))
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// CreateServiceAccount makes a service account to serve for permanent jobs.
func CreateServiceAccount(userCopy *apps_v1alpha.User, accountType string, ownerReferences []metav1.OwnerReference) (*corev1.ServiceAccount, error) {
	// Set the name of service account according to the type
	name := userCopy.GetName()
	serviceAccount := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name, OwnerReferences: ownerReferences}}
	serviceAccountCreated, err := Clientset.CoreV1().ServiceAccounts(userCopy.GetNamespace()).Create(serviceAccount)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return serviceAccountCreated, nil
}

// CreateConfig checks serviceaccount of the user and then it gets that secret to use CA and token information.
// Subsequently, that reads cluster and server info of the current context from the config file to be consumed
// on the creation of kubeconfig.
func CreateConfig(serviceAccount *corev1.ServiceAccount) string {
	// To find out the secret name to use
	accountSecretName := ""
	for _, accountSecret := range serviceAccount.Secrets {
		match, _ := regexp.MatchString("([a-z0-9]+)-token-([a-z0-9]+)", accountSecret.Name)
		if match {
			accountSecretName = accountSecret.Name
			break
		}
	}
	// If there is no matching secret terminate this function as generating kubeconfig file is not possible
	if accountSecretName == "" {
		log.Printf("Serviceaccount %s in %s doesn't have a serviceaccount token", serviceAccount.GetName(), serviceAccount.GetNamespace())
		return fmt.Sprintf("Serviceaccount %s doesn't have a serviceaccount token\n", serviceAccount.GetName())
	}
	secret, err := Clientset.CoreV1().Secrets(serviceAccount.GetNamespace()).Get(accountSecretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Printf("Secret for %s in %s not found", serviceAccount.GetName(), serviceAccount.GetNamespace())
		return fmt.Sprintf("Secret %s not found\n", serviceAccount.GetName())
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Printf("Error getting secret %s in %s: %v", serviceAccount.GetName(), serviceAccount.GetNamespace(), statusError.ErrStatus)
		return fmt.Sprintf("Error getting secret %s: %v\n", serviceAccount.GetName(), statusError.ErrStatus)
	} else if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	// Define the cluster and server by taking advantage of the current config file
	cluster, server, _, err := util.GetClusterServerOfCurrentContext()
	if err != nil {
		log.Println(err)
		return fmt.Sprintf("Err: %s", err)
	}
	// Put the collected data into new kubeconfig file
	newKubeConfig := kubeconfigutil.CreateWithToken(server, cluster, serviceAccount.GetName(), secret.Data["ca.crt"], string(secret.Data["token"]))
	newKubeConfig.Contexts[newKubeConfig.CurrentContext].Namespace = serviceAccount.GetNamespace()
	kubeconfigutil.WriteToDisk(fmt.Sprintf("../../assets/kubeconfigs/edgenet-%s-%s.cfg", serviceAccount.GetNamespace(), serviceAccount.GetName()), newKubeConfig)
	// Check whether the creation process is completed
	dat, err := ioutil.ReadFile(fmt.Sprintf("../../assets/kubeconfigs/edgenet-%s-%s.cfg", serviceAccount.GetNamespace(), serviceAccount.GetName()))
	if err != nil {
		log.Println(err)
		return fmt.Sprintf("Err: %s", err)
	}
	return string(dat)
}
