package release_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/release"
	"github.com/lukascivil/trama/internal/testutil"
)

// clone returns a clone of a fresh remote, and the remote to push commits to.
func clone(t *testing.T) (gitx.Repo, *testutil.Remote) {
	t.Helper()
	remote := testutil.NewRemote(t, "main")
	dir := filepath.Join(t.TempDir(), "clone")
	if err := gitx.Clone(remote.URL, "main", dir); err != nil {
		t.Fatal(err)
	}
	return gitx.Repo{Dir: dir}, remote
}

func tag(t *testing.T, r gitx.Repo, args ...string) {
	t.Helper()
	testutil.Git(t, r.Dir, append([]string{"tag"}, args...)...)
}

func latest(t *testing.T, r gitx.Repo) *release.Release {
	t.Helper()
	rel, err := release.Latest(r, "origin/main")
	if err != nil {
		t.Fatal(err)
	}
	return rel
}

func TestLatestPicksHighestSemver(t *testing.T) {
	r, _ := clone(t)
	for _, name := range []string{"v0.9.0", "v0.10.0", "v0.10.0-rc.1", "start", "release-2024"} {
		tag(t, r, name, "origin/main")
	}
	if rel := latest(t, r); rel == nil || rel.Version != "v0.10.0" {
		t.Fatalf("Latest = %+v, want v0.10.0", rel)
	}
}

func TestLatestNoSemverTag(t *testing.T) {
	r, _ := clone(t)
	tag(t, r, "start", "origin/main")
	if rel := latest(t, r); rel != nil {
		t.Fatalf("Latest = %+v, want nil", rel)
	}
}

func TestLatestIgnoresTagOffMain(t *testing.T) {
	r, _ := clone(t)
	tag(t, r, "v0.1.0", "origin/main")
	testutil.Git(t, r.Dir, "switch", "-q", "-c", "side")
	testutil.Git(t, r.Dir, "commit", "-q", "--allow-empty", "-m", "side")
	tag(t, r, "v9.0.0")
	if rel := latest(t, r); rel == nil || rel.Version != "v0.1.0" {
		t.Fatalf("Latest = %+v, want v0.1.0", rel)
	}
}

func TestLatestCountsCommitsAfter(t *testing.T) {
	r, remote := clone(t)
	tag(t, r, "v0.1.0", "origin/main")
	remote.Commit(t, "a.txt", "a\n")
	remote.Commit(t, "b.txt", "b\n")
	if err := r.Fetch(); err != nil {
		t.Fatal(err)
	}
	if rel := latest(t, r); rel == nil || rel.Version != "v0.1.0" || rel.CommitsAfter != 2 {
		t.Fatalf("Latest = %+v, want v0.1.0 with 2 commits after", rel)
	}
}

func TestLatestDates(t *testing.T) {
	t.Setenv("GIT_COMMITTER_DATE", "2026-01-01T12:00:00Z")
	r, remote := clone(t)
	remote.Commit(t, "a.txt", "a\n")
	if err := r.Fetch(); err != nil {
		t.Fatal(err)
	}
	tag(t, r, "v0.1.0", "origin/main")
	if rel := latest(t, r); rel == nil || !strings.HasPrefix(rel.Date, "2026-01-01") {
		t.Fatalf("lightweight tag: Latest = %+v, want the commit date 2026-01-01", rel)
	}
	t.Setenv("GIT_COMMITTER_DATE", "2026-02-01T12:00:00Z")
	tag(t, r, "-a", "v0.2.0", "-m", "v0.2.0", "origin/main")
	if rel := latest(t, r); rel == nil || rel.Version != "v0.2.0" || !strings.HasPrefix(rel.Date, "2026-02-01") {
		t.Fatalf("annotated tag: Latest = %+v, want v0.2.0 with the tagger date 2026-02-01", rel)
	}
}

func TestLatestMissingRef(t *testing.T) {
	r, _ := clone(t)
	if _, err := release.Latest(r, "origin/nope"); err == nil {
		t.Fatal("Latest with a missing ref: want an error")
	}
}
