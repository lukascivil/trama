package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const cliSpec = `openapi: 3.1.0
paths:
  /hello:
    get:
      responses:
        "200":
          content:
            application/json:
              schema: { type: object, properties: { message: { type: string } } }
  /bye:
    get: {}
`

func TestBodiesWritesWhatTheSpecDeclares(t *testing.T) {
	ws := catalogWorkspace(t, cliAPI, cliWeb)
	writeRepoFile(t, ws, "repos/api/openapi.yaml", cliSpec)

	out, err := run(t, "bodies", "--dry-run", "--dir", ws)
	want := "api: GET /hello <- openapi.yaml:4 (contract, response 1 field)\n" +
		"api: openapi.yaml declares 1 route the card does not list: GET /bye\n"
	if err != nil || out != want {
		t.Fatalf("%v\n%s\nwant:\n%s", err, out, want)
	}
	if card, _ := os.ReadFile(filepath.Join(ws, "catalog/api.yaml")); string(card) != cliAPI {
		t.Fatalf("--dry-run wrote the card:\n%s", card)
	}

	out, err = run(t, "bodies", "api", "--dir", ws)
	if err != nil || out != want+"\nwrote 1 provide; review the diff in catalog/ and run \"trama validate\"\n" {
		t.Fatalf("%v\n%s", err, out)
	}
	card, _ := os.ReadFile(filepath.Join(ws, "catalog/api.yaml"))
	if !strings.Contains(string(card), "    contract: openapi.yaml\n    response:\n      message: string\n") {
		t.Fatalf("card:\n%s", card)
	}
	if out, err := run(t, "validate", "--dir", ws); err != nil || !strings.HasPrefix(out, "ok: 2 card(s), 1 relation(s), 0 warning(s)") {
		t.Fatalf("validate after bodies: %v\n%s", err, out)
	}

	out, err = run(t, "bodies", "--dir", ws)
	if err != nil || out != "api: openapi.yaml declares 1 route the card does not list: GET /bye\nnothing to write\n" {
		t.Fatalf("second run: %v\n%s", err, out)
	}
}

func TestBodiesNothingToWrite(t *testing.T) {
	ws := catalogWorkspace(t, cliAPI, cliWeb)
	out, err := run(t, "bodies", "--dir", ws)
	if err != nil || out != "nothing to write\n" {
		t.Fatalf("%v\n%q", err, out)
	}
}

func TestBodiesUnknownRepo(t *testing.T) {
	ws := catalogWorkspace(t, cliAPI, cliWeb)
	if _, err := run(t, "bodies", "nope", "--dir", ws); err == nil || err.Error() != "repo nope is not in workspace.yaml" {
		t.Fatalf("err = %v", err)
	}
}
