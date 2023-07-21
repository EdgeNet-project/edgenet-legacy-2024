package fedmanctl

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

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
	Short: "Federate a workload cluster with the generated token and a namespace.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context, true)

		if err != nil {
			panic(err.Error())
		}

		err = f.FederateWorkloadCluster(args[0], args[1])

		if err != nil {
			panic(err.Error())
		}

		fmt.Println("Federated workload cluster")
	},
}

var managerSeparateCmd = &cobra.Command{
	Use:   "separate <uid> <namespace>",
	Short: "Separate a workload cluster with the uid and namespace.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context, true)

		if err != nil {
			panic(err.Error())
		}

		clusterUID := args[0]

		// If user inputs cluster-XXX format, convert it to normal uid
		clusterUID = strings.Replace(clusterUID, "cluster-", "", 1)

		err = f.SeparateWorkloadCluster(clusterUID, args[1])

		if err != nil {
			panic(err.Error())
		}

		fmt.Println("Sperated the workload cluster")
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

		clusters, err := f.ListWorkloadClusters()

		if err != nil {
			panic(err.Error())
		}

		if len(clusters) == 0 {
			fmt.Println("No workload clusters available")
		} else {
			w := tabwriter.NewWriter(os.Stdout, 20, 20, 1, ' ', 0)

			// Display these properties
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\n", "CLUSTER NAME", "CLUSTER NAMESPACE", "VISIBILITY", "ENABLED", "STATE")

			for _, cluster := range clusters {
				fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\n", cluster.ObjectMeta.Name, cluster.ObjectMeta.Namespace, cluster.Spec.Visibility, cluster.Spec.Enabled, cluster.Status.State)
			}
			w.Flush()
		}
	},
}

func init() {
	managerCmd.AddCommand(managerFederateCmd)
	managerCmd.AddCommand(managerSeparateCmd)
	managerCmd.AddCommand(managerListCmd)
}
