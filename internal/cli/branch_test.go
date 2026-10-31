package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/testutil"
)

// branchWorkspace is a workspace with repo "api" cloned from remote.
func branchWorkspace(t *testing.T) (string, *testutil.Remote) {
	t.Helper()
	ws := newWorkspace(t)
	remote := testutil.NewRemote(t, "main")
	for _, args := range [][]string{{"add", remote.URL, "--dir", ws, "--name", "api"}, {"sync", "--dir", ws}} {
		if out, err := run(t, args...); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	return ws, remote
}

func repoOf(ws string) gitx.Repo { return gitx.Repo{Dir: filepath.Join(ws, "repos", "api")} }

func TestBranchCreatesFromOrigin(t *testing.T) {
	ws, remote := branchWorkspace(t)
	sha := remote.Commit(t, "new.txt", "new") // local main is now behind origin
	out, err := run(t, "branch", "api", "trama/x", "--dir", ws)
	if err != nil || !strings.Contains(out, "repos/api is on trama/x, from origin/main "+sha[:7]) {
		t.Fatalf("%v\n%s", err, out)
	}
	if head, _ := repoOf(ws).RevParse("HEAD"); head != sha {
		t.Fatalf("HEAD = %s, want %s", head, sha)
	}
}

func TestBranchRefusesDirty(t *testing.T) {
	ws, _ := branchWorkspace(t)
	readme := filepath.Join(ws, "repos", "api", "README.md")
	if err := os.WriteFile(readme, []byte("local edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := run(t, "branch", "api", "trama/x", "--dir", ws)
	if err == nil || err.Error() != "repos/api has local changes; commit or move them first" {
		t.Fatalf("err = %v", err)
	}
	if b, _ := repoOf(ws).CurrentBranch(); b != "main" {
		t.Fatalf("branch = %q", b)
	}
	if got, _ := os.ReadFile(readme); string(got) != "local edit\n" {
		t.Fatalf("local edit lost: %q", got)
	}
}

func TestBranchRefusesExisting(t *testing.T) {
	ws, _ := branchWorkspace(t)
	if _, err := run(t, "branch", "api", "trama/x", "--dir", ws); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "branch", "api", "--main", "--dir", ws); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "branch", "api", "trama/x", "--dir", ws); err == nil || err.Error() != "branch trama/x already exists in repos/api" {
		t.Fatalf("err = %v", err)
	}
}

func TestBranchBackToMain(t *testing.T) {
	ws, _ := branchWorkspace(t)
	if _, err := run(t, "branch", "api", "trama/x", "--dir", ws); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "branch", "api", "--main", "--dir", ws)
	if err != nil || !strings.Contains(out, "repos/api is back on main") {
		t.Fatalf("%v\n%s", err, out)
	}
	if b, _ := repoOf(ws).CurrentBranch(); b != "main" {
		t.Fatalf("branch = %q", b)
	}
}

func TestBranchRejectsBadName(t *testing.T) {
	ws, _ := branchWorkspace(t)
	if _, err := run(t, "branch", "api", "-x", "--dir", ws); err == nil {
		t.Fatal("bad branch name accepted")
	}
}
