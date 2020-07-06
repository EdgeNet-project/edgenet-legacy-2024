package main

import (
	"edgenet/pkg/authorization"
	"edgenet/pkg/client/clientset/versioned"
	"edgenet/pkg/controller/v1alpha/nodecontribution"
	"log"

	"k8s.io/client-go/kubernetes"
)

func main(kubernetes kubernetes.Interface, edgenet versioned.Interface) {
	// Set kubeconfig to be used to create clientsets
	authorization.SetKubeConfig()
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	edgenetClientset, err := authorization.CreateEdgeNetClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	// Start the controller to provide the functionalities of nodecontribution resource
	nodecontribution.Start(clientset, edgenetClientset)
}
