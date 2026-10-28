package gitx_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/testutil"
)

func cloneOf(t *testing.T, remote *testutil.Remote, branch string) gitx.Repo {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "clone")
	if err := gitx.Clone(remote.URL, branch, dir); err != nil {
		t.Fatal(err)
	}
	return gitx.Repo{Dir: dir}
}

func TestCloneAndDefaultBranch(t *testing.T) {
	remote := testutil.NewRemote(t, "master")
	b, err := gitx.DefaultBranch(remote.URL)
	if err != nil || b != "master" {
		t.Fatalf("DefaultBranch = %q, %v; want master", b, err)
	}
	r := cloneOf(t, remote, "master")
	if !r.IsRepo() {
		t.Fatal("clone is not a repo")
	}
	if got, _ := r.CurrentBranch(); got != "master" {
		t.Fatalf("branch = %q, want master", got)
	}
	if o, _ := r.OriginURL(); !gitx.SameRemote(o, remote.URL) {
		t.Fatalf("origin = %q, want %q", o, remote.URL)
	}
}

func TestDefaultBranchFailsFastOnBadURL(t *testing.T) {
	if _, err := gitx.DefaultBranch(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("want error for a missing remote")
	}
}

func TestIsRepoFalseForPlainDir(t *testing.T) {
	if (gitx.Repo{Dir: t.TempDir()}).IsRepo() {
		t.Fatal("plain directory reported as repo")
	}
}

func TestFastForwardMovesHead(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	r := cloneOf(t, remote, "main")
	sha := remote.Commit(t, "a.txt", "a")
	if err := r.Fetch(); err != nil {
		t.Fatal(err)
	}
	moved, err := r.FastForward("main")
	if err != nil || !moved {
		t.Fatalf("FastForward = %v, %v; want true, nil", moved, err)
	}
	if head, _ := r.RevParse("HEAD"); head != sha {
		t.Fatalf("HEAD = %s, want %s", head, sha)
	}
	moved, err = r.FastForward("main")
	if err != nil || moved {
		t.Fatalf("second FastForward = %v, %v; want false, nil", moved, err)
	}
}

func TestFastForwardDiverged(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	r := cloneOf(t, remote, "main")
	if err := os.WriteFile(filepath.Join(r.Dir, "local.txt"), []byte("local"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.Git(t, r.Dir, "add", "local.txt")
	testutil.Git(t, r.Dir, "commit", "-m", "local")
	local, _ := r.RevParse("HEAD")
	remote.Commit(t, "remote.txt", "remote")
	if err := r.Fetch(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.FastForward("main"); !errors.Is(err, gitx.ErrDiverged) {
		t.Fatalf("FastForward err = %v, want ErrDiverged", err)
	}
	if head, _ := r.RevParse("HEAD"); head != local {
		t.Fatalf("HEAD moved to %s, want %s", head, local)
	}
}

func TestIsCleanCountsUntracked(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	r := cloneOf(t, remote, "main")
	if clean, err := r.IsClean(); err != nil || !clean {
		t.Fatalf("fresh clone IsClean = %v, %v", clean, err)
	}
	if err := os.WriteFile(filepath.Join(r.Dir, "new.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if clean, _ := r.IsClean(); clean {
		t.Fatal("untracked file not reported")
	}
}

func TestAheadBehind(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	r := cloneOf(t, remote, "main")
	remote.Commit(t, "a.txt", "a")
	remote.Commit(t, "b.txt", "b")
	if err := r.Fetch(); err != nil {
		t.Fatal(err)
	}
	ahead, behind, err := r.AheadBehind("main", "origin/main")
	if err != nil || ahead != 0 || behind != 2 {
		t.Fatalf("AheadBehind = %d, %d, %v; want 0, 2, nil", ahead, behind, err)
	}
}

func TestCurrentBranchDetached(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	r := cloneOf(t, remote, "main")
	testutil.Git(t, r.Dir, "checkout", "--detach")
	if b, err := r.CurrentBranch(); err != nil || b != "" {
		t.Fatalf("CurrentBranch = %q, %v; want empty", b, err)
	}
}

func TestInitCreatesRepoOnBranch(t *testing.T) {
	dir := t.TempDir()
	if err := gitx.Init(dir, "main"); err != nil {
		t.Fatal(err)
	}
	r := gitx.Repo{Dir: dir}
	if b, _ := r.CurrentBranch(); b != "main" {
		t.Fatalf("branch = %q, want main", b)
	}
}

func TestSameRemote(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"https://github.com/a/b.git", "https://github.com/a/b", true},
		{"https://github.com/A/b/", "https://github.com/a/b.git", true},
		{"https://github.com/a/b", "https://github.com/a/c", false},
	}
	for _, c := range cases {
		if got := gitx.SameRemote(c.a, c.b); got != c.want {
			t.Errorf("SameRemote(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
