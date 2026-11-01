package scaffold_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/scaffold"
)

func TestInitAllowsClaudeDirWithoutSettings(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, ".claude", "settings.local.json")
	if err := os.MkdirAll(filepath.Dir(local), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(local, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := scaffold.Init(scaffold.Options{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(local); string(b) != "{}" {
		t.Fatalf("settings.local.json changed: %q", b)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "settings.json")); err != nil {
		t.Fatal(err)
	}
}

func TestInitRefusesExistingClaudeSettings(t *testing.T) {
	dir := t.TempDir()
	settings := filepath.Join(dir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := scaffold.Init(scaffold.Options{Dir: dir})
	if !errors.Is(err, scaffold.ErrNotEmpty) || !strings.Contains(err.Error(), ".claude/settings.json") {
		t.Fatalf("err = %v", err)
	}
	if b, _ := os.ReadFile(settings); string(b) != "{}" {
		t.Fatalf("settings.json overwritten: %q", b)
	}
}
