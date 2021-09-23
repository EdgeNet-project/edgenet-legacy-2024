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

package permission

import (
	"context"
	"fmt"
	"log"

	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Clientset to be synced by the custom resources
var Clientset kubernetes.Interface
var labels = map[string]string{"edge-net.io/generated": "true"}

// CheckAuthorization returns true if the user is holder of a role
func CheckAuthorization(namespace, email, resource, resourceName, scope string) bool {
	authorized := false

	checkRules := func(rule rbacv1.PolicyRule) {
		for _, ruleResource := range rule.Resources {
			if ruleResource == resource {
				for _, verb := range rule.Verbs {
					if verb == "create" || verb == "update" || verb == "patch" || verb == "delete" || verb == "*" {
						if len(rule.ResourceNames) > 0 {
							for _, ruleResourceName := range rule.ResourceNames {
								if ruleResourceName == resourceName {
									authorized = true
								}
							}
						} else {
							authorized = true
						}
					}
				}
			}
		}
	}
	if scope == "namespace" {
		roleBindingRaw, _ := Clientset.RbacV1().RoleBindings(namespace).List(context.TODO(), metav1.ListOptions{})
		for _, roleBindingRow := range roleBindingRaw.Items {
			for _, subject := range roleBindingRow.Subjects {
				if subject.Kind == "User" && subject.Name == email {
					if roleBindingRow.RoleRef.Kind == "Role" {
						role, err := Clientset.RbacV1().Roles(namespace).Get(context.TODO(), roleBindingRow.RoleRef.Name, metav1.GetOptions{})
						if err == nil {
							for _, rule := range role.Rules {
								checkRules(rule)
							}
						}
					} else if roleBindingRow.RoleRef.Kind == "ClusterRole" {
						role, err := Clientset.RbacV1().ClusterRoles().Get(context.TODO(), roleBindingRow.RoleRef.Name, metav1.GetOptions{})
						if err == nil {
							for _, rule := range role.Rules {
								checkRules(rule)
							}
						}
					}
				}
			}
		}
	} else {
		clusterRoleBindingRaw, _ := Clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})
		for _, clusterRoleBindingRow := range clusterRoleBindingRaw.Items {
			for _, subject := range clusterRoleBindingRow.Subjects {
				if subject.Kind == "User" && subject.Name == email {
					clusterRole, err := Clientset.RbacV1().ClusterRoles().Get(context.TODO(), clusterRoleBindingRow.RoleRef.Name, metav1.GetOptions{})
					if err == nil {
						for _, rule := range clusterRole.Rules {
							checkRules(rule)
						}
					}
				}
			}
		}

	}
	return authorized
}

