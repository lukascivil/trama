package linker_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/catalog"
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
identity:
  ports: [{ value: 8080, evidence: ["cmd/api/main.go:11"] }]
  topics: [{ value: greeting_fct_helloSaid_0, evidence: ["internal/api/api.go:42"] }]
provides:
  - kind: http
    id: "GET /hello"
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

// unknownWebConsume puts the web consume back to where a fresh map leaves
// it: a provider nobody has proven yet.
func (f *fx) unknownWebConsume() {
	f.editWeb("    from: trama-demo-api\n    via: env\n", "    from: unknown\n")
}

// secondAPI adds a repository that serves the very same route.
func (f *fx) secondAPI() {
	f.write(workspace.FileName, demoWorkspace+`  - name: trama-demo-legacy
    url: https://github.com/lukascivil/trama-demo-legacy.git
    branch: main
`)
	f.card("trama-demo-legacy", strings.Replace(demoAPI, "name: trama-demo-api", "name: trama-demo-legacy", 1))
	f.file("trama-demo-legacy", "cmd/api/main.go", 20)
	f.file("trama-demo-legacy", "internal/api/api.go", 45)
}

func (f *fx) editWeb(old, new string) { f.edit("trama-demo-web", old, new) }
func (f *fx) editAPI(old, new string) { f.edit("trama-demo-api", old, new) }

func (f *fx) edit(name, old, new string) {
	f.t.Helper()
	data := f.readCard(name)
	if !strings.Contains(data, old) {
		f.t.Fatalf("%s lacks %q", name, old)
	}
	f.card(name, strings.Replace(data, old, new, 1))
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

func (f *fx) file(repo, path string, n int) {
	f.t.Helper()
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	f.write(filepath.Join("repos", repo, path), b.String())
}
