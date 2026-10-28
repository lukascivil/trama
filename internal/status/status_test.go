package status_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/status"
	"github.com/lukascivil/trama/internal/syncer"
	"github.com/lukascivil/trama/internal/testutil"
	"github.com/lukascivil/trama/internal/workspace"
)

type fixture struct {
	ws     *workspace.Workspace
	remote *testutil.Remote
}

// newFixture creates a workspace with one repo "api" backed by a remote.
// When synced is true the repo is cloned.
func newFixture(t *testing.T, synced bool) fixture {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, workspace.FileName), []byte("version: 1\nname: test\nrepos: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	remote := testutil.NewRemote(t, "main")
	if err := ws.AddRepo(workspace.Repo{Name: "api", URL: remote.URL, Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	f := fixture{ws: ws, remote: remote}
	if synced {
		f.sync(t)
	}
	return f
}

func (f fixture) sync(t *testing.T) {
	t.Helper()
	if _, err := syncer.Sync(f.ws, nil); err != nil {
		t.Fatal(err)
	}
}

func (f fixture) writeCard(t *testing.T, sha string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(f.ws.Root, "catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	card := "version: 1\nname: api\nmapped_at_commit: " + sha + "\n"
	if err := os.WriteFile(catalog.Path(f.ws.Root, "api"), []byte(card), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (f fixture) originMain(t *testing.T) string {
	t.Helper()
	sha, err := gitx.Repo{Dir: f.ws.RepoDir("api")}.RevParse("origin/main")
	if err != nil {
		t.Fatal(err)
	}
	return sha
}

func only(t *testing.T, ws *workspace.Workspace) status.RepoStatus {
	t.Helper()
	sts := status.Collect(ws)
	if len(sts) != 1 {
		t.Fatalf("got %d statuses", len(sts))
	}
	return sts[0]
}

func TestStatusNotCloned(t *testing.T) {
	f := newFixture(t, false)
	s := only(t, f.ws)
	if s.Cloned {
		t.Fatal("reported as cloned")
	}
	if p := s.Problems(); len(p) == 0 || p[0] != `not cloned: run "trama sync"` {
		t.Fatalf("problems = %q", p)
	}
}

func TestStatusNotClonedWithCardIsUnknown(t *testing.T) {
	f := newFixture(t, false)
	f.writeCard(t, "abc1234")
	if s := only(t, f.ws); s.Card != status.CardUnknown {
		t.Fatalf("card = %q, want unknown", s.Card)
	}
}

func TestStatusCardMissing(t *testing.T) {
	f := newFixture(t, true)
	s := only(t, f.ws)
	if s.Card != status.CardMissing {
		t.Fatalf("card = %q, want missing", s.Card)
	}
	if p := s.Problems(); len(p) != 1 || p[0] != `no catalog card: run "/trama:map api"` {
		t.Fatalf("problems = %q", p)
	}
}

func TestStatusCardCurrent(t *testing.T) {
	f := newFixture(t, true)
	f.writeCard(t, f.originMain(t))
	s := only(t, f.ws)
	if s.Card != status.CardCurrent || !s.Cloned || s.Branch != "main" || s.Main != "main" || !s.Clean {
		t.Fatalf("status = %+v", s)
	}
	if p := s.Problems(); len(p) != 0 {
		t.Fatalf("problems = %q", p)
	}
}

func TestStatusCardCurrentWithShortSHA(t *testing.T) {
	f := newFixture(t, true)
	f.writeCard(t, f.originMain(t)[:7])
	if s := only(t, f.ws); s.Card != status.CardCurrent {
		t.Fatalf("card = %q, want current", s.Card)
	}
}

func TestStatusCardStaleAfterRemoteMoves(t *testing.T) {
	f := newFixture(t, true)
	f.writeCard(t, f.originMain(t))
	f.remote.Commit(t, "new.txt", "new")
	f.sync(t)
	s := only(t, f.ws)
	if s.Card != status.CardStale {
		t.Fatalf("card = %q, want stale", s.Card)
	}
	if p := s.Problems(); len(p) != 1 || p[0] != `catalog card is stale: run "/trama:map api"` {
		t.Fatalf("problems = %q", p)
	}
}

func TestStatusCardStaleForUnknownCommit(t *testing.T) {
	f := newFixture(t, true)
	f.writeCard(t, "deadbeef")
	if s := only(t, f.ws); s.Card != status.CardStale {
		t.Fatalf("card = %q, want stale", s.Card)
	}
}

func TestStatusBehind(t *testing.T) {
	f := newFixture(t, true)
	if err := os.WriteFile(filepath.Join(f.ws.RepoDir("api"), "README.md"), []byte("edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.remote.Commit(t, "new.txt", "new")
	f.sync(t) // fetches, cannot fast-forward a dirty tree
	s := only(t, f.ws)
	if s.Behind != 1 || s.Ahead != 0 {
		t.Fatalf("ahead/behind = %d/%d, want 0/1", s.Ahead, s.Behind)
	}
	want := `1 commit(s) behind origin/main: run "trama sync"`
	found := false
	for _, p := range s.Problems() {
		found = found || p == want
	}
	if !found {
		t.Fatalf("problems = %q, want %q", s.Problems(), want)
	}
}

func TestStatusInfoFields(t *testing.T) {
	f := newFixture(t, true)
	f.writeCard(t, f.originMain(t))
	dir := f.ws.RepoDir("api")
	testutil.Git(t, dir, "checkout", "-q", "-b", "feature")
	if err := os.WriteFile(filepath.Join(dir, "scratch.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := only(t, f.ws)
	if s.Branch != "feature" || s.Clean {
		t.Fatalf("status = %+v", s)
	}
	if p := s.Problems(); len(p) != 0 {
		t.Fatalf("branch and local changes must not be problems: %q", p)
	}
}

func TestStatusNotARepo(t *testing.T) {
	f := newFixture(t, false)
	if err := os.MkdirAll(f.ws.RepoDir("api"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := only(t, f.ws)
	if s.Err == "" || len(s.Problems()) == 0 {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusDoesNotUseNetwork(t *testing.T) {
	f := newFixture(t, true)
	f.writeCard(t, f.originMain(t))
	if err := os.RemoveAll(f.remote.URL); err != nil {
		t.Fatal(err)
	}
	s := only(t, f.ws)
	if s.Err != "" || s.Card != status.CardCurrent {
		t.Fatalf("status = %+v", s)
	}
}

func TestRenderQuietHealthyPrintsNothing(t *testing.T) {
	var buf bytes.Buffer
	status.Render(&buf, []status.RepoStatus{{Name: "api", Cloned: true, Branch: "main", Main: "main", Clean: true, Card: status.CardCurrent}}, true)
	if buf.Len() != 0 {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestRenderQuietPrintsProblems(t *testing.T) {
	var buf bytes.Buffer
	status.Render(&buf, []status.RepoStatus{{Name: "api", Cloned: true, Branch: "main", Main: "main", Clean: true, Card: status.CardMissing}}, true)
	if got, want := buf.String(), "trama: api: no catalog card: run \"/trama:map api\"\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRenderTable(t *testing.T) {
	var buf bytes.Buffer
	status.Render(&buf, []status.RepoStatus{{Name: "api", Cloned: true, Branch: "main", Main: "main", Clean: false, Ahead: 1, Behind: 2, Card: status.CardStale}}, false)
	out := buf.String()
	for _, want := range []string{"api", "main", "dirty", "+1 -2", "stale", "- api: catalog card is stale"} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
}
