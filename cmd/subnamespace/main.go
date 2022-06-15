package main

import (
	"flag"
	"log"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha1/subnamespace"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/signals"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/informers/internalinterfaces"
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
	// Start the controller to provide the functionalities of subnamespace resource
	var listOptionsFunc = func(labelSelector string) internalinterfaces.TweakListOptionsFunc {
		return func(listOptions *metav1.ListOptions) {
			listOptions.LabelSelector = labelSelector
		}
	}
	informerOption := kubeinformers.WithTweakListOptions(listOptionsFunc("edge-net.io/tenant"))
	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(kubeclientset, 0, informerOption)
	edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, 0)

	controller := subnamespace.NewController(kubeclientset,
		edgenetclientset,
		kubeInformerFactory.Rbac().V1().Roles(),
		kubeInformerFactory.Rbac().V1().RoleBindings(),
		kubeInformerFactory.Networking().V1().NetworkPolicies(),
		kubeInformerFactory.Core().V1().LimitRanges(),
		kubeInformerFactory.Core().V1().Secrets(),
		kubeInformerFactory.Core().V1().ConfigMaps(),
		kubeInformerFactory.Core().V1().ServiceAccounts(),
		edgenetInformerFactory.Core().V1alpha1().SubNamespaces())

	kubeInformerFactory.Start(stopCh)
	edgenetInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}
