package gitx_test

import (
	"errors"
	"fmt"
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

func TestReadFilesBatch(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	remote.Commit(t, "a.txt", "alpha\n")
	remote.Commit(t, "db/init.sql", "CREATE TABLE x ();\n")
	r := cloneOf(t, remote, "main")
	got, err := r.ReadFiles("origin/main", []string{"db/init.sql", "a.txt"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]byte{"a.txt": []byte("alpha\n"), "db/init.sql": []byte("CREATE TABLE x ();\n")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadFiles = %q, want %q", got, want)
	}
}

func TestReadFilesManyLargeBlobs(t *testing.T) {
	// Far more output than a pipe buffer holds: reading must not wait on
	// writing, or the batch deadlocks and this test times out.
	remote := testutil.NewRemote(t, "main")
	r := cloneOf(t, remote, "main")
	body := strings.Repeat("x", 2048)
	var paths []string
	for i := range 200 {
		p := fmt.Sprintf("models/m%03d.sql", i)
		full := filepath.Join(r.Dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(fmt.Sprintf("%d %s\n", i, body)), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	testutil.Git(t, r.Dir, "add", "models")
	testutil.Git(t, r.Dir, "commit", "-q", "-m", "models")
	got, err := r.ReadFiles("HEAD", paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 200 || string(got["models/m137.sql"]) != fmt.Sprintf("137 %s\n", body) {
		t.Fatalf("read %d files; m137 = %.20q", len(got), got["models/m137.sql"])
	}
}

func TestReadFilesRefusesWhatReadFileRefuses(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	r := cloneOf(t, remote, "main")
	if err := os.Symlink("/etc/hosts", filepath.Join(r.Dir, "lnk")); err != nil {
		t.Fatal(err)
	}
	testutil.Git(t, r.Dir, "add", "lnk")
	testutil.Git(t, r.Dir, "commit", "-q", "-m", "link")
	if _, err := r.ReadFiles("HEAD", []string{"README.md", "lnk"}); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("symlink: err = %v", err)
	}
	if _, err := r.ReadFiles("HEAD", []string{"missing.sql"}); !errors.Is(err, gitx.ErrNotExist) {
		t.Fatalf("missing: err = %v", err)
	}
}
