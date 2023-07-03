package fedmanctl

import (
	"fmt"

	"github.com/EdgeNet-project/edgenet/pkg/fedmanctl"
	"github.com/spf13/cobra"
)

var managerCmd = &cobra.Command{
	Use:     "manager",
	Aliases: []string{"m"},
	Short:   "Manage manager clusters",
}

var managerLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "Link a worker cluster with the generated token.",
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context)

		if err != nil {
			panic(err.Error())
		}

		fmt.Printf("Not Implemented command of fedmanctl version: %v\n", f.Version())
	},
}

var managerUnlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Unlink a worker cluster with the uid.",
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context)

		if err != nil {
			panic(err.Error())
		}

		fmt.Printf("Not Implemented command of fedmanctl version: %v\n", f.Version())
	},
}

var managerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all of the federated clusters.",
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context)

		if err != nil {
			panic(err.Error())
		}

		fmt.Printf("Not Implemented command of fedmanctl version: %v\n", f.Version())
	},
}

func init() {
	managerCmd.AddCommand(managerLinkCmd)
	managerCmd.AddCommand(managerUnlinkCmd)
	managerCmd.AddCommand(managerListCmd)
}