// CreateClusterRoles generate a cluster role for tenant owners, admins, and collaborators
func CreateClusterRoles() error {
	policyRule := []rbacv1.PolicyRule{{APIGroups: []string{"core.edgenet.io"}, Resources: []string{"subnamespaces"}, Verbs: []string{"*"}},
		{APIGroups: []string{"core.edgenet.io"}, Resources: []string{"subnamespaces/status"}, Verbs: []string{"get", "list", "watch"}},
		{APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"selectivedeployments"}, Verbs: []string{"*"}},
		{APIGroups: []string{"rbac.authorization.k8s.io"}, Resources: []string{"roles", "rolebindings"}, Verbs: []string{"*"}},
		{APIGroups: []string{""}, Resources: []string{"configmaps", "endpoints", "persistentvolumeclaims", "pods", "pods/exec", "pods/log", "pods/attach", "replicationcontrollers", "services", "secrets", "serviceaccounts"}, Verbs: []string{"*"}},
		{APIGroups: []string{"apps"}, Resources: []string{"daemonsets", "deployments", "replicasets", "statefulsets"}, Verbs: []string{"*"}},
		{APIGroups: []string{"autoscaling"}, Resources: []string{"horizontalpodautoscalers"}, Verbs: []string{"*"}},
		{APIGroups: []string{"batch"}, Resources: []string{"cronjobs", "jobs"}, Verbs: []string{"*"}},
		{APIGroups: []string{"extensions"}, Resources: []string{"daemonsets", "deployments", "ingresses", "networkpolicies", "replicasets", "replicationcontrollers"}, Verbs: []string{"*"}},
		{APIGroups: []string{"networking.k8s.io"}, Resources: []string{"ingresses", "networkpolicies"}, Verbs: []string{"*"}},
		{APIGroups: []string{""}, Resources: []string{"events", "controllerrevisions"}, Verbs: []string{"get", "list", "watch"}}}
	ownerRole := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "edgenet:tenant-owner"}, Rules: policyRule}
	ownerRole.SetLabels(labels)
	_, err := Clientset.RbacV1().ClusterRoles().Create(context.TODO(), ownerRole, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Couldn't create tenant owner cluster role: %s", err)
		if errors.IsAlreadyExists(err) {
			currentClusterRole, err := Clientset.RbacV1().ClusterRoles().Get(context.TODO(), ownerRole.GetName(), metav1.GetOptions{})
			if err == nil {
				currentClusterRole.Rules = policyRule
				_, err = Clientset.RbacV1().ClusterRoles().Update(context.TODO(), currentClusterRole, metav1.UpdateOptions{})
				if err == nil {
					log.Println("Tenant owner cluster role updated")
				} else {
					return err
				}
			}
		}
	}
	adminRole := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "edgenet:tenant-admin"}, Rules: policyRule}
	adminRole.SetLabels(labels)
	_, err = Clientset.RbacV1().ClusterRoles().Create(context.TODO(), adminRole, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Couldn't create tenant admin cluster role: %s", err)
		if errors.IsAlreadyExists(err) {
			currentClusterRole, err := Clientset.RbacV1().ClusterRoles().Get(context.TODO(), adminRole.GetName(), metav1.GetOptions{})
			if err == nil {
				currentClusterRole.Rules = policyRule
				_, err = Clientset.RbacV1().ClusterRoles().Update(context.TODO(), currentClusterRole, metav1.UpdateOptions{})
				if err == nil {
					log.Println("Tenant admin cluster role updated")
				} else {
					return err
				}
			}
		}
	}

	policyRule = []rbacv1.PolicyRule{{APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"selectivedeployments"}, Verbs: []string{"*"}},
		{APIGroups: []string{""}, Resources: []string{"configmaps", "endpoints", "persistentvolumeclaims", "pods", "pods/exec", "pods/log", "pods/attach", "replicationcontrollers", "services", "secrets", "serviceaccounts"}, Verbs: []string{"*"}},
		{APIGroups: []string{"apps"}, Resources: []string{"daemonsets", "deployments", "replicasets", "statefulsets"}, Verbs: []string{"*"}},
		{APIGroups: []string{"autoscaling"}, Resources: []string{"horizontalpodautoscalers"}, Verbs: []string{"*"}},
		{APIGroups: []string{"batch"}, Resources: []string{"cronjobs", "jobs"}, Verbs: []string{"*"}},
		{APIGroups: []string{"extensions"}, Resources: []string{"daemonsets", "deployments", "replicasets", "replicationcontrollers"}, Verbs: []string{"*"}},
		{APIGroups: []string{""}, Resources: []string{"events", "controllerrevisions"}, Verbs: []string{"get", "list", "watch"}}}
	collaboratorRole := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "edgenet:tenant-collaborator"}, Rules: policyRule}
	collaboratorRole.SetLabels(labels)
	_, err = Clientset.RbacV1().ClusterRoles().Create(context.TODO(), collaboratorRole, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Couldn't create tenant collaborator cluster role: %s", err)
		if errors.IsAlreadyExists(err) {
			currentClusterRole, err := Clientset.RbacV1().ClusterRoles().Get(context.TODO(), collaboratorRole.GetName(), metav1.GetOptions{})
			if err == nil {
				currentClusterRole.Rules = policyRule
				_, err = Clientset.RbacV1().ClusterRoles().Update(context.TODO(), currentClusterRole, metav1.UpdateOptions{})
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
func CreateObjectSpecificClusterRole(tenant, apiGroup, resource, resourceName, name string, verbs []string, ownerReferences []metav1.OwnerReference) error {
	objectName := fmt.Sprintf("edgenet:%s:%s:%s-%s", tenant, resource, resourceName, name)
	policyRule := []rbacv1.PolicyRule{{APIGroups: []string{apiGroup}, Resources: []string{resource}, ResourceNames: []string{resourceName}, Verbs: verbs},
		{APIGroups: []string{apiGroup}, Resources: []string{fmt.Sprintf("%s/status", resource)}, ResourceNames: []string{resourceName}, Verbs: []string{"get", "list", "watch"}},
	}
	role := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: objectName, OwnerReferences: ownerReferences},
		Rules: policyRule}
	roleLabels := map[string]string{"edge-net.io/tenant": tenant}
	for key, value := range labels {
		roleLabels[key] = value
	}
	role.SetLabels(roleLabels)
	_, err := Clientset.RbacV1().ClusterRoles().Create(context.TODO(), role, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Couldn't create %s cluster role: %s", objectName, err)
		if errors.IsAlreadyExists(err) {
			currentRole, err := Clientset.RbacV1().ClusterRoles().Get(context.TODO(), role.GetName(), metav1.GetOptions{})
			if err == nil {
				currentRole.Rules = policyRule
				_, err = Clientset.RbacV1().ClusterRoles().Update(context.TODO(), currentRole, metav1.UpdateOptions{})
				if err == nil {
					log.Printf("Updated: %s cluster role updated", objectName)
					return err
				}
			}
		}
	}
	return err
}

// CreateObjectSpecificClusterRoleBinding links the cluster role up with the user
func CreateObjectSpecificClusterRoleBinding(tenant, roleName, username, email string, roleBindLabels map[string]string, ownerReferences []metav1.OwnerReference) error {
	objectName := fmt.Sprintf("%s-%s", roleName, username)
	roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: roleName}
	rbSubjects := []rbacv1.Subject{{Kind: "User", Name: email, APIGroup: "rbac.authorization.k8s.io"}}
	roleBind := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: objectName},
		Subjects: rbSubjects, RoleRef: roleRef}
	roleBind.ObjectMeta.OwnerReferences = ownerReferences
	for key, value := range labels {
		roleBindLabels[key] = value
	}
	roleBind.SetLabels(roleBindLabels)
	_, err := Clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), roleBind, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Couldn't create %s cluster role binding: %s", objectName, err)
		if errors.IsAlreadyExists(err) {
			currentRoleBind, err := Clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), roleBind.GetName(), metav1.GetOptions{})
			if err == nil {
				currentRoleBind.Subjects = rbSubjects
				currentRoleBind.RoleRef = roleRef
				_, err = Clientset.RbacV1().ClusterRoleBindings().Update(context.TODO(), currentRoleBind, metav1.UpdateOptions{})
				if err == nil {
					log.Printf("Updated: %s cluster role binding updated", objectName)
					return err
				}
			}
		}
		return err
	}
	return err
}

