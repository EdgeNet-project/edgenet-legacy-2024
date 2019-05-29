package registration

import (
	"bytes"
	"fmt"
	"os"

	"headnode/pkg/namespace"

	apiv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rbacv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/cert"
	cmdconfig "k8s.io/kubernetes/pkg/kubectl/cmd/config"
)

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}

// MakeUser to make user
func MakeUser(kubeconfig *string, user string) bool {
	userNamespace, err := namespace.Create(kubeconfig, user)
	if err != nil {
		fmt.Printf("Namespace couldn't be created: %q.\n", userNamespace)
		panic(err)
	}
	fmt.Printf("Created namespace %q.\n", userNamespace)
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
		panic(err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	clientset, err := rbacv1.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	policyRule := []apiv1.PolicyRule{{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}}}
	var userRole *apiv1.Role
	userRole = &apiv1.Role{ObjectMeta: metav1.ObjectMeta{Namespace: user, Name: fmt.Sprintf("%s-admin", user)},
		Rules: policyRule}
	roleResult, err := clientset.Roles(user).Create(userRole)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Role: %s", roleResult)

	subjects := []apiv1.Subject{{Kind: "User", Name: user, APIGroup: ""}}
	roleRef := apiv1.RoleRef{Kind: "Role", Name: "deployment-manager", APIGroup: ""}
	var roleBinding *apiv1.RoleBinding
	roleBinding = &apiv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: user, Name: fmt.Sprintf("%s-binding", user)},
		Subjects: subjects, RoleRef: roleRef}
	roleBindingResult, err := clientset.RoleBindings(user).Create(roleBinding)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Role Binding: %s", roleBindingResult)

	rbSubjects := []apiv1.Subject{{Kind: "ServiceAccount", Name: "default", Namespace: user}}
	roleBindRef := apiv1.RoleRef{Kind: "ClusterRole", Name: "admin"}
	var roleBind *apiv1.RoleBinding
	roleBind = &apiv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: user, Name: fmt.Sprintf("%s-rolebind", user)},
		Subjects: rbSubjects, RoleRef: roleBindRef}
	roleBindResult, err := clientset.RoleBindings(user).Create(roleBind)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Role Bind (Clusterrole Admin): %s", roleBindResult)

	exist, err := namespace.GetNamespaceByName(kubeconfig, user)
	if err == nil && exist == "true" {
		fmt.Printf("User %s is created successfully", user)
		return true
	}
	return false
}
