package syncer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/syncer"
	"github.com/lukascivil/trama/internal/testutil"
	"github.com/lukascivil/trama/internal/workspace"
)

func TestSyncWithURLRewrite(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	cfg := filepath.Join(t.TempDir(), "gitconfig")
	rule := "[url \"" + filepath.Dir(remote.URL) + "/\"]\n\tinsteadOf = https://example.invalid/org/\n"
	if err := os.WriteFile(cfg, []byte(rule), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", cfg)
	ws := newWorkspace(t, workspace.Repo{Name: "a", URL: "https://example.invalid/org/remote.git", Branch: "main"})
	if got := mustSync(t, ws)[0]; got.Outcome != syncer.Cloned {
		t.Fatalf("first sync = %+v", got)
	}
	if got := mustSync(t, ws)[0]; got.Outcome != syncer.UpToDate {
		t.Fatalf("second sync = %+v, want up-to-date", got)
	}
}

func TestSyncReportsMissingUpstreamBranch(t *testing.T) {
	remote := testutil.NewRemote(t, "master")
	ws := newWorkspace(t, repoFor("a", remote, "master"))
	mustSync(t, ws)
	testutil.Git(t, remote.URL, "branch", "-m", "master", "main")
	got := mustSync(t, ws)[0]
	if got.Outcome != syncer.Failed || !strings.Contains(got.Detail, "origin/master no longer exists") || !strings.Contains(got.Detail, "main") {
		t.Fatalf("result = %+v", got)
	}
}
