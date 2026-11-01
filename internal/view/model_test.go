package view_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/testutil"
	"github.com/lukascivil/trama/internal/view"
)

func edgeOf(t *testing.T, m *view.Model, consumer, id string) view.Edge {
	t.Helper()
	for _, e := range m.Edges {
		if e.Consumer == consumer && e.ID == id {
			return e
		}
	}
	t.Fatalf("no edge %s %s in %+v", consumer, id, m.Edges)
	return view.Edge{}
}

func TestBuildDemo(t *testing.T) {
	m := demo(t).build()
	if m.Workspace != "demo" || len(m.Repos) != 2 {
		t.Fatalf("model = %s with %d repos", m.Workspace, len(m.Repos))
	}
	e := edgeOf(t, m, "trama-demo-web", "GET /hello")
	if e.Provider != "trama-demo-api" || e.Status != view.Proven || e.Via != "env" {
		t.Fatalf("edge = %+v, want proven from trama-demo-api via env", e)
	}
	want := view.Link{Text: "src/api/hello.ts:10", URL: "https://github.com/lukascivil/trama-demo-web/blob/697bd1b/src/api/hello.ts#L10"}
	if len(e.Evidence) != 1 || e.Evidence[0] != want {
		t.Fatalf("evidence = %+v, want %+v", e.Evidence, want)
	}
	api := repoOf(t, m, "trama-demo-api")
	if len(api.Affected.Direct) != 1 || api.Affected.Direct[0].Repo != "trama-demo-web" {
		t.Fatalf("api affected = %+v, want trama-demo-web as direct", api.Affected)
	}
	if api.URL != "https://github.com/lukascivil/trama-demo-api" || api.Description != "Backend that serves the greeting" {
		t.Fatalf("api = %+v", api)
	}
	if got := []view.Command{{Kind: "dev", Cmd: "go run ./cmd/api", Port: 8080}, {Kind: "test", Cmd: "go test ./..."}}; !reflect.DeepEqual(api.Run, got) {
		t.Fatalf("run = %+v, want %+v", api.Run, got)
	}
	if len(api.Provides) != 1 || api.Provides[0].Evidence[0].URL != "https://github.com/lukascivil/trama-demo-api/blob/1cb5a15/internal/api/api.go#L42" {
		t.Fatalf("provides = %+v", api.Provides)
	}
}

func TestBuildEvidenceRange(t *testing.T) {
	f := demo(t)
	f.edit("trama-demo-web", `"src/api/hello.ts:10"`, `"src/api/hello.ts:10-12"`)
	e := edgeOf(t, f.build(), "trama-demo-web", "GET /hello")
	if !strings.HasSuffix(e.Evidence[0].URL, "/src/api/hello.ts#L10-L12") {
		t.Fatalf("url = %q, want a #L10-L12 anchor", e.Evidence[0].URL)
	}
}

func TestBuildUnverifiedOnValidationError(t *testing.T) {
	f := demo(t)
	f.edit("trama-demo-web", `"src/api/hello.ts:10"`, `"src/api/hello.ts:999"`)
	m := f.build()
	e := edgeOf(t, m, "trama-demo-web", "GET /hello")
	if e.Status != view.Unverified || !containsText(e.Problems, "line 999") {
		t.Fatalf("edge = %+v, want unverified with the line 999 problem", e)
	}
	api := repoOf(t, m, "trama-demo-api")
	if len(api.Affected.Direct) != 0 || len(api.Affected.Possible) != 1 || api.Affected.Possible[0].Reason != "unverified" {
		t.Fatalf("api affected = %+v, want the web only as possible (unverified)", api.Affected)
	}
	if !containsText(m.Warnings, "line 999") {
		t.Fatalf("warnings = %v, want the validation error", m.Warnings)
	}
}

