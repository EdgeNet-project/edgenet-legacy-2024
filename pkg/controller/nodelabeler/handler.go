/*
Copyright 2019 Sorbonne Université

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nodelabeler

import (
	"headnode/pkg/node"

	log "github.com/Sirupsen/logrus"
	api_v1 "k8s.io/api/core/v1"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init() error
	SetNodeGeolocation(obj interface{})
}

// Handler is a sample implementation of Handler
type Handler struct{}

// Init handles any handler initialization
func (t *Handler) Init() error {
	log.Info("Handler.Init")
	return nil
}

// SetNodeGeolocation is called when an object is created or updated
func (t *Handler) SetNodeGeolocation(obj interface{}) {
	log.Info("Handler.ObjectCreated")
	// Get internal and external IP addresses of the node
	internalIP, externalIP := node.GetNodeIPAddresses(obj.(*api_v1.Node))
	result := false
	// Check if the external IP exists to use it in the first place
	if externalIP != "" {
		log.Infof("External IP: %s", externalIP)
		result = node.GetGeolocationByIP(obj.(*api_v1.Node).Name, externalIP)
	}
	// Check if the internal IP exists and
	// the result of detecting geolocation by external IP is false
	if internalIP != "" && result == false {
		log.Infof("Internal IP: %s", internalIP)
		node.GetGeolocationByIP(obj.(*api_v1.Node).Name, internalIP)
	}
}
