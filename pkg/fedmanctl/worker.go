package fedmanctl

import (
	"context"

	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CreateWorkerToken(kubeclientset *kubernetes.Clientset, edgenetclientset *versioned.Clientset) (string, error) {
	kubeNamespace, err := kubeclientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", v1.GetOptions{})

	if err != nil {
		return "", err
	}

	token := &WorkerClusterToken{
		CACertificate: "",
		Namespace:     "",
		Token:         "",
		UID:           string(kubeNamespace.UID),
	}

	return Tokenize(token)
}
