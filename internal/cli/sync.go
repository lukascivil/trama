package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/syncer"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync [repo...]",
		Short: "Clone missing repositories and fast-forward clean ones",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			results, err := syncer.Sync(ws, args)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			failed := 0
			for _, r := range results {
				fmt.Fprintln(out, strings.TrimRight(fmt.Sprintf("%-24s %-24s %s", r.Repo, r.Outcome, r.Detail), " "))
				if r.Outcome == syncer.Failed {
					failed++
				}
			}
			if failed > 0 {
				return fmt.Errorf("sync failed for %d repo(s)", failed)
			}
			return nil
		},
	}
}
