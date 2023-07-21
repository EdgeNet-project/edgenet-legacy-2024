package fedmanctl

import (
	"fmt"
	"strings"

	"github.com/EdgeNet-project/edgenet/pkg/fedmanctl"
	"github.com/spf13/cobra"
)

var managerCmd = &cobra.Command{
	Use:     "manager",
	Aliases: []string{"m"},
	Short:   "Manage manager clusters",
}

var managerFederateCmd = &cobra.Command{
	Use:   "federate <token> <namespace>",
	Short: "Federate a worker cluster with the generated token.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context, true)

		if err != nil {
			panic(err.Error())
		}

		err = f.FederateToManagerCluster(args[0], args[1])

		if err != nil {
			panic(err.Error())
		}

		fmt.Println("Linked worker cluster")
	},
}

var managerUnfederateCmd = &cobra.Command{
	Use:   "unfederate <uid> <namespace>",
	Short: "Unfederate a worker cluster with the uid.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context, true)

		if err != nil {
			panic(err.Error())
		}

		clusterUID := args[0]

		// If user inputs cluster-XXX format, convert it to normal uid
		clusterUID = strings.Replace(clusterUID, "cluster-", "", 1)

		err = f.UnfederateFromManagerCluster(clusterUID, args[1])

		if err != nil {
			panic(err.Error())
		}

		fmt.Println("Unlinked worker cluster")
	},
}

var managerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all of the federated clusters.",
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context, true)

		if err != nil {
			panic(err.Error())
		}

		clusters, err := f.ListWorkerClusters()

		if err != nil {
			panic(err.Error())
		}

		if len(clusters) == 0 {
			fmt.Println("no worker clusters available")
		} else {
			fmt.Printf("%v\t%v\n", "Cluster Name", "Cluster Namespace")
			// Just display the uids for now
			for _, cluster := range clusters {
				fmt.Printf("%v\t%v\n", cluster.ObjectMeta.Name, cluster.ObjectMeta.Namespace)
			}
		}
	},
}

func init() {
	managerCmd.AddCommand(managerFederateCmd)
	managerCmd.AddCommand(managerUnfederateCmd)
	managerCmd.AddCommand(managerListCmd)
}
