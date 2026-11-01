package bodies_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/bodies"
	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/testutil"
	"github.com/lukascivil/trama/internal/validate"
	"github.com/lukascivil/trama/internal/workspace"
)

const ws = `version: 1
name: demo
repos:
  - name: api
    url: https://github.com/x/api.git
    branch: main
  - name: web
    url: https://github.com/x/web.git
    branch: main
`

const apiCard = `version: 1
name: api
description: Orders API
stack: [go]
mapped_at_commit: 1cb5a15
mapped_at: 2026-10-28
provides:
  # the order routes
  - kind: http
    id: "POST /orders"
    evidence: ["router.go:1"]
  - { kind: http, id: "GET /orders/{id}", evidence: ["router.go:2"] } # read one
  - kind: http
    id: "DELETE /orders/{id}"
    request: { reason: string }
    evidence: ["router.go:3"]
`

const spec = `openapi: 3.0.3
paths:
  /orders:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                email: { type: string }
                items: { type: array, items: { type: object, properties: { sku: { type: string } } } }
      responses:
        "201":
          content:
            application/json:
              schema:
                type: object
                properties:
                  id: { type: string }
                  tags: { type: array, items: { type: string } }
  /orders/{order_id}:
    get:
      responses:
        "200":
          content:
            application/json:
              schema: { type: object, properties: { id: { type: string } } }
    delete:
      responses:
        "204": { description: deleted }
`

type fx struct {
	t    *testing.T
	root string
}

func fixture(t *testing.T) *fx {
	t.Helper()
	f := &fx{t: t, root: t.TempDir()}
	f.write("workspace.yaml", ws)
	f.write("catalog/api.yaml", apiCard)
	f.write("repos/api/router.go", "a\nb\nc\n")
	f.write("repos/api/openapi.yaml", spec)
	f.write("repos/web/app.ts", "a\n")
	return f
}

