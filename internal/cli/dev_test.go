package cli

import (
	"strings"
	"testing"
)

const setupAndDev = "run:\n  setup: { cmd: \"echo S\" }\n  dev: { cmd: \"echo D\" }\n"

func TestDevRunsSetupOnce(t *testing.T) {
	ws := runWorkspace(t, setupAndDev)
	out, err := run(t, "dev", "api", "--dir", ws)
	s, d := strings.Index(out, "$ echo S"), strings.Index(out, "$ echo D")
	if err != nil || s < 0 || d < s {
		t.Fatalf("first run: %v\n%s", err, out)
	}
	out, err = run(t, "dev", "api", "--dir", ws)
	if err != nil || strings.Contains(out, "$ echo S") || !strings.Contains(out, "$ echo D") {
		t.Fatalf("second run: %v\n%s", err, out)
	}
}

func TestDevStopsWhenSetupFails(t *testing.T) {
	ws := runWorkspace(t, "run:\n  setup: { cmd: \"exit 2\" }\n  dev: { cmd: \"echo D\" }\n")
	out, err := run(t, "dev", "api", "--dir", ws)
	if err == nil || strings.Contains(out, "$ echo D") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestDevWarnsStaleCard(t *testing.T) {
	ws := runWorkspace(t, "mapped_at_commit: deadbee\n"+setupAndDev)
	out, err := run(t, "dev", "api", "--dir", ws)
	if err != nil || !strings.HasPrefix(out, "warning: the catalog card of api is stale") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestDevNeedsDevCommand(t *testing.T) {
	ws := runWorkspace(t, "run:\n  setup: { cmd: \"echo S\" }\n")
	if _, err := run(t, "dev", "api", "--dir", ws); err == nil || err.Error() != "api has no run.dev in its catalog card" {
		t.Fatalf("err = %v", err)
	}
}
