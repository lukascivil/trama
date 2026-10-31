package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/testutil"
)

func TestSaveRefusesUnpushedNonRecordCommits(t *testing.T) {
	ws, bare := changesWorkspace(t)
	if err := os.WriteFile(filepath.Join(ws, "catalog", "wip.yaml"), []byte("x: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.Git(t, ws, "add", "catalog/wip.yaml")
	testutil.Git(t, ws, "commit", "-q", "-m", "wip")
	_, err := run(t, "changes", "save", changeID, "--dir", ws)
	if err == nil || !strings.Contains(err.Error(), "unpushed commits outside changes/") {
		t.Fatalf("err = %v", err)
	}
	if got := lastSubject(t, bare); got != "init" {
		t.Fatalf("remote got %q", got)
	}
	if got := lastSubject(t, ws); !strings.HasPrefix(got, "change: ") {
		t.Fatalf("record not committed locally: %q", got)
	}
}

func TestSaveLeavesStagedFilesAlone(t *testing.T) {
	ws, bare := changesWorkspace(t)
	if err := os.WriteFile(filepath.Join(ws, "docs", "staged.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.Git(t, ws, "add", "docs/staged.md")
	if _, err := run(t, "changes", "save", changeID, "--dir", ws); err != nil {
		t.Fatal(err)
	}
	if files := testutil.Git(t, bare, "show", "--name-only", "--format=", "main"); files != "changes/"+changeID+".md" {
		t.Fatalf("pushed files = %q", files)
	}
	if staged := testutil.Git(t, ws, "diff", "--cached", "--name-only"); staged != "docs/staged.md" {
		t.Fatalf("staged = %q", staged)
	}
}

func TestSaveOnMasterWorkspace(t *testing.T) {
	isolateGit(t)
	ws := t.TempDir()
	testutil.Git(t, ws, "init", "-q", "-b", "master")
	if out, err := run(t, "init", ws); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	bare := filepath.Join(t.TempDir(), "ws.git")
	testutil.Git(t, filepath.Dir(bare), "init", "-q", "--bare", "-b", "master", bare)
	testutil.Git(t, ws, "remote", "add", "origin", bare)
	testutil.Git(t, ws, "add", "-A")
	testutil.Git(t, ws, "commit", "-q", "-m", "init")
	testutil.Git(t, ws, "push", "-q", "-u", "origin", "master")
	if err := os.WriteFile(filepath.Join(ws, "changes", changeID+".md"), []byte(changeRecord), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := run(t, "changes", "save", changeID, "--dir", ws); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if got := testutil.Git(t, bare, "log", "-1", "--format=%s", "master"); !strings.HasPrefix(got, "change: ") {
		t.Fatalf("remote master = %q", got)
	}
}

func TestSaveReportsBehindOrigin(t *testing.T) {
	ws, bare := changesWorkspace(t)
	other := filepath.Join(t.TempDir(), "other")
	testutil.Git(t, filepath.Dir(other), "clone", "-q", bare, other)
	if err := os.WriteFile(filepath.Join(other, "notes.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.Git(t, other, "add", "notes.md")
	testutil.Git(t, other, "commit", "-q", "-m", "teammate")
	testutil.Git(t, other, "push", "-q", "origin", "main")
	_, err := run(t, "changes", "save", changeID, "--dir", ws)
	if err == nil || !strings.Contains(err.Error(), "git pull --rebase") {
		t.Fatalf("err = %v", err)
	}
}

func TestChangesRefusesOtherBranchBeforeWriting(t *testing.T) {
	ws, _ := changesWorkspace(t)
	if _, err := run(t, "changes", "save", changeID, "--dir", ws); err != nil {
		t.Fatal(err)
	}
	testutil.Git(t, ws, "switch", "-q", "-c", "side")
	withStates(t, func(string) (string, error) { return "MERGED", nil })
	if _, err := run(t, "changes", "--dir", ws); err == nil {
		t.Fatal("changes ran on another branch")
	}
	if dirty := testutil.Git(t, ws, "status", "--porcelain", "--", "changes/"); dirty != "" {
		t.Fatalf("record written on the wrong branch: %q", dirty)
	}
}

func TestChangesRetriesUnpushed(t *testing.T) {
	ws, bare := changesWorkspace(t)
	if _, err := run(t, "changes", "save", changeID, "--dir", ws); err != nil {
		t.Fatal(err)
	}
	withStates(t, func(string) (string, error) { return "MERGED", nil })
	if _, err := run(t, "changes", "--no-push", "--dir", ws); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "changes", "--dir", ws); err != nil {
		t.Fatal(err)
	}
	if got := lastSubject(t, bare); got != "change: update status" {
		t.Fatalf("remote subject = %q", got)
	}
}

func TestChangesCommitsDirtyRecord(t *testing.T) {
	ws, bare := changesWorkspace(t)
	if _, err := run(t, "changes", "save", changeID, "--dir", ws); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(ws, "changes", changeID+".md")
	data, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(data), "Say greeting.", "Say greeting, please.", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	withStates(t, func(string) (string, error) { return "OPEN", nil })
	if _, err := run(t, "changes", "--dir", ws); err != nil {
		t.Fatal(err)
	}
	if got := lastSubject(t, bare); got != "change: update status" {
		t.Fatalf("remote subject = %q", got)
	}
}

func TestChangesFailsWhenStatesUnreadable(t *testing.T) {
	ws, _ := changesWorkspace(t)
	if _, err := run(t, "changes", "save", changeID, "--dir", ws); err != nil {
		t.Fatal(err)
	}
	withStates(t, func(string) (string, error) { return "", os.ErrNotExist })
	if _, err := run(t, "changes", "--dir", ws); err == nil || !strings.Contains(err.Error(), "could not read the state of 2 pull request(s)") {
		t.Fatalf("err = %v", err)
	}
}

func TestSaveErrorMatchesExactRecord(t *testing.T) {
	ws, _ := changesWorkspace(t)
	if err := os.WriteFile(filepath.Join(ws, "changes", "2026-11-04-rename-message.md"), []byte("broken\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "changes", "save", "message", "--dir", ws); err == nil || err.Error() != "no change record changes/message.md" {
		t.Fatalf("err = %v", err)
	}
}

func TestBranchRefusesOtherBranch(t *testing.T) {
	ws, _ := branchWorkspace(t)
	testutil.Git(t, filepath.Join(ws, "repos", "api"), "switch", "-q", "-c", "my-feature")
	_, err := run(t, "branch", "api", "trama/x", "--dir", ws)
	if err == nil || err.Error() != "repos/api is on my-feature; switch it to main first" {
		t.Fatalf("err = %v", err)
	}
	if b, _ := repoOf(ws).CurrentBranch(); b != "my-feature" {
		t.Fatalf("branch = %q", b)
	}
}

func TestPRChecksBodyBeforePush(t *testing.T) {
	fakeGH(t)
	ws, remote := branchWorkspace(t)
	if _, err := run(t, "branch", "api", "trama/x", "--dir", ws); err != nil {
		t.Fatal(err)
	}
	commitIn(t, filepath.Join(ws, "repos", "api"))
	if _, err := run(t, "pr", "api", "--title", "T", "--body-file", filepath.Join(t.TempDir(), "missing.md"), "--dir", ws); err == nil {
		t.Fatal("missing body file accepted")
	}
	if heads := testutil.Git(t, remote.URL, "for-each-ref", "--format=%(refname)", "refs/heads"); heads != "refs/heads/main" {
		t.Fatalf("pushed before checking the body: %q", heads)
	}
}
