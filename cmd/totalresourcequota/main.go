package main

import (
	"edgenet/pkg/bootstrap"
	"edgenet/pkg/controller/v1alpha/totalresourcequota"
	"log"
)

func main() {
	// Set kubeconfig to be used to create clientsets
	bootstrap.SetKubeConfig()
	clientset, err := bootstrap.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	edgenetClientset, err := bootstrap.CreateEdgeNetClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	// Start the controller to provide the functionalities of total resource quota resource
	totalresourcequota.Start(clientset, edgenetClientset)
}
