// Package cli implements the trama command line.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time with
// -ldflags "-X github.com/lukascivil/trama/internal/cli.version=<version>".
var version = "dev"

// Execute runs the trama command and returns the process exit code.
func Execute() int {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "trama: %v\n", err)
		return 1
	}
	return 0
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "trama",
		Short:         "Local meta-harness for many repositories",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringP("dir", "C", "", "run as if trama was started in this directory")
	root.AddCommand(newVersionCmd(), newInitCmd(), newAddCmd())
	return root
}
