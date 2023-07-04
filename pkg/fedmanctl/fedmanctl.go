package fedmanctl

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/EdgeNet-project/edgenet/pkg/apis/federation/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	corev1 "k8s.io/api/core/v1"
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
	GenerateWorkerClusterToken(clusterIP, clusterPort, visibility string, labels map[string]string) (string, error)

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
	clusterIP        string
	clusterPort      string
}

// Create a new interface for fedmanctl
func NewFedmanctl(kubeconfig, context string, silent bool) (Fedmanctl, error) {
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

	hostname := config.Host

	if strings.Contains(config.Host, "//") {
		if !silent {
			fmt.Println("Warning: The token generated contains a url instead of an ip:port. This might mean there is a proxy and federation might not work, override the ip and port using --ip and --port options.")
			fmt.Println("")
		}
		hostname = strings.Split(hostname, "//")[1]
	}

	hostnames := strings.Split(hostname, ":")

	return fedmanctl{
		edgenetClientset: edgenetclientset,
		kubeClientset:    kubeclientset,
		clusterIP:        hostnames[0],
		clusterPort:      hostnames[1],
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
	sa := &corev1.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name: "fedmanager",
		},
	}
	_, err := f.GetKubeClientset().CoreV1().ServiceAccounts("edgenet").Create(context.TODO(), sa, v1.CreateOptions{})

	if err != nil {
		return err
	}

	s := &corev1.Secret{
		Type: "kubernetes.io/service-account-token",
		ObjectMeta: v1.ObjectMeta{
			Name:      "fedmanager",
			Namespace: "edgenet",
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": "fedmanager",
			},
		},
	}

	_, err = f.GetKubeClientset().CoreV1().Secrets("edgenet").Create(context.TODO(), s, v1.CreateOptions{})

	if err != nil {
		return err
	}

	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name: "edgenet:fedmanager",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: "edgenet",
				Name:      "fedmanager",
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "edgenet:federation:remotecluster",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	_, err = f.GetKubeClientset().RbacV1().ClusterRoleBindings().Create(context.TODO(), crb, v1.CreateOptions{})

	if err != nil {
		return err
	}

	return nil
}

// Delete and remove all the configuration of the worker cluster.
func (f fedmanctl) ResetWorkerCluster() error {
	f.GetKubeClientset().RbacV1().ClusterRoleBindings().Delete(context.TODO(), "edgenet:fedmanager", v1.DeleteOptions{})
	f.GetKubeClientset().CoreV1().ServiceAccounts("edgenet").Delete(context.TODO(), "fedmanager", v1.DeleteOptions{})
	f.GetKubeClientset().CoreV1().Secrets("edgenet").Delete(context.TODO(), "fedmanager", v1.DeleteOptions{})
	return nil
}

// Generate the token of the worker cluster with the given labels
func (f fedmanctl) GenerateWorkerClusterToken(clusterIP, clusterPort, visibility string, labels map[string]string) (string, error) {
	clusterUID, err := f.getClusterUID()

	if err != nil {
		return "", err
	}

	secret, err := f.getSecret()

	if err != nil {
		return "", err
	}

	requiredDataFieldList := []string{"ca.crt", "token", "namespace"}
	// Check for specific fields that we require from the secret
	for _, field := range requiredDataFieldList {
		if _, ok := secret.Data[field]; !ok {
			return "", errors.New("worker cluster secret does not have required data, reset the worker or wait until controllers create secret certificate")
		}
	}

	nClusterIP := f.clusterIP
	nClusterPort := f.clusterPort

	if clusterIP != "" {
		nClusterIP = clusterIP
	}

	if clusterPort != "" {
		nClusterPort = clusterPort
	}

	// base64 encoded secrets except the cluster uid
	token := &WorkerClusterInfo{
		CACertificate: string(secret.Data["ca.crt"]),
		Namespace:     string(secret.Data["namespace"]),
		Token:         string(secret.Data["token"]),
		UID:           clusterUID,
		ClusterIP:     nClusterIP,
		ClusterPort:   nClusterPort,
		Visibility:    visibility,
		Labels:        labels,
	}
	return TokenizeWorkerClusterInfo(token)
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
	clusters, err := f.GetEdgeNetClientset().FederationV1alpha1().Clusters("edgenet-federation").List(context.TODO(), v1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return clusters.Items, nil
}

func (f fedmanctl) Version() string {
	return "v1.0.0"
}

// Get the current cluster's UID (kube-system uid)
func (f fedmanctl) getClusterUID() (string, error) {
	kubeNamespace, err := f.GetKubeClientset().CoreV1().Namespaces().Get(context.TODO(), "kube-system", v1.GetOptions{})

	if err != nil {
		return "", err
	}

	return string(kubeNamespace.UID), nil
}

// Get the secret of the fedmanager
func (f fedmanctl) getSecret() (*corev1.Secret, error) {
	secret, err := f.GetKubeClientset().CoreV1().Secrets("edgenet").Get(context.TODO(), "fedmanager", v1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return secret, nil
}
