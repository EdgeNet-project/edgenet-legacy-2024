package main

import (
	"headnode/pkg/authorization"
	"headnode/pkg/controller/v1alpha/authorityrequest"
)

func main() {
	// Set kubeconfig to be used to create clientsets
	authorization.SetKubeConfig()
	// Start the controller to provide the functionalities of authorityrequest resource
	authorityrequest.Start()
}
