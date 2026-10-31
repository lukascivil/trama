package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/catalog"
)

func newUsesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uses <id>",
		Short: `List the repositories that provide or consume an id (e.g. "GET /hello"), with their evidence`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			id := args[0]
			var lines, unmapped []string
			for _, r := range ws.Repos {
				card, err := catalog.Load(ws.Root, r.Name)
				if err != nil {
					return err
				}
				if card == nil {
					unmapped = append(unmapped, r.Name)
					continue
				}
				for _, p := range card.Provides {
					if p.ID == id {
						lines = append(lines, fmt.Sprintf("  provided by %s  %s", r.Name, strings.Join(p.Evidence, ", ")))
					}
				}
				for _, c := range card.Consumes {
					if c.ID != id {
						continue
					}
					evidence := strings.Join(c.Evidence, ", ")
					if c.From == catalog.Unknown {
						lines = append(lines, fmt.Sprintf("  pending in  %s  candidates: %s  %s", r.Name, orNone(c.Candidates), evidence))
						continue
					}
					how := "via " + c.Via
					if len(c.FieldsUsed) > 0 {
						how += ", fields: " + strings.Join(c.FieldsUsed, ", ")
					}
					lines = append(lines, fmt.Sprintf("  consumed by %s  %s  %s", r.Name, how, evidence))
				}
			}
			if len(lines) == 0 {
				return fmt.Errorf("no catalog card provides or consumes %q", id)
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, id)
			for _, l := range lines {
				fmt.Fprintln(out, l)
			}
			if len(unmapped) > 0 {
				fmt.Fprintf(out, "not mapped: %s\n", strings.Join(unmapped, ", "))
			}
			return nil
		},
	}
}

func orNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}
