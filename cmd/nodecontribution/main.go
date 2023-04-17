package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha1/nodecontribution"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/signals"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/klog"
)

func main() {
	klog.InitFlags(nil)
	flag.String("kubeconfig-path", bootstrap.GetDefaultKubeconfigPath(), "Path to the kubeconfig file's directory")
	flag.String("ssh-path", "/edgenet/.ssh", "Path to the SSH keys")
	flag.String("configs-path", "/edgenet/configs", "Path to the config files")
	flag.String("ca-path", "/etc/kubernetes/pki/ca.crt", "Path to the CA")
	flag.String("aws-id-path", "/edgenet/aws/id", "Path to the AWS ID")
	flag.String("aws-secret-path", "/edgenet/aws/secret", "Path to the AWS key")
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

	// Start the controller to provide the functionalities of nodecontribution resource
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclientset, time.Hour*3)
	edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, 0)
	hostedZone := strings.TrimSpace(os.Getenv("ROUTE53_HOSTED_ZONE"))
	domain := strings.TrimSpace(os.Getenv("DOMAIN_NAME"))

	controller := nodecontribution.NewController(kubeclientset,
		edgenetclientset,
		kubeInformerFactory.Core().V1().Nodes(),
		edgenetInformerFactory.Core().V1alpha1().NodeContributions(),
		hostedZone,
		domain)

	kubeInformerFactory.Start(stopCh)
	edgenetInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}
