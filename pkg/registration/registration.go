package registration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"headnode/pkg/authorization"
	"headnode/pkg/namespace"

	apiv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	rbacv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/cert"
	kubeconfigutil "k8s.io/kubernetes/cmd/kubeadm/app/util/kubeconfig"
	cmdconfig "k8s.io/kubernetes/pkg/kubectl/cmd/config"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}

// MakeUser to make user
func MakeUser(kubeconfig *string, user string) string {
	userNamespace, err := namespace.Create(kubeconfig, user)
	if err != nil {
		fmt.Printf("Namespace %s couldn't be created.\n", user)
		return fmt.Sprintf("Namespace %s couldn't be created.\n", user)
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
		return fmt.Sprintf("unexpected error executing command: %v,kubectl config set-credentials  args: %v", err, user)
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return fmt.Sprintf("unexpected error executing command: %v,kubectl config set-credentials  args: %v", err, user)
	}
	clientset, err := rbacv1.NewForConfig(config)
	if err != nil {
		return fmt.Sprintf("Err: %s", err)
	}

	policyRule := []apiv1.PolicyRule{{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}}}
	var userRole *apiv1.Role
	userRole = &apiv1.Role{ObjectMeta: metav1.ObjectMeta{Namespace: user, Name: fmt.Sprintf("%s-admin", user)},
		Rules: policyRule}
	_, err = clientset.Roles(user).Create(userRole)
	if err != nil {
		return fmt.Sprintf("Err: %s", err)
	}

	subjects := []apiv1.Subject{{Kind: "User", Name: user, APIGroup: ""}}
	roleRef := apiv1.RoleRef{Kind: "Role", Name: "deployment-manager", APIGroup: ""}
	var roleBinding *apiv1.RoleBinding
	roleBinding = &apiv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: user, Name: fmt.Sprintf("%s-binding", user)},
		Subjects: subjects, RoleRef: roleRef}
	_, err = clientset.RoleBindings(user).Create(roleBinding)
	if err != nil {
		return fmt.Sprintf("Err: %s", err)
	}

	rbSubjects := []apiv1.Subject{{Kind: "ServiceAccount", Name: "default", Namespace: user}}
	roleBindRef := apiv1.RoleRef{Kind: "ClusterRole", Name: "admin"}
	var roleBind *apiv1.RoleBinding
	roleBind = &apiv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: user, Name: fmt.Sprintf("%s-rolebind", user)},
		Subjects: rbSubjects, RoleRef: roleBindRef}
	_, err = clientset.RoleBindings(user).Create(roleBind)
	if err != nil {
		return fmt.Sprintf("Err: %s", err)
	}

	exist, err := namespace.GetNamespaceByName(kubeconfig, user)
	if err == nil && exist == "true" {
		resultMap := map[string]string{"status": "Acknowledged"}
		result, _ := json.Marshal(resultMap)
		return string(result)
	}
	return fmt.Sprintf("Err: %s", err)
}

// MakeConfig to make config
func MakeConfig(kubeconfig *string, user string) string {
	clientset, err := authorization.CreateClientSet(kubeconfig)
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

	pathOptions := clientcmd.NewDefaultPathOptions()
	buf := bytes.NewBuffer([]byte{})
	currentContextCmd := cmdconfig.NewCmdConfigCurrentContext(buf, pathOptions)
	if err := currentContextCmd.Execute(); err != nil {
		fmt.Printf("unexpected error executing command: %v", err)
		return fmt.Sprintf("unexpected error executing command: %v", err)
	}

	streamsIn := &bytes.Buffer{}
	streamsOut := &bytes.Buffer{}
	streamsErrOut := &bytes.Buffer{}
	streams := genericclioptions.IOStreams{
		In:     streamsIn,
		Out:    streamsOut,
		ErrOut: streamsErrOut,
	}
	configCmd := cmdconfig.NewCmdConfigView(cmdutil.NewFactory(genericclioptions.NewConfigFlags(false)), streams, pathOptions)
	// "context" is a global flag, inherited from base kubectl command in the real world
	configCmd.Flags().String("context", "", "The name of the kubeconfig context to use")
	configCmd.Flags().Parse([]string{"--output=json"})
	if err := configCmd.Execute(); err != nil {
		fmt.Printf("unexpected error executing command: %v", err)
		return fmt.Sprintf("unexpected error executing command: %v", err)
	}

	type ClusterDetails struct {
		Server string `json:"server"`
	}
	type Clusters struct {
		Cluster ClusterDetails `json:"cluster"`
		Name    string         `json:"name"`
	}
	type ContextDetails struct {
		Cluster string `json:"cluster"`
		User    string `json:"user"`
	}
	type Contexts struct {
		Context ContextDetails `json:"context"`
		Name    string         `json:"name"`
	}
	type ConfigView struct {
		Clusters       []Clusters `json:"clusters"`
		Contexts       []Contexts `json:"contexts"`
		CurrentContext string     `json:"current-context"`
	}

	output := fmt.Sprint(streams.Out)
	var configViewDet ConfigView
	err = json.Unmarshal([]byte(output), &configViewDet)
	if err != nil {
		fmt.Printf("Err: %s", err)
		return fmt.Sprintf("Err: %s", err)
	}
	currentContext := configViewDet.CurrentContext
	var cluster string
	for _, contextRaw := range configViewDet.Contexts {
		if contextRaw.Name == currentContext {
			cluster = contextRaw.Context.Cluster
		}
	}
	var server string
	for _, clusterRaw := range configViewDet.Clusters {
		if clusterRaw.Name == cluster {
			server = clusterRaw.Cluster.Server
		}
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
