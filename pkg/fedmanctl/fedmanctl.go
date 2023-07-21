package fedmanctl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	federationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/federation/v1alpha1"
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
	InitWorkloadCluster() error

	// Delete and remove all the configuration of the worker cluster.
	ResetWorkloadCluster() error

	// Generate the token of the worker cluster with the given labels
	GenerateWorkloadClusterToken(clusterIP, clusterPort, visibility string, debug bool, labels map[string]string) (string, error)

	// Link the worker cluster to the manager cluster. Configures the manager cluster by the token. Called by the manager context.
	FederateWorkloadCluster(token, namespace string) error

	// Unlinks the worker cluster from manager cluster. Called by the manager context.
	SeparateWorkloadCluster(uid, namespace string) error

	// List the worker cluster objects
	ListWorkloadClusters() ([]federationv1alpha1.Cluster, error)

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

	hostport := config.Host

	if strings.Contains(config.Host, "//") {
		if !silent {
			fmt.Println("Warning: The token generated contains a url instead of an ip:port. This might mean there is a proxy and federation might not work, override the ip and port using --ip and --port options.")
			fmt.Println("")
		}
		hostport = strings.Split(hostport, "//")[1]
	}

	// Split it like that so it also supports ipv6 addresses
	host, port, err := net.SplitHostPort(hostport)

	if err != nil {
		return nil, err
	}

	return fedmanctl{
		edgenetClientset: edgenetclientset,
		kubeClientset:    kubeclientset,
		clusterIP:        host,
		clusterPort:      port,
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
func (f fedmanctl) InitWorkloadCluster() error {
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
func (f fedmanctl) ResetWorkloadCluster() error {
	f.GetKubeClientset().RbacV1().ClusterRoleBindings().Delete(context.TODO(), "edgenet:fedmanager", v1.DeleteOptions{})
	f.GetKubeClientset().CoreV1().ServiceAccounts("edgenet").Delete(context.TODO(), "fedmanager", v1.DeleteOptions{})
	f.GetKubeClientset().CoreV1().Secrets("edgenet").Delete(context.TODO(), "fedmanager", v1.DeleteOptions{})
	return nil
}

// Generate the token of the worker cluster with the given labels
func (f fedmanctl) GenerateWorkloadClusterToken(clusterIP, clusterPort, visibility string, debug bool, labels map[string]string) (string, error) {
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
	token := &WorkloadClusterInfo{
		CACertificate: string(secret.Data["ca.crt"]),
		Namespace:     string(secret.Data["namespace"]),
		Token:         string(secret.Data["token"]),
		UID:           clusterUID,
		ClusterIP:     nClusterIP,
		ClusterPort:   nClusterPort,
		Visibility:    visibility,
		Labels:        labels,
	}

	if debug {
		b, err := json.Marshal(token)

		if err != nil {
			return "", err
		}

		return string(b), nil
	}

	return TokenizeWorkloadClusterInfo(token)
}

// Link the worker cluster to the manager cluster. Configures the manager cluster by the token. Called by the manager context.
func (f fedmanctl) FederateWorkloadCluster(token, namespace string) error {
	workerClusterInfo, err := DetokenizeWorkloadClusterInfo(token)

	if err != nil {
		return err
	}

	// Create the federated namespace
	federationUID, err := f.getClusterUID()

	if err != nil {
		return err
	}

	// Check if federation namespace "federation-XXX-XXX-XX-XXXX" exists, if not create.
	federationNamespaceName := fmt.Sprintf(federationv1alpha1.FederationManagerNamespace, federationUID)

	_, err = f.GetKubeClientset().CoreV1().Namespaces().Get(context.TODO(), federationNamespaceName, v1.GetOptions{})
	if err != nil {
		ns := &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: federationNamespaceName,
			},
		}
		_, err = f.GetKubeClientset().CoreV1().Namespaces().Create(context.TODO(), ns, v1.CreateOptions{})

		if err != nil {
			return err
		}
	}

	// Check if the secret storage namespace exsits. If not create? No it is up to the user to create and give permissions.
	_, err = f.GetKubeClientset().CoreV1().Namespaces().Get(context.TODO(), namespace, v1.GetOptions{})
	if err != nil {
		if err != nil {
			return fmt.Errorf("cannot find given namespace, please ensure the namespace %q exist", namespace)
		}
	}

	// secretName has the convention "secret-XXX" where XXX is the UID of the cluster
	secretName := strings.Join([]string{"secret", workerClusterInfo.UID}, "-")

	// clusterName has the convention "cluster-XXX" where XXX is the UID of the cluster
	clusterName := strings.Join([]string{"cluster", workerClusterInfo.UID}, "-")

	// Create the secret in the given namespace
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"ca.crt":    []byte(workerClusterInfo.CACertificate),
			"token":     []byte(workerClusterInfo.Token),
			"namespace": []byte(workerClusterInfo.Namespace),
		},
	}

	_, err = f.GetKubeClientset().CoreV1().Secrets(namespace).Create(context.TODO(), secret, v1.CreateOptions{})

	if err != nil {
		return err
	}

	// create the cluster in the given namespace
	cluster := &federationv1alpha1.Cluster{
		ObjectMeta: v1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
			Labels:    workerClusterInfo.Labels,
		},
		Spec: federationv1alpha1.ClusterSpec{
			UID:        workerClusterInfo.UID,
			Role:       "Workload",
			Server:     strings.Join([]string{workerClusterInfo.ClusterIP, workerClusterInfo.ClusterPort}, ":"),
			Visibility: workerClusterInfo.Visibility,
			Enabled:    true,
			SecretName: secretName,
			// For now we do not employ any referance mechanism
		},
	}

	_, err = f.GetEdgeNetClientset().FederationV1alpha1().Clusters(namespace).Create(context.TODO(), cluster, v1.CreateOptions{})

	if err != nil {
		return err
	}

	return nil
}

// Unlinks the worker cluster from manager cluster. Called by the manager context.
func (f fedmanctl) SeparateWorkloadCluster(uid, namespace string) error {
	secretName := strings.Join([]string{"secret", uid}, "-")
	clusterName := strings.Join([]string{"cluster", uid}, "-")

	f.GetKubeClientset().CoreV1().Secrets(namespace).Delete(context.TODO(), secretName, v1.DeleteOptions{})
	f.GetEdgeNetClientset().FederationV1alpha1().Clusters(namespace).Delete(context.TODO(), clusterName, v1.DeleteOptions{})

	return nil
}

// List the worker cluster objects
func (f fedmanctl) ListWorkloadClusters() ([]federationv1alpha1.Cluster, error) {
	// Get all the cluster objects from all of the namespaces
	clusters, err := f.GetEdgeNetClientset().FederationV1alpha1().Clusters("").List(context.TODO(), v1.ListOptions{})

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
