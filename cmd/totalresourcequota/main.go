package main

import (
	"edgenet/pkg/authorization"
	"edgenet/pkg/controller/v1alpha/totalresourcequota"
)

func main() {
	// Set kubeconfig to be used to create clientsets
	authorization.SetKubeConfig()
	// Start the controller to provide the functionalities of total resource quota resource
	totalresourcequota.Start()
}
