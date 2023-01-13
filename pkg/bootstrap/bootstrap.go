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

// The credit for this namecheap API communication goes to:
// https://github.com/billputer/go-namecheap

package bootstrap

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"gopkg.in/yaml.v2"

	antrea "antrea.io/antrea/pkg/client/clientset/versioned"
	namecheap "github.com/billputer/go-namecheap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Structure of Namecheap access credentials
type namecheapCred struct {
	App      string `yaml:"app"`
	APIUser  string `yaml:"apiUser"`
	APIToken string `yaml:"apiToken"`
	Username string `yaml:"username"`
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}

// GetDefaultKubeconfigPath returns the default kubeconfig path
func GetDefaultKubeconfigPath() string {
	var kubeconfigPath string
	if home := homeDir(); home != "" {
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	} else {
		kubeconfigPath = "/edgenet/.kube/config"
	}
	return kubeconfigPath
}

func getKubeconfigPath() string {
	kubeconfigPath := GetDefaultKubeconfigPath()
	if flag.Lookup("kubeconfig-path") != nil {
		kubeconfigPath = flag.Lookup("kubeconfig-path").Value.(flag.Getter).Get().(string)
	}
	return kubeconfigPath
}

// CreateEdgeNetClientset generates the clientset to interact with the custom resources
func CreateEdgeNetClientset(by string) (*clientset.Clientset, error) {
	var edgenetclientset *clientset.Clientset
	var generateClientset = func(config *rest.Config) *clientset.Clientset {
		// Create the clientset
		edgenetclientset, err := clientset.NewForConfig(config)
		if err != nil {
			// TODO: Error handling
			panic(err.Error())
		}
		return edgenetclientset
	}

	if by == "kubeconfig" {
		// Use the current context in kubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", getKubeconfigPath())
		if err != nil {
			log.Println(err.Error())
			panic(err.Error())
		}
		edgenetclientset = generateClientset(config)
	} else {
		// Creates the in-cluster config
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		edgenetclientset = generateClientset(config)
	}
	return edgenetclientset, nil
}

// CreateClientset generates the clientset to interact with the Kubernetes resources
func CreateClientset(by string) (*kubernetes.Clientset, error) {
	var kubeclientset *kubernetes.Clientset
	var generateClientset = func(config *rest.Config) *kubernetes.Clientset {
		// Create the clientset
		kubeclientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			// TODO: Error handling
			panic(err.Error())
		}
		return kubeclientset
	}

	if by == "kubeconfig" {
		// Use the current context in kubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", getKubeconfigPath())
		if err != nil {
			// TODO: Error handling
			panic(err.Error())
		}
		kubeclientset = generateClientset(config)
	} else {
		// Creates the in-cluster config
		config, err := rest.InClusterConfig()
		if err != nil {
			// TODO: Error handling
			panic(err.Error())
		}
		kubeclientset = generateClientset(config)
	}
	return kubeclientset, nil
}

// CreateNamecheapClient generates the client to interact with Namecheap API
func CreateNamecheapClient() (*namecheap.Client, error) {
	apiuser, apitoken, username, err := getNamecheapCredentials()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	client := namecheap.NewClient(apiuser, apitoken, username)
	return client, nil
}

// CreateAntreaClientset generates the clientset to interact with the Antrea resources
func CreateAntreaClientset(by string) (*antrea.Clientset, error) {
	var antreaclientset *antrea.Clientset
	var generateClientset = func(config *rest.Config) *antrea.Clientset {
		// Create the clientset
		antreaclientset, err := antrea.NewForConfig(config)
		if err != nil {
			// TODO: Error handling
			panic(err.Error())
		}
		return antreaclientset
	}

	if by == "kubeconfig" {
		// Use the current context in kubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", getKubeconfigPath())
		if err != nil {
			log.Println(err.Error())
			panic(err.Error())
		}
		antreaclientset = generateClientset(config)
	} else {
		// Creates the in-cluster config
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		antreaclientset = generateClientset(config)
	}
	return antreaclientset, nil
}

// getNamecheapCredentials provides authentication info to have API Access
func getNamecheapCredentials() (string, string, string, error) {
	// The path of the yaml config file of namecheap
	namecheapPath := "."
	if flag.Lookup("configs-path") != nil {
		namecheapPath = flag.Lookup("configs-path").Value.(flag.Getter).Get().(string)
	}
	file, err := os.Open(fmt.Sprintf("%s/namecheap.yaml", namecheapPath))
	if err != nil {
		log.Printf("unexpected error executing command: %v", err)
		return "", "", "", err
	}

	decoder := yaml.NewDecoder(file)
	var namecheapCred namecheapCred
	err = decoder.Decode(&namecheapCred)
	if err != nil {
		log.Printf("unexpected error executing command: %v", err)
		return "", "", "", err
	}
	return namecheapCred.APIUser, namecheapCred.APIToken, namecheapCred.Username, nil
}
