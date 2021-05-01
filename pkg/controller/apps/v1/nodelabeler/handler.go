package nodelabeler

import (
	"github.com/EdgeNet-project/edgenet/pkg/node"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init(kubernetes kubernetes.Interface)
	SetNodeGeolocation(obj interface{})
}

// Handler is a sample implementation of Handler
type Handler struct {
	clientset kubernetes.Interface
}

// Init handles any handler initialization
func (t *Handler) Init(kubernetes kubernetes.Interface) {
	log.Info("Handler.Init")
	t.clientset = kubernetes
	node.Clientset = t.clientset
}

// SetNodeGeolocation is called when an object is created or updated
func (t *Handler) SetNodeGeolocation(obj interface{}) {
	log.Info("Handler.ObjectCreated")
	// Get internal and external IP addresses of the node
	internalIP, externalIP := node.GetNodeIPAddresses(obj.(*corev1.Node))
	result := false
	// Check if the external IP exists to use it in the first place
	if externalIP != "" {
		log.Infof("External IP: %s", externalIP)
		result = node.GetGeolocationByIP(obj.(*corev1.Node).Name, externalIP)
	}
	// Check if the internal IP exists and
	// the result of detecting geolocation by external IP is false
	if internalIP != "" && result == false {
		log.Infof("Internal IP: %s", internalIP)
		node.GetGeolocationByIP(obj.(*corev1.Node).Name, internalIP)
	}
}
