package fedmanctl

import "github.com/spf13/cobra"

var workerCmd = &cobra.Command{
	Use:     "worker",
	Aliases: []string{"w"},
	Short:   "Manage worker clusters",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {

}
