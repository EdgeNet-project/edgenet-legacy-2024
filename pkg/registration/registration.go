package registration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

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
func MakeUser(user string) string {
	userNamespace, err := namespace.Create(user)
	if err != nil {
		//fmt.Printf("Namespace %s couldn't be created.\n", user)
		return fmt.Sprintf("Namespace %s couldn't be created.\n", user)
	}
	//fmt.Printf("Created namespace %q.\n", userNamespace)
	cert.GenerateSelfSignedCertKeyWithFixtures(userNamespace, nil, nil, "../../cmd/user_files/keys")
	pathOptions := clientcmd.NewDefaultPathOptions()
	buf := bytes.NewBuffer([]byte{})
	kcmd := cmdconfig.NewCmdConfigSetAuthInfo(buf, pathOptions)
	kcmd.SetArgs([]string{user})
	kcmd.Flags().Parse([]string{
		fmt.Sprintf("--client-certificate=../../cmd/user_files/keys/%s__.crt", user),
		fmt.Sprintf("--client-key=../../cmd/user_files/keys/%s__.key", user),
	})

	if err := kcmd.Execute(); err != nil {
		fmt.Printf("unexpected error executing command: %v,kubectl config set-credentials  args: %v", err, user)
		return fmt.Sprintf("unexpected error executing command: %v,kubectl config set-credentials  args: %v", err, user)
	}

	clientset, err := authorization.CreateClientSet()
	if err != nil {
		panic(err.Error())
	}

	policyRule := []apiv1.PolicyRule{{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}}}
	var userRole *apiv1.Role
	userRole = &apiv1.Role{ObjectMeta: metav1.ObjectMeta{Namespace: user, Name: fmt.Sprintf("%s-admin", user)},
		Rules: policyRule}
	_, err = clientset.RbacV1().Roles(user).Create(userRole)
	if err != nil {
		return fmt.Sprintf("Err: %s", err)
	}

	subjects := []apiv1.Subject{{Kind: "User", Name: user, APIGroup: ""}}
	roleRef := apiv1.RoleRef{Kind: "Role", Name: "deployment-manager", APIGroup: ""}
	var roleBinding *apiv1.RoleBinding
	roleBinding = &apiv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: user, Name: fmt.Sprintf("%s-binding", user)},
		Subjects: subjects, RoleRef: roleRef}
	_, err = clientset.RbacV1().RoleBindings(user).Create(roleBinding)
	if err != nil {
		return fmt.Sprintf("Err: %s", err)
	}

	rbSubjects := []apiv1.Subject{{Kind: "ServiceAccount", Name: "default", Namespace: user}}
	roleBindRef := apiv1.RoleRef{Kind: "ClusterRole", Name: "admin"}
	var roleBind *apiv1.RoleBinding
	roleBind = &apiv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: user, Name: fmt.Sprintf("%s-rolebind", user)},
		Subjects: rbSubjects, RoleRef: roleBindRef}
	_, err = clientset.RbacV1().RoleBindings(user).Create(roleBind)
	if err != nil {
		return fmt.Sprintf("Err: %s", err)
	}

	exist, err := namespace.GetNamespaceByName(user)
	if err == nil && exist == "true" {
		resultMap := map[string]string{"status": "Acknowledged"}
		result, _ := json.Marshal(resultMap)
		return string(result)
	}
	return fmt.Sprintf("Err: %s", err)
}

// MakeConfig checks/gets serviceaccount of the user (actually, the namespace), and if the serviceaccount exists
// this function checks/gets its secret, and then CA and token info of the secret. Subsequently, this reads cluster
// and server info of the current context from the config file to use them on the creation of kubeconfig.
func MakeConfig(user string) string {
	clientset, err := authorization.CreateClientSet()
	if err != nil {
		panic(err.Error())
	}

	serviceAccount, err := clientset.CoreV1().ServiceAccounts(user).Get("default", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("Serviceaccount %s not found\n", user)
		return fmt.Sprintf("Serviceaccount %s not found\n", user)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting serviceaccount %s: %v\n", user, statusError.ErrStatus)
		return fmt.Sprintf("Error getting serviceaccount %s: %v\n", user, statusError.ErrStatus)
	} else if err != nil {
		panic(err.Error())
	}
	accountSecret := serviceAccount.Secrets[0].Name
	if accountSecret == "" {
		fmt.Printf("Serviceaccount %s doesn't have a serviceaccount token\n", user)
		return fmt.Sprintf("Serviceaccount %s doesn't have a serviceaccount token\n", user)
	}

	secret, err := clientset.CoreV1().Secrets(user).Get(accountSecret, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("Secret %s not found\n", user)
		return fmt.Sprintf("Secret %s not found\n", user)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting secret %s: %v\n", user, statusError.ErrStatus)
		return fmt.Sprintf("Error getting secret %s: %v\n", user, statusError.ErrStatus)
	} else if err != nil {
		panic(err.Error())
	}

	cluster, server, err := custconfig.GetClusterServerOfCurrentContext()
	if err != nil {
		fmt.Printf("Err: %s", err)
		return fmt.Sprintf("Err: %s", err)
	}

	newKubeConfig := kubeconfigutil.CreateWithToken(server, cluster, "default", secret.Data["ca.crt"], string(secret.Data["token"]))
	newKubeConfig.Contexts[newKubeConfig.CurrentContext].Namespace = user
	newKubeConfig.Contexts["kubernetes-admin@kubernetes"] = newKubeConfig.Contexts[newKubeConfig.CurrentContext]
	delete(newKubeConfig.Contexts, newKubeConfig.CurrentContext)
	newKubeConfig.CurrentContext = "kubernetes-admin@kubernetes"
	kubeconfigutil.WriteToDisk(fmt.Sprintf("../../cmd/user_files/keys/edgenet_%s.cfg", user), newKubeConfig)

	dat, err := ioutil.ReadFile(fmt.Sprintf("../../cmd/user_files/keys/edgenet_%s.cfg", user))
	if err != nil {
		fmt.Printf("Err: %s", err)
		return fmt.Sprintf("Err: %s", err)
	}
	return string(dat)
}
