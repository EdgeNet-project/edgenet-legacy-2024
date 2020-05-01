package main

import (
	"edgenet/pkg/authorization"
	"edgenet/pkg/controller/v1alpha/authority"
)

func main() {
	// Set kubeconfig to be used to create clientsets
	authorization.SetKubeConfig()
	// Start the controller to provide the functionalities of authority resource
	authority.Start()
}
