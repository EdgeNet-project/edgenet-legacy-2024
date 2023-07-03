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

		err = f.InitWorkerCluster()

		if err != nil {
			panic(err.Error())
		}

		fmt.Println("Initialized worker cluster")
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

		err = f.ResetWorkerCluster()

		if err != nil {
			panic(err.Error())
		}

		fmt.Println("Reset worker cluster")
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

		city, _ := cmd.Flags().GetString("city")
		country, _ := cmd.Flags().GetString("country")

		// more types of labels can be added here
		labels := map[string]string{
			"edge-net.io/city":    city,
			"edge-net.io/country": country,
		}

		token, err := f.GenerateWorkerClusterToken(labels)

		if err != nil {
			panic(err.Error())
		}

		silent, _ := cmd.Flags().GetBool("silent")

		if !silent {
			fmt.Println("Token generated use the following command to link the worker cluster with your manager cluster.")
			fmt.Println("")
			fmt.Printf("fedmanctl manager link %v\n", token)
		} else {
			fmt.Printf("%v\n", token)
		}

	},
}

func init() {
	workerTokenCmd.Flags().Bool("silent", false, "Only print the token")
	workerTokenCmd.Flags().String("city", "", "Override the city label of the cluster")
	workerTokenCmd.Flags().String("country", "", "Override the country label of the cluster")

	workerCmd.AddCommand(workerInitCmd)
	workerCmd.AddCommand(workerResetCmd)
	workerCmd.AddCommand(workerTokenCmd)
}
