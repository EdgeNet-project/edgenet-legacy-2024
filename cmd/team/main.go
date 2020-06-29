package main

import (
	"edgenet/pkg/authorization"
	"edgenet/pkg/client/clientset/versioned"
	"edgenet/pkg/controller/v1alpha/team"

	"k8s.io/client-go/kubernetes"
)

func main(clientset kubernetes.Interface, edgenetClientset versioned.Interface) {
	// Set kubeconfig to be used to create clientsets
	authorization.SetKubeConfig()
	// Start the controller to provide the functionalities of team resource
	team.Start(clientset, edgenetClientset)
}
