package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusMentionsALeftoverBinaryPin(t *testing.T) {
	dir := t.TempDir()
	if _, err := run(t, "init", dir, "--name", "demo"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mise.toml"), []byte("[tools]\n\"github:lukascivil/trama\" = \"0.0.8\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "-C", dir, "status")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "mise.toml pins the trama binary") {
		t.Fatalf("status did not mention the leftover pin:\n%s", out)
	}
	quiet, err := run(t, "-C", dir, "status", "--quiet")
	if err != nil {
		t.Fatalf("%v\n%s", err, quiet)
	}
	if strings.Contains(quiet, "mise.toml") {
		t.Fatalf("--quiet mentioned the pin:\n%s", quiet)
	}
}
