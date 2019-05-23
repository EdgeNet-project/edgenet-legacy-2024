package namespace

import (
	"fmt"

	"headnode/pkg/authorization"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

func getNamespaceByName(kubeconfig *string, name string) (string, error) {
	clientset, err := authorization.CreateClientSet(kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// Examples for error handling:
	// - Use helper functions like e.g. errors.IsNotFound()
	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	_, err = clientset.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Println("Namespace %s not found\n", name)
		return "Namespace not found", err
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Println("Error getting namespace %s: %v\n", name, statusError.ErrStatus)
		return "Error getting namespace", err
	} else if err != nil {
		panic(err.Error())
	} else {
		return "", nil
	}
}
