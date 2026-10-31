package cli

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/runner"
	"github.com/lukascivil/trama/internal/status"
	"github.com/lukascivil/trama/internal/workspace"
)

// targets returns the repositories a command acts on: the ones named, or
// every repository of the workspace. explicit tells which.
func targets(ws *workspace.Workspace, args []string) (names []string, explicit bool) {
	if len(args) > 0 {
		seen := map[string]bool{}
		for _, a := range args {
			if !seen[a] {
				seen[a] = true
				names = append(names, a)
			}
		}
		return names, true
	}
	for _, r := range ws.Repos {
		names = append(names, r.Name)
	}
	return names, false
}

// warnStale prints a warning for every named repository whose catalog card
// no longer describes its main branch.
func warnStale(out io.Writer, ws *workspace.Workspace, names []string) {
	for _, s := range status.Collect(ws) {
		if s.Card == status.CardStale && slices.Contains(names, s.Name) {
			fmt.Fprintf(out, "warning: the catalog card of %s is stale\n", s.Name)
		}
	}
}

func exitText(err error) string {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return "killed by a signal (" + ws.Signal().String() + ")"
		}
		return fmt.Sprintf("exit %d", exitErr.ExitCode())
	}
	return err.Error()
}

// runEach runs run.<kind> of each target repository in turn, then prints one
// summary line per repository. With setupFirst, each repository's setup runs
// first when needed. onOK runs after each success. It returns how many
// repositories failed.
func runEach(cmd *cobra.Command, args []string, kind runner.Kind, okWord string, setupFirst bool, onOK func(*workspace.Workspace, runner.Job) error) (int, error) {
	ws, err := loadWorkspace(cmd)
	if err != nil {
		return 0, err
	}
	names, explicit := targets(ws, args)
	out := cmd.OutOrStdout()
	warnStale(out, ws, names)
	var summary []string
	failed := 0
	for _, name := range names {
		j, err := runner.JobFor(ws, name, kind)
		if err != nil {
			if !explicit && errors.Is(err, runner.ErrNoCommand) {
				summary = append(summary, fmt.Sprintf("%s: no run.%s, skipped", name, kind))
				continue
			}
			summary = append(summary, fmt.Sprintf("%s: %v", name, err))
			failed++
			continue
		}
		foreground := func(j runner.Job) error {
			return runner.Run(cmd.Context(), j, cmd.InOrStdin(), out, cmd.ErrOrStderr())
		}
		if setupFirst {
			if err := setupIfNeeded(ws, name, out, foreground); err != nil {
				if errors.Is(err, runner.ErrInterrupted) {
					summary = append(summary, name+": interrupted")
					printLines(out, summary)
					return failed, err
				}
				summary = append(summary, fmt.Sprintf("%s: %v", name, err))
				failed++
				continue
			}
		}
		fmt.Fprintln(out, j.Describe())
		if err := foreground(j); err != nil {
			if errors.Is(err, runner.ErrInterrupted) {
				summary = append(summary, name+": interrupted")
				printLines(out, summary)
				return failed, err
			}
			summary = append(summary, fmt.Sprintf("%s: failed (%s)", name, exitText(err)))
			failed++
			continue
		}
		if onOK != nil {
			if err := onOK(ws, j); err != nil {
				summary = append(summary, fmt.Sprintf("%s: %v", name, err))
				failed++
				continue
			}
		}
		summary = append(summary, fmt.Sprintf("%s: %s", name, okWord))
	}
	printLines(out, summary)
	return failed, nil
}

func printLines(out io.Writer, lines []string) {
	for _, line := range lines {
		fmt.Fprintln(out, line)
	}
}

func newSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup [repo...]",
		Short: "Run run.setup from each repository's catalog card (e.g. npm install)",
		RunE: func(cmd *cobra.Command, args []string) error {
			failed, err := runEach(cmd, args, runner.Setup, "ok", false, runner.MarkSetup)
			if err != nil {
				return err
			}
			if failed > 0 {
				return fmt.Errorf("setup failed for %d repo(s)", failed)
			}
			return nil
		},
	}
}

func newTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test [repo...]",
		Short: "Run run.test from each repository's catalog card, after its setup when needed",
		RunE: func(cmd *cobra.Command, args []string) error {
			failed, err := runEach(cmd, args, runner.Test, "passed", true, nil)
			if err != nil {
				return err
			}
			if failed > 0 {
				return fmt.Errorf("tests failed for %d repo(s)", failed)
			}
			return nil
		},
	}
}
