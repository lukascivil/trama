package runner

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Result is how one job of Up ended.
type Result struct {
	Repo     string
	ExitCode int  // -1 when it could not start or was killed by a signal
	Stopped  bool // ended because Up stopped it
	Err      error
}

// Up starts every job at once, each in its own process group with stdin
// closed, and prefixes its output with the repo name padded to the longest
// one. It returns when all jobs ended. When ctx is cancelled it sends
// SIGTERM to every group, waits up to grace, then sends SIGKILL.
func Up(ctx context.Context, jobs []Job, out io.Writer, grace time.Duration) []Result {
	width := 0
	for _, j := range jobs {
		width = max(width, len(j.Repo))
	}
	var mu sync.Mutex
	var stopping atomic.Bool
	var wg sync.WaitGroup
	results := make([]Result, len(jobs))
	var groups []int

	for i, j := range jobs {
		w := NewPrefixWriter(out, fmt.Sprintf("%*s", width, j.Repo), &mu)
		args := j.Args()
		c := exec.Command(args[0], args[1:]...)
		c.Dir = j.Dir
		c.Stdout = w
		c.Stderr = w
		c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		c.WaitDelay = grace
		if err := c.Start(); err != nil {
			fmt.Fprintf(w, "could not start: %v\n", err)
			w.Close()
			results[i] = Result{Repo: j.Repo, ExitCode: -1, Err: err}
			continue
		}
		groups = append(groups, c.Process.Pid)
		wg.Add(1)
		go func(i int, c *exec.Cmd, w io.WriteCloser) {
			defer wg.Done()
			err := c.Wait()
			w.Close()
			results[i] = Result{Repo: jobs[i].Repo, ExitCode: c.ProcessState.ExitCode(), Stopped: stopping.Load(), Err: err}
		}(i, c, w)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		stopping.Store(true)
		signalGroups(groups, syscall.SIGTERM)
		select {
		case <-done:
		case <-time.After(grace):
			signalGroups(groups, syscall.SIGKILL)
			<-done
		}
	}
	return results
}

func signalGroups(pids []int, sig syscall.Signal) {
	for _, pid := range pids {
		_ = syscall.Kill(-pid, sig)
	}
}

// BusyPorts lists "<repo>: port <n> is already in use" for every job whose
// Port is set and taken.
func BusyPorts(jobs []Job) []string {
	var busy []string
	for _, j := range jobs {
		if j.Port != 0 && !PortFree(j.Port) {
			busy = append(busy, fmt.Sprintf("%s: port %d is already in use", j.Repo, j.Port))
		}
	}
	return busy
}
