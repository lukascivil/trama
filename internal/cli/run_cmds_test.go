package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/runner"
	"github.com/lukascivil/trama/internal/testutil"
	"github.com/lukascivil/trama/internal/workspace"
)

// runWorkspace creates a workspace with repo "api" cloned and its card set
// to card (a card body without version/name header lines).
func runWorkspace(t *testing.T, card string) string {
	t.Helper()
	ws := newWorkspace(t)
	remote := testutil.NewRemote(t, "main")
	for _, args := range [][]string{{"add", remote.URL, "--dir", ws, "--name", "api"}, {"sync", "--dir", ws}} {
		if out, err := run(t, args...); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	full := "version: 1\nname: api\ndescription: API\nstack: [sh]\n" + card
	if err := os.WriteFile(filepath.Join(ws, "catalog", "api.yaml"), []byte(full), 0o644); err != nil {
		t.Fatal(err)
	}
	return ws
}

func TestSetupCommandMarks(t *testing.T) {
	ws := runWorkspace(t, "run:\n  setup: { cmd: \"echo setup\" }\n")
	out, err := run(t, "setup", "--dir", ws)
	if err != nil || !strings.Contains(out, "$ echo setup  (in repos/api)") || !strings.Contains(out, "api: ok") {
		t.Fatalf("%v\n%s", err, out)
	}
	w, _ := workspace.Load(ws)
	j, err := runner.JobFor(w, "api", runner.Setup)
	if err != nil {
		t.Fatal(err)
	}
	if need, _ := runner.NeedsSetup(w, j); need {
		t.Fatal("setup not marked as done")
	}
}

func TestSetupCommandSkipsMissing(t *testing.T) {
	ws := runWorkspace(t, "run:\n  dev: { cmd: \"echo dev\" }\n")
	out, err := run(t, "setup", "--dir", ws)
	if err != nil || !strings.Contains(out, "api: no run.setup, skipped") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestTestCommandReportsFailure(t *testing.T) {
	ws := runWorkspace(t, "run:\n  test: { cmd: \"exit 1\" }\n")
	out, err := run(t, "test", "--dir", ws)
	if err == nil || err.Error() != "tests failed for 1 repo(s)" || !strings.Contains(out, "api: failed (exit 1)") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestTestCommandPasses(t *testing.T) {
	ws := runWorkspace(t, "run:\n  test: { cmd: \"true\" }\n")
	out, err := run(t, "test", "api", "--dir", ws)
	if err != nil || !strings.Contains(out, "api: passed") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestTestRunsSetupFirst(t *testing.T) {
	ws := runWorkspace(t, "run:\n  setup: { cmd: \"touch installed\" }\n  test: { cmd: \"test -f installed\" }\n")
	out, err := run(t, "test", "--dir", ws)
	if err != nil || !strings.Contains(out, "$ touch installed  (in repos/api)") || !strings.Contains(out, "api: passed") {
		t.Fatalf("%v\n%s", err, out)
	}
}
