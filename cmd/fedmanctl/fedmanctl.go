package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/fedmanctl"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig       string
	context          string
	kubeclientset    *kubernetes.Clientset
	edgenetclientset *versioned.Clientset
)

func loadConfig() {
	var config *rest.Config
	var err error

	if context != "" {
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
			&clientcmd.ConfigOverrides{
				CurrentContext: context,
			}).ClientConfig()

		if err != nil {
			panic(err.Error())
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)

		if err != nil {
			panic(err.Error())
		}
	}

	kubeclientset, err = bootstrap.CreateKubeClientset(config)
	if err != nil {
		panic(err.Error())
	}

	edgenetclientset, err = bootstrap.CreateEdgeNetClientset(config)
	if err != nil {
		panic(err.Error())
	}
}

var rootCmd = &cobra.Command{
	Use:   "fedmanctl",
	Short: "fedmanctl - a simple CLI for federating Kubernetes clusters",
	Long:  `fedmanctl is a simple CLI for federating Kubernetes clusters`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Please refer to fedmanctl --help for more information")
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of fedmanctl",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("v1.0.0")
	},
}

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Manage worker cluster operations",
}

var workerInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize federation on a worker cluster",
	Run: func(cmd *cobra.Command, args []string) {
		loadConfig()
		p := &fedmanctl.WorkerFederationPerformer{
			Kubeclientset:    kubeclientset,
			Edgenetclientset: edgenetclientset,
		}

		token, err := p.CreateWorkerToken()

		if err != nil {
			fmt.Println("Canot create token an error occured:")
			panic(err.Error())
		}

		fmt.Println("Created the token. Use the following command on your federation cluster to complete the federation of your worker cluster.")
		fmt.Println("")
		fmt.Printf("fedmanctl manager init %v\n", token)
	},
}

var workerResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the federation extensions from a worker cluster",
	Run: func(cmd *cobra.Command, args []string) {
		loadConfig()
		p := &fedmanctl.WorkerFederationPerformer{
			Kubeclientset:    kubeclientset,
			Edgenetclientset: edgenetclientset,
		}

		p.ResetWorkerClusterFederation()
	},
}

var managerCmd = &cobra.Command{
	Use:   "manager",
	Short: "Manage worker cluster operations",
}

var managerInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Manage worker cluster operations",
	Run: func(cmd *cobra.Command, args []string) {
		loadConfig()
		if len(args) != 1 {
			panic(errors.New("init command only needs <base4> token"))
		}

		token := args[0]
		err := fedmanctl.FederateByWorkerToken(kubeclientset, edgenetclientset, token)

		if err != nil {
			panic(err.Error())
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", os.Getenv("KUBECONFIG"), "Kubeconfig file to be used")
	rootCmd.PersistentFlags().StringVar(&context, "context", "", "The context specified in the kubeconfig file")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(workerCmd)
	rootCmd.AddCommand(managerCmd)

	workerCmd.AddCommand(workerInitCmd)
	workerCmd.AddCommand(workerResetCmd)

	managerCmd.AddCommand(managerInitCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}
