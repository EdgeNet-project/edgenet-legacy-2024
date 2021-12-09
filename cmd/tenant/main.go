package main

import (
	"flag"
	"log"
	"time"

	"k8s.io/klog"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha/tenant"
	"github.com/EdgeNet-project/edgenet/pkg/signals"

	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	kubeinformers "k8s.io/client-go/informers"
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
	// Start the controller to provide the functionalities of nodecontribution resource
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclientset, time.Second*30)
	edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, 0)

	controller := tenant.NewController(kubeclientset,
		edgenetclientset,
		edgenetInformerFactory.Core().V1alpha().Tenants())

	kubeInformerFactory.Start(stopCh)
	edgenetInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}
