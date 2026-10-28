package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const markedAgents = "# demo\n\n<!-- trama:begin -->\n_none_\n<!-- trama:end -->\n\nRules stay here.\n"

func TestGraphCommandRefusesInvalidCatalog(t *testing.T) {
	ws := catalogWorkspace(t, strings.Replace(cliAPI, `evidence: ["main.go:2"]`, `evidence: []`, 1), cliWeb)
	if err := os.WriteFile(filepath.Join(ws, "AGENTS.md"), []byte(markedAgents), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := run(t, "graph", "--dir", ws)
	if err == nil || err.Error() != `fix 1 validation error(s) first: run "trama validate"` {
		t.Fatalf("err = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(ws, "docs", "graph.md")); !os.IsNotExist(statErr) {
		t.Fatal("docs/graph.md written for an invalid catalog")
	}
}

func TestGraphCommandWrites(t *testing.T) {
	ws := catalogWorkspace(t, cliAPI, cliWeb)
	agents := filepath.Join(ws, "AGENTS.md")
	if err := os.WriteFile(agents, []byte(markedAgents), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "graph", "--dir", ws)
	if err != nil || !strings.Contains(out, "wrote docs/graph.md") || !strings.Contains(out, "updated AGENTS.md") {
		t.Fatalf("%v\n%s", err, out)
	}
	graph1, _ := os.ReadFile(filepath.Join(ws, "docs", "graph.md"))
	agents1, _ := os.ReadFile(agents)
	if _, err := run(t, "graph", "--dir", ws); err != nil {
		t.Fatal(err)
	}
	graph2, _ := os.ReadFile(filepath.Join(ws, "docs", "graph.md"))
	agents2, _ := os.ReadFile(agents)
	if string(graph1) != string(graph2) || string(agents1) != string(agents2) {
		t.Fatal("second run changed the output")
	}
	if !strings.Contains(string(graph1), `n1 -->|"GET /hello"| n0`) || !strings.Contains(string(agents1), "Rules stay here.") || !strings.Contains(string(agents1), "| web | node | — | GET /hello ← api |") {
		t.Fatalf("graph.md:\n%s\nAGENTS.md:\n%s", graph1, agents1)
	}
}
