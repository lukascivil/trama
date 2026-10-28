package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/testutil"
)

func TestSyncCommandFailsWhenARepoFails(t *testing.T) {
	ws := newWorkspace(t)
	good := testutil.NewRemote(t, "main")
	if out, err := run(t, "add", good.URL, "--dir", ws, "--name", "good"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	missing := filepath.Join(t.TempDir(), "missing")
	if out, err := run(t, "add", missing, "--dir", ws, "--name", "bad", "--branch", "main"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	out, err := run(t, "sync", "--dir", ws)
	if err == nil || err.Error() != "sync failed for 1 repo(s)" {
		t.Fatalf("err = %v\n%s", err, out)
	}
	for _, want := range []string{"good", "cloned", "bad", "failed"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
}
