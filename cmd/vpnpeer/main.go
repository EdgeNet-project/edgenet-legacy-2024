package main

import (
	"github.com/EdgeNet-project/edgenet/pkg/controller/networking/v1alpha/vpnpeer"
	"log"
	"os"
	"strings"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/signals"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/klog"
)

func main() {
	stopCh := signals.SetupSignalHandler()
	bootstrap.SetKubeConfig()
	kubeclientset, err := bootstrap.CreateClientset("kubeconfig")
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	edgenetclientset, err := bootstrap.CreateEdgeNetClientset("kubeconfig")
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclientset, time.Second*30)
	edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, 0)

	linkName := strings.TrimSpace(os.Getenv("LINKNAME"))
	if linkName == "" {
		linkName = "edgenetmesh0"
	}

	controller := vpnpeer.NewController(
		kubeclientset,
		edgenetclientset,
		edgenetInformerFactory.Networking().V1alpha().VPNPeers(),
		linkName,
	)

	kubeInformerFactory.Start(stopCh)
	edgenetInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}
