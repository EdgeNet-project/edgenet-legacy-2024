package main

import (
	"edgenet/pkg/client/clientset/versioned"
	"edgenet/pkg/controller/v1alpha/nodecontribution"

	"k8s.io/client-go/kubernetes"
)

func main(kubernetes kubernetes.Interface, edgenet versioned.Interface) {
	// Set kubeconfig to be used to create clientsets
	//authorization.SetKubeConfig()
	clientset := kubernetes
	edgenetClientset := edgenet
	// Start the controller to provide the functionalities of nodecontribution resource
	nodecontribution.Start(clientset, edgenetClientset)
}
