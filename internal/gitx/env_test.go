package gitx

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const batchSSH = "GIT_SSH_COMMAND=ssh -o BatchMode=yes"

func TestGitEnvDisablesPrompts(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "")
	t.Setenv("GIT_SSH", "")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	env := gitEnv("")
	for _, want := range []string{"GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=", batchSSH} {
		if !slices.Contains(env, want) {
			t.Errorf("env lacks %q", want)
		}
	}
}

func TestGitEnvKeepsUserSSHCommand(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "ssh -i mykey")
	if slices.Contains(gitEnv(""), batchSSH) {
		t.Fatal("overrode the user's GIT_SSH_COMMAND")
	}
}

func TestGitEnvKeepsConfiguredSSHCommand(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "")
	t.Setenv("GIT_SSH", "")
	cfg := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(cfg, []byte("[core]\n\tsshCommand = ssh -i mykey\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", cfg)
	if slices.Contains(gitEnv(""), batchSSH) {
		t.Fatal("overrode core.sshCommand")
	}
}

// fakeGit puts a git first in PATH that logs each call to calls and, for
// any command but config, prints the GIT_SSH_COMMAND it got.
func fakeGit(t *testing.T) (calls string) {
	t.Helper()
	dir := t.TempDir()
	calls = filepath.Join(dir, "calls")
	script := "#!/bin/sh\necho \"$1\" >> " + calls + "\nif [ \"$1\" = config ]; then exit 1; fi\necho \"$GIT_SSH_COMMAND\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GIT_SSH_COMMAND", "")
	t.Setenv("GIT_SSH", "")
	return calls
}

func TestLocalCommandsKeepBatchModeSSH(t *testing.T) {
	fakeGit(t)
	for _, sub := range []string{"switch", "commit", "merge", "fetch"} {
		out, err := run(t.TempDir(), sub)
		if err != nil || out != "ssh -o BatchMode=yes" {
			t.Errorf("git %s got GIT_SSH_COMMAND=%q (%v), want BatchMode: hooks and LFS can run ssh", sub, out, err)
		}
	}
}

func TestSSHCommandLookupOncePerConfig(t *testing.T) {
	calls := fakeGit(t)
	dir := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(dir, "a"))
	for range 3 {
		if _, err := run(dir, "fetch"); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(dir, "b"))
	if _, err := run(dir, "fetch"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(calls)
	if n := strings.Count(string(data), "config\n"); n != 2 {
		t.Fatalf("core.sshCommand looked up %d times, want once per directory and config (2)", n)
	}
}
