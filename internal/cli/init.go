package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/scaffold"
)

func newInitCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Create a trama workspace in an empty directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := dirFlag(cmd)
			if len(args) == 1 {
				dir = args[0]
			}
			if dir == "" {
				dir = "."
			}
			created, err := scaffold.Init(scaffold.Options{Dir: dir, Name: name})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, p := range created {
				fmt.Fprintf(out, "created %s\n", p)
			}
			fmt.Fprintln(out, "Next: trama add <url>, then trama sync")
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "workspace name (default: the directory name)")
	return cmd
}

// dirFlag returns the value of the global --dir flag.
func dirFlag(cmd *cobra.Command) string {
	if f := cmd.Flag("dir"); f != nil {
		return f.Value.String()
	}
	return ""
}
