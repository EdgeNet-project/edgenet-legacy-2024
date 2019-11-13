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

package registration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"headnode/pkg/authorization"
	custconfig "headnode/pkg/config"
	"headnode/pkg/namespace"

	apiv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/cert"
	kubeconfigutil "k8s.io/kubernetes/cmd/kubeadm/app/util/kubeconfig"
	cmdconfig "k8s.io/kubernetes/pkg/kubectl/cmd/config"
)

// MakeUser generates key and certificate and then set credentials into the config file. As the next step,
// this function creates user role and role bindings for the namespace. Lastly, this checks the namespace
// created successfully or not.
func MakeUser(user string) ([]byte, int) {
	userNamespace, err := namespace.Create(user)
	if err != nil {
		log.Printf("Namespace %s couldn't be created.", user)
		resultMap := map[string]string{"status": "Failure"}
		result, _ := json.Marshal(resultMap)
		return result, 500
	}
	cert.GenerateSelfSignedCertKeyWithFixtures(userNamespace, nil, nil, "../../assets/certs")
	pathOptions := clientcmd.NewDefaultPathOptions()
	buf := bytes.NewBuffer([]byte{})
	kcmd := cmdconfig.NewCmdConfigSetAuthInfo(buf, pathOptions)
	kcmd.SetArgs([]string{user})
	kcmd.Flags().Parse([]string{
		fmt.Sprintf("--client-certificate=../../assets/certs/%s__.crt", user),
		fmt.Sprintf("--client-key=../../assets/certs/%s__.key", user),
	})

	if err := kcmd.Execute(); err != nil {
		log.Printf("Couldn't set auth info on the kubeconfig file: %s", user)
		resultMap := map[string]string{"status": "Failure"}
		result, _ := json.Marshal(resultMap)
		return result, 500
	}

	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	policyRule := []apiv1.PolicyRule{{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}}}
	var userRole *apiv1.Role
	userRole = &apiv1.Role{ObjectMeta: metav1.ObjectMeta{Namespace: user, Name: fmt.Sprintf("%s-admin", user)},
		Rules: policyRule}
	_, err = clientset.RbacV1().Roles(user).Create(userRole)
	if err != nil {
		log.Printf("Couldn't create user role: %s", user)
		resultMap := map[string]string{"status": "Failure"}
		result, _ := json.Marshal(resultMap)
		return result, 500
	}

	/*subjects := []apiv1.Subject{{Kind: "User", Name: user, APIGroup: ""}}
	roleRef := apiv1.RoleRef{Kind: "Role", Name: "deployment-manager", APIGroup: ""}
	var roleBinding *apiv1.RoleBinding
	roleBinding = &apiv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: user, Name: fmt.Sprintf("%s-binding", user)},
		Subjects: subjects, RoleRef: roleRef}
	_, err = clientset.RbacV1().RoleBindings(user).Create(roleBinding)
	if err != nil {
		log.Printf("Couldn't create user role binding: %s", user)
		resultMap := map[string]string{"status": "Failure"}
		result, _ := json.Marshal(resultMap)
		return result, 500
	}*/

	rbSubjects := []apiv1.Subject{{Kind: "ServiceAccount", Name: "default", Namespace: user}}
	roleBindRef := apiv1.RoleRef{Kind: "ClusterRole", Name: "admin"}
	var roleBind *apiv1.RoleBinding
	roleBind = &apiv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: user, Name: fmt.Sprintf("%s-rolebind", user)},
		Subjects: rbSubjects, RoleRef: roleBindRef}
	_, err = clientset.RbacV1().RoleBindings(user).Create(roleBind)
	if err != nil {
		log.Printf("Couldn't create user admin role binding: %s", user)
		resultMap := map[string]string{"status": "Failure"}
		result, _ := json.Marshal(resultMap)
		return result, 500
	}

	rbSubjectsEdgenet := []apiv1.Subject{{Kind: "ServiceAccount", Name: "default", Namespace: user}}
	roleBindRefEdgenet := apiv1.RoleRef{Kind: "ClusterRole", Name: "edgenet-admin"}
	var roleBindEdgenet *apiv1.RoleBinding
	roleBindEdgenet = &apiv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: user, Name: fmt.Sprintf("%s-rolebind-edgenet", user)},
		Subjects: rbSubjectsEdgenet, RoleRef: roleBindRefEdgenet}
	_, err = clientset.RbacV1().RoleBindings(user).Create(roleBindEdgenet)
	if err != nil {
		log.Printf("Couldn't create user admin role binding: %s", user)
		resultMap := map[string]string{"status": "Failure"}
		result, _ := json.Marshal(resultMap)
		return result, 500
	}

	exist, err := namespace.GetNamespaceByName(user)
	if err == nil && exist == "true" {
		resultMap := map[string]string{"status": "Acknowledged"}
		result, _ := json.Marshal(resultMap)
		return result, 200
	}

	log.Printf("Namespace couldn't be created: %s", user)
	resultMap := map[string]string{"status": "Failure"}
	result, _ := json.Marshal(resultMap)
	return result, 500
}

// MakeConfig checks/gets serviceaccount of the user (actually, the namespace), and if the serviceaccount exists
// this function checks/gets its secret, and then CA and token info of the secret. Subsequently, this reads cluster
// and server info of the current context from the config file to use them on the creation of kubeconfig.
func MakeConfig(user string) string {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	serviceAccount, err := clientset.CoreV1().ServiceAccounts(user).Get("default", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Printf("Serviceaccount %s not found", user)
		return fmt.Sprintf("Serviceaccount %s not found\n", user)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Printf("Error getting serviceaccount %s: %v", user, statusError.ErrStatus)
		return fmt.Sprintf("Error getting serviceaccount %s: %v\n", user, statusError.ErrStatus)
	} else if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	accountSecret := serviceAccount.Secrets[0].Name
	if accountSecret == "" {
		log.Printf("Serviceaccount %s doesn't have a serviceaccount token", user)
		return fmt.Sprintf("Serviceaccount %s doesn't have a serviceaccount token\n", user)
	}

	secret, err := clientset.CoreV1().Secrets(user).Get(accountSecret, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Printf("Secret %s not found", user)
		return fmt.Sprintf("Secret %s not found\n", user)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Printf("Error getting secret %s: %v", user, statusError.ErrStatus)
		return fmt.Sprintf("Error getting secret %s: %v\n", user, statusError.ErrStatus)
	} else if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	cluster, server, err := custconfig.GetClusterServerOfCurrentContext()
	if err != nil {
		log.Println(err)
		return fmt.Sprintf("Err: %s", err)
	}

	newKubeConfig := kubeconfigutil.CreateWithToken(server, cluster, "default", secret.Data["ca.crt"], string(secret.Data["token"]))
	newKubeConfig.Contexts[newKubeConfig.CurrentContext].Namespace = user
	newKubeConfig.Contexts["kubernetes-admin@kubernetes"] = newKubeConfig.Contexts[newKubeConfig.CurrentContext]
	delete(newKubeConfig.Contexts, newKubeConfig.CurrentContext)
	newKubeConfig.CurrentContext = "kubernetes-admin@kubernetes"
	kubeconfigutil.WriteToDisk(fmt.Sprintf("../../assets/kubeconfigs/edgenet_%s.cfg", user), newKubeConfig)

	dat, err := ioutil.ReadFile(fmt.Sprintf("../../assets/kubeconfigs/edgenet_%s.cfg", user))
	if err != nil {
		log.Println(err)
		return fmt.Sprintf("Err: %s", err)
	}
	return string(dat)
}
