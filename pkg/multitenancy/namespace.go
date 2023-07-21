/*
Copyright 2021 Contributors to the EdgeNet project.

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

package multitenancy

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

// MakeOwnerReferenceForNamespace creates an owner reference for the given namespace.
func MakeOwnerReferenceForNamespace(namespace *corev1.Namespace) metav1.OwnerReference {
	// The section below makes namespace the owner
	newNamespaceRef := *metav1.NewControllerRef(namespace, corev1.SchemeGroupVersion.WithKind("Namespace"))
	takeControl := true
	newNamespaceRef.Controller = &takeControl
	return newNamespaceRef
}

// EligibilityCheck checks whether namespace, in which object exists, is local to the cluster or is propagated along with a federated deployment.
// If another cluster propagates the namespace, we skip checking the owner tenant's status as the Selective Deployment entity manages this life-cycle.
func (m *Manager) EligibilityCheck(objNamespace string) (bool, *corev1.Namespace, map[string]string) {
	systemNamespace, err := m.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err != nil {
		klog.Infoln(err)
		return false, nil, nil
	}
	namespace, err := m.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), objNamespace, metav1.GetOptions{})
	if err != nil {
		klog.Infoln(err)
		return false, nil, nil
	}
	namespaceLabels := namespace.GetLabels()
	if namespaceLabels["edge-net.io/cluster-uid"] != "" {
		if systemNamespace.GetUID() == types.UID(namespaceLabels["edge-net.io/cluster-uid"]) {
			tenant, err := m.edgenetclientset.CoreV1alpha1().Tenants().Get(context.TODO(), strings.ToLower(namespaceLabels["edge-net.io/tenant"]), metav1.GetOptions{})
			if err != nil {
				klog.Infoln(err)
				return false, nil, nil
			}
			if tenant.GetUID() != types.UID(namespaceLabels["edge-net.io/tenant-uid"]) || !tenant.Spec.Enabled {
				return false, nil, nil
			}
		} else {
			return false, nil, nil
		}
	} else {
		namespaceLabels["edge-net.io/cluster-uid"] = string(systemNamespace.GetUID())
	}
	return true, namespace, namespaceLabels
}
