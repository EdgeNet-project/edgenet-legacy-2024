package fedmanctl

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fedmanctl",
	Short: "fedmanctl - a simple CLI for federating Kubernetes clusters",
	Long:  `fedmanctl is a simple CLI for federating Kubernetes clusters`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}
