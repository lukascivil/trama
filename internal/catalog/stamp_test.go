package catalog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/testutil"
	"github.com/lukascivil/trama/internal/workspace"
)

var today = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

const stampCard = `# card for api
version: 1
name: api
description: API
stack: [go]
mapped_at_commit: ""
mapped_at: ""
provides:
  - { kind: http, id: "GET /hello", evidence: ["README.md:1"] }
`

type stampFx struct {
	ws     *workspace.Workspace
	remote *testutil.Remote
}

// newStampFx creates a workspace with repo "api"; clone controls whether
// repos/api is cloned and card whether catalog/api.yaml exists.
func newStampFx(t *testing.T, clone bool, card string) stampFx {
	t.Helper()
	remote := testutil.NewRemote(t, "main")
	root := t.TempDir()
	yml := "version: 1\nname: t\nrepos:\n  - {name: api, url: \"" + remote.URL + "\", branch: main}\n"
	if err := os.WriteFile(filepath.Join(root, workspace.FileName), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if clone {
		if err := gitx.Clone(remote.URL, "main", ws.RepoDir("api")); err != nil {
			t.Fatal(err)
		}
	}
	if card != "" {
		writeCard(t, root, "api", card)
	}
	return stampFx{ws: ws, remote: remote}
}

func (f stampFx) origin(t *testing.T) string {
	t.Helper()
	sha, err := gitx.Repo{Dir: f.ws.RepoDir("api")}.RevParse("origin/main")
	if err != nil {
		t.Fatal(err)
	}
	return sha
}

func TestStampWritesCommitAndDate(t *testing.T) {
	f := newStampFx(t, true, stampCard)
	sha, err := catalog.Stamp(f.ws, "api", today)
	if err != nil {
		t.Fatal(err)
	}
	want := f.origin(t)
	if sha != want || len(sha) != 40 {
		t.Fatalf("sha = %q, want %q", sha, want)
	}
	c, err := catalog.LoadStrict(f.ws.Root, "api")
	if err != nil {
		t.Fatal(err)
	}
	if c.MappedAtCommit != want || c.MappedAt != "2026-10-28" || len(c.Provides) != 1 {
		t.Fatalf("card = %+v", c)
	}
	raw, _ := os.ReadFile(catalog.Path(f.ws.Root, "api"))
	if !strings.Contains(string(raw), "# card for api") {
		t.Fatalf("comment lost:\n%s", raw)
	}
}

func TestStampAddsMissingKeys(t *testing.T) {
	card := strings.Replace(strings.Replace(stampCard, "mapped_at_commit: \"\"\n", "", 1), "mapped_at: \"\"\n", "", 1)
	f := newStampFx(t, true, card)
	if _, err := catalog.Stamp(f.ws, "api", today); err != nil {
		t.Fatal(err)
	}
	c, err := catalog.LoadStrict(f.ws.Root, "api")
	if err != nil || c.MappedAtCommit != f.origin(t) || c.MappedAt != "2026-10-28" {
		t.Fatalf("card = %+v, %v", c, err)
	}
}

func TestStampRefuses(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T) stampFx
		want  string
	}{
		{"unknown repo", func(t *testing.T) stampFx { return newStampFx(t, true, stampCard) }, "repo ghost is not in workspace.yaml"},
		{"no card", func(t *testing.T) stampFx { return newStampFx(t, true, "") }, "no card at catalog/api.yaml: write it first"},
		{"not cloned", func(t *testing.T) stampFx { return newStampFx(t, false, stampCard) }, `repos/api is not cloned: run "trama sync api"`},
		{"other branch", func(t *testing.T) stampFx {
			f := newStampFx(t, true, stampCard)
			testutil.Git(t, f.ws.RepoDir("api"), "checkout", "-q", "-b", "feature")
			return f
		}, "repos/api is on feature, not main"},
		{"dirty", func(t *testing.T) stampFx {
			f := newStampFx(t, true, stampCard)
			if err := os.WriteFile(filepath.Join(f.ws.RepoDir("api"), "x.txt"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			return f
		}, "repos/api has local changes"},
		{"behind", func(t *testing.T) stampFx {
			f := newStampFx(t, true, stampCard)
			f.remote.Commit(t, "new.txt", "new")
			if err := (gitx.Repo{Dir: f.ws.RepoDir("api")}).Fetch(); err != nil {
				t.Fatal(err)
			}
			return f
		}, `repos/api is not at origin/main: run "trama sync api"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := c.setup(t)
			name := "api"
			if c.name == "unknown repo" {
				name = "ghost"
			}
			before, _ := os.ReadFile(catalog.Path(f.ws.Root, "api"))
			_, err := catalog.Stamp(f.ws, name, today)
			if err == nil || err.Error() != c.want {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
			after, _ := os.ReadFile(catalog.Path(f.ws.Root, "api"))
			if string(before) != string(after) {
				t.Fatal("card changed although stamp refused")
			}
		})
	}
}

func TestStampKeepsEmptyTables(t *testing.T) {
	// "tables: []" says the mapper looked and this repository owns no table;
	// it has to survive the stamp, or the next stamp brings the warning back.
	card := strings.Replace(stampCard, "provides:", "identity:\n  tables: []\nprovides:", 1)
	f := newStampFx(t, true, card)
	if _, err := catalog.Stamp(f.ws, "api", today); err != nil {
		t.Fatal(err)
	}
	c, err := catalog.LoadStrict(f.ws.Root, "api")
	if err != nil {
		t.Fatal(err)
	}
	if c.Identity.Tables == nil || len(c.Identity.Tables) != 0 {
		t.Fatalf("tables = %#v, want an empty list, not a missing one", c.Identity.Tables)
	}
}
