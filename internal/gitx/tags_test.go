package gitx_test

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/testutil"
)

func TestFetchBringsTagOnKnownCommit(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	r := cloneOf(t, remote, "main")
	other := filepath.Join(t.TempDir(), "other")
	testutil.Git(t, filepath.Dir(other), "clone", "-q", remote.URL, other)
	testutil.Git(t, other, "tag", "v1.0.0")
	testutil.Git(t, other, "push", "-q", "origin", "v1.0.0")
	if err := r.Fetch(); err != nil {
		t.Fatal(err)
	}
	tags, err := r.TagsMerged("origin/main")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(tags, func(tg gitx.Tag) bool { return tg.Name == "v1.0.0" && tg.Date != "" }) {
		t.Fatalf("tags = %+v, want v1.0.0 with a date", tags)
	}
}

func TestCountCommits(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	r := cloneOf(t, remote, "main")
	base := testutil.Git(t, r.Dir, "rev-parse", "origin/main")
	remote.Commit(t, "a.txt", "a\n")
	if err := r.Fetch(); err != nil {
		t.Fatal(err)
	}
	if n, err := r.CountCommits(base, "origin/main"); err != nil || n != 1 {
		t.Fatalf("CountCommits = %d, %v; want 1", n, err)
	}
}
