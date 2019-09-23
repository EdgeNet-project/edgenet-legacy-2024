package main

import (
	"headnode/pkg/authorization"
	"headnode/pkg/controller/selectivedeployment"
)

func main() {
	// Set kubeconfig to be used to create clientsets
	authorization.SetKubeConfig()
	// Start the controller to provide the functionalities of selectivedeployment resource
	selectivedeployment.Start()
}
