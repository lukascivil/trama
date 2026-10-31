package runner_test

import (
	"testing"

	"github.com/lukascivil/trama/internal/runner"
	"github.com/lukascivil/trama/internal/syncer"
)

func setupJob(t *testing.T, ws interface{ RepoDir(string) string }, cmd string) runner.Job {
	t.Helper()
	return runner.Job{Repo: "api", Kind: runner.Setup, Dir: ws.RepoDir("api"), Cmd: cmd}
}

func TestNeedsSetupFirstTime(t *testing.T) {
	ws, _ := newRepoWS(t, runCard, true)
	j := setupJob(t, ws, "npm install")
	if need, err := runner.NeedsSetup(ws, j); err != nil || !need {
		t.Fatalf("first time NeedsSetup = %v, %v", need, err)
	}
	if err := runner.MarkSetup(ws, j); err != nil {
		t.Fatal(err)
	}
	if need, err := runner.NeedsSetup(ws, j); err != nil || need {
		t.Fatalf("after MarkSetup NeedsSetup = %v, %v", need, err)
	}
}

func TestNeedsSetupAfterHeadMoves(t *testing.T) {
	ws, remote := newRepoWS(t, runCard, true)
	j := setupJob(t, ws, "npm install")
	if err := runner.MarkSetup(ws, j); err != nil {
		t.Fatal(err)
	}
	remote.Commit(t, "package.json", "{}")
	if _, err := syncer.Sync(ws, nil); err != nil {
		t.Fatal(err)
	}
	if need, err := runner.NeedsSetup(ws, j); err != nil || !need {
		t.Fatalf("after new commits NeedsSetup = %v, %v", need, err)
	}
}

func TestNeedsSetupWhenCommandChanges(t *testing.T) {
	ws, _ := newRepoWS(t, runCard, true)
	if err := runner.MarkSetup(ws, setupJob(t, ws, "npm install")); err != nil {
		t.Fatal(err)
	}
	if need, err := runner.NeedsSetup(ws, setupJob(t, ws, "npm ci")); err != nil || !need {
		t.Fatalf("after command change NeedsSetup = %v, %v", need, err)
	}
}
