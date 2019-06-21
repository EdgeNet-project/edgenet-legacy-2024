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
	internalIP, externalIP := node.GetNodeIPAddresses(obj.(*api_v1.Node))
	if internalIP != "" {
		log.Infof("Internal IP: %s", internalIP)
	}
	if externalIP != "" {
		log.Infof("External IP: %s", externalIP)
	}
}
