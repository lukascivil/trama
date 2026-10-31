package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newPRCmd() *cobra.Command {
	var title, bodyFile string
	var draft bool
	cmd := &cobra.Command{
		Use:   "pr <repo> --title <title> --body-file <file>",
		Short: "Push the current branch of a repository and open a pull request (never from its main branch)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			repo, main, err := cleanClone(ws, args[0])
			if err != nil {
				return err
			}
			branch, err := repo.CurrentBranch()
			if err != nil {
				return err
			}
			switch branch {
			case "":
				return errors.New("refusing to open a pull request from a detached HEAD")
			case main:
				return fmt.Errorf("refusing to open a pull request from %s", main)
			}
			body, err := filepath.Abs(bodyFile)
			if err != nil {
				return err
			}
			if err := repo.Push(branch); err != nil {
				return err
			}
			ghArgs := []string{"pr", "create", "--base", main, "--head", branch, "--title", title, "--body-file", body}
			if draft {
				ghArgs = append(ghArgs, "--draft")
			}
			gh := exec.Command("gh", ghArgs...)
			gh.Dir = repo.Dir
			gh.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GH_PROMPT_DISABLED=1")
			var stdout, stderr bytes.Buffer
			gh.Stdout = &stdout
			gh.Stderr = &stderr
			if err := gh.Run(); err != nil {
				return fmt.Errorf("gh pr create: %s", strings.TrimSpace(stderr.String()+" "+err.Error()))
			}
			lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
			fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(lines[len(lines)-1]))
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "pull request title")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "file with the pull request body")
	cmd.Flags().BoolVar(&draft, "draft", false, "open the pull request as a draft")
	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("body-file")
	return cmd
}
