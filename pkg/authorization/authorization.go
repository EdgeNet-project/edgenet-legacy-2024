package authorization

import (
	"flag"
	"os"
	"path/filepath"
	"log"

	"headnode/pkg/config"
	geolocationclientset "headnode/pkg/client/clientset/versioned"

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

// CreateGeoLocationClientSet generates the clientset to interact with geolocation custom resource
func CreateGeoLocationClientSet() (*geolocationclientset.Clientset, error) {
	// Use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	// Create the clientset
	clientset, err := geolocationclientset.NewForConfig(config)
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