func TestBuildFromRepoOutsideWorkspace(t *testing.T) {
	f := demo(t)
	f.edit("trama-demo-web", "from: trama-demo-api", "from: ghost")
	m := f.build()
	if e := edgeOf(t, m, "trama-demo-web", "GET /hello"); e.Status != view.Unverified || len(e.Problems) == 0 {
		t.Fatalf("edge = %+v, want unverified with a problem", e)
	}
	names := map[string]bool{}
	for _, r := range m.Repos {
		names[r.Name] = true
	}
	for _, e := range m.Edges {
		if e.Status != view.Unverified && e.Provider != "" && !names[e.Provider] {
			t.Fatalf("edge %+v points to a repo outside the workspace", e)
		}
	}
}

func TestBuildPending(t *testing.T) {
	f := demo(t)
	f.edit("trama-demo-web", "from: trama-demo-api\n    via: env", "from: unknown\n    candidates: [trama-demo-api]")
	m := f.build()
	e := edgeOf(t, m, "trama-demo-web", "GET /hello")
	if e.Status != view.Pending || e.Provider != "" || !reflect.DeepEqual(e.Candidates, []string{"trama-demo-api"}) {
		t.Fatalf("edge = %+v, want pending with trama-demo-api as candidate", e)
	}
	if api := repoOf(t, m, "trama-demo-api"); len(api.Affected.Possible) != 1 || api.Affected.Possible[0].Reason != "unknown" {
		t.Fatalf("api affected = %+v, want the web as possible (unknown)", api.Affected)
	}
}

func TestBuildDeclaredLinkWithoutCard(t *testing.T) {
	f := demo(t)
	f.write("workspace.yaml", demoWorkspace+"links:\n  - consumer: trama-demo-web\n    provider: trama-demo-api\n    id: \"GET /other\"\n")
	m := f.build()
	e := edgeOf(t, m, "trama-demo-web", "GET /other")
	if e.Status != view.Declared || e.Provider != "trama-demo-api" || len(e.Evidence) != 1 || e.Evidence[0].Text != "workspace.yaml" {
		t.Fatalf("edge = %+v, want declared with workspace.yaml as evidence", e)
	}
	direct := repoOf(t, m, "trama-demo-api").Affected.Direct
	if len(direct) != 2 || direct[1].ID != "GET /other" {
		t.Fatalf("direct = %+v, want GET /hello and GET /other", direct)
	}
}

