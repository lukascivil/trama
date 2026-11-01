package gitx_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/testutil"
)

func TestReadFileAtRevision(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	remote.Commit(t, "a/b.txt", "\n\nthird\n")
	r := cloneOf(t, remote, "main")
	got, err := r.ReadFile("origin/main", "a/b.txt")
	if err != nil || string(got) != "\n\nthird\n" {
		t.Fatalf("ReadFile = %q, %v", got, err)
	}
	if _, err := r.ReadFile("origin/main", "missing.txt"); !errors.Is(err, gitx.ErrNotExist) || err.Error() != "missing.txt does not exist at origin/main" {
		t.Fatalf("missing file: err = %v", err)
	}
}

func TestReadFileRejectsSymlink(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	r := cloneOf(t, remote, "main")
	if err := os.Symlink("/etc/hosts", filepath.Join(r.Dir, "lnk")); err != nil {
		t.Fatal(err)
	}
	testutil.Git(t, r.Dir, "add", "lnk")
	testutil.Git(t, r.Dir, "commit", "-q", "-m", "link")
	if _, err := r.ReadFile("HEAD", "lnk"); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("symlink: err = %v", err)
	}
}

func TestListFilesOnlyRegular(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	remote.Commit(t, "db/schema.rb", "x\n")
	r := cloneOf(t, remote, "main")
	if err := os.WriteFile(filepath.Join(r.Dir, "run.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/hosts", filepath.Join(r.Dir, "lnk")); err != nil {
		t.Fatal(err)
	}
	head := testutil.Git(t, r.Dir, "rev-parse", "HEAD")
	testutil.Git(t, r.Dir, "add", "run.sh", "lnk")
	// A gitlink is how a submodule is committed: an entry of type commit.
	testutil.Git(t, r.Dir, "update-index", "--add", "--cacheinfo", "160000,"+head+",sub")
	testutil.Git(t, r.Dir, "commit", "-q", "-m", "mixed")
	got, err := r.ListFiles("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"README.md", "db/schema.rb", "run.sh"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListFiles = %v, want %v (no symlink, no submodule)", got, want)
	}
}
