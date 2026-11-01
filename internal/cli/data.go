package cli

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/catalog"
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
			term := strings.ToLower(strings.TrimSpace(args[0]))
			if term == "" {
				return fmt.Errorf(`give a field or column to look for, e.g. "email"`)
			}
			cards := map[string]*catalog.Card{}
			var unmapped []string
			for _, r := range ws.Repos {
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

			var lines []string
			// Endpoints first: a field that already comes from a route is one
			// the caller can use today, with nobody else's work.
			for _, body := range []string{"response", "request"} {
				for _, r := range ws.Repos {
					card := cards[r.Name]
					if card == nil {
						continue
					}
					for _, p := range card.Provides {
						fields := p.Response
						if body == "request" {
							fields = p.Request
						}
						for _, path := range slices.Sorted(maps.Keys(fields)) {
							if !strings.Contains(strings.ToLower(path), term) {
								continue
							}
							lines = append(lines, fmt.Sprintf("  %-8s  %s  %s  %s: %s  %s",
								body, r.Name, p.ID, path, fields[path], strings.Join(p.Evidence, ", ")))
						}
					}
				}
			}
			// Then the tables: the data exists and has an owner, but no
			// endpoint serves it yet.
			for _, r := range ws.Repos {
				card := cards[r.Name]
				if card == nil {
					continue
				}
				for _, tb := range card.Identity.Tables {
					for _, col := range tb.Columns {
						if !strings.Contains(strings.ToLower(tb.Value+"."+col), term) {
							continue
						}
						lines = append(lines, fmt.Sprintf("  %-8s  %s  %s.%s  %s",
							"table", r.Name, tb.Value, col, strings.Join(tb.Evidence, ", ")))
					}
				}
			}
			if len(lines) == 0 {
				return fmt.Errorf("no catalog card has a field or column matching %q", args[0])
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, args[0])
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
