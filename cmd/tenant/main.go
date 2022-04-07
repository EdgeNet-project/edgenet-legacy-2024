package main

import (
	"flag"
	"log"

	"k8s.io/klog"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha1/tenant"
	"github.com/EdgeNet-project/edgenet/pkg/signals"

	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	stopCh := signals.SetupSignalHandler()
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
	antreaclientset, err := bootstrap.CreateAntreaClientset("serviceaccount")
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	// Start the controller to provide the functionalities of tenant resource
	edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, 0)

	controller := tenant.NewController(kubeclientset,
		edgenetclientset,
		antreaclientset,
		edgenetInformerFactory.Core().V1alpha1().Tenants())

	edgenetInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}
