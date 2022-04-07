package main

import (
	"flag"
	"log"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha1/slice"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/signals"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/klog"
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
	// Start the controller to provide the functionalities of slice resource
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclientset, time.Second*30)
	edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, 0)

	controller := slice.NewController(kubeclientset,
		edgenetclientset,
		edgenetInformerFactory.Core().V1alpha1().SliceClaims(),
		edgenetInformerFactory.Core().V1alpha1().Slices())

	kubeInformerFactory.Start(stopCh)
	edgenetInformerFactory.Start(stopCh)

	if err = controller.Run(1, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}
