package runner_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/lukascivil/trama/internal/runner"
)

// syncBuffer is a bytes.Buffer safe to read while Up writes to it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestUpPrefixesOutput(t *testing.T) {
	var out syncBuffer
	jobs := []runner.Job{
		{Repo: "api", Dir: t.TempDir(), Cmd: "echo one"},
		{Repo: "web-app", Dir: t.TempDir(), Cmd: "echo two"},
	}
	results := runner.Up(context.Background(), jobs, &out, time.Second)
	got := out.String()
	if !strings.Contains(got, "    api | one\n") || !strings.Contains(got, "web-app | two\n") {
		t.Fatalf("output:\n%s", got)
	}
	for _, r := range results {
		if r.ExitCode != 0 || r.Stopped {
			t.Fatalf("results = %+v", results)
		}
	}
}

func TestUpKeepsOthersWhenOneExits(t *testing.T) {
	var out syncBuffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	jobs := []runner.Job{
		{Repo: "a", Dir: t.TempDir(), Cmd: "exit 4"},
		{Repo: "b", Dir: t.TempDir(), Cmd: "echo up; sleep 30"},
	}
	done := make(chan []runner.Result)
	go func() { done <- runner.Up(ctx, jobs, &out, 2*time.Second) }()
	waitFor(t, "b to start", func() bool { return strings.Contains(out.String(), "b | up") })
	time.Sleep(300 * time.Millisecond) // a has exited; b must still be running
	select {
	case <-done:
		t.Fatal("Up returned while b was still running")
	default:
	}
	cancel()
	results := <-done
	if results[0].ExitCode != 4 || results[0].Stopped {
		t.Fatalf("a = %+v", results[0])
	}
	if !results[1].Stopped {
		t.Fatalf("b = %+v", results[1])
	}
}

func TestUpStopsProcessTrees(t *testing.T) {
	dir := t.TempDir()
	pidfile := filepath.Join(dir, "pid")
	ctx, cancel := context.WithCancel(context.Background())
	job := runner.Job{Repo: "tree", Dir: dir, Cmd: "sh -c 'echo $$ > pid; sleep 60' & wait"}
	done := make(chan []runner.Result)
	var out syncBuffer
	go func() { done <- runner.Up(ctx, []runner.Job{job}, &out, 2*time.Second) }()
	var pid int
	waitFor(t, "the grandchild to write its pid", func() bool {
		b, err := os.ReadFile(pidfile)
		if err != nil {
			return false
		}
		pid, err = strconv.Atoi(strings.TrimSpace(string(b)))
		return err == nil
	})
	start := time.Now()
	cancel()
	select {
	case <-done:
	case <-time.After(7 * time.Second):
		t.Fatal("Up did not return within 7s of cancel")
	}
	t.Logf("stopped in %v", time.Since(start))
	waitFor(t, fmt.Sprintf("pid %d to disappear", pid), func() bool {
		return syscall.Kill(pid, 0) != nil
	})
}

func TestBusyPorts(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	freeL, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	free := freeL.Addr().(*net.TCPAddr).Port
	freeL.Close()
	busyPort := busy.Addr().(*net.TCPAddr).Port
	got := runner.BusyPorts([]runner.Job{{Repo: "api", Port: busyPort}, {Repo: "web", Port: free}, {Repo: "lib"}})
	want := fmt.Sprintf("api: port %d is already in use", busyPort)
	if len(got) != 1 || got[0] != want {
		t.Fatalf("BusyPorts = %q", got)
	}
}
