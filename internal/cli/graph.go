package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/graph"
	"github.com/lukascivil/trama/internal/validate"
)

func newGraphCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "graph",
		Short: "Write docs/graph.md and the repository table in AGENTS.md from the catalog",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			if n := validate.Errors(validate.Workspace(ws)); n > 0 {
				return fmt.Errorf(`fix %d validation error(s) first: run "trama validate"`, n)
			}
			if err := graph.Write(ws); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "wrote docs/graph.md")
			fmt.Fprintln(out, "updated AGENTS.md")
			return nil
		},
	}
}
