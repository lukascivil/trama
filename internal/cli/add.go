package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/workspace"
)

func newAddCmd() *cobra.Command {
	var name, branch string
	var pick, noOpen bool
	var port int
	cmd := &cobra.Command{
		Use:   "add <url> | add --pick [<org>]",
		Short: "Add a repository to workspace.yaml (does not clone it)",
		Long: `Add a repository to workspace.yaml. It does not clone it: run "trama sync".

With --pick, open a page on this machine that lists the repositories of a GitHub
organization (the token comes from gh, GH_TOKEN or GITHUB_TOKEN), with a filter and
checkboxes, and add the ones you choose. The names it added go to stdout, one per line.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if pick {
				return cobra.MaximumNArgs(1)(cmd, args)
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if pick && (cmd.Flags().Changed("name") || cmd.Flags().Changed("branch")) {
				return errors.New("--name and --branch do not work with --pick")
			}
			if !pick && (noOpen || cmd.Flags().Changed("port")) {
				return errors.New("--no-open and --port need --pick")
			}
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			if pick {
				org := ""
				if len(args) == 1 {
					org = args[0]
				}
				return runPick(cmd, ws, org, port, cmd.Flags().Changed("port"), !noOpen)
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			url := workspace.LocalURL(args[0], cwd)
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
	cmd.Flags().BoolVar(&pick, "pick", false, "choose repositories of a GitHub organization in a page on this machine")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "with --pick, do not open the browser")
	cmd.Flags().IntVar(&port, "port", defaultPort, "port for --pick; 0 picks a free one")
	return cmd
}
