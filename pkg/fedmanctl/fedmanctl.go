package fedmanctl

import (
	"os"

	"github.com/EdgeNet-project/edgenet/pkg/apis/federation/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Fedmanctl interface {
	// Getter for EdgeNet clientset
	GetEdgeNetClientset() *versioned.Clientset

	// Getter for Kubernetes clientset
	GetKubeClientset() *kubernetes.Clientset

	// Initialize the given cluster as the worker cluster. Do not generate a token.
	InitWorkerCluster() error

	// Delete and remove all the configuration of the worker cluster.
	ResetWorkerCluster() error

	// Generate the token of the worker cluster with the given labels
	GenerateWorkerClusterToken(labels map[string]string) (string, error)

	// Link the worker cluster to the manager cluster. Configures the manager cluster by the token. Called by the manager context.
	LinkToManagerCluster(token string) error

	// Unlinks the worker cluster from manager cluster. Called by the manager context.
	UnlinkFromManagerCluster(uid string) error

	// List the worker cluster objects
	ListWorkerClusters() ([]v1alpha1.Cluster, error)

	Version() string
}

type fedmanctl struct {
	Fedmanctl
	edgenetClientset *versioned.Clientset
	kubeClientset    *kubernetes.Clientset
}

var _ Fedmanctl = (*fedmanctl)(nil)

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

// Getter for EdgeNet clientset
func (f fedmanctl) GetEdgeNetClientset() *versioned.Clientset {
	return f.edgenetClientset
}

// Getter for Kubernetes clientset
func (f fedmanctl) GetKubeClientset() *kubernetes.Clientset {
	return f.kubeClientset
}

// Initialize the given cluster as the worker cluster. Do not generate a token.
func (f fedmanctl) InitWorkerCluster() error {
	return nil
}

// Delete and remove all the configuration of the worker cluster.
func (f fedmanctl) ResetWorkerCluster() error {
	return nil
}

// Generate the token of the worker cluster with the given labels
func (f fedmanctl) GenerateWorkerClusterToken(labels map[string]string) (string, error) {
	return "", nil
}

// Link the worker cluster to the manager cluster. Configures the manager cluster by the token. Called by the manager context.
func (f fedmanctl) LinkToManagerCluster(token string) error {
	return nil
}

// Unlinks the worker cluster from manager cluster. Called by the manager context.
func (f fedmanctl) UnlinkFromManagerCluster(uid string) error {
	return nil
}

// List the worker cluster objects
func (f fedmanctl) ListWorkerClusters() ([]v1alpha1.Cluster, error) {
	return []v1alpha1.Cluster{}, nil
}

func (f fedmanctl) Version() string {
	return "v1.0.0"
}
