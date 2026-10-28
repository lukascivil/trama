package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lukascivil/trama/internal/workspace"
)

func sameDir(t *testing.T, got, want string) {
	t.Helper()
	g, _ := filepath.EvalSymlinks(got)
	w, _ := filepath.EvalSymlinks(want)
	if g != w {
		t.Fatalf("Find = %q, want %q", got, want)
	}
}

func TestFindSkipsForeignWorkspaceYAML(t *testing.T) {
	root := writeWorkspace(t, twoRepos)
	inner := filepath.Join(root, "repos", "x")
	if err := os.MkdirAll(filepath.Join(inner, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inner, workspace.FileName), []byte("load_from: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := workspace.Find(filepath.Join(inner, "src"))
	if err != nil {
		t.Fatal(err)
	}
	sameDir(t, got, root)
}

func TestFindStopsAtOtherVersion(t *testing.T) {
	root := writeWorkspace(t, "version: 2\nname: future\nrepos: []\n")
	got, err := workspace.Find(root)
	if err != nil {
		t.Fatal(err)
	}
	sameDir(t, got, root)
}

func TestLocalURL(t *testing.T) {
	cases := map[string]string{
		"https://github.com/x/y.git": "https://github.com/x/y.git",
		"git@github.com:x/y.git":     "git@github.com:x/y.git",
		"github.com:x/y.git":         "github.com:x/y.git",
		"../y.git":                   "/work/y.git",
		"./y.git":                    "/work/sub/y.git",
		"/abs/y.git":                 "/abs/y.git",
		"--upload-pack=x":            "--upload-pack=x",
	}
	for in, want := range cases {
		if got := workspace.LocalURL(in, "/work/sub"); got != want {
			t.Errorf("LocalURL(%q) = %q, want %q", in, got, want)
		}
	}
}
