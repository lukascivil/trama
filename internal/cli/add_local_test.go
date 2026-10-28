package cli

import (
	"path/filepath"
	"testing"

	"github.com/lukascivil/trama/internal/testutil"
	"github.com/lukascivil/trama/internal/workspace"
)

func TestAddStoresLocalPathAsAbsolute(t *testing.T) {
	ws := newWorkspace(t)
	remote := testutil.NewRemote(t, "main")
	t.Chdir(filepath.Dir(remote.URL))
	if out, err := run(t, "add", "./remote.git", "--dir", ws); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	w, err := workspace.Load(ws)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := w.Repo("remote")
	want, _ := filepath.EvalSymlinks(remote.URL)
	got, _ := filepath.EvalSymlinks(r.URL)
	if !ok || got != want || !filepath.IsAbs(r.URL) {
		t.Fatalf("stored url = %q, want absolute %q", r.URL, remote.URL)
	}
}
