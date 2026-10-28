package catalog_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lukascivil/trama/internal/catalog"
)

func TestLoadMissingReturnsNil(t *testing.T) {
	e, err := catalog.Load(t.TempDir(), "api")
	if err != nil || e != nil {
		t.Fatalf("Load = %+v, %v; want nil, nil", e, err)
	}
}

func TestLoadIgnoresUnknownFields(t *testing.T) {
	root := t.TempDir()
	card := `version: 1
name: trama-demo-web
description: Front que mostra a saudação vinda da API
stack: [node, react, typescript]
tools:
  node: "22"
mapped_at_commit: 8b77d02aa1
mapped_at: 2026-10-27
run:
  dev: { cmd: "npm run dev", port: 5173 }
consumes:
  - kind: http
    id: "GET /hello"
    from: trama-demo-api
    via: env
    evidence: ["src/api/hello.ts:5"]
`
	if err := os.MkdirAll(filepath.Join(root, "catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(catalog.Path(root, "trama-demo-web"), []byte(card), 0o644); err != nil {
		t.Fatal(err)
	}
	e, err := catalog.Load(root, "trama-demo-web")
	if err != nil {
		t.Fatal(err)
	}
	if e.Name != "trama-demo-web" || e.MappedAtCommit != "8b77d02aa1" || e.Version != 1 {
		t.Fatalf("entry = %+v", e)
	}
}

func TestLoadReportsBrokenYAML(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(catalog.Path(root, "api"), []byte("name: [unclosed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Load(root, "api"); err == nil {
		t.Fatal("want error for broken YAML")
	}
}
