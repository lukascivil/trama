package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/linker"
)

func newLinkCmd() *cobra.Command {
	var dry bool
	cmd := &cobra.Command{
		Use:   "link",
		Short: "Link every from: unknown to the one repository of the workspace that provides it",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			changes, err := linker.Plan(ws)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(changes) == 0 {
				fmt.Fprintln(out, "nothing to link")
				return nil
			}
			linked := 0
			for _, ch := range changes {
				if ch.From != "" {
					linked++
					fmt.Fprintf(out, "%s: %s -> %s (via %s)\n", ch.Repo, ch.ID, ch.From, ch.Via)
					continue
				}
				fmt.Fprintf(out, "%s: %s stays unknown, served by %s\n", ch.Repo, ch.ID, strings.Join(ch.Candidates, ", "))
			}
			if dry {
				return nil
			}
			if err := linker.Apply(ws.Root, changes); err != nil {
				return err
			}
			fmt.Fprintf(out, "\nlinked %d of %d; review the diff in catalog/ and run \"trama validate\"\n", linked, len(changes))
			return nil
		},
	}
	cmd.Flags().BoolVar(&dry, "dry-run", false, "print what would change and write nothing")
	return cmd
}
