package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/lukascivil/trama/internal/runner"
	"github.com/lukascivil/trama/internal/testutil"
)

// multiWorkspace creates a workspace whose repos are named by the keys of
// cards, each cloned and given the card body (without header lines).
func multiWorkspace(t *testing.T, names []string, cards map[string]string) string {
	t.Helper()
	ws := newWorkspace(t)
	for _, name := range names {
		remote := testutil.NewRemote(t, "main")
		if out, err := run(t, "add", remote.URL, "--dir", ws, "--name", name); err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
	}
	if out, err := run(t, "sync", "--dir", ws); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, name := range names {
		full := "version: 1\nname: " + name + "\ndescription: x\nstack: [sh]\n" + cards[name]
		if err := os.WriteFile(filepath.Join(ws, "catalog", name+".yaml"), []byte(full), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return ws
}

func waitPid(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(path); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil {
				return pid
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no pid in %s", path)
	return 0
}

func waitGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		if time.Now().After(deadline) {
			t.Fatalf("pid %d still running", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestUpStopsOnSIGHUP(t *testing.T) {
	ws := runWorkspace(t, "run:\n  dev: { cmd: \"sleep 60 & echo $! > pid; wait\" }\n")
	done := make(chan error, 1)
	go func() { _, err := run(t, "up", "--dir", ws); done <- err }()
	pid := waitPid(t, filepath.Join(ws, "repos", "api", "pid"))
	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		t.Fatal("up did not stop on SIGHUP")
	}
	waitGone(t, pid)
}

func TestTestStopsOnInterrupt(t *testing.T) {
	ws := multiWorkspace(t, []string{"alpha", "beta"}, map[string]string{
		"alpha": "run:\n  test: { cmd: \"echo $$ > started; sleep 5\" }\n",
		"beta":  "run:\n  test: { cmd: \"touch ran\" }\n",
	})
	done := make(chan error, 1)
	go func() { _, err := run(t, "test", "--dir", ws); done <- err }()
	waitPid(t, filepath.Join(ws, "repos", "alpha", "started"))
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	err := <-done
	if !errors.Is(err, runner.ErrInterrupted) {
		t.Fatalf("err = %v, want ErrInterrupted", err)
	}
	if _, statErr := os.Stat(filepath.Join(ws, "repos", "beta", "ran")); statErr == nil {
		t.Fatal("beta ran after the interrupt")
	}
}

func TestExitCode(t *testing.T) {
	if code, quiet := exitCode(runner.ErrInterrupted); code != 130 || !quiet {
		t.Fatalf("interrupted → %d, quiet %v", code, quiet)
	}
	if code, quiet := exitCode(errors.New("boom")); code != 1 || quiet {
		t.Fatalf("error → %d, quiet %v", code, quiet)
	}
}

func TestUpChecksPortsBeforeSetup(t *testing.T) {
	ws, port := busyPortWorkspace(t, "setup: { cmd: \"touch installed\" }\n")
	_, err := run(t, "up", "--dir", ws)
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("port %d is already in use", port)) {
		t.Fatalf("err = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(ws, "repos", "api", "installed")); statErr == nil {
		t.Fatal("setup ran although up could not start")
	}
}

func TestUpSkipsRepoWhoseSetupFails(t *testing.T) {
	ws := multiWorkspace(t, []string{"alpha", "beta"}, map[string]string{
		"alpha": "run:\n  setup: { cmd: \"exit 3\" }\n  dev: { cmd: \"echo A\" }\n",
		"beta":  "run:\n  dev: { cmd: \"echo B\" }\n",
	})
	out, err := run(t, "up", "--dir", ws)
	if err == nil || !strings.Contains(out, "alpha: setup failed (exit 3); skipped") || !strings.Contains(out, "beta | B") || strings.Contains(out, "alpha | A") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestUpPadsSetupLines(t *testing.T) {
	ws := multiWorkspace(t, []string{"a", "bbb"}, map[string]string{
		"a":   "run:\n  setup: { cmd: \"echo S\" }\n  dev: { cmd: \"true\" }\n",
		"bbb": "run:\n  dev: { cmd: \"true\" }\n",
	})
	out, err := run(t, "up", "--dir", ws)
	if err != nil || !strings.Contains(out, "  a | S\n") {
		t.Fatalf("%v\n%q", err, out)
	}
}

func TestUpReportsSignalDeath(t *testing.T) {
	ws := runWorkspace(t, "run:\n  dev: { cmd: \"echo started; kill -TERM $$\" }\n")
	out, err := run(t, "up", "--dir", ws)
	if err == nil || !strings.Contains(out, "api was killed by a signal (terminated)") || strings.Contains(out, "could not start") {
		t.Fatalf("%v\n%s", err, out)
	}
}