func TestBuildBrokenCardIsWarning(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-web", "version: [\n")
	m := f.build()
	repoOf(t, m, "trama-demo-web")
	for _, e := range m.Edges {
		if e.Consumer == "trama-demo-web" {
			t.Fatalf("edge %+v from a card that does not open", e)
		}
	}
	if !containsText(m.Warnings, "trama-demo-web") {
		t.Fatalf("warnings = %v, want one about the broken card", m.Warnings)
	}
}

func TestBuildKeepsHostileText(t *testing.T) {
	f := demo(t)
	f.edit("trama-demo-api", "description: Backend that serves the greeting", `description: "<script>alert(1)</script> \"quoted\" ação"`)
	m := f.build()
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var back view.Model
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if got := repoOf(t, &back, "trama-demo-api").Description; got != `<script>alert(1)</script> "quoted" ação` {
		t.Fatalf("description after a JSON round trip = %q", got)
	}
}

func TestBuildOpenChanges(t *testing.T) {
	f := demo(t)
	f.write("changes/2026-11-03-rename-message.md", `---
id: 2026-11-03-rename-message
title: Rename message to greeting
status: open
steps:
  - {order: 1, repo: trama-demo-api, action: Add greeting, branch: trama/x, pr: "https://github.com/o/a/pull/1", state: merged}
  - {order: 2, repo: trama-demo-web, action: Read greeting, branch: trama/x, pr: "https://github.com/o/w/pull/1", state: open}
---

## Why

Say greeting.
`)
	m := f.build()
	web := repoOf(t, m, "trama-demo-web")
	if len(web.OpenChanges) != 1 || web.OpenChanges[0].ID != "2026-11-03-rename-message" || web.OpenChanges[0].Title != "Rename message to greeting" {
		t.Fatalf("web open changes = %+v", web.OpenChanges)
	}
	if api := repoOf(t, m, "trama-demo-api"); len(api.OpenChanges) != 0 {
		t.Fatalf("api open changes = %+v, want none (its step is merged)", api.OpenChanges)
	}
}

// gitWorkspace makes a workspace of clones of fresh remotes. cards maps a
// repo to the body of its card after name and mapped_at_commit; clone says
// which repos get cloned. links is appended to workspace.yaml.
func gitWorkspace(t testing.TB, names []string, clone func(string) bool, cards map[string]string, links string) (string, map[string]*testutil.Remote) {
	t.Helper()
	root := t.TempDir()
	remotes := map[string]*testutil.Remote{}
	yaml := "version: 1\nname: git\nrepos:\n"
	for _, n := range names {
		remotes[n] = testutil.NewRemote(t, "main")
		yaml += fmt.Sprintf("  - name: %s\n    url: %s\n    branch: main\n", n, remotes[n].URL)
	}
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(yaml+links), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		dir := filepath.Join(root, "repos", n)
		sha := "0000000"
		if clone(n) {
			if err := gitx.Clone(remotes[n].URL, "main", dir); err != nil {
				t.Fatal(err)
			}
			sha = testutil.Git(t, dir, "rev-parse", "origin/main")
		}
		card := fmt.Sprintf("version: 1\nname: %s\nmapped_at_commit: %s\n%s", n, sha, cards[n])
		if err := os.WriteFile(filepath.Join(root, "catalog", n+".yaml"), []byte(card), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root, remotes
}

func TestBuildReleaseAndCard(t *testing.T) {
	root, remotes := gitWorkspace(t, []string{"a", "b", "c"}, func(n string) bool { return n != "c" }, nil, "")
	a := gitx.Repo{Dir: filepath.Join(root, "repos", "a")}
	testutil.Git(t, a.Dir, "tag", "v0.1.0", "origin/main")
	remotes["a"].Commit(t, "x.txt", "x\n")
	remotes["a"].Commit(t, "y.txt", "y\n")
	if err := a.Fetch(); err != nil {
		t.Fatal(err)
	}
	sha := testutil.Git(t, a.Dir, "rev-parse", "origin/main")
	if err := os.WriteFile(filepath.Join(root, "catalog", "a.yaml"), []byte("version: 1\nname: a\nmapped_at_commit: "+sha+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := view.NewBuilder(root).Build()
	if err != nil {
		t.Fatal(err)
	}
	ra := repoOf(t, m, "a")
	if ra.Release == nil || ra.Release.Version != "v0.1.0" || ra.Release.CommitsAfter != 2 || ra.Card != "current" {
		t.Fatalf("a = %+v release %+v, want v0.1.0, 2 commits after, current card", ra, ra.Release)
	}
	if rb := repoOf(t, m, "b"); !rb.Cloned || rb.Release != nil {
		t.Fatalf("b = %+v, want cloned without release", rb)
	}
	if rc := repoOf(t, m, "c"); rc.Cloned || rc.Release != nil {
		t.Fatalf("c = %+v, want not cloned and without release", rc)
	}
}

func TestBuildCachesValidation(t *testing.T) {
	f := demo(t)
	var n int
	view.CountChecks(t.Cleanup, &n)
	b := view.NewBuilder(f.root)
	for range 2 {
		if _, err := b.Build(); err != nil {
			t.Fatal(err)
		}
	}
	if n != 1 {
		t.Fatalf("validation ran %d times for an unchanged workspace, want 1", n)
	}
	f.edit("trama-demo-api", "Backend that serves", "Backend that returns")
	if _, err := b.Build(); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("validation ran %d times after a card changed, want 2", n)
	}
}

func TestBuildHundredRepos(t *testing.T) {
	if os.Getenv("TRAMA_PERF") != "1" {
		t.Skip("set TRAMA_PERF=1 to measure 100 repositories")
	}
	var names []string
	cards := map[string]string{}
	links := "links:\n"
	for i := range 100 {
		n := fmt.Sprintf("r%02d", i)
		names = append(names, n)
		cards[n] = fmt.Sprintf("provides:\n  - kind: http\n    id: \"GET /%s\"\n    evidence: [\"README.md:1\"]\n", n)
		if i > 0 {
			prev := fmt.Sprintf("r%02d", i-1)
			cards[n] += fmt.Sprintf("consumes:\n  - kind: http\n    id: \"GET /%s\"\n    from: %s\n    via: declared\n    evidence: [\"README.md:1\"]\n", prev, prev)
			links += fmt.Sprintf("  - consumer: %s\n    provider: %s\n    id: \"GET /%s\"\n", n, prev, prev)
		}
	}
	root, _ := gitWorkspace(t, names, func(string) bool { return true }, cards, links)
	b := view.NewBuilder(root)
	start := time.Now()
	m, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	first := time.Since(start)
	start = time.Now()
	if _, err := b.Build(); err != nil {
		t.Fatal(err)
	}
	second := time.Since(start)
	t.Logf("100 repos, %d edges, %d warnings: first build %v, second %v", len(m.Edges), len(m.Warnings), first, second)
	if second > 2*time.Second {
		t.Fatalf("second build took %v, want under 2s", second)
	}
}

func TestBuildProviderCardBrokenIsUnverified(t *testing.T) {
	f := demo(t)
	f.edit("trama-demo-api", "stack: [go]", "stack: [go]\ntypo_field: 1")
	m := f.build()
	if e := edgeOf(t, m, "trama-demo-web", "GET /hello"); e.Status != view.Unverified || !containsText(e.Problems, "trama-demo-api") {
		t.Fatalf("edge = %+v, want unverified because the provider's card has errors", e)
	}
	api := repoOf(t, m, "trama-demo-api")
	if len(api.Affected.Direct) != 0 || len(api.Affected.Possible) != 1 {
		t.Fatalf("api affected = %+v, want the web only as possible", api.Affected)
	}
}

// withTable adds a table to the api card, citing a file the fixture writes.
func withTable(f *fx) {
	f.edit("trama-demo-api",
		`  ports: [{ value: 8080, evidence: ["cmd/api/main.go:11"] }]`,
		`  ports: [{ value: 8080, evidence: ["cmd/api/main.go:11"] }]
  tables:
    - value: greetings
      columns: [id, message]
      evidence: ["cmd/api/main.go:11"]`)
}

func TestModelCarriesData(t *testing.T) {
	f := demo(t)
	withTable(f)
	m := f.build()
	var got []string
	for _, d := range m.Data {
		got = append(got, d.Layer+" "+d.Repo+" "+d.Match)
	}
	want := []string{
		"response trama-demo-api message",
		"table trama-demo-api greetings.id",
		"table trama-demo-api greetings.message",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("data = %v, want %v", got, want)
	}
}

func TestModelDataEmptyWithoutFieldsOrTables(t *testing.T) {
	f := demo(t)
	f.edit("trama-demo-api", "    response: { message: string }\n", "")
	if m := f.build(); len(m.Data) != 0 {
		t.Fatalf("data = %+v, want none", m.Data)
	}
}

func TestModelDataEvidenceIsLinked(t *testing.T) {
	m := demo(t).build()
	if len(m.Data) != 1 {
		t.Fatalf("data = %+v, want one entry", m.Data)
	}
	ev := m.Data[0].Evidence
	want := "https://github.com/lukascivil/trama-demo-api/blob/1cb5a15/internal/api/api.go#L42"
	if len(ev) != 1 || ev[0].Text != "internal/api/api.go:42" || ev[0].URL != want {
		t.Fatalf("evidence = %+v, want a link to %s", ev, want)
	}
}

func TestModelDataJSONKey(t *testing.T) {
	f := demo(t)
	withTable(f)
	data, err := json.Marshal(f.build())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"data":[`, `"layer":"response"`, `"repo":"trama-demo-api"`, `"match":"message"`, `"evidence":[`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("json lacks %s:\n%s", want, data)
		}
	}
	empty := demo(t)
	empty.edit("trama-demo-api", "    response: { message: string }\n", "")
	plain, err := json.Marshal(empty.build())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(plain), `"data"`) {
		t.Fatalf("json should omit an empty data:\n%s", plain)
	}
}
