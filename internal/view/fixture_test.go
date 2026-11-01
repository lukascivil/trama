package view_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/view"
	"github.com/lukascivil/trama/internal/workspace"
)

const demoWorkspace = `version: 1
name: demo
repos:
  - name: trama-demo-api
    url: https://github.com/lukascivil/trama-demo-api.git
    branch: main
  - name: trama-demo-web
    url: https://github.com/lukascivil/trama-demo-web.git
    branch: main
`

const demoAPI = `version: 1
name: trama-demo-api
description: Backend that serves the greeting
stack: [go]
mapped_at_commit: 1cb5a15
mapped_at: 2026-10-28
run:
  dev: { cmd: "go run ./cmd/api", port: 8080 }
  test: { cmd: "go test ./..." }
identity:
  ports: [{ value: 8080, evidence: ["cmd/api/main.go:11"] }]
provides:
  - kind: http
    id: "GET /hello"
    response: { message: string }
    evidence: ["internal/api/api.go:42"]
`

const demoWeb = `version: 1
name: trama-demo-web
description: Frontend that shows the greeting from the API
stack: [node, react, typescript]
mapped_at_commit: 697bd1b
mapped_at: 2026-10-28
run:
  dev: { cmd: "npm run dev", port: 5173 }
consumes:
  - kind: http
    id: "GET /hello"
    from: trama-demo-api
    via: env
    fields_used: [message]
    evidence: ["src/api/hello.ts:10"]
config:
  - env: VITE_API_URL
    value: "http://localhost:8080"
    points_to: trama-demo-api
    evidence: [".env.example:1"]
`

type fx struct {
	t    *testing.T
	root string
}

// demo is the two-repo demo with both cards and every cited file, without git.
func demo(t *testing.T) *fx {
	t.Helper()
	f := &fx{t: t, root: t.TempDir()}
	f.write(workspace.FileName, demoWorkspace)
	f.card("trama-demo-api", demoAPI)
	f.card("trama-demo-web", demoWeb)
	f.file("trama-demo-api", "cmd/api/main.go", 20)
	f.file("trama-demo-api", "internal/api/api.go", 45)
	f.file("trama-demo-web", "src/api/hello.ts", 16)
	f.file("trama-demo-web", ".env.example", 1)
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

func (f *fx) card(name, content string) {
	f.t.Helper()
	rel, _ := filepath.Rel(f.root, catalog.Path(f.root, name))
	f.write(rel, content)
}

// edit replaces old with new in the card of name.
func (f *fx) edit(name, old, new string) {
	f.t.Helper()
	data, err := os.ReadFile(catalog.Path(f.root, name))
	if err != nil || !strings.Contains(string(data), old) {
		f.t.Fatalf("%v: %q not in the card of %s", err, old, name)
	}
	f.card(name, strings.Replace(string(data), old, new, 1))
}

// file writes repos/<repo>/<path> with n lines.
func (f *fx) file(repo, path string, n int) {
	f.t.Helper()
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	f.write(filepath.Join("repos", repo, path), b.String())
}

func (f *fx) build() *view.Model {
	f.t.Helper()
	m, err := view.NewBuilder(f.root).Build()
	if err != nil {
		f.t.Fatal(err)
	}
	return m
}

func repoOf(t *testing.T, m *view.Model, name string) view.Repo {
	t.Helper()
	for _, r := range m.Repos {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no repo %s in %+v", name, m.Repos)
	return view.Repo{}
}

func containsText(list []string, s string) bool {
	for _, l := range list {
		if strings.Contains(l, s) {
			return true
		}
	}
	return false
}
