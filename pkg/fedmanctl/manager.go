package fedmanctl

import (
	"context"
	"fmt"

	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func FederateByWorkerToken(kubeclientset *kubernetes.Clientset, edgenetclientset *versioned.Clientset, tokenString string) error {
	kubeNamespace, err := kubeclientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", v1.GetOptions{})

	if err != nil {
		return err
	}

	token, err := Detokenize(tokenString)

	if err != nil {
		return err
	}

	fmt.Printf("Federated cluster-%v by cluster-%v\n", token.UID, kubeNamespace.UID)
	return nil
}
