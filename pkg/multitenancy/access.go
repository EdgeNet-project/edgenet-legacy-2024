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
	"fmt"
	"log"

	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"

	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

var labels = map[string]string{"edge-net.io/generated": "true"}

// GrantObjectOwnership configures permission for the object owner
func (m *Manager) GrantObjectOwnership(apiGroup, resource, resourceName, subject string, ownerReferences []metav1.OwnerReference) error {
	clusterRole, err := m.createObjectSpecificClusterRole(apiGroup, resource, resourceName, "owner", []string{"get", "update", "patch", "delete"}, ownerReferences)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		klog.Infof("Couldn't create owner cluster role %s: %s", subject, err)
		return err
	}
	if err := m.createObjectSpecificClusterRoleBinding(clusterRole, subject, ownerReferences); err != nil && !k8serrors.IsAlreadyExists(err) {
		klog.Infof("Couldn't create cluster role binding %s: %s", subject, err)
		return err
	}
	return nil
}

// CreateClusterRoles generate a cluster role for tenant owners, admins, and collaborators
func (m *Manager) CreateClusterRoles() error {
	policyRule := []rbacv1.PolicyRule{{APIGroups: []string{"core.edgenet.io"}, Resources: []string{"subnamespaces"}, Verbs: []string{"*"}},
		{APIGroups: []string{"core.edgenet.io"}, Resources: []string{"subnamespaces/status"}, Verbs: []string{"get", "list", "watch"}},
		{APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"selectivedeployments"}, Verbs: []string{"*"}},
		{APIGroups: []string{"rbac.authorization.k8s.io"}, Resources: []string{"roles", "rolebindings"}, Verbs: []string{"*"}},
		{APIGroups: []string{""}, Resources: []string{"configmaps", "endpoints", "persistentvolumeclaims", "pods", "pods/exec", "pods/log", "pods/attach", "pods/portforward", "replicationcontrollers", "services", "secrets", "serviceaccounts"}, Verbs: []string{"*"}},
		{APIGroups: []string{"apps"}, Resources: []string{"daemonsets", "deployments", "replicasets", "statefulsets"}, Verbs: []string{"*"}},
		{APIGroups: []string{"autoscaling"}, Resources: []string{"horizontalpodautoscalers"}, Verbs: []string{"*"}},
		{APIGroups: []string{"batch"}, Resources: []string{"cronjobs", "jobs"}, Verbs: []string{"*"}},
		{APIGroups: []string{"extensions"}, Resources: []string{"daemonsets", "deployments", "ingresses", "networkpolicies", "replicasets", "replicationcontrollers"}, Verbs: []string{"*"}},
		{APIGroups: []string{"networking.k8s.io"}, Resources: []string{"ingresses", "networkpolicies"}, Verbs: []string{"*"}},
		{APIGroups: []string{""}, Resources: []string{"events", "controllerrevisions"}, Verbs: []string{"get", "list", "watch"}}}
	ownerRole := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: corev1alpha1.TenantOwnerClusterRoleName}, Rules: policyRule}
	ownerRole.SetLabels(labels)
	_, err := m.kubeclientset.RbacV1().ClusterRoles().Create(context.TODO(), ownerRole, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Couldn't create tenant owner cluster role: %s", err)
		if k8serrors.IsAlreadyExists(err) {
			currentClusterRole, err := m.kubeclientset.RbacV1().ClusterRoles().Get(context.TODO(), ownerRole.GetName(), metav1.GetOptions{})
			if err == nil {
				currentClusterRole.Rules = policyRule
				_, err = m.kubeclientset.RbacV1().ClusterRoles().Update(context.TODO(), currentClusterRole, metav1.UpdateOptions{})
				if err == nil {
					log.Println("Tenant owner cluster role updated")
				} else {
					return err
				}
			}
		}
	}
	adminRole := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: corev1alpha1.TenantAdminClusterRoleName}, Rules: policyRule}
	adminRole.SetLabels(labels)
	_, err = m.kubeclientset.RbacV1().ClusterRoles().Create(context.TODO(), adminRole, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Couldn't create tenant admin cluster role: %s", err)
		if k8serrors.IsAlreadyExists(err) {
			currentClusterRole, err := m.kubeclientset.RbacV1().ClusterRoles().Get(context.TODO(), adminRole.GetName(), metav1.GetOptions{})
			if err == nil {
				currentClusterRole.Rules = policyRule
				_, err = m.kubeclientset.RbacV1().ClusterRoles().Update(context.TODO(), currentClusterRole, metav1.UpdateOptions{})
				if err == nil {
					log.Println("Tenant admin cluster role updated")
				} else {
					return err
				}
			}
		}
	}

	policyRule = []rbacv1.PolicyRule{{APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"selectivedeployments"}, Verbs: []string{"*"}},
		{APIGroups: []string{""}, Resources: []string{"configmaps", "endpoints", "persistentvolumeclaims", "pods", "pods/exec", "pods/log", "pods/attach", "pods/portforward", "replicationcontrollers", "services", "secrets", "serviceaccounts"}, Verbs: []string{"*"}},
		{APIGroups: []string{"apps"}, Resources: []string{"daemonsets", "deployments", "replicasets", "statefulsets"}, Verbs: []string{"*"}},
		{APIGroups: []string{"autoscaling"}, Resources: []string{"horizontalpodautoscalers"}, Verbs: []string{"*"}},
		{APIGroups: []string{"batch"}, Resources: []string{"cronjobs", "jobs"}, Verbs: []string{"*"}},
		{APIGroups: []string{"extensions"}, Resources: []string{"daemonsets", "deployments", "replicasets", "replicationcontrollers"}, Verbs: []string{"*"}},
		{APIGroups: []string{""}, Resources: []string{"events", "controllerrevisions"}, Verbs: []string{"get", "list", "watch"}}}
	collaboratorRole := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: corev1alpha1.TenantCollaboratorClusterRoleName}, Rules: policyRule}
	collaboratorRole.SetLabels(labels)
	_, err = m.kubeclientset.RbacV1().ClusterRoles().Create(context.TODO(), collaboratorRole, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Couldn't create tenant collaborator cluster role: %s", err)
		if k8serrors.IsAlreadyExists(err) {
			currentClusterRole, err := m.kubeclientset.RbacV1().ClusterRoles().Get(context.TODO(), collaboratorRole.GetName(), metav1.GetOptions{})
			if err == nil {
				currentClusterRole.Rules = policyRule
				_, err = m.kubeclientset.RbacV1().ClusterRoles().Update(context.TODO(), currentClusterRole, metav1.UpdateOptions{})
				if err == nil {
					log.Println("Tenant collaborator cluster role updated")
					return err
				}
			}
		}
	}

	return err
}

