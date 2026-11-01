// Package cli implements the trama command line.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/runner"
)

// version is set at build time with
// -ldflags "-X github.com/lukascivil/trama/internal/cli.version=<version>".
var version = "dev"

// Execute runs the trama command and returns the process exit code.
func Execute() int {
	if err := newRootCmd().Execute(); err != nil {
		code, quiet := exitCode(err)
		if !quiet {
			fmt.Fprintf(os.Stderr, "trama: %v\n", err)
		}
		return code
	}
	return 0
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "trama",
		Short:         "Local meta-harness for many repositories",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate("{{.Version}}\n")
	root.PersistentFlags().StringP("dir", "C", "", "run as if trama was started in this directory")
	root.AddCommand(newVersionCmd(), newInitCmd(), newAddCmd(), newSyncCmd(), newStatusCmd(), newValidateCmd(), newStampCmd(), newBodiesCmd(), newLinkCmd(), newGraphCmd(), newSetupCmd(), newTestCmd(), newDevCmd(), newUpCmd(), newUsesCmd(), newDataCmd(), newSchemasCmd(), newBranchCmd(), newPRCmd(), newChangesCmd())
	return root
}

// exitCode maps an error to the process exit code and whether to stay
// quiet: an interrupt exits 130 without a message, like a shell.
func exitCode(err error) (int, bool) {
	if errors.Is(err, runner.ErrInterrupted) {
		return 130, true
	}
	return 1, false
}
