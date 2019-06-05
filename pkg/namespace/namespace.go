package namespace

import (
	"fmt"

	"headnode/pkg/authorization"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create function checks namespace occupied or not and uses clientset to create a namespace
func Create(name string) (string, error) {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		panic(err.Error())
	}
  // Check namespace occupied or not 
	exist, err := GetNamespaceByName(name)
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

// GetList uses clientset, this function gets list of namespaces by eliminating "default", "kube-system", and "kube-public"
func GetList() []string {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		panic(err.Error())
	}
	// FieldSelector allows getting filtered results
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

// GetNamespaceByName uses clientset to get namespace requested
func GetNamespaceByName(name string) (string, error) {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		panic(err.Error())
	}

	// Examples for error handling:
	// - Use helper functions like e.g. errors.IsNotFound()
	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	_, err = clientset.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		//fmt.Printf("Namespace %s not found\n", name)
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
