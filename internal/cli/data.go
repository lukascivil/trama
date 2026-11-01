package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/datafind"
)

func newDataCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "data <term>",
		Short: `List the endpoints and tables that hold a piece of data (e.g. "email"), with their evidence`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			if strings.TrimSpace(args[0]) == "" {
				return fmt.Errorf(`give a field or column to look for, e.g. "email"`)
			}
			names := make([]string, 0, len(ws.Repos))
			cards := map[string]*catalog.Card{}
			var unmapped []string
			for _, r := range ws.Repos {
				names = append(names, r.Name)
				card, err := catalog.Load(ws.Root, r.Name)
				if err != nil {
					return err
				}
				if card == nil {
					unmapped = append(unmapped, r.Name)
					continue
				}
				cards[r.Name] = card
			}

			hits := datafind.Find(names, cards, args[0])
			if len(hits) == 0 {
				return fmt.Errorf("no catalog card has a field or column matching %q", args[0])
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, args[0])
			for _, h := range hits {
				evidence := strings.Join(h.Evidence, ", ")
				if h.Layer == "table" {
					fmt.Fprintf(out, "  %-8s  %s  %s.%s  %s\n", h.Layer, h.Repo, h.Table, h.Path, evidence)
					continue
				}
				fmt.Fprintf(out, "  %-8s  %s  %s  %s: %s  %s\n", h.Layer, h.Repo, h.ID, h.Path, h.Type, evidence)
			}
			if len(unmapped) > 0 {
				fmt.Fprintf(out, "not mapped: %s\n", strings.Join(unmapped, ", "))
			}
			return nil
		},
	}
}
