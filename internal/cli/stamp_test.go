package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/testutil"
)

func TestStampCommand(t *testing.T) {
	ws := newWorkspace(t)
	remote := testutil.NewRemote(t, "main")
	for _, args := range [][]string{{"add", remote.URL, "--dir", ws, "--name", "api"}, {"sync", "--dir", ws}} {
		if out, err := run(t, args...); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	card := "version: 1\nname: api\ndescription: API\nstack: [go]\n"
	if err := os.WriteFile(filepath.Join(ws, "catalog", "api.yaml"), []byte(card), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "stamp", "api", "--dir", ws)
	if err != nil || !strings.HasPrefix(out, "stamped api at ") {
		t.Fatalf("%v\n%s", err, out)
	}
}
