package namespace

import (
	"fmt"

	"headnode/pkg/authorization"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Create(kubeconfig *string, name string) (string, error) {
	clientset, err := authorization.CreateClientSet(kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	exist, err := GetNamespaceByName(kubeconfig, name)
	if (err == nil && exist == "true") || (err != nil && exist == "error") {
		if err == nil {
			err = errors.NewGone(exist)
		}
		return "", err
	}

	userNamespace := &apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	result, err := clientset.CoreV1().Namespaces().Create(userNamespace)
	if err != nil {
		return "", err
	}

	return result.GetObjectMeta().GetName(), nil
}

func GetList(kubeconfig *string) []string {
	clientset, err := authorization.CreateClientSet(kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	namespacesRaw, err := clientset.CoreV1().Namespaces().List(
		metav1.ListOptions{FieldSelector: "metadata.name!=default,metadata.name!=kube-system,metadata.name!=kube-public"})
	if err != nil {
		panic(err.Error())
	}
	namespaces := make([]string, len(namespacesRaw.Items))
	for i, namespacesRaw := range namespacesRaw.Items {
		namespaces[i] = namespacesRaw.Name
	}

	return namespaces
}

func GetNamespaceByName(kubeconfig *string, name string) (string, error) {
	clientset, err := authorization.CreateClientSet(kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// Examples for error handling:
	// - Use helper functions like e.g. errors.IsNotFound()
	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	_, err = clientset.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("Namespace %s not found\n", name)
		return "false", err
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting namespace %s: %v\n", name, statusError.ErrStatus)
		return "error", err
	} else if err != nil {
		panic(err.Error())
	} else {
		return "true", nil
	}
}
