package runner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/testutil"
	"github.com/lukascivil/trama/internal/workspace"
)

// newRepoWS creates a workspace with repo "api" cloned from a fresh remote;
// card, when not empty, is written to catalog/api.yaml.
func newRepoWS(t *testing.T, card string, clone bool) (*workspace.Workspace, *testutil.Remote) {
	t.Helper()
	remote := testutil.NewRemote(t, "main")
	root := t.TempDir()
	yml := "version: 1\nname: t\nrepos:\n  - {name: api, url: \"" + remote.URL + "\", branch: main}\n"
	if err := os.WriteFile(filepath.Join(root, workspace.FileName), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if clone {
		if err := gitx.Clone(remote.URL, "main", ws.RepoDir("api")); err != nil {
			t.Fatal(err)
		}
	}
	if card != "" {
		if err := os.MkdirAll(filepath.Join(root, "catalog"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(catalog.Path(root, "api"), []byte(card), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return ws, remote
}

const runCard = `version: 1
name: api
description: API
stack: [node]
tools:
  node: "24"
run:
  setup: { cmd: "npm install" }
  dev: { cmd: "npm run dev", port: 5173 }
`