// CreateObjectSpecificRoleBinding links the cluster role up with the user
func CreateObjectSpecificRoleBinding(tenant, namespace, roleName string, user *registrationv1alpha.UserRequest) error {
	userLabels := user.GetLabels()
	objectName := fmt.Sprintf("%s-%s", roleName, fmt.Sprintf("%s-%s", user.GetName(), userLabels["edge-net.io/user-template-hash"]))
	roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: roleName}
	rbSubjects := []rbacv1.Subject{{Kind: "User", Name: user.Spec.Email, APIGroup: "rbac.authorization.k8s.io"}}
	roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: objectName, Namespace: namespace},
		Subjects: rbSubjects, RoleRef: roleRef}

	roleBindLabels := map[string]string{"edge-net.io/tenant": tenant, "edge-net.io/username": user.GetName(), "edge-net.io/user-template-hash": userLabels["edge-net.io/user-template-hash"]}
	for key, value := range labels {
		roleBindLabels[key] = value
	}
	roleBind.SetLabels(roleBindLabels)
	_, err := Clientset.RbacV1().RoleBindings(namespace).Create(context.TODO(), roleBind, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Couldn't create %s role binding: %s", objectName, err)
		if errors.IsAlreadyExists(err) {
			currentRoleBind, err := Clientset.RbacV1().RoleBindings(namespace).Get(context.TODO(), roleBind.GetName(), metav1.GetOptions{})
			if err == nil {
				currentRoleBind.Subjects = rbSubjects
				currentRoleBind.RoleRef = roleRef
				_, err = Clientset.RbacV1().RoleBindings(namespace).Update(context.TODO(), currentRoleBind, metav1.UpdateOptions{})
				if err == nil {
					log.Printf("Updated: %s role binding updated", objectName)
					return err
				}
			}
		}
	}
	return err
}
