package main

import (
	"flag"
	"log"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha1/sliceclaim"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/signals"

	"k8s.io/klog"
)

func main() {
	klog.InitFlags(nil)
	provisioning := flag.String("provisioning", "Dynamic", "Working mode to automate slice creation")
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
	// Start the controller to provide the functionalities of sliceclaim resource
	edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, time.Second*30)

	controller := sliceclaim.NewController(kubeclientset,
		edgenetclientset,
		edgenetInformerFactory.Core().V1alpha1().SubNamespaces(),
		edgenetInformerFactory.Core().V1alpha1().SliceClaims(),
		*provisioning)

	edgenetInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}
