package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"k8s.io/klog"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha1/tenant"
	"github.com/EdgeNet-project/edgenet/pkg/signals"

	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
)

func main() {
	klog.InitFlags(nil)
	flag.String("kubeconfig-path", bootstrap.GetDefaultKubeconfigPath(), "Path to the kubeconfig file's directory")
	flag.Parse()

	stopCh := signals.SetupSignalHandler()
	var authentication string
	if authentication = strings.TrimSpace(os.Getenv("AUTHENTICATION_STRATEGY")); authentication != "kubeconfig" {
		authentication = "serviceaccount"
	}
	config, err := bootstrap.GetRestConfig(authentication)
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	kubeclientset, err := bootstrap.CreateKubeClientset(config)
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	edgenetclientset, err := bootstrap.CreateEdgeNetClientset(config)
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	antreaclientset, err := bootstrap.CreateAntreaClientset(config)
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
