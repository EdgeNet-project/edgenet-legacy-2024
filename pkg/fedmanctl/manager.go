package fedmanctl

import (
	"context"
	"fmt"

	"github.com/EdgeNet-project/edgenet/pkg/apis/federation/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ManagerFederationPerformer struct {
	Kubeclientset    *kubernetes.Clientset
	Edgenetclientset *versioned.Clientset
}

func (m ManagerFederationPerformer) FederateByWorkerToken(tokenString string) error {
	kubeNamespace, err := m.Kubeclientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", v1.GetOptions{})

	if err != nil {
		return err
	}

	_, err = m.Kubeclientset.CoreV1().Namespaces().Get(context.TODO(), "edgenet-federation", v1.GetOptions{})

	// Create missing edgenet-federation namespace if it doesn't exist
	if err != nil {
		federationNamespace := &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: "edgenet-federation",
			},
		}
		_, err = m.Kubeclientset.CoreV1().Namespaces().Create(context.TODO(), federationNamespace, v1.CreateOptions{})

		if err != nil {
			return err
		}
	}

	token, err := Detokenize(tokenString)

	if err != nil {
		return err
	}

	fmt.Printf("Federated cluster-%v by cluster-%v\n", token.UID, kubeNamespace.UID)
	return nil
}

// Removes the secrets and the CRDs from the federation cluster.
func (m ManagerFederationPerformer) ResetWorkerClusterFederation(clusterUID string) error {
	return nil
}

// Gets a list of federation clusters
func (m ManagerFederationPerformer) ListWorkerClusters() ([]v1alpha1.Cluster, error) {
	clusters, err := m.Edgenetclientset.FederationV1alpha1().Clusters("edgenet-federation").List(context.TODO(), v1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return clusters.Items, nil
}
