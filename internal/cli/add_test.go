package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/testutil"
	"github.com/lukascivil/trama/internal/workspace"
)

// newWorkspace runs "trama init" in a fresh directory and returns it.
func newWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if out, err := run(t, "init", dir); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	return dir
}

func TestAddDetectsDefaultBranch(t *testing.T) {
	ws := newWorkspace(t)
	remote := testutil.NewRemote(t, "develop")
	out, err := run(t, "add", remote.URL, "--dir", ws)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	w, err := workspace.Load(ws)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := w.Repo("remote")
	if !ok || r.Branch != "develop" || r.URL != remote.URL {
		t.Fatalf("repo = %+v, %v", r, ok)
	}
	if !strings.Contains(out, "added remote") || !strings.Contains(out, `run "trama sync"`) {
		t.Fatalf("output:\n%s", out)
	}
}

func TestAddUsesNameAndBranchFlags(t *testing.T) {
	ws := newWorkspace(t)
	remote := testutil.NewRemote(t, "main")
	if out, err := run(t, "add", remote.URL, "--dir", ws, "--name", "api", "--branch", "release"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	w, _ := workspace.Load(ws)
	if r, ok := w.Repo("api"); !ok || r.Branch != "release" {
		t.Fatalf("repo = %+v, %v", r, ok)
	}
}

func TestAddFailsOnUnreachableURL(t *testing.T) {
	ws := newWorkspace(t)
	before, _ := os.ReadFile(filepath.Join(ws, workspace.FileName))
	_, err := run(t, "add", filepath.Join(t.TempDir(), "missing"), "--dir", ws)
	if err == nil || !strings.Contains(err.Error(), "could not read the default branch") {
		t.Fatalf("err = %v", err)
	}
	after, _ := os.ReadFile(filepath.Join(ws, workspace.FileName))
	if !bytes.Equal(before, after) {
		t.Fatalf("workspace.yaml changed:\n%s", after)
	}
}

func TestAddRejectsDuplicate(t *testing.T) {
	ws := newWorkspace(t)
	remote := testutil.NewRemote(t, "main")
	if out, err := run(t, "add", remote.URL, "--dir", ws); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if _, err := run(t, "add", remote.URL, "--dir", ws, "--name", "other"); err == nil {
		t.Fatal("same URL added twice")
	}
}

func TestAddFromSubdirectory(t *testing.T) {
	ws := newWorkspace(t)
	remote := testutil.NewRemote(t, "main")
	if out, err := run(t, "add", remote.URL, "--dir", filepath.Join(ws, "catalog")); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	w, _ := workspace.Load(ws)
	if _, ok := w.Repo("remote"); !ok {
		t.Fatal("repo not added to the workspace root")
	}
}

func TestAddOutsideWorkspace(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	_, err := run(t, "add", remote.URL, "--dir", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "not inside a trama workspace") {
		t.Fatalf("err = %v", err)
	}
}
