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
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Clientset to be synced by the custom resources
var Clientset kubernetes.Interface

// List uses clientset, this function provides the list of namespaces by eliminating "default", "kube-system", and "kube-public"
func List() []string {
	// FieldSelector allows getting filtered results
	namespaceRaw, err := Clientset.CoreV1().Namespaces().List(context.TODO(),
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

// GetNamespace uses clientset to get namespace requested
func GetNamespace(name string) (*corev1.Namespace, error) {
	// Examples for error handling:
	// - Use helper functions like e.g. errors.IsNotFound()
	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	namespace, err := Clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Printf("Namespace %s not found", name)
		return nil, err
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Printf("Error getting namespace %s: %v", name, statusError.ErrStatus)
		return nil, err
	} else if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	return namespace, nil
}

// SetAsOwnerReference returns the namespace as owner
func SetAsOwnerReference(namespace *corev1.Namespace) []metav1.OwnerReference {
	// The section below makes namespace the owner
	newNamespaceRef := *metav1.NewControllerRef(namespace, corev1.SchemeGroupVersion.WithKind("Namespace"))
	takeControl := false
	newNamespaceRef.Controller = &takeControl
	namespaceOwnerReferences := []metav1.OwnerReference{newNamespaceRef}
	return namespaceOwnerReferences
}
