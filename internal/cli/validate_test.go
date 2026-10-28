package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const cliWorkspace = `version: 1
name: demo
repos:
  - name: api
    url: https://github.com/x/api.git
    branch: main
  - name: web
    url: https://github.com/x/web.git
    branch: main
`

const cliAPI = `version: 1
name: api
description: API
stack: [go]
mapped_at_commit: 1cb5a15
mapped_at: 2026-10-28
identity:
  ports: [{ value: 8080, evidence: ["main.go:1"] }]
provides:
  - { kind: http, id: "GET /hello", evidence: ["main.go:2"] }
`

const cliWeb = `version: 1
name: web
description: Web
stack: [node]
mapped_at_commit: 697bd1b
mapped_at: 2026-10-28
consumes:
  - { kind: http, id: "GET /hello", from: api, via: env, evidence: ["app.ts:1"] }
config:
  - { env: API_URL, value: "http://localhost:8080", points_to: api, evidence: [".env:1"] }
`

// catalogWorkspace writes a workspace with the two cards and their files.
func catalogWorkspace(t *testing.T, api, web string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"workspace.yaml":    cliWorkspace,
		"catalog/api.yaml":  api,
		"catalog/web.yaml":  web,
		"repos/api/main.go": "a\nb\n",
		"repos/web/app.ts":  "a\n",
		"repos/web/.env":    "API_URL=http://localhost:8080\n",
	}
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestValidateCommandOK(t *testing.T) {
	ws := catalogWorkspace(t, cliAPI, cliWeb)
	out, err := run(t, "validate", "--dir", ws)
	if err != nil || !strings.HasPrefix(out, "ok: 2 card(s), 1 relation(s), 0 warning(s)") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestValidateCommandReportsErrors(t *testing.T) {
	ws := catalogWorkspace(t, strings.Replace(cliAPI, `evidence: ["main.go:2"]`, `evidence: []`, 1), cliWeb)
	out, err := run(t, "validate", "--dir", ws)
	if err == nil || err.Error() != "catalog has 1 error(s)" || !strings.Contains(out, "error: catalog/api.yaml: provides[0] GET /hello: no evidence") {
		t.Fatalf("%v\n%s", err, out)
	}
}
