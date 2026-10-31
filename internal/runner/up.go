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
	Started  bool // false when the command could not start (see Err)
	ExitCode int  // -1 when it could not start or was killed by a signal
	Stopped  bool // ended because Up stopped it
	Err      error
}

// Up starts every job at once, each in its own process group with stdin
// closed, and prefixes its output with the repo name padded to the longest
// one. It returns when all jobs ended. When ctx is cancelled, or writing to
// out fails, it sends SIGTERM to every running group, waits up to grace,
// then sends SIGKILL. A job that ends on its own gets its group stopped
// too, so its background children do not outlive it.
func Up(ctx context.Context, jobs []Job, out io.Writer, grace time.Duration) []Result {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	out = &stopOnError{w: out, stop: cancel}
	width := 0
	for _, j := range jobs {
		width = max(width, len(j.Repo))
	}
	var mu sync.Mutex
	var stopping atomic.Bool
	var wg sync.WaitGroup
	results := make([]Result, len(jobs))
	var groupsMu sync.Mutex
	active := map[int]bool{} // process groups of jobs still running

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
		pgid := c.Process.Pid
		groupsMu.Lock()
		active[pgid] = true
		groupsMu.Unlock()
		wg.Add(1)
		go func(i int, c *exec.Cmd, w io.WriteCloser) {
			defer wg.Done()
			err := c.Wait()
			w.Close()
			groupsMu.Lock()
			delete(active, pgid)
			groupsMu.Unlock()
			stopped := stopping.Load()
			if !stopped {
				_ = syscall.Kill(-pgid, syscall.SIGTERM) // background children of a job that ended
			}
			results[i] = Result{Repo: jobs[i].Repo, Started: true, ExitCode: c.ProcessState.ExitCode(), Stopped: stopped, Err: err}
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
		signalGroups := func(sig syscall.Signal) {
			groupsMu.Lock()
			defer groupsMu.Unlock()
			for pgid := range active {
				_ = syscall.Kill(-pgid, sig)
			}
		}
		signalGroups(syscall.SIGTERM)
		select {
		case <-done:
		case <-time.After(grace):
			signalGroups(syscall.SIGKILL)
			<-done
		}
	}
	return results
}

// stopOnError stops Up when its output can no longer be written (a closed
// pipe), so servers never keep running unseen.
type stopOnError struct {
	w    io.Writer
	once sync.Once
	stop func()
}

func (s *stopOnError) Write(p []byte) (int, error) {
	n, err := s.w.Write(p)
	if err != nil {
		s.once.Do(s.stop)
	}
	return n, err
}

// BusyPorts lists "<repo>: port <n> is already in use" for every job whose
// Port is set and taken, and "<a> and <b>: both use port <n>" when two jobs
// declare the same port.
func BusyPorts(jobs []Job) []string {
	var busy []string
	owner := map[int]string{}
	for _, j := range jobs {
		if j.Port == 0 {
			continue
		}
		if first, ok := owner[j.Port]; ok {
			busy = append(busy, fmt.Sprintf("%s and %s: both use port %d", first, j.Repo, j.Port))
			continue
		}
		owner[j.Port] = j.Repo
	}
	for _, j := range jobs {
		if j.Port != 0 && !PortFree(j.Port) {
			busy = append(busy, fmt.Sprintf("%s: port %d is already in use", j.Repo, j.Port))
		}
	}
	return busy
}
