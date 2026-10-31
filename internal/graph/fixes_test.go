package graph_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lukascivil/trama/internal/graph"
)

func TestReplaceMarkedNeedsWholeLineMarkers(t *testing.T) {
	doc := "The table between `<!-- trama:begin -->` and `<!-- trama:end -->` is generated.\n\n<!-- trama:begin -->\nold\n<!-- trama:end -->\n"
	got, err := graph.ReplaceMarked(doc, "new")
	if err != nil {
		t.Fatal(err)
	}
	want := "The table between `<!-- trama:begin -->` and `<!-- trama:end -->` is generated.\n\n<!-- trama:begin -->\nnew\n<!-- trama:end -->\n"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	twice := "<!-- trama:begin -->\na\n<!-- trama:end -->\n<!-- trama:begin -->\nb\n<!-- trama:end -->\n"
	if _, err := graph.ReplaceMarked(twice, "x"); err == nil {
		t.Fatal("duplicate markers accepted")
	}
}

func TestGraphDoesNotFollowSymlinks(t *testing.T) {
	ws := demoWorkspace(t)
	if err := os.WriteFile(filepath.Join(ws.Root, "AGENTS.md"), []byte("<!-- trama:begin -->\n<!-- trama:end -->\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(ws.Root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(ws.Root, "docs", "graph.md")); err != nil {
		t.Fatal(err)
	}
	if err := graph.Write(ws); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(outside); string(b) != "keep" {
		t.Fatalf("file outside the workspace was overwritten: %q", b)
	}
	if fi, err := os.Lstat(filepath.Join(ws.Root, "docs", "graph.md")); err != nil || !fi.Mode().IsRegular() {
		t.Fatalf("docs/graph.md is not a regular file: %v %v", fi, err)
	}
}
