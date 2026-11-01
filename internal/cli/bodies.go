package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/bodies"
)

func newBodiesCmd() *cobra.Command {
	var dry bool
	cmd := &cobra.Command{
		Use:   "bodies [repo...]",
		Short: "Write the request and response of every route an OpenAPI 3 spec of the repository declares",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			reports, err := bodies.Plan(ws, args)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			var changes []bodies.Change
			for _, r := range reports {
				for _, ch := range r.Changes {
					fmt.Fprintf(out, "%s: %s <- %s:%d (%s)\n", ch.Repo, ch.ID, ch.Spec, ch.Line, ch.Summary())
				}
				for _, n := range r.Notes {
					fmt.Fprintf(out, "%s: %s\n", r.Repo, n)
				}
				changes = append(changes, r.Changes...)
			}
			if len(changes) == 0 {
				fmt.Fprintln(out, "nothing to write")
				return nil
			}
			if dry {
				return nil
			}
			if err := bodies.Apply(ws.Root, changes); err != nil {
				return err
			}
			noun := "provides"
			if len(changes) == 1 {
				noun = "provide"
			}
			fmt.Fprintf(out, "\nwrote %d %s; review the diff in catalog/ and run \"trama validate\"\n", len(changes), noun)
			return nil
		},
	}
	cmd.Flags().BoolVar(&dry, "dry-run", false, "print what would change and write nothing")
	return cmd
}
