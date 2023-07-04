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

var managerLinkCmd = &cobra.Command{
	Use:   "link <token>",
	Short: "Link a worker cluster with the generated token.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context, true)

		if err != nil {
			panic(err.Error())
		}

		err = f.LinkToManagerCluster(args[0])

		if err != nil {
			panic(err.Error())
		}

		fmt.Println("Linked worker cluster")
	},
}

var managerUnlinkCmd = &cobra.Command{
	Use:   "unlink <uid>",
	Short: "Unlink a worker cluster with the uid.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context, true)

		if err != nil {
			panic(err.Error())
		}

		clusterUID := args[0]

		// If user inputs cluster-XXX format, convert it to normal uid
		clusterUID = strings.Replace(clusterUID, "cluster-", "", 1)

		err = f.UnlinkFromManagerCluster(clusterUID)

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
			// Just display the uids for now
			for _, cluster := range clusters {
				fmt.Printf("%v\n", cluster.ObjectMeta.Name)
			}
		}
	},
}

func init() {
	managerCmd.AddCommand(managerLinkCmd)
	managerCmd.AddCommand(managerUnlinkCmd)
	managerCmd.AddCommand(managerListCmd)
}
