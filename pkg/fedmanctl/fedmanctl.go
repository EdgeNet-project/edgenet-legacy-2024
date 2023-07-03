package fedmanctl

import (
	"os"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Fedmanctl interface {
	GetEdgeNetClientset() *versioned.Clientset
	GetKubClientset() *kubernetes.Clientset
	Version() string
}

type fedmanctl struct {
	Fedmanctl
	edgenetClientset *versioned.Clientset
	kubeClientset    *kubernetes.Clientset
}

// Create a new interface for fedmanctl
func NewFedmanctl(kubeconfig, context string) (Fedmanctl, error) {
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}

	// Get the specified context if context variable is a non-empty string
	if context != "" {
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
			&clientcmd.ConfigOverrides{
				CurrentContext: context,
			}).ClientConfig()

		if err != nil {
			return nil, err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)

		if err != nil {
			return nil, err
		}
	}

	edgenetclientset, err := bootstrap.CreateEdgeNetClientset(config)
	if err != nil {
		return nil, err
	}

	kubeclientset, err := bootstrap.CreateKubeClientset(config)
	if err != nil {
		return nil, err
	}

	return fedmanctl{
		edgenetClientset: edgenetclientset,
		kubeClientset:    kubeclientset,
	}, nil
}

func (f fedmanctl) Version() string {
	return "v1.0.0"
}
