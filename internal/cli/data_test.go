package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const dataAPI = `version: 1
name: api
description: API
stack: [go]
mapped_at_commit: 1cb5a15
mapped_at: 2026-10-28
identity:
  ports: [{ value: 8080, evidence: ["main.go:1"] }]
  tables:
    - value: users
      columns: [id, email]
      evidence: ["main.go:1"]
provides:
  - kind: http
    id: "GET /hello"
    response: { "customer.email": string }
    evidence: ["main.go:2"]
  - kind: http
    id: "POST /orders"
    request: { email: string }
    evidence: ["main.go:2"]
`

func TestDataFindsResponseRequestAndTable(t *testing.T) {
	ws := catalogWorkspace(t, dataAPI, cliWeb)
	out, err := run(t, "data", "email", "--dir", ws)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, want := range []string{
		"email\n",
		"  response  api  GET /hello  customer.email: string  main.go:2\n",
		"  request   api  POST /orders  email: string  main.go:2\n",
		"  table     api  users.email  main.go:1\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
}

func TestDataOrdersEndpointsBeforeTables(t *testing.T) {
	ws := catalogWorkspace(t, dataAPI, cliWeb)
	out, err := run(t, "data", "email", "--dir", ws)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	response, request, table := strings.Index(out, "  response"), strings.Index(out, "  request"), strings.Index(out, "  table")
	if response < 0 || request < 0 || table < 0 || !(response < request && request < table) {
		t.Fatalf("want response < request < table, got %d %d %d:\n%s", response, request, table, out)
	}
}

func TestDataIsCaseInsensitive(t *testing.T) {
	ws := catalogWorkspace(t, dataAPI, cliWeb)
	out, err := run(t, "data", "EMAIL", "--dir", ws)
	if err != nil || !strings.Contains(out, "  table     api  users.email") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestDataMatchesSubstringNotRegex(t *testing.T) {
	ws := catalogWorkspace(t, dataAPI, cliWeb)
	_, err := run(t, "data", ".*", "--dir", ws)
	if err == nil || err.Error() != `no catalog card has a field or column matching ".*"` {
		t.Fatalf("err = %v", err)
	}
}

func TestDataListsUnmapped(t *testing.T) {
	ws := catalogWorkspace(t, dataAPI, cliWeb)
	if err := os.Remove(filepath.Join(ws, "catalog", "web.yaml")); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "data", "email", "--dir", ws)
	if err != nil || !strings.Contains(out, "not mapped: web\n") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestDataUnknownTerm(t *testing.T) {
	ws := catalogWorkspace(t, dataAPI, cliWeb)
	if _, err := run(t, "data", "cpf", "--dir", ws); err == nil || err.Error() != `no catalog card has a field or column matching "cpf"` {
		t.Fatalf("err = %v", err)
	}
}

func TestDataRejectsEmptyTerm(t *testing.T) {
	ws := catalogWorkspace(t, dataAPI, cliWeb)
	out, err := run(t, "data", "", "--dir", ws)
	if err == nil || !strings.Contains(err.Error(), "give a field or column") {
		t.Fatalf("err = %v, out:\n%s", err, out)
	}
}
