package main

import (
	"headnode/pkg/authorization"
	"headnode/pkg/controller/nodelabeler"
)

func main() {
	// Set kubeconfig to be used to create clientsets
	authorization.SetKubeConfig()
	// Start the controller to watch nodes and attach the labels to them
	nodelabeler.Start()
}