func (f *fx) write(rel, content string) {
	f.t.Helper()
	full := filepath.Join(f.root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fx) edit(rel, old, new string) {
	f.t.Helper()
	data := f.read(rel)
	if !strings.Contains(data, old) {
		f.t.Fatalf("%s lacks %q", rel, old)
	}
	f.write(rel, strings.Replace(data, old, new, 1))
}

func (f *fx) read(rel string) string {
	f.t.Helper()
	data, err := os.ReadFile(filepath.Join(f.root, rel))
	if err != nil {
		f.t.Fatal(err)
	}
	return string(data)
}

func (f *fx) ws() *workspace.Workspace {
	f.t.Helper()
	w, err := workspace.Load(f.root)
	if err != nil {
		f.t.Fatal(err)
	}
	return w
}

func (f *fx) plan(names ...string) []bodies.Report {
	f.t.Helper()
	reports, err := bodies.Plan(f.ws(), names)
	if err != nil {
		f.t.Fatal(err)
	}
	return reports
}

// lines renders a plan the way trama bodies prints it.
func lines(reports []bodies.Report) []string {
	var out []string
	for _, r := range reports {
		for _, ch := range r.Changes {
			out = append(out, ch.ID+" <- "+ch.Spec+" ("+ch.Summary()+")")
		}
		for _, n := range r.Notes {
			out = append(out, r.Repo+": "+n)
		}
	}
	return out
}

func changes(reports []bodies.Report) []bodies.Change {
	var out []bodies.Change
	for _, r := range reports {
		out = append(out, r.Changes...)
	}
	return out
}

func want(t *testing.T, got []string, want ...string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("got:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

func TestFindsTheOnlySpecAndWritesTheContract(t *testing.T) {
	f := fixture(t)
	want(t, lines(f.plan()),
		"POST /orders <- openapi.yaml (contract, request 2 fields, response 2 fields)",
		"GET /orders/{id} <- openapi.yaml (contract, response 1 field)",
		"DELETE /orders/{id} <- openapi.yaml (contract, no request)",
	)
	ch := changes(f.plan())[0]
	if ch.Line != 4 || ch.Request.Fields[1].Path != "items[].sku" {
		t.Fatalf("change = %+v", ch)
	}
}

func TestFillsFromTheContract(t *testing.T) {
	f := fixture(t)
	f.write("repos/api/docs/api.yaml", spec)
	f.edit("catalog/api.yaml", `id: "POST /orders"`, `id: "POST /orders"`+"\n    contract: docs/api.yaml")
	got := lines(f.plan())
	if got[0] != "POST /orders <- docs/api.yaml (request 2 fields, response 2 fields)" {
		t.Fatalf("got %v", got)
	}
	if !slices.Contains(got, "api: GET /orders/{id}: declared by docs/api.yaml and openapi.yaml: set contract to the one that serves it") {
		t.Fatalf("an ambiguous route was written: %v", got)
	}
}

func TestRouteNotInTheContract(t *testing.T) {
	f := fixture(t)
	f.write("repos/api/other.yaml", "openapi: 3.0.0\npaths:\n  /other:\n    get: {}\n")
	f.edit("catalog/api.yaml", `id: "POST /orders"`, `id: "POST /orders"`+"\n    contract: other.yaml")
	got := lines(f.plan())
	if !slices.Contains(got, "api: POST /orders: not in contract other.yaml") {
		t.Fatalf("got %v", got)
	}
}

func TestCountsRoutesInNoSpecAndNamesRoutesTheCardMisses(t *testing.T) {
	f := fixture(t)
	f.edit("catalog/api.yaml", `id: "DELETE /orders/{id}"`, `id: "DELETE /legacy"`)
	f.edit("catalog/api.yaml", "  - { kind: http, id: \"GET /orders/{id}\", evidence: [\"router.go:2\"] } # read one\n", "  - { kind: http, id: \"GET /health\", evidence: [\"router.go:2\"] }\n")
	want(t, lines(f.plan()),
		"POST /orders <- openapi.yaml (contract, request 2 fields, response 2 fields)",
		"api: 2 routes are in no OpenAPI spec of the repository",
		"api: openapi.yaml declares 2 routes the card does not list: GET /orders/{order_id}, DELETE /orders/{order_id}",
	)
}

func TestSilentWithoutSpecs(t *testing.T) {
	f := fixture(t)
	if err := os.Remove(filepath.Join(f.root, "repos/api/openapi.yaml")); err != nil {
		t.Fatal(err)
	}
	f.write("repos/api/config.yaml", "port: 8080\n")
	if got := f.plan(); len(got) != 0 {
		t.Fatalf("got %v", lines(got))
	}
}

func TestABodyTramaCannotReadStaysAsItIs(t *testing.T) {
	f := fixture(t)
	f.edit("repos/api/openapi.yaml", "schema: { type: object, properties: { id: { type: string } } }", "schema: { type: array, items: { type: string } }")
	f.edit("catalog/api.yaml", `id: "GET /orders/{id}", evidence`, `id: "GET /orders/{id}", response: { id: string }, evidence`)
	got := lines(f.plan())
	if !slices.Contains(got, "GET /orders/{id} <- openapi.yaml (contract)") || !slices.Contains(got, "api: GET /orders/{id}: response is a list, not an object: left as it is") {
		t.Fatalf("got %v", got)
	}
}

func TestACardThatMatchesTheSpecDoesNotChange(t *testing.T) {
	f := fixture(t)
	if err := bodies.Apply(f.root, changes(f.plan())); err != nil {
		t.Fatal(err)
	}
	if got := f.plan(); len(got) != 0 {
		t.Fatalf("second run: %v", lines(got))
	}
}

func TestContractProblems(t *testing.T) {
	cases := map[string]string{
		"../x.yaml":    `api: contract "../x.yaml": path must be relative to the repository root, without ..`,
		"missing.yaml": "api: contract missing.yaml: missing.yaml does not exist",
		"router.go":    "",
		"broken.yaml":  "api: contract broken.yaml is not YAML: ",
		"chart.yaml":   "api: contract chart.yaml is not an OpenAPI 3 spec",
	}
	for contract, note := range cases {
		f := fixture(t)
		f.write("repos/api/broken.yaml", "a: [\n")
		f.write("repos/api/chart.yaml", "apiVersion: v2\n")
		for _, id := range []string{`id: "POST /orders"`, `id: "DELETE /orders/{id}"`} {
			f.edit("catalog/api.yaml", id, id+"\n    contract: "+contract)
		}
		got := lines(f.plan())
		var notes []string
		for _, l := range got {
			if strings.HasPrefix(l, "api: contract") {
				notes = append(notes, l)
			}
		}
		if note == "" {
			if len(notes) != 0 {
				t.Errorf("%s: notes %v", contract, notes)
			}
			continue
		}
		if len(notes) != 1 || !strings.HasPrefix(notes[0], note) {
			t.Errorf("%s: notes %v, want one starting %q", contract, notes, note)
		}
		if strings.Contains(strings.Join(got, "\n"), "POST /orders <-") {
			t.Errorf("%s: a route with a broken contract was written: %v", contract, got)
		}
	}
}

func TestUnknownRepoAndRepoWithoutCard(t *testing.T) {
	f := fixture(t)
	if _, err := bodies.Plan(f.ws(), []string{"nope"}); err == nil || err.Error() != "repo nope is not in workspace.yaml" {
		t.Fatalf("err = %v", err)
	}
	if _, err := bodies.Plan(f.ws(), []string{"web"}); err == nil || err.Error() != "no card at catalog/web.yaml: write it first" {
		t.Fatalf("err = %v", err)
	}
}

func TestNotCloned(t *testing.T) {
	f := fixture(t)
	if err := os.RemoveAll(filepath.Join(f.root, "repos/api")); err != nil {
		t.Fatal(err)
	}
	want(t, lines(f.plan("api")), `api: repos/api is not cloned: run "trama sync api"`)
}

func TestApplyWritesBlocksKeepsCommentsAndLeavesAValidCard(t *testing.T) {
	f := fixture(t)
	if err := bodies.Apply(f.root, changes(f.plan())); err != nil {
		t.Fatal(err)
	}
	wantCard := `version: 1
name: api
description: Orders API
stack: [go]
mapped_at_commit: 1cb5a15
mapped_at: 2026-10-28
provides:
  # the order routes
  - kind: http
    id: "POST /orders"
    contract: openapi.yaml
    request:
      email: string
      items[].sku: string
    response:
      id: string
      tags: "[]string"
    evidence: ["router.go:1"]
  - kind: http
    id: "GET /orders/{id}"
    contract: openapi.yaml
    response:
      id: string
    evidence: ["router.go:2"] # read one
  - kind: http
    id: "DELETE /orders/{id}"
    contract: openapi.yaml
    evidence: ["router.go:3"]
`
	if got := f.read("catalog/api.yaml"); got != wantCard {
		t.Fatalf("card:\n%s\nwant:\n%s", got, wantCard)
	}
	for _, finding := range validate.Workspace(f.ws()) {
		if finding.Severity == validate.Error || strings.HasPrefix(finding.File, "catalog/api") {
			t.Errorf("finding after bodies: %s", finding)
		}
	}
	if _, err := catalog.LoadStrict(f.root, "api"); err != nil {
		t.Fatal(err)
	}
}

func TestApplyRefusesACardThatChanged(t *testing.T) {
	f := fixture(t)
	chs := changes(f.plan())
	f.edit("catalog/api.yaml", `id: "POST /orders"`, `id: "PUT /orders"`)
	err := bodies.Apply(f.root, chs)
	if err == nil || !strings.HasSuffix(err.Error(), "provides[0] is no longer POST /orders") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadsTheSpecAtOrigin(t *testing.T) {
	f := fixture(t)
	if err := os.RemoveAll(filepath.Join(f.root, "repos/api")); err != nil {
		t.Fatal(err)
	}
	remote := testutil.NewRemote(t, "main")
	remote.Commit(t, "router.go", "a\nb\nc\n")
	remote.Commit(t, "api/openapi.yaml", spec)
	dir := filepath.Join(f.root, "repos/api")
	if err := gitx.Clone(remote.URL, "main", dir); err != nil {
		t.Fatal(err)
	}
	// Only on disk: the card describes origin/main, so this must not count.
	f.write("repos/api/local.yaml", spec)
	f.edit("catalog/api.yaml", `id: "POST /orders"`, `id: "POST /orders"`+"\n    contract: ./api/openapi.yaml")
	want(t, lines(f.plan()),
		"POST /orders <- api/openapi.yaml (request 2 fields, response 2 fields)",
		"GET /orders/{id} <- api/openapi.yaml (contract, response 1 field)",
		"DELETE /orders/{id} <- api/openapi.yaml (contract, no request)",
	)
}
