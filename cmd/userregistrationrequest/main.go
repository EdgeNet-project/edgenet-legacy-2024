package main

import (
	"log"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/controller/registration/v1alpha/userrequest"
)

func main() {
	// TODO: Pass an argument to select using kubeconfig or service account for clients
	// bootstrap.SetKubeConfig()
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
	// Start the controller to provide the functionalities of userrequest resource
	userrequest.Start(kubeclientset, edgenetclientset)
}
