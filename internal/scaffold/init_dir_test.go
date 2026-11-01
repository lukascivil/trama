package scaffold_test

import (
	"path/filepath"
	"testing"

	"github.com/lukascivil/trama/internal/scaffold"
	"github.com/lukascivil/trama/internal/workspace"
)

func TestInitCreatesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new", "ws")
	if _, err := scaffold.Init(scaffold.Options{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.Load(dir); err != nil {
		t.Fatal(err)
	}
}

func TestInitQuotesSpecialNames(t *testing.T) {
	for _, name := range []string{"null", "true", "123", "~"} {
		dir := t.TempDir()
		if _, err := scaffold.Init(scaffold.Options{Dir: dir, Name: name}); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		ws, err := workspace.Load(dir)
		if err != nil || ws.Name != name {
			t.Errorf("name %q: loaded %+v, %v", name, ws, err)
		}
	}
}
