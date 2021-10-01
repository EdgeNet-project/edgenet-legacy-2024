package main

import (
	"flag"
	"k8s.io/klog"
	"log"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha/acceptableusepolicy"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

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
	// Start the controller to provide the functionalities of acceptableusepolicy resource
	acceptableusepolicy.Start(kubeclientset, edgenetclientset)
}
