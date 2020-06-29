package main

import (
	"edgenet/pkg/authorization"
	"edgenet/pkg/client/clientset/versioned"
	"edgenet/pkg/controller/v1alpha/user"

	"k8s.io/client-go/kubernetes"
)

func main(clientset kubernetes.Interface, edgenetClientset versioned.Interface) {
	// Set kubeconfig to be used to create clientsets
	authorization.SetKubeConfig()
	// Start the controller to provide the functionalities of user resource
	user.Start(clientset, edgenetClientset)
}
