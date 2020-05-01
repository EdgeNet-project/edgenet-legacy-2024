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

// The credit for this namecheap API communication goes to:
// https://github.com/billputer/go-namecheap

package authorization

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	edgenetclientset "edgenet/pkg/client/clientset/versioned"
	"edgenet/pkg/config"

	namecheap "github.com/billputer/go-namecheap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeconfig string

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}

// SetKubeConfig declares the options and calls parse before using them to set kubeconfig variable
func SetKubeConfig() {
	if home := homeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
}

// CreateEdgeNetClientSet generates the clientset to interact with custom resources of selective deployment, authority, user, and slice
func CreateEdgeNetClientSet() (*edgenetclientset.Clientset, error) {
	// Use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	// Create the clientset
	clientset, err := edgenetclientset.NewForConfig(config)
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	return clientset, err
}

// CreateClientSet generates the clientset to interact with Kubernetes
func CreateClientSet() (*kubernetes.Clientset, error) {
	// Use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	return clientset, err
}

// CreateNameCheapClient generates the client to interact with Namecheap API
func CreateNamecheapClient() (*namecheap.Client, error) {
	apiuser, apitoken, username, err := config.GetNamecheapCredentials()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	client := namecheap.NewClient(apiuser, apitoken, username)
	return client, nil
}
