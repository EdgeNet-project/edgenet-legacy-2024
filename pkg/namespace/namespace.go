/*
Copyright 2019 Sorbonne Université

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package namespace

import (
	"log"

	"k8s.io/client-go/kubernetes"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create function checks namespace occupied or not and uses clientset to create a namespace
func Create(clientset kubernetes.Interface, name string) (string, error) {
	exist, err := GetNamespaceByName(clientset, name)
	if (err == nil && exist == "true") || (err != nil && exist == "error") {
		if err == nil {
			err = errors.NewGone(exist)
			log.Println(err)
		}
		return "", err
	}
	userNamespace := &apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	result, err := clientset.CoreV1().Namespaces().Create(userNamespace)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return result.GetObjectMeta().GetName(), nil
}

// Delete function checks whether namespace exists, and uses clientset to delete the namespace
func Delete(clientset kubernetes.Interface, namespace string) (string, error) {

	// Check namespace exists or not
	exist, err := GetNamespaceByName(clientset, namespace)
	if err == nil && exist == "true" {
		err := clientset.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{})
		if err != nil {
			log.Println(err)
			return "", err
		}
		return "deleted", nil
	}
	if err == nil {
		err = errors.NewGone(exist)
		log.Println(err)
	}
	return "", err
}

// GetList uses clientset, this function gets list of namespaces by eliminating "default", "kube-system", and "kube-public"
func GetList(clientset kubernetes.Interface) []string {

	// FieldSelector allows getting filtered results
	namespaceRaw, err := clientset.CoreV1().Namespaces().List(
		metav1.ListOptions{FieldSelector: "metadata.name!=default,metadata.name!=kube-system,metadata.name!=kube-public"})
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	namespaces := make([]string, len(namespaceRaw.Items))
	for i, namespaceRow := range namespaceRaw.Items {
		namespaces[i] = namespaceRow.Name
	}
	return namespaces
}

// GetNamespaceByName uses clientset to get namespace requested
func GetNamespaceByName(clientset kubernetes.Interface, name string) (string, error) {
	// Examples for error handling:
	// - Use helper functions like e.g. errors.IsNotFound()
	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	_, err := clientset.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Printf("Namespace %s not found", name)
		return "false", err
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Printf("Error getting namespace %s: %v", name, statusError.ErrStatus)
		return "error", err
	} else if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	} else {
		return "true", nil
	}
}
