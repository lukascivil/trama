package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/workspace"
)

// run executes the trama command with args and returns its output.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestInitCommandCreatesWorkspace(t *testing.T) {
	dir := t.TempDir()
	out, err := run(t, "init", dir, "--name", "demo")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	ws, err := workspace.Load(dir)
	if err != nil || ws.Name != "demo" {
		t.Fatalf("workspace = %+v, %v", ws, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mise.toml")); !os.IsNotExist(err) {
		t.Fatalf("init pinned a binary version in the workspace: %v", err)
	}
	if !strings.Contains(out, "workspace.yaml") || !strings.Contains(out, "trama add <url>") {
		t.Fatalf("output:\n%s", out)
	}
}
