package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/testutil"
)

const changeID = "2026-11-03-rename-message"

const changeRecord = `---
id: 2026-11-03-rename-message
title: Rename message to greeting in GET /hello
status: open
steps:
  - {order: 1, repo: trama-demo-api, action: Add greeting, branch: trama/rename-message, pr: "https://github.com/lukascivil/trama-demo-api/pull/1", state: open}
  - {order: 2, repo: trama-demo-web, action: Read greeting, branch: trama/rename-message, pr: "https://github.com/lukascivil/trama-demo-web/pull/1", state: open}
  - {order: 3, repo: trama-demo-api, action: Remove message, state: pending}
---

## Why

Say greeting.
`

// isolateGit makes commits in tests independent of the machine's git config.
func isolateGit(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME"} {
		t.Setenv(k, "trama-test")
	}
	for _, k := range []string{"GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "test@example.invalid")
	}
}

// changesWorkspace is an initialized workspace pushed to a bare origin, with
// the record written (not committed). It returns the workspace and the bare.
func changesWorkspace(t *testing.T) (string, string) {
	t.Helper()
	isolateGit(t)
	ws := newWorkspace(t)
	bare := filepath.Join(t.TempDir(), "ws.git")
	testutil.Git(t, filepath.Dir(bare), "init", "-q", "--bare", "-b", "main", bare)
	testutil.Git(t, ws, "remote", "add", "origin", bare)
	testutil.Git(t, ws, "add", "-A")
	testutil.Git(t, ws, "commit", "-q", "-m", "init")
	testutil.Git(t, ws, "push", "-q", "-u", "origin", "main")
	if err := os.WriteFile(filepath.Join(ws, "changes", changeID+".md"), []byte(changeRecord), 0o644); err != nil {
		t.Fatal(err)
	}
	return ws, bare
}

func lastSubject(t *testing.T, repo string) string {
	t.Helper()
	return testutil.Git(t, repo, "log", "-1", "--format=%s", "main")
}

