package syncer_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/syncer"
	"github.com/lukascivil/trama/internal/testutil"
	"github.com/lukascivil/trama/internal/workspace"
)

// newWorkspace writes a workspace.yaml listing repos and loads it.
func newWorkspace(t *testing.T, repos ...workspace.Repo) *workspace.Workspace {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, workspace.FileName), []byte("version: 1\nname: test\nrepos: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range repos {
		if err := ws.AddRepo(r); err != nil {
			t.Fatal(err)
		}
	}
	return ws
}

func repoFor(name string, remote *testutil.Remote, branch string) workspace.Repo {
	return workspace.Repo{Name: name, URL: remote.URL, Branch: branch}
}

func outcomes(results []syncer.Result) []syncer.Outcome {
	var out []syncer.Outcome
	for _, r := range results {
		out = append(out, r.Outcome)
	}
	return out
}

func mustSync(t *testing.T, ws *workspace.Workspace, only ...string) []syncer.Result {
	t.Helper()
	results, err := syncer.Sync(ws, only)
	if err != nil {
		t.Fatal(err)
	}
	return results
}

func head(t *testing.T, dir string) string {
	t.Helper()
	sha, err := gitx.Repo{Dir: dir}.RevParse("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return sha
}

func TestSyncClonesMissing(t *testing.T) {
	a := testutil.NewRemote(t, "main")
	b := testutil.NewRemote(t, "master")
	ws := newWorkspace(t, repoFor("a", a, "main"), repoFor("b", b, "master"))
	got := outcomes(mustSync(t, ws))
	if want := []syncer.Outcome{syncer.Cloned, syncer.Cloned}; !reflect.DeepEqual(got, want) {
		t.Fatalf("outcomes = %v, want %v", got, want)
	}
	if br, _ := (gitx.Repo{Dir: ws.RepoDir("b")}).CurrentBranch(); br != "master" {
		t.Fatalf("b is on %q, want master", br)
	}
}

func TestSyncUpdatesCleanRepo(t *testing.T) {
	a := testutil.NewRemote(t, "main")
	ws := newWorkspace(t, repoFor("a", a, "main"))
	mustSync(t, ws)
	sha := a.Commit(t, "new.txt", "new")
	if got := mustSync(t, ws)[0]; got.Outcome != syncer.Updated {
		t.Fatalf("result = %+v, want updated", got)
	}
	if h := head(t, ws.RepoDir("a")); h != sha {
		t.Fatalf("HEAD = %s, want %s", h, sha)
	}
	if got := mustSync(t, ws)[0]; got.Outcome != syncer.UpToDate {
		t.Fatalf("result = %+v, want up-to-date", got)
	}
}

func TestSyncNeverTouchesLocalChanges(t *testing.T) {
	a := testutil.NewRemote(t, "main")
	ws := newWorkspace(t, repoFor("a", a, "main"))
	mustSync(t, ws)
	dir := ws.RepoDir("a")
	before := head(t, dir)
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("my local edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha := a.Commit(t, "new.txt", "new")
	if got := mustSync(t, ws)[0]; got.Outcome != syncer.SkippedDirty {
		t.Fatalf("result = %+v, want skipped-dirty", got)
	}
	if b, _ := os.ReadFile(readme); string(b) != "my local edit\n" {
		t.Fatalf("local edit lost: %q", b)
	}
	if h := head(t, dir); h != before {
		t.Fatalf("HEAD moved to %s", h)
	}
	if o, _ := (gitx.Repo{Dir: dir}).RevParse("origin/main"); o != sha {
		t.Fatalf("origin/main = %s, want fetched %s", o, sha)
	}
}

func TestSyncSkipsOtherBranch(t *testing.T) {
	a := testutil.NewRemote(t, "main")
	ws := newWorkspace(t, repoFor("a", a, "main"))
	mustSync(t, ws)
	dir := ws.RepoDir("a")
	testutil.Git(t, dir, "checkout", "-q", "-b", "feature")
	sha := a.Commit(t, "new.txt", "new")
	if got := mustSync(t, ws)[0]; got.Outcome != syncer.SkippedBranch {
		t.Fatalf("result = %+v, want skipped-branch", got)
	}
	r := gitx.Repo{Dir: dir}
	if br, _ := r.CurrentBranch(); br != "feature" {
		t.Fatalf("branch = %q, want feature", br)
	}
	if o, _ := r.RevParse("origin/main"); o != sha {
		t.Fatalf("origin/main = %s, want fetched %s", o, sha)
	}
}

func TestSyncDiverged(t *testing.T) {
	a := testutil.NewRemote(t, "main")
	ws := newWorkspace(t, repoFor("a", a, "main"))
	mustSync(t, ws)
	dir := ws.RepoDir("a")
	if err := os.WriteFile(filepath.Join(dir, "local.txt"), []byte("l"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.Git(t, dir, "add", "local.txt")
	testutil.Git(t, dir, "commit", "-q", "-m", "local")
	local := head(t, dir)
	a.Commit(t, "remote.txt", "r")
	if got := mustSync(t, ws)[0]; got.Outcome != syncer.SkippedDiverged {
		t.Fatalf("result = %+v, want skipped-diverged", got)
	}
	if h := head(t, dir); h != local {
		t.Fatalf("HEAD = %s, want local commit %s", h, local)
	}
}

func TestSyncLeavesNonRepoDirAlone(t *testing.T) {
	a := testutil.NewRemote(t, "main")
	ws := newWorkspace(t, repoFor("a", a, "main"))
	dir := ws.RepoDir("a")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := mustSync(t, ws)[0]; got.Outcome != syncer.SkippedNotRepo {
		t.Fatalf("result = %+v, want skipped-not-a-repo", got)
	}
	if b, _ := os.ReadFile(file); string(b) != "keep" {
		t.Fatalf("file changed: %q", b)
	}
}

func TestSyncOriginMismatch(t *testing.T) {
	a := testutil.NewRemote(t, "main")
	b := testutil.NewRemote(t, "main")
	ws := newWorkspace(t, repoFor("x", b, "main"))
	if err := gitx.Clone(a.URL, "main", ws.RepoDir("x")); err != nil {
		t.Fatal(err)
	}
	if got := mustSync(t, ws)[0]; got.Outcome != syncer.SkippedOrigin {
		t.Fatalf("result = %+v, want skipped-origin-mismatch", got)
	}
}

func TestSyncContinuesAfterFailure(t *testing.T) {
	good := testutil.NewRemote(t, "main")
	bad := workspace.Repo{Name: "bad", URL: filepath.Join(t.TempDir(), "missing"), Branch: "main"}
	ws := newWorkspace(t, bad, repoFor("good", good, "main"))
	results := mustSync(t, ws)
	if got, want := outcomes(results), []syncer.Outcome{syncer.Failed, syncer.Cloned}; !reflect.DeepEqual(got, want) {
		t.Fatalf("outcomes = %v, want %v", got, want)
	}
	if results[0].Detail == "" {
		t.Fatal("failed result has no detail")
	}
}

func TestSyncOnlyFilter(t *testing.T) {
	a := testutil.NewRemote(t, "main")
	b := testutil.NewRemote(t, "main")
	ws := newWorkspace(t, repoFor("a", a, "main"), repoFor("b", b, "main"))
	results := mustSync(t, ws, "b")
	if len(results) != 1 || results[0].Repo != "b" {
		t.Fatalf("results = %+v", results)
	}
	if _, err := os.Stat(ws.RepoDir("a")); !os.IsNotExist(err) {
		t.Fatal("a was cloned although only b was asked")
	}
	if _, err := syncer.Sync(ws, []string{"zzz"}); err == nil {
		t.Fatal("unknown repo accepted")
	}
}
