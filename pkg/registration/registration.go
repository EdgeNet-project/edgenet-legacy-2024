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
	"context"
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

	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"

	yaml "gopkg.in/yaml.v2"
	certv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/cert"
	//cmdconfig "k8s.io/kubernetes/pkg/kubectl/cmd/config"
)

// headnode implementation
type headnode struct {
	DNS string `yaml:"dns"`
	IP  string `yaml:"ip"`
}

// Clientset to be synced by the custom resources
var Clientset kubernetes.Interface
var dir = "../.."

// MakeUser generates key and certificate and then set user credentials into the config file.
func MakeUser(tenant, username, email string) ([]byte, []byte, error) {
	// The code below inits dir
	if flag.Lookup("dir") != nil {
		dir = flag.Lookup("dir").Value.(flag.Getter).Get().(string)
	}
	path := fmt.Sprintf("%s/assets/certs/%s", dir, email)
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
		Organization: []string{tenant},
	}
	// Opening the default path for headnode
	file, err := os.Open(fmt.Sprintf("%s/configs/headnode.yaml", dir))
	if err != nil {
		log.Printf("Registration: unexpected error executing command: %v", err)
		return nil, nil, err
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
	csrRequest, _ := cert.MakeCSR(key, &subject, dnsSANs, ipSANs)

	var csrObject *certv1.CertificateSigningRequest = new(certv1.CertificateSigningRequest)
	csrObject.Name = fmt.Sprintf("%s-%s", tenant, username)
	csrObject.Spec.Usages = []certv1.KeyUsage{"client auth"}
	csrObject.Spec.Request = csrRequest
	csrObject.Spec.SignerName = "kubernetes.io/kube-apiserver-client"
	csr, err := Clientset.CertificatesV1().CertificateSigningRequests().Create(context.TODO(), csrObject, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, err
	}

	csr.Status.Conditions = append(csr.Status.Conditions, certv1.CertificateSigningRequestCondition{
		Type:           certv1.CertificateApproved,
		Reason:         "User creation is completed",
		Message:        "This CSR has been approved automatically by EdgeNet",
		Status:         "True",
		LastUpdateTime: metav1.Now(),
	})

	_, err = Clientset.CertificatesV1().CertificateSigningRequests().UpdateApproval(context.TODO(), csr.GetName(), csr, metav1.UpdateOptions{})
	if err != nil {
		return nil, nil, err
	}
	timeout := time.After(5 * time.Minute)
	ticker := time.Tick(3 * time.Second)
check:
	for {
		select {
		case <-timeout:
			return nil, nil, err
		case <-ticker:
			csr, err = Clientset.CertificatesV1().CertificateSigningRequests().Get(context.TODO(), csr.GetName(), metav1.GetOptions{})
			if err != nil {
				return nil, nil, err
			}
			if len(csr.Status.Certificate) != 0 {
				break check
			}
		}
	}
	err = ioutil.WriteFile(fmt.Sprintf("%s.crt", path), csr.Status.Certificate, 0700)
	if err != nil {
		return nil, nil, err
	}
	err = ioutil.WriteFile(fmt.Sprintf("%s.key", path), pemdata, 0700)
	if err != nil {
		return nil, nil, err
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	// if you want to change the loading rules (which files in which order), you can do so here

	configOverrides := &clientcmd.ConfigOverrides{}
	// if you want to change override values or bind them to flags, there are methods to help you

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		// Log the error to debug
		log.Println(err)
		return nil, nil, err
	}
	authInfo := api.AuthInfo{}
	authInfo.Username = email
	authInfo.ClientCertificateData = csr.Status.Certificate
	authInfo.ClientKeyData = pemdata
	rawConfig.AuthInfos[email] = &authInfo
	err = clientcmd.ModifyConfig(kubeConfig.ConfigAccess(), rawConfig, false)
	if err != nil {
		log.Println(err)
		return nil, nil, err
	}
	/*pathOptions := clientcmd.NewDefaultPathOptions()
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
	}*/

	return csr.Status.Certificate, pemdata, nil
}

