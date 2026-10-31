package catalog_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/catalog"
)

const webCard = `version: 1
name: trama-demo-web
description: Frontend that shows the greeting from the API
stack: [node, react, typescript]
tools:
  node: "22"
mapped_at_commit: 8b77d02
mapped_at: 2026-10-27
run:
  dev:  { cmd: "npm run dev", port: 5173 }
  test: { cmd: "npm test" }
  setup: { cmd: "npm install" }
provides: []
consumes:
  - kind: http
    id: "GET /hello"
    from: trama-demo-api
    via: env
    fields_used: [message]
    evidence: ["src/api/hello.ts:5", "src/App.tsx:12"]
config:
  - env: VITE_API_URL
    value: "http://localhost:8080"
    points_to: trama-demo-api
    evidence: [".env.example:1"]
`

const apiCard = `version: 1
name: trama-demo-api
description: Backend that serves the greeting
stack: [go]
tools:
  go: "1.25"
mapped_at_commit: 3f2a91c
mapped_at: 2026-10-27
run:
  dev: { cmd: "go run ./cmd/api", port: 8080 }
  test: { cmd: "go test ./..." }
identity:
  ports: [{ value: 8080, evidence: ["cmd/api/main.go:10"] }]
  hosts: []
  package: null
  topics: []
provides:
  - kind: http
    id: "GET /hello"
    contract: null
    shape: { message: string }
    evidence: ["internal/http/hello.go:14"]
`

func writeCard(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(catalog.Path(root, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadFullCard(t *testing.T) {
	root := t.TempDir()
	writeCard(t, root, "trama-demo-web", webCard)
	c, err := catalog.LoadStrict(root, "trama-demo-web")
	if err != nil {
		t.Fatal(err)
	}
	want := catalog.Consume{Kind: "http", ID: "GET /hello", From: "trama-demo-api", Via: "env",
		FieldsUsed: []string{"message"}, Evidence: []string{"src/api/hello.ts:5", "src/App.tsx:12"}}
	if len(c.Consumes) != 1 || !reflect.DeepEqual(c.Consumes[0], want) {
		t.Fatalf("consumes = %+v", c.Consumes)
	}
	if c.Config[0].Value != "http://localhost:8080" || c.Config[0].PointsTo != "trama-demo-api" {
		t.Fatalf("config = %+v", c.Config)
	}
	if c.Run["dev"] != (catalog.Command{Cmd: "npm run dev", Port: 5173}) || c.MappedAt != "2026-10-27" || c.Tools["node"] != "22" {
		t.Fatalf("card = %+v", c)
	}
}

func TestLoadIdentity(t *testing.T) {
	root := t.TempDir()
	writeCard(t, root, "trama-demo-api", apiCard)
	c, err := catalog.LoadStrict(root, "trama-demo-api")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Identity.Ports) != 1 || c.Identity.Ports[0].Value != 8080 || c.Identity.Ports[0].Evidence[0] != "cmd/api/main.go:10" {
		t.Fatalf("identity = %+v", c.Identity)
	}
	if c.Identity.Package != nil || c.Provides[0].Contract != "" || c.Provides[0].Shape["message"] != "string" {
		t.Fatalf("card = %+v", c)
	}
}

func TestLoadStrictRejectsUnknownField(t *testing.T) {
	root := t.TempDir()
	writeCard(t, root, "trama-demo-web", strings.Replace(webCard, `    evidence: ["src/api`, `    evidnce: ["src/api`, 1))
	_, err := catalog.LoadStrict(root, "trama-demo-web")
	if err == nil || !strings.Contains(err.Error(), "evidnce") || !strings.Contains(err.Error(), "line ") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadToleratesUnknownField(t *testing.T) {
	root := t.TempDir()
	writeCard(t, root, "trama-demo-web", webCard+"extra: field\n")
	if c, err := catalog.Load(root, "trama-demo-web"); err != nil || c.Name != "trama-demo-web" {
		t.Fatalf("Load = %+v, %v", c, err)
	}
}

func TestLoadStrictMissingReturnsNil(t *testing.T) {
	if c, err := catalog.LoadStrict(t.TempDir(), "x"); c != nil || err != nil {
		t.Fatalf("LoadStrict = %+v, %v", c, err)
	}
}

func TestNames(t *testing.T) {
	root := t.TempDir()
	writeCard(t, root, "b", "version: 1\n")
	writeCard(t, root, "a", "version: 1\n")
	for _, extra := range []string{"notes.md", ".gitkeep"} {
		if err := os.WriteFile(filepath.Join(root, "catalog", extra), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := catalog.Names(root)
	if err != nil || !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("Names = %v, %v", got, err)
	}
}

func TestLoadStrictRejectsSecondDocument(t *testing.T) {
	root := t.TempDir()
	writeCard(t, root, "trama-demo-web", webCard+"---\nname: x\n")
	if _, err := catalog.LoadStrict(root, "trama-demo-web"); err == nil || !strings.Contains(err.Error(), "more than one YAML document") {
		t.Fatalf("err = %v", err)
	}
}

func TestIsCommitSHA(t *testing.T) {
	ok := []string{"1cb5a15", strings.Repeat("a", 40)}
	bad := []string{"main", "1CB5A15", "abc", strings.Repeat("a", 41), ""}
	for _, s := range ok {
		if !catalog.IsCommitSHA(s) {
			t.Errorf("IsCommitSHA(%q) = false", s)
		}
	}
	for _, s := range bad {
		if catalog.IsCommitSHA(s) {
			t.Errorf("IsCommitSHA(%q) = true", s)
		}
	}
}
