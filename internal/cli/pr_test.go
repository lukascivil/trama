package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/testutil"
)

// fakeGH puts a gh on PATH that logs its arguments to the returned file and
// prints a pull request URL.
func fakeGH(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(t.TempDir(), "gh.log")
	script := "#!/bin/sh\necho \"$@\" >> \"$GH_LOG\"\necho https://github.com/example/api/pull/7\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	t.Setenv("GH_LOG", log)
	return log
}

func bodyFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(path, []byte("Part of a trama change.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func commitIn(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "change.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.Git(t, dir, "add", "change.txt")
	testutil.Git(t, dir, "commit", "-q", "-m", "change")
}

func TestPRRefusesMainBranch(t *testing.T) {
	log := fakeGH(t)
	ws, remote := branchWorkspace(t)
	commitIn(t, filepath.Join(ws, "repos", "api"))
	_, err := run(t, "pr", "api", "--title", "T", "--body-file", bodyFile(t), "--dir", ws)
	if err == nil || err.Error() != "refusing to open a pull request from main" {
		t.Fatalf("err = %v", err)
	}
	if _, statErr := os.Stat(log); statErr == nil {
		t.Fatal("gh was called")
	}
	if heads := testutil.Git(t, remote.URL, "for-each-ref", "--format=%(refname)", "refs/heads"); heads != "refs/heads/main" {
		t.Fatalf("remote branches = %q", heads)
	}
}

func TestPRPushesAndOpens(t *testing.T) {
	log := fakeGH(t)
	ws, remote := branchWorkspace(t)
	if _, err := run(t, "branch", "api", "trama/x", "--dir", ws); err != nil {
		t.Fatal(err)
	}
	commitIn(t, filepath.Join(ws, "repos", "api"))
	out, err := run(t, "pr", "api", "--title", "T", "--body-file", bodyFile(t), "--dir", ws)
	if err != nil || out != "https://github.com/example/api/pull/7\n" {
		t.Fatalf("%v\n%q", err, out)
	}
	head, _ := repoOf(ws).RevParse("HEAD")
	if got := testutil.Git(t, remote.URL, "rev-parse", "trama/x"); got != head {
		t.Fatalf("remote trama/x = %s, want %s", got, head)
	}
	logged, _ := os.ReadFile(log)
	if !strings.Contains(string(logged), "pr create --base main --head trama/x --title T --body-file") {
		t.Fatalf("gh args = %q", logged)
	}
}

func TestPRRefusesDirty(t *testing.T) {
	fakeGH(t)
	ws, _ := branchWorkspace(t)
	if _, err := run(t, "branch", "api", "trama/x", "--dir", ws); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "repos", "api", "README.md"), []byte("edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "pr", "api", "--title", "T", "--body-file", bodyFile(t), "--dir", ws); err == nil {
		t.Fatal("pr accepted a dirty repo")
	}
}
