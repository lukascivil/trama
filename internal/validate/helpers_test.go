package validate_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/validate"
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
tools:
  go: "1.25"
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
    shape: { message: string }
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

// fixture writes workspace.yaml in a fresh directory.
func fixture(t *testing.T, workspaceYAML string) *fx {
	t.Helper()
	f := &fx{t: t, root: t.TempDir()}
	f.write(workspace.FileName, workspaceYAML)
	return f
}

// demo is the two-repo demo with both cards and every cited file.
func demo(t *testing.T) *fx {
	t.Helper()
	f := fixture(t, demoWorkspace)
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

// file writes repos/<repo>/<path> with n lines.
func (f *fx) file(repo, path string, n int) {
	f.t.Helper()
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	f.write(filepath.Join("repos", repo, path), b.String())
}

func (f *fx) findings() []validate.Finding {
	f.t.Helper()
	ws, err := workspace.Load(f.root)
	if err != nil {
		f.t.Fatal(err)
	}
	return validate.Workspace(ws)
}

// only returns the single finding, failing otherwise.
func only(t *testing.T, fs []validate.Finding) validate.Finding {
	t.Helper()
	if len(fs) != 1 {
		t.Fatalf("want 1 finding, got %d: %v", len(fs), fs)
	}
	return fs[0]
}

func hasError(fs []validate.Finding, where, text string) bool {
	for _, f := range fs {
		if f.Severity == validate.Error && f.Where == where && strings.Contains(f.Message, text) {
			return true
		}
	}
	return false
}
