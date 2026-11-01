package scaffold_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/scaffold"
	"github.com/lukascivil/trama/internal/workspace"
)

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestInitCreatesLayout(t *testing.T) {
	dir := t.TempDir()
	created, err := scaffold.Init(scaffold.Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".claude/settings.json", ".gitignore", "AGENTS.md", "CLAUDE.md",
		"catalog/.gitkeep", "changes/.gitkeep", "docs/.gitkeep", "workspace.yaml"}
	if !reflect.DeepEqual(created, want) {
		t.Fatalf("created = %v\nwant      %v", created, want)
	}
	if !(gitx.Repo{Dir: dir}).IsRepo() {
		t.Fatal("git repository not initialized")
	}
	ws, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != filepath.Base(dir) || len(ws.Repos) != 0 {
		t.Fatalf("workspace = %+v", ws)
	}
	if gi := read(t, filepath.Join(dir, ".gitignore")); !strings.Contains(gi, "repos/") {
		t.Fatalf(".gitignore = %q", gi)
	}
	if c := read(t, filepath.Join(dir, "CLAUDE.md")); strings.TrimSpace(c) != "@AGENTS.md" {
		t.Fatalf("CLAUDE.md = %q", c)
	}
	agents := read(t, filepath.Join(dir, "AGENTS.md"))
	if !strings.Contains(agents, "<!-- trama:begin -->") || !strings.Contains(agents, "<!-- trama:end -->") {
		t.Fatalf("AGENTS.md lacks generated-section markers:\n%s", agents)
	}
	if !strings.Contains(agents, "## Running") || !strings.Contains(agents, "`trama up`") {
		t.Fatalf("AGENTS.md lacks the Running section:\n%s", agents)
	}
	var settings struct {
		EnabledPlugins map[string]bool `json:"enabledPlugins"`
	}
	if err := json.Unmarshal([]byte(read(t, filepath.Join(dir, ".claude", "settings.json"))), &settings); err != nil {
		t.Fatal(err)
	}
	if !settings.EnabledPlugins["trama@trama"] {
		t.Fatalf("settings = %+v", settings)
	}
}

func TestInitUsesGivenName(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(scaffold.Options{Dir: dir, Name: "acme"}); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Load(dir)
	if err != nil || ws.Name != "acme" {
		t.Fatalf("workspace = %+v, %v", ws, err)
	}
	if agents := read(t, filepath.Join(dir, "AGENTS.md")); !strings.HasPrefix(agents, "# acme — trama workspace") {
		t.Fatalf("AGENTS.md title:\n%s", agents)
	}
}

// The plugin shim runs the binary whose version matches the plugin, so a
// version in the workspace could only ever disagree with it.
func TestInitWritesNoMiseFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(scaffold.Options{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mise.toml")); !os.IsNotExist(err) {
		t.Fatalf("mise.toml was created: %v", err)
	}
}

func TestInitAllowsExistingGit(t *testing.T) {
	dir := t.TempDir()
	if err := gitx.Init(dir, "main"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".DS_Store"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := scaffold.Init(scaffold.Options{Dir: dir}); err != nil {
		t.Fatal(err)
	}
}

func TestInitRefusesNonEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := scaffold.Init(scaffold.Options{Dir: dir})
	if !errors.Is(err, scaffold.ErrNotEmpty) || !strings.Contains(err.Error(), "README.md") {
		t.Fatalf("err = %v, want ErrNotEmpty naming README.md", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, workspace.FileName)); !os.IsNotExist(statErr) {
		t.Fatal("workspace.yaml was created in a non-empty directory")
	}
}
