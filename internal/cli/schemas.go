package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/schemafind"
)

func newSchemasCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schemas <repo>",
		Short: "List the files of a repository that declare tables: schema files first, then migrations, then .sql files that create a table",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			name := args[0]
			r, ok := ws.Repo(name)
			if !ok {
				return fmt.Errorf("repo %s is not in workspace.yaml", name)
			}
			files := ws.Files(r)
			if !files.Cloned {
				return fmt.Errorf(`repos/%s is not cloned: run "trama sync %s"`, name, name)
			}
			found, err := schemafind.Scan(files.List, files.Read)
			if err != nil {
				return fmt.Errorf("%s: %w", files.Place, err)
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, name)
			if len(found) == 0 {
				fmt.Fprintln(out, "  no schema files found")
				return nil
			}
			for _, f := range found {
				fmt.Fprintf(out, "  %-9s  %s\n", f.Kind, f.Path)
			}
			return nil
		},
	}
}
