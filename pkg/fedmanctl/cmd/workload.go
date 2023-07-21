package fedmanctl

import (
	"fmt"

	"github.com/EdgeNet-project/edgenet/pkg/fedmanctl"
	"github.com/spf13/cobra"
)

var workloadCmd = &cobra.Command{
	Use:     "workload",
	Aliases: []string{"w"},
	Short:   "Manage workload clusters",
}

var workloadInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the cluster as the workload cluster. Configure such that, it can receive and send workloads.",
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context, true)

		if err != nil {
			panic(err.Error())
		}

		err = f.InitWorkloadCluster()

		if err != nil {
			panic(err.Error())
		}

		fmt.Println("Initialized workload cluster")
	},
}

var workloadResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the configuration of the cluster as a workload cluster.",
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context, true)

		if err != nil {
			panic(err.Error())
		}

		err = f.ResetWorkloadCluster()

		if err != nil {
			panic(err.Error())
		}

		fmt.Println("Reset workload cluster")
	},
}

var workloadTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Generate the token to be fed to the manager cluster. Geo tags can be overwritten.",
	Run: func(cmd *cobra.Command, args []string) {
		silent, _ := cmd.Flags().GetBool("silent")
		debug, _ := cmd.Flags().GetBool("debug")

		f, err := fedmanctl.NewFedmanctl(kubeconfig, context, silent)

		if err != nil {
			panic(err.Error())
		}

		city, _ := cmd.Flags().GetString("city")
		country, _ := cmd.Flags().GetString("country")

		ip, _ := cmd.Flags().GetString("ip")
		port, _ := cmd.Flags().GetString("port")

		// more types of labels can be added here
		labels := map[string]string{
			"edge-net.io/city":    city,
			"edge-net.io/country": country,
		}

		visibility, _ := cmd.Flags().GetString("visibility")

		token, err := f.GenerateWorkloadClusterToken(ip, port, visibility, debug, labels)

		if err != nil {
			panic(err.Error())
		}

		if !silent {
			fmt.Println("Token generated use the following command to link the workload cluster with your manager cluster.")
			fmt.Println("")
			fmt.Printf("fedmanctl manager link %v\n", token)
		} else {
			fmt.Printf("%v\n", token)
		}

	},
}

func init() {
	workloadTokenCmd.Flags().Bool("silent", false, "Only print the token")
	workloadTokenCmd.Flags().Bool("debug", false, "Print the token in json format")

	workloadTokenCmd.Flags().String("visibility", "Public", "Visibility of the cluster, Public or Private")
	workloadTokenCmd.Flags().String("ip", "", "IP address of the kube-apiserver")
	workloadTokenCmd.Flags().String("port", "", "Port of the kube-apiserver")

	workloadTokenCmd.Flags().String("city", "", "Override the city label of the cluster")
	workloadTokenCmd.Flags().String("country", "", "Override the country label of the cluster")

	workloadCmd.AddCommand(workloadInitCmd)
	workloadCmd.AddCommand(workloadResetCmd)
	workloadCmd.AddCommand(workloadTokenCmd)
}
