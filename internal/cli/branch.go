package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/workspace"
)

// cleanClone returns the clone of name and its main branch, failing unless
// the clone exists and has no local changes.
func cleanClone(ws *workspace.Workspace, name string) (gitx.Repo, string, error) {
	r, ok := ws.Repo(name)
	if !ok {
		return gitx.Repo{}, "", fmt.Errorf("repo %s is not in workspace.yaml", name)
	}
	repo := gitx.Repo{Dir: ws.RepoDir(name)}
	if _, err := os.Stat(repo.Dir); err != nil || !repo.IsRepo() {
		return gitx.Repo{}, "", fmt.Errorf(`repos/%s is not cloned: run "trama sync %s"`, name, name)
	}
	clean, err := repo.IsClean()
	if err != nil {
		return gitx.Repo{}, "", err
	}
	if !clean {
		return gitx.Repo{}, "", fmt.Errorf("repos/%s has local changes; commit or move them first", name)
	}
	return repo, r.Branch, nil
}

func newBranchCmd() *cobra.Command {
	var backToMain bool
	cmd := &cobra.Command{
		Use:   "branch <repo> [<name> | --main]",
		Short: "Create a branch from the up-to-date main branch of a repository, or go back to main",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			name := args[0]
			out := cmd.OutOrStdout()
			if backToMain {
				if len(args) != 1 {
					return errors.New("use either a branch name or --main")
				}
				repo, main, err := cleanClone(ws, name)
				if err != nil {
					return err
				}
				if err := repo.Switch(main); err != nil {
					return err
				}
				fmt.Fprintf(out, "repos/%s is back on %s\n", name, main)
				return nil
			}
			if len(args) != 2 {
				return errors.New("give the new branch name, or --main")
			}
			branch := args[1]
			if err := workspace.ValidBranch(branch); err != nil {
				return err
			}
			repo, main, err := cleanClone(ws, name)
			if err != nil {
				return err
			}
			current, err := repo.CurrentBranch()
			if err != nil {
				return err
			}
			if current != main {
				if current == "" {
					current = "a detached HEAD"
				}
				return fmt.Errorf("repos/%s is on %s; switch it to %s first", name, current, main)
			}
			if repo.BranchExists(branch) {
				return fmt.Errorf("branch %s already exists in repos/%s", branch, name)
			}
			if err := repo.Fetch(); err != nil {
				return err
			}
			start := "origin/" + main
			if err := repo.SwitchNew(branch, start); err != nil {
				return err
			}
			sha, _ := repo.RevParse("HEAD")
			fmt.Fprintf(out, "repos/%s is on %s, from %s %s\n", name, branch, start, short(sha))
			return nil
		},
	}
	cmd.Flags().BoolVar(&backToMain, "main", false, "switch the repository back to its main branch")
	return cmd
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
