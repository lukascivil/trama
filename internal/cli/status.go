package cli

import (
	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/status"
)

func newStatusCmd() *cobra.Command {
	var quiet bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show every repository and what needs attention (offline)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			status.Render(cmd.OutOrStdout(), status.Collect(ws), quiet)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "print only problems, and nothing when there are none")
	return cmd
}
