package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/workspace"
)

func newAddCmd() *cobra.Command {
	var name, branch string
	cmd := &cobra.Command{
		Use:   "add <url>",
		Short: "Add a repository to workspace.yaml (does not clone it)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			nameGiven := name != ""
			if !nameGiven {
				name = workspace.NameFromURL(url)
			}
			if err := ws.CheckNew(name, url); err != nil {
				if !nameGiven && errors.Is(err, workspace.ErrInvalidName) {
					return fmt.Errorf("%w; choose a name with --name", err)
				}
				return err
			}
			if branch == "" {
				if branch, err = gitx.DefaultBranch(url); err != nil {
					return fmt.Errorf("could not read the default branch of %s: %w", url, err)
				}
			}
			if err := ws.AddRepo(workspace.Repo{Name: name, URL: url, Branch: branch}); err != nil {
				return err
			}
			if err := ws.Save(); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "added %s (%s, branch %s)\n", name, url, branch)
			fmt.Fprintln(out, `run "trama sync" to clone it`)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "repository name (default: derived from the URL)")
	cmd.Flags().StringVar(&branch, "branch", "", "main branch (default: the remote's default branch)")
	return cmd
}