// MakeConfig reads cluster, server, and CA info of the current context from the config file
// to use them on the creation of kubeconfig. Then generates kubeconfig by certs.
func MakeConfig(tenant, username, email string, clientCert, clientKey []byte) error {
	// Define the cluster and server by taking advantage of the current config file
	/*cluster, server, CA, err := util.GetClusterServerOfCurrentContext()
	if err != nil {
		log.Println(err)
		return err
	}*/
	// Put the collected data into new kubeconfig file
	/*newKubeConfig := kubeconfigutil.CreateWithCerts(server, cluster, email, CA, clientKey, clientCert)
	newKubeConfig.Contexts[newKubeConfig.CurrentContext].Namespace = tenant
	kubeconfigutil.WriteToDisk(fmt.Sprintf("../../assets/kubeconfigs/%s-%s.cfg", tenant, username), newKubeConfig)*/
	// Check if the creation process is completed
	/*_, err = ioutil.ReadFile(fmt.Sprintf("../../assets/kubeconfigs/%s-%s.cfg", tenant, username))
	if err != nil {
		log.Println(err)
		return err
	}*/
	// The code below inits dir
	if flag.Lookup("dir") != nil {
		dir = flag.Lookup("dir").Value.(flag.Getter).Get().(string)
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	// if you want to change the loading rules (which files in which order), you can do so here

	configOverrides := &clientcmd.ConfigOverrides{}
	// if you want to change override values or bind them to flags, there are methods to help you

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		// Log the error to debug
		return err
	}
	rawConfig.AuthInfos = map[string]*api.AuthInfo{}
	userContext := rawConfig.Contexts[rawConfig.CurrentContext]
	userContext.Namespace = tenant
	userContext.AuthInfo = email
	rawConfig.Contexts = map[string]*api.Context{}
	rawConfig.Contexts[tenant] = userContext
	rawConfig.CurrentContext = tenant

	authInfo := api.AuthInfo{}
	authInfo.Username = email
	authInfo.ClientCertificateData = clientCert
	authInfo.ClientKeyData = clientKey
	rawConfig.AuthInfos[email] = &authInfo

	clientcmd.WriteToFile(rawConfig, fmt.Sprintf("%s/assets/kubeconfigs/%s-%s.cfg", dir, tenant, username))
	_, err = ioutil.ReadFile(fmt.Sprintf("%s/assets/kubeconfigs/%s-%s.cfg", dir, tenant, username))
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// CreateServiceAccount makes a service account to serve for permanent jobs.
func CreateServiceAccount(user registrationv1alpha.UserRequest, accountType string, ownerReferences []metav1.OwnerReference) (*corev1.ServiceAccount, error) {
	// Set the name of service account according to the type
	name := user.GetName()
	serviceAccount := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name, OwnerReferences: ownerReferences}}
	serviceAccountCreated, err := Clientset.CoreV1().ServiceAccounts(user.Spec.Tenant).Create(context.TODO(), serviceAccount, metav1.CreateOptions{})
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
		log.Printf("Serviceaccount %s in %s doesn't have a token", serviceAccount.GetName(), serviceAccount.GetNamespace())
		return fmt.Sprintf("Serviceaccount %s doesn't have a token", serviceAccount.GetName())
	}
	secret, err := Clientset.CoreV1().Secrets(serviceAccount.GetNamespace()).Get(context.TODO(), accountSecretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Printf("Secret for %s in %s not found", serviceAccount.GetName(), serviceAccount.GetNamespace())
		return fmt.Sprintf("Secret %s not found", serviceAccount.GetName())
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Printf("Error getting secret %s in %s: %v", serviceAccount.GetName(), serviceAccount.GetNamespace(), statusError.ErrStatus)
		return fmt.Sprintf("Error getting secret %s: %v", serviceAccount.GetName(), statusError.ErrStatus)
	} else if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	// Define the cluster and server by taking advantage of the current config file
	/*cluster, server, _, err := util.GetClusterServerOfCurrentContext()
	if err != nil {
		log.Println(err)
		return fmt.Sprintf("Err: %s", err)
	}
	// Put the collected data into new kubeconfig file
	newKubeConfig := kubeconfigutil.CreateWithToken(server, cluster, serviceAccount.GetName(), secret.Data["ca.crt"], string(secret.Data["token"]))
	newKubeConfig.Contexts[newKubeConfig.CurrentContext].Namespace = serviceAccount.GetNamespace()
	kubeconfigutil.WriteToDisk(fmt.Sprintf("../../assets/kubeconfigs/edgenet-%s-%s.cfg", serviceAccount.GetNamespace(), serviceAccount.GetName()), newKubeConfig)
	*/

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	// if you want to change the loading rules (which files in which order), you can do so here

	configOverrides := &clientcmd.ConfigOverrides{}
	// if you want to change override values or bind them to flags, there are methods to help you

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		// Log the error to debug
		return fmt.Sprintf("Err: %s", err)
	}
	rawConfig.AuthInfos = map[string]*api.AuthInfo{}
	userContext := rawConfig.Contexts[rawConfig.CurrentContext]
	userContext.Namespace = serviceAccount.GetNamespace()
	userContext.AuthInfo = serviceAccount.GetName()
	rawConfig.Contexts = map[string]*api.Context{}
	rawConfig.Contexts[serviceAccount.GetNamespace()] = userContext
	rawConfig.CurrentContext = serviceAccount.GetNamespace()

	authInfo := api.AuthInfo{}
	authInfo.Username = serviceAccount.GetName()
	authInfo.Token = string(secret.Data["token"])
	rawConfig.AuthInfos[serviceAccount.GetName()] = &authInfo

	clientcmd.WriteToFile(rawConfig, fmt.Sprintf("%s/assets/kubeconfigs/%s-%s.cfg", dir, serviceAccount.GetNamespace(), serviceAccount.GetName()))
	// Check whether the creation process is completed
	dat, err := ioutil.ReadFile(fmt.Sprintf("%s/assets/kubeconfigs/%s-%s.cfg", dir, serviceAccount.GetNamespace(), serviceAccount.GetName()))
	if err != nil {
		log.Println(err)
		return fmt.Sprintf("Err: %s", err)
	}
	return string(dat)
}
