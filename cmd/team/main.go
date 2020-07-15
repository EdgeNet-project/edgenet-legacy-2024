package main

import (
	"edgenet/pkg/bootstrap"
	"edgenet/pkg/controller/v1alpha/team"
)

func main() {
	// Set kubeconfig to be used to create clientsets
	bootstrap.SetKubeConfig()
	// Start the controller to provide the functionalities of team resource
	team.Start()
}
