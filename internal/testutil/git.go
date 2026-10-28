// Package testutil holds helpers for tests that need real git repositories.
package testutil

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Git runs git in dir with a throwaway identity and returns trimmed stdout.
// It fails the test when git fails.
func Git(t testing.TB, dir string, args ...string) string {
	t.Helper()
	full := append([]string{
		"-c", "user.name=trama-test",
		"-c", "user.email=test@example.invalid",
		"-c", "commit.gpgsign=false",
		"-c", "init.defaultBranch=main",
	}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(stdout.String())
}

// Remote is a bare repository that tests clone from.
type Remote struct {
	URL    string // path of the bare repository
	branch string
	work   string // working clone used to create commits
}

// NewRemote creates a bare repository whose branch has one commit (README.md).
func NewRemote(t testing.TB, branch string) *Remote {
	t.Helper()
	base := t.TempDir()
	r := &Remote{
		URL:    filepath.Join(base, "remote.git"),
		branch: branch,
		work:   filepath.Join(base, "work"),
	}
	Git(t, base, "init", "--bare", "-b", branch, r.URL)
	Git(t, base, "init", "-b", branch, r.work)
	Git(t, r.work, "remote", "add", "origin", r.URL)
	r.Commit(t, "README.md", "# remote\n")
	return r
}

// Commit writes path with content, commits it, pushes it to the remote
// branch and returns the new commit SHA.
func (r *Remote) Commit(t testing.TB, path, content string) string {
	t.Helper()
	full := filepath.Join(r.work, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	Git(t, r.work, "add", path)
	Git(t, r.work, "commit", "-m", "update "+path)
	Git(t, r.work, "push", "-q", "origin", r.branch)
	return Git(t, r.work, "rev-parse", "HEAD")
}
