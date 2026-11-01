package status_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/status"
)

func withMise(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	if content != "" {
		if err := os.WriteFile(filepath.Join(root, "mise.toml"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestStalePinSaysNothingWithoutOne(t *testing.T) {
	for _, content := range []string{
		"",
		"[tools]\nnode = \"22\"\n",
		"# \"github:lukascivil/trama\" = \"0.0.8\"\n",
	} {
		if lines := status.StalePin(withMise(t, content)); len(lines) != 0 {
			t.Errorf("mise.toml %q reported %v", content, lines)
		}
	}
}

func TestStalePinReportsAPinnedBinary(t *testing.T) {
	root := withMise(t, "[tools]\n\"github:lukascivil/trama\" = \"0.0.8\"\nnode = \"22\"\n")
	lines := status.StalePin(root)
	if len(lines) != 1 {
		t.Fatalf("lines = %v, want 1", lines)
	}
	if !strings.Contains(lines[0], "mise.toml") || !strings.Contains(lines[0], "plugin") {
		t.Fatalf("line = %q", lines[0])
	}
}

func TestStalePinAcceptsSingleQuotesAndSpacing(t *testing.T) {
	for _, content := range []string{
		"[tools]\n'github:lukascivil/trama'='0.0.8'\n",
		"[tools]\n  \"github:lukascivil/trama\"   =  \"0.0.8\"\n",
	} {
		if lines := status.StalePin(withMise(t, content)); len(lines) != 1 {
			t.Errorf("mise.toml %q reported %v", content, lines)
		}
	}
}
