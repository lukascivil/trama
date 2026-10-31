package cli

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/runner"
)

func newUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up [repo...]",
		Short: "Run every repository with run.dev at once, output prefixed by repository; Ctrl+C stops them all",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			names, explicit := targets(ws, args)
			out := cmd.OutOrStdout()
			var jobs []runner.Job
			var running []string
			for _, name := range names {
				j, err := runner.JobFor(ws, name, runner.Dev)
				if err != nil {
					if explicit {
						return err
					}
					if errors.Is(err, runner.ErrNoCommand) {
						fmt.Fprintf(out, "%s: no run.dev, skipped\n", name)
					} else {
						fmt.Fprintf(out, "%s: %v; skipped\n", name, err)
					}
					continue
				}
				jobs = append(jobs, j)
				running = append(running, name)
			}
			if len(jobs) == 0 {
				return errors.New("nothing to run: no repository has run.dev in its catalog card")
			}
			warnStale(out, ws, running)

			var mu sync.Mutex
			for _, name := range running {
				prefixed := func(j runner.Job) error {
					w := runner.NewPrefixWriter(out, name, &mu)
					defer w.Close()
					return runner.Run(cmd.Context(), j, nil, w, w)
				}
				if err := setupIfNeeded(ws, name, out, prefixed); err != nil {
					return err
				}
			}
			if busy := runner.BusyPorts(jobs); len(busy) > 0 {
				return fmt.Errorf("cannot start: %s", strings.Join(busy, "; "))
			}
			for _, j := range jobs {
				fmt.Fprintln(out, j.Describe())
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			failed := 0
			for _, r := range runner.Up(ctx, jobs, out, 5*time.Second) {
				if r.Stopped {
					continue
				}
				if r.ExitCode == -1 && r.Err != nil {
					fmt.Fprintf(out, "%s could not start: %v\n", r.Repo, r.Err)
					failed++
					continue
				}
				fmt.Fprintf(out, "%s exited with code %d\n", r.Repo, r.ExitCode)
				if r.ExitCode != 0 {
					failed++
				}
			}
			if failed > 0 {
				return fmt.Errorf("%d repo(s) exited with an error", failed)
			}
			return nil
		},
	}
}
