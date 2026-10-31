package runner_test

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/runner"
)

func TestRunPassesOutputAndExitCode(t *testing.T) {
	var out bytes.Buffer
	j := runner.Job{Repo: "api", Dir: t.TempDir(), Cmd: "echo hello; exit 3"}
	err := runner.Run(context.Background(), j, nil, &out, &out)
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 3 || out.String() != "hello\n" {
		t.Fatalf("err = %v, output %q", err, out.String())
	}
}

func TestRunUsesRepoDir(t *testing.T) {
	var out bytes.Buffer
	dir := t.TempDir()
	if err := runner.Run(context.Background(), runner.Job{Dir: dir, Cmd: "pwd"}, nil, &out, &out); err != nil {
		t.Fatal(err)
	}
	got, _ := filepath.EvalSymlinks(strings.TrimSpace(out.String()))
	want, _ := filepath.EvalSymlinks(dir)
	if got != want {
		t.Fatalf("pwd = %q, want %q", got, want)
	}
}
