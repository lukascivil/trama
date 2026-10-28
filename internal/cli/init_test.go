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

func TestInitCommandPinsBinaryVersion(t *testing.T) {
	version = "0.0.1"
	t.Cleanup(func() { version = "dev" })
	dir := t.TempDir()
	out, err := run(t, "init", dir, "--name", "demo")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	ws, err := workspace.Load(dir)
	if err != nil || ws.Name != "demo" {
		t.Fatalf("workspace = %+v, %v", ws, err)
	}
	mise, _ := os.ReadFile(filepath.Join(dir, "mise.toml"))
	if !strings.Contains(string(mise), `"github:lukascivil/trama" = "0.0.1"`) {
		t.Fatalf("mise.toml:\n%s", mise)
	}
	if !strings.Contains(out, "workspace.yaml") || !strings.Contains(out, "trama add <url>") {
		t.Fatalf("output:\n%s", out)
	}
}
