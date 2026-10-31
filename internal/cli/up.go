package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
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
			if busy := runner.BusyPorts(jobs); len(busy) > 0 {
				return fmt.Errorf("cannot start: %s", strings.Join(busy, "; "))
			}

			width := 0
			for _, name := range running {
				width = max(width, len(name))
			}
			var mu sync.Mutex
			failed := 0
			var ready []runner.Job
			for _, j := range jobs {
				label := fmt.Sprintf("%*s", width, j.Repo)
				prefixed := func(sj runner.Job) error {
					w := runner.NewPrefixWriter(out, label, &mu)
					defer w.Close()
					return runner.Run(cmd.Context(), sj, nil, w, w)
				}
				err := setupIfNeeded(ws, j.Repo, out, prefixed)
				var se *setupError
				switch {
				case err == nil:
					ready = append(ready, j)
				case errors.As(err, &se) && !explicit:
					fmt.Fprintf(out, "%s: setup failed (%s); skipped\n", j.Repo, se.reason)
					failed++
				default:
					return err
				}
			}
			if len(ready) == 0 {
				return fmt.Errorf("nothing to run: setup failed for %d repo(s)", failed)
			}
			for _, j := range ready {
				fmt.Fprintln(out, j.Describe())
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
			defer stop()
			pipe := make(chan os.Signal, 1)
			signal.Notify(pipe, syscall.SIGPIPE) // a closed output becomes a write error, which stops up
			defer signal.Stop(pipe)
			for _, r := range runner.Up(ctx, ready, out, 5*time.Second) {
				var exitErr *exec.ExitError
				switch {
				case r.Stopped:
				case !r.Started:
					fmt.Fprintf(out, "%s could not start: %v\n", r.Repo, r.Err)
					failed++
				case errors.As(r.Err, &exitErr) && exitErr.ExitCode() == -1:
					fmt.Fprintf(out, "%s was %s\n", r.Repo, exitText(r.Err))
					failed++
				default:
					fmt.Fprintf(out, "%s exited with code %d\n", r.Repo, r.ExitCode)
					if r.ExitCode != 0 {
						failed++
					}
				}
			}
			if failed > 0 {
				return fmt.Errorf("%d repo(s) exited with an error", failed)
			}
			return nil
		},
	}
}
