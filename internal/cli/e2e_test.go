package cli

import (
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/testutil"
)

func TestInitAddSyncStatus(t *testing.T) {
	ws := newWorkspace(t)
	api := testutil.NewRemote(t, "main")
	web := testutil.NewRemote(t, "master")
	for _, args := range [][]string{
		{"add", api.URL, "--dir", ws, "--name", "api"},
		{"add", web.URL, "--dir", ws, "--name", "web"},
	} {
		if out, err := run(t, args...); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	out, err := run(t, "sync", "--dir", ws)
	if err != nil || strings.Count(out, "cloned") != 2 {
		t.Fatalf("sync: %v\n%s", err, out)
	}
	out, err = run(t, "status", "--dir", ws)
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	for _, want := range []string{"api", "web", "master", `- api: no catalog card: run "/trama:map api"`, `- web: no catalog card: run "/trama:map web"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("status output lacks %q:\n%s", want, out)
		}
	}
	quiet, err := run(t, "status", "--quiet", "--dir", ws)
	if err != nil || strings.Count(quiet, "\n") != 2 || !strings.HasPrefix(quiet, "trama: api: ") {
		t.Fatalf("status --quiet: %v\n%s", err, quiet)
	}
}

func TestStatusOutsideWorkspaceFails(t *testing.T) {
	if _, err := run(t, "status", "--dir", t.TempDir()); err == nil {
		t.Fatal("status outside a workspace succeeded")
	}
}
