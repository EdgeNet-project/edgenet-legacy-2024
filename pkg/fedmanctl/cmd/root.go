package fedmanctl

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// kubeconfig file path
	kubeconfig string

	// context on the kubeconfig file
	context string
)

var rootCmd = &cobra.Command{
	Use:   "fedmanctl",
	Short: "fedmanctl federate Kubernetes clusters",
	Long: `fedmanctl is a simple CLI for federating Kubernetes clusters using EdgeNet features. For more info 
please visit the EdgeNet GitHub page available https://github.com/edgenet-project/edgenet`,
	Run: func(cmd *cobra.Command, args []string) {},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Kubeconfig file to be used")
	rootCmd.PersistentFlags().StringVar(&context, "context", "", "The context specified in the kubeconfig file")

	rootCmd.AddCommand(workerCmd)
	rootCmd.AddCommand(managerCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "An error occured while executing fedmanctl: '%s'", err)
		os.Exit(1)
	}
}
