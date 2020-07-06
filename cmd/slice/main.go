package main

import (
	"edgenet/pkg/authorization"
	"edgenet/pkg/controller/v1alpha/slice"
	"log"
)

func main() {
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
	// Start the controller to provide the functionalities of slice resource
	slice.Start(clientset, edgenetClientset)
}
