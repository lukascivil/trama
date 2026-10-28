package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/catalog"
)

func newStampCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stamp <repo>",
		Short: "Record in the catalog card the commit it describes (origin/<branch>)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			today := time.Now()
			sha, err := catalog.Stamp(ws, args[0], today)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "stamped %s at %s (%s)\n", args[0], sha[:7], today.Format("2006-01-02"))
			return nil
		},
	}
}
