package fedmanctl

import (
	"fmt"

	"github.com/EdgeNet-project/edgenet/pkg/fedmanctl"
	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:     "worker",
	Aliases: []string{"w"},
	Short:   "Manage worker clusters",
}

var workerInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the cluster as the worker cluster. Configure such that, it can receive and send workloads.",
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context)

		if err != nil {
			panic(err.Error())
		}

		fmt.Printf("Not Implemented command of fedmanctl version: %v\n", f.Version())
	},
}

var workerResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the configuration of the cluster as a workload cluster.",
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context)

		if err != nil {
			panic(err.Error())
		}

		fmt.Printf("Not Implemented command of fedmanctl version: %v\n", f.Version())
	},
}

var workerTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Generate the token to be fed to the manager cluster. Geo tags can be overwritten.",
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context)

		if err != nil {
			panic(err.Error())
		}

		fmt.Printf("Not Implemented command of fedmanctl version: %v\n", f.Version())
	},
}

func init() {
	workerCmd.AddCommand(workerInitCmd)
	workerCmd.AddCommand(workerResetCmd)
	workerCmd.AddCommand(workerTokenCmd)
}