// CreateObjectSpecificClusterRole generates a object specific cluster role to allow the user access
func (m *Manager) createObjectSpecificClusterRole(apiGroup, resource, resourceName, name string, verbs []string, ownerReferences []metav1.OwnerReference) (string, error) {
	objectName := fmt.Sprintf("edgenet:%s:%s-%s", resource, resourceName, name)
	policyRule := []rbacv1.PolicyRule{{APIGroups: []string{apiGroup}, Resources: []string{resource}, ResourceNames: []string{resourceName}, Verbs: verbs},
		{APIGroups: []string{apiGroup}, Resources: []string{fmt.Sprintf("%s/status", resource)}, ResourceNames: []string{resourceName}, Verbs: []string{"get", "list", "watch"}},
	}
	role := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: objectName, OwnerReferences: ownerReferences},
		Rules: policyRule}

	_, err := m.kubeclientset.RbacV1().ClusterRoles().Create(context.TODO(), role, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Couldn't create %s cluster role: %s", objectName, err)
		if k8serrors.IsAlreadyExists(err) {
			currentRole, err := m.kubeclientset.RbacV1().ClusterRoles().Get(context.TODO(), role.GetName(), metav1.GetOptions{})
			if err == nil {
				currentRole.Rules = policyRule
				_, err = m.kubeclientset.RbacV1().ClusterRoles().Update(context.TODO(), currentRole, metav1.UpdateOptions{})
				if err == nil {
					log.Printf("Updated: %s cluster role updated", objectName)
					return objectName, err
				}
			}
		}
	}
	return objectName, err
}

// CreateObjectSpecificClusterRoleBinding links the cluster role up with the user
func (m *Manager) createObjectSpecificClusterRoleBinding(roleName, email string, ownerReferences []metav1.OwnerReference) error {
	roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: roleName}
	rbSubjects := []rbacv1.Subject{{Kind: "User", Name: email, APIGroup: "rbac.authorization.k8s.io"}}
	roleBind := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: roleName},
		Subjects: rbSubjects, RoleRef: roleRef}
	roleBind.ObjectMeta.OwnerReferences = ownerReferences
	roleBind.SetLabels(labels)
	_, err := m.kubeclientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), roleBind, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Couldn't create %s cluster role binding: %s", roleName, err)
		if k8serrors.IsAlreadyExists(err) {
			currentRoleBind, err := m.kubeclientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), roleName, metav1.GetOptions{})
			if err == nil {
				currentRoleBind.Subjects = []rbacv1.Subject{{Kind: "User", Name: email, APIGroup: "rbac.authorization.k8s.io"}}
				currentRoleBind.SetLabels(labels)
				if _, err = m.kubeclientset.RbacV1().ClusterRoleBindings().Update(context.TODO(), currentRoleBind, metav1.UpdateOptions{}); err == nil {
					log.Printf("Updated: %s cluster role binding updated", roleName)
					return err
				}
			}
		}
	}
	return err
}
