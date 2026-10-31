package gitx_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lukascivil/trama/internal/testutil"
)

func TestSwitchNewAndSwitch(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	r := cloneOf(t, remote, "main")
	if r.BranchExists("trama/x") {
		t.Fatal("branch exists before creation")
	}
	if err := r.SwitchNew("trama/x", "origin/main"); err != nil {
		t.Fatal(err)
	}
	if b, _ := r.CurrentBranch(); b != "trama/x" || !r.BranchExists("trama/x") {
		t.Fatalf("branch = %q", b)
	}
	if err := r.Switch("main"); err != nil {
		t.Fatal(err)
	}
	if b, _ := r.CurrentBranch(); b != "main" {
		t.Fatalf("branch = %q", b)
	}
}

func TestPushCreatesRemoteBranch(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	r := cloneOf(t, remote, "main")
	if err := r.Push(""); err == nil {
		t.Fatal("Push accepted an empty branch")
	}
	if err := r.SwitchNew("trama/x", "origin/main"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.Dir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.Git(t, r.Dir, "add", "f.txt")
	testutil.Git(t, r.Dir, "commit", "-q", "-m", "f")
	if err := r.Push("trama/x"); err != nil {
		t.Fatal(err)
	}
	head, _ := r.RevParse("HEAD")
	if got := testutil.Git(t, remote.URL, "rev-parse", "trama/x"); got != head {
		t.Fatalf("remote trama/x = %s, want %s", got, head)
	}
}
