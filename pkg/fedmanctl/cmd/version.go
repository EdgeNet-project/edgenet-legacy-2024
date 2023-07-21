package fedmanctl

import (
	"fmt"

	"github.com/EdgeNet-project/edgenet/pkg/fedmanctl"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print the version of the fedmanctl",
	Run: func(cmd *cobra.Command, args []string) {
		f, err := fedmanctl.NewFedmanctl(kubeconfig, context, false)

		if err != nil {
			panic(err.Error())
		}

		fmt.Printf("%v\n", f.Version())
	},
}
