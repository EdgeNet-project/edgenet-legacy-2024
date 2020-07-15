package main

import (
	"edgenet/pkg/bootstrap"
	"edgenet/pkg/controller/v1alpha/userregistrationrequest"
)

func main() {
	// Set kubeconfig to be used to create clientsets
	bootstrap.SetKubeConfig()
	// Start the controller to provide the functionalities of userregistrationrequest resource
	userregistrationrequest.Start()
}
