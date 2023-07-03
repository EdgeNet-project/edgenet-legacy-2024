package fedmanctl

import (
	"context"
	"fmt"

	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
)

type WorkerFederationPerformer struct {
	Kubeclientset    *kubernetes.Clientset
	Edgenetclientset *versioned.Clientset
}

// Creates a service account, rolebinding, and the secret. Then encodes them as a token. This token
// can be deserialized by the federation init command.
func (w WorkerFederationPerformer) CreateWorkerToken() (string, error) {
	clusterUID, err := w.getClusterUID()

	if err != nil {
		return "", err
	}

	secret, err := w.configureFederationClusterAccess()

	if err != nil {
		return "", err
	}

	// base64 encoded secrets except the cluster uid
	token := &WorkerClusterToken{
		CACertificate: string(secret.Data["ca.crt"]),
		Namespace:     string(secret.Data["namespace"]),
		Token:         string(secret.Data["token"]),
		UID:           clusterUID,
	}

	return Tokenize(token)
}

// Receives the uid of the kube-system namespace. This is used as the cluster identifier by edgenet.
func (w WorkerFederationPerformer) getClusterUID() (string, error) {
	kubeNamespace, err := w.Kubeclientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", v1.GetOptions{})

	if err != nil {
		return "", err
	}

	return string(kubeNamespace.UID), nil
}

// This is run on the worker node. It will create a service account, a secret and the rolebinding to
// give outside permission to the manager cluster.
func (w WorkerFederationPerformer) configureFederationClusterAccess() (*corev1.Secret, error) {
	sa := &corev1.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name: "fedmanager",
		},
	}
	_, err := w.Kubeclientset.CoreV1().ServiceAccounts("edgenet").Create(context.TODO(), sa, v1.CreateOptions{})

	if err != nil {
		return nil, err
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

	_, err = w.Kubeclientset.CoreV1().Secrets("edgenet").Create(context.TODO(), s, v1.CreateOptions{})

	if err != nil {
		return nil, err
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

	_, err = w.Kubeclientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), crb, v1.CreateOptions{})

	if err != nil {
		return nil, err
	}

	var secret *corev1.Secret
	requiredDataFieldList := []string{"ca.crt", "token", "namespace"}

	// We have to wait until the secret certificates are generated. We check periodicaly
	// if the fields are added to the
	for {
		secret, err := w.Kubeclientset.CoreV1().Secrets("edgenet").Get(context.TODO(), "fedmanager", v1.GetOptions{})

		if err != nil {
			return nil, err
		}

		numFields := 0
		// Check for specific fields that we require from the secret
		for _, field := range requiredDataFieldList {
			if _, ok := secret.Data[field]; !ok {
				return nil, fmt.Errorf("created kubernetes secret does not contian the field '%v'", field)
			}
			numFields += 1
		}

		if numFields == len(requiredDataFieldList) {
			break
		}
	}

	return secret, nil
}

// Removes the secretes, rolebindings and service accounts configured for external federation cluster access.
func (w WorkerFederationPerformer) removeFederationClusterAccess() {
	w.Kubeclientset.RbacV1().ClusterRoleBindings().Delete(context.TODO(), "edgenet:fedmanager", v1.DeleteOptions{})
	w.Kubeclientset.CoreV1().ServiceAccounts("edgenet").Delete(context.TODO(), "fedmanager", v1.DeleteOptions{})
	w.Kubeclientset.CoreV1().Secrets("edgenet").Delete(context.TODO(), "fedmanager", v1.DeleteOptions{})
}

// Reset the objects in the class
func (w WorkerFederationPerformer) ResetWorkerClusterFederation() {
	w.removeFederationClusterAccess()
}
