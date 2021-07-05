package main

import (
	"log"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha/tenant"
)

func main() {
	kubeclientset, err := bootstrap.CreateClientset("serviceaccount")
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	edgenetclientset, err := bootstrap.CreateEdgeNetClientset("serviceaccount")
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	// Start the controller to provide the functionalities of tenant resource
	tenant.Start(kubeclientset, edgenetclientset)
}
