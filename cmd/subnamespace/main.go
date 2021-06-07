package main

import (
	"log"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha/subnamespace"
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
	// Start the controller to provide the functionalities of subnamespace resource
	subnamespace.Start(clientset, edgenetClientset)
}
