package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/validate"
)

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Check the catalog cards: fields, evidence and proof for every relation",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			r := validate.Check(ws)
			out := cmd.OutOrStdout()
			for _, f := range r.Findings {
				fmt.Fprintln(out, f)
			}
			errs := validate.Errors(r.Findings)
			if errs > 0 {
				return fmt.Errorf("catalog has %d error(s)", errs)
			}
			fmt.Fprintf(out, "ok: %d card(s), %d relation(s), %d warning(s)\n", r.Cards, r.Relations, len(r.Findings))
			return nil
		},
	}
}
