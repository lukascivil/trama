package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/runner"
	"github.com/lukascivil/trama/internal/workspace"
)

// setupIfNeeded runs run.setup of name when it has not run since HEAD or
// the command last changed, and records it on success. run executes a job.
func setupIfNeeded(ws *workspace.Workspace, name string, out io.Writer, run func(runner.Job) error) error {
	j, err := runner.JobFor(ws, name, runner.Setup)
	if errors.Is(err, runner.ErrNoCommand) {
		return nil
	}
	if err != nil {
		return err
	}
	need, err := runner.NeedsSetup(ws, j)
	if err != nil || !need {
		return err
	}
	fmt.Fprintln(out, j.Describe())
	if err := run(j); err != nil {
		if errors.Is(err, runner.ErrInterrupted) {
			return err
		}
		return &setupError{name: name, reason: exitText(err)}
	}
	return runner.MarkSetup(ws, j)
}

// setupError is a failed setup; up skips the repository, dev and test stop.
type setupError struct{ name, reason string }

func (e *setupError) Error() string { return fmt.Sprintf("setup of %s failed (%s)", e.name, e.reason) }

func newDevCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dev <repo>",
		Short: "Run a repository with run.dev from its catalog card, after its setup when needed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			name := args[0]
			dev, err := runner.JobFor(ws, name, runner.Dev)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			warnStale(out, ws, []string{name})
			foreground := func(j runner.Job) error {
				return runner.Run(cmd.Context(), j, cmd.InOrStdin(), out, cmd.ErrOrStderr())
			}
			if err := setupIfNeeded(ws, name, out, foreground); err != nil {
				return err
			}
			fmt.Fprintln(out, dev.Describe())
			return foreground(dev)
		},
	}
}