func TestSaveCommitsAndPushes(t *testing.T) {
	ws, bare := changesWorkspace(t)
	if err := os.WriteFile(filepath.Join(ws, "notes.txt"), []byte("unrelated"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "changes", "save", changeID, "--dir", ws)
	if err != nil || !strings.Contains(out, "saved "+changeID) {
		t.Fatalf("%v\n%s", err, out)
	}
	if got := lastSubject(t, bare); got != "change: Rename message to greeting in GET /hello ("+changeID+")" {
		t.Fatalf("remote subject = %q", got)
	}
	if files := testutil.Git(t, bare, "show", "--name-only", "--format=", "main"); files != "changes/"+changeID+".md" {
		t.Fatalf("committed files = %q", files)
	}
}

func TestSaveRefusesOtherBranch(t *testing.T) {
	ws, bare := changesWorkspace(t)
	testutil.Git(t, ws, "switch", "-q", "-c", "x")
	_, err := run(t, "changes", "save", changeID, "--dir", ws)
	if err == nil || err.Error() != "workspace is on x; change records go to main" {
		t.Fatalf("err = %v", err)
	}
	if got := lastSubject(t, bare); got != "init" {
		t.Fatalf("remote got %q", got)
	}
}

func TestSaveWithoutRemote(t *testing.T) {
	ws, _ := changesWorkspace(t)
	testutil.Git(t, ws, "remote", "remove", "origin")
	out, err := run(t, "changes", "save", changeID, "--dir", ws)
	if err != nil || !strings.Contains(out, "committed locally; the workspace has no origin remote") {
		t.Fatalf("%v\n%s", err, out)
	}
	if got := lastSubject(t, ws); !strings.HasPrefix(got, "change: ") {
		t.Fatalf("local subject = %q", got)
	}
}

func TestSaveReportsRejectedPush(t *testing.T) {
	ws, bare := changesWorkspace(t)
	hook := filepath.Join(bare, "hooks", "pre-receive")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := run(t, "changes", "save", changeID, "--dir", ws)
	if err == nil || !strings.Contains(err.Error(), "push to main was rejected") {
		t.Fatalf("err = %v", err)
	}
	if got := lastSubject(t, ws); !strings.HasPrefix(got, "change: ") {
		t.Fatalf("local commit missing: %q", got)
	}
}

func withStates(t *testing.T, f func(string) (string, error)) {
	t.Helper()
	old := stateFunc
	stateFunc = f
	t.Cleanup(func() { stateFunc = old })
}

func TestChangesMarksMerged(t *testing.T) {
	ws, bare := changesWorkspace(t)
	if _, err := run(t, "changes", "save", changeID, "--dir", ws); err != nil {
		t.Fatal(err)
	}
	withStates(t, func(string) (string, error) { return "MERGED", nil })
	out, err := run(t, "changes", "--dir", ws)
	if err != nil || !strings.Contains(out, changeID+": 1 open step(s)") {
		t.Fatalf("%v\n%s", err, out)
	}
	if got := lastSubject(t, bare); got != "change: update status" {
		t.Fatalf("remote subject = %q", got)
	}
	data, _ := os.ReadFile(filepath.Join(ws, "changes", changeID+".md"))
	if strings.Count(string(data), "state: merged") != 2 || !strings.Contains(string(data), "## Why\n\nSay greeting.") {
		t.Fatalf("record:\n%s", data)
	}
}

func TestChangesNoPush(t *testing.T) {
	ws, bare := changesWorkspace(t)
	if _, err := run(t, "changes", "save", changeID, "--dir", ws); err != nil {
		t.Fatal(err)
	}
	withStates(t, func(string) (string, error) { return "MERGED", nil })
	if _, err := run(t, "changes", "--no-push", "--dir", ws); err != nil {
		t.Fatal(err)
	}
	if got := lastSubject(t, ws); got != "change: update status" {
		t.Fatalf("local subject = %q", got)
	}
	if got := lastSubject(t, bare); got == "change: update status" {
		t.Fatal("pushed despite --no-push")
	}
}

func TestChangesSkipsBrokenRecord(t *testing.T) {
	ws, _ := changesWorkspace(t)
	if err := os.WriteFile(filepath.Join(ws, "changes", "2026-11-05-broken.md"), []byte("no front matter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	withStates(t, func(string) (string, error) { return "MERGED", nil })
	out, err := run(t, "changes", "--no-push", "--dir", ws)
	if err != nil || !strings.Contains(out, "warning: 2026-11-05-broken.md") || !strings.Contains(out, changeID+": 1 open step(s)") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestChangesKeepsStateOnGhError(t *testing.T) {
	ws, bare := changesWorkspace(t)
	if _, err := run(t, "changes", "save", changeID, "--dir", ws); err != nil {
		t.Fatal(err)
	}
	withStates(t, func(string) (string, error) { return "", errors.New("gh: not logged in") })
	out, err := run(t, "changes", "--dir", ws)
	if err == nil || !strings.Contains(out, "warning:") || !strings.Contains(out, "trama-demo-api") {
		t.Fatalf("%v\n%s", err, out)
	}
	if got := lastSubject(t, bare); got == "change: update status" {
		t.Fatal("committed although nothing changed")
	}
}

func TestStatusCommandShowsOpenChanges(t *testing.T) {
	ws, _ := changesWorkspace(t)
	out, err := run(t, "status", "--dir", ws)
	if err != nil || !strings.Contains(out, "- change "+changeID+": 3 step(s) not merged") {
		t.Fatalf("%v\n%s", err, out)
	}
	quiet, err := run(t, "status", "--quiet", "--dir", ws)
	if err != nil || !strings.Contains(quiet, "trama: change "+changeID+": 3 step(s) not merged") {
		t.Fatalf("%v\n%s", err, quiet)
	}
}
