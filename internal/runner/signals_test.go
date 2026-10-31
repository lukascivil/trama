package runner_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/lukascivil/trama/internal/runner"
)

// readPid waits for path to hold a pid and returns it.
func readPid(t *testing.T, path string) int {
	t.Helper()
	var pid int
	waitFor(t, "pid in "+filepath.Base(path), func() bool {
		b, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		pid, err = strconv.Atoi(strings.TrimSpace(string(b)))
		return err == nil
	})
	return pid
}

func gone(pid int) func() bool { return func() bool { return syscall.Kill(pid, 0) != nil } }

func runAndSignal(t *testing.T, sig syscall.Signal) {
	t.Helper()
	dir := t.TempDir()
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(context.Background(), runner.Job{Repo: "api", Dir: dir, Cmd: "sleep 30 & echo $! > pid; wait"}, nil, io.Discard, io.Discard)
	}()
	pid := readPid(t, filepath.Join(dir, "pid"))
	if err := syscall.Kill(os.Getpid(), sig); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, runner.ErrInterrupted) {
			t.Fatalf("err = %v, want ErrInterrupted", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("Run did not return within 3s of %v", sig)
	}
	waitFor(t, "the grandchild to stop", gone(pid))
}

func TestRunForwardsSIGTERM(t *testing.T) { runAndSignal(t, syscall.SIGTERM) }
func TestRunForwardsSIGINT(t *testing.T)  { runAndSignal(t, syscall.SIGINT) }

func TestUpKillsBackgroundChildrenOfExitedJob(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	jobs := []runner.Job{
		{Repo: "a", Dir: dir, Cmd: "sleep 43 & echo $! > bg; exit 0"},
		{Repo: "b", Dir: t.TempDir(), Cmd: "sleep 30"},
	}
	var out syncBuffer
	done := make(chan []runner.Result)
	go func() { done <- runner.Up(ctx, jobs, &out, time.Second) }()
	pid := readPid(t, filepath.Join(dir, "bg"))
	waitFor(t, "the background child of the exited job to stop", gone(pid))
	cancel()
	<-done
}

func TestUpEscalatesToSIGKILL(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	job := runner.Job{Repo: "stubborn", Dir: dir, Cmd: `trap "" TERM; echo $$ > pid; while :; do sleep 0.1; done`}
	var out syncBuffer
	done := make(chan []runner.Result)
	go func() { done <- runner.Up(ctx, []runner.Job{job}, &out, 500*time.Millisecond) }()
	pid := readPid(t, filepath.Join(dir, "pid"))
	cancel()
	select {
	case <-done:
	case <-time.After(4 * time.Second):
		t.Fatal("Up did not return after the grace period")
	}
	waitFor(t, "the job that ignores SIGTERM to be killed", gone(pid))
}

func TestNeedsSetupWhenToolsChange(t *testing.T) {
	ws, _ := newRepoWS(t, runCard, true)
	j := runner.Job{Repo: "api", Kind: runner.Setup, Dir: ws.RepoDir("api"), Cmd: "npm install", Tools: map[string]string{"node": "22"}}
	if err := runner.MarkSetup(ws, j); err != nil {
		t.Fatal(err)
	}
	j.Tools = map[string]string{"node": "24"}
	if need, err := runner.NeedsSetup(ws, j); err != nil || !need {
		t.Fatalf("after tools change NeedsSetup = %v, %v", need, err)
	}
}

func TestDescribeShowsToolsAndEscapes(t *testing.T) {
	j := runner.Job{Repo: "web", Cmd: "npm run dev\x1b[2K\r", Tools: map[string]string{"node": "24"}}
	want := `$ npm run dev\x1b[2K\x0d  (in repos/web, with node@24 via mise)`
	if got := j.Describe(); got != want {
		t.Fatalf("Describe = %q\nwant       %q", got, want)
	}
}

func TestJobForRejectsBadTools(t *testing.T) {
	ws, _ := newRepoWS(t, strings.Replace(runCard, `node: "24"`, `"npm:evil": "1"`, 1), true)
	if _, err := runner.JobFor(ws, "api", runner.Dev); err == nil || !strings.Contains(err.Error(), "tools.npm:evil") {
		t.Fatalf("err = %v", err)
	}
}
