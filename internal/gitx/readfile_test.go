package gitx_test

import (
	"errors"
	"os"
	"path/filepath"
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
