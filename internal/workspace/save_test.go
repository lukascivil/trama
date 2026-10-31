package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lukascivil/trama/internal/workspace"
)

func TestSaveDoesNotFollowSymlink(t *testing.T) {
	root := writeWorkspace(t, twoRepos)
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, workspace.FileName+".tmp")); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.AddRepo(workspace.Repo{Name: "new", URL: "https://github.com/x/new.git", Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.Save(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(outside); string(b) != "keep" {
		t.Fatalf("outside file overwritten: %q", b)
	}
}
