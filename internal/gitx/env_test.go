package gitx

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

const batchSSH = "GIT_SSH_COMMAND=ssh -o BatchMode=yes"

func TestGitEnvDisablesPrompts(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "")
	t.Setenv("GIT_SSH", "")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	env := gitEnv("", []string{"fetch"})
	for _, want := range []string{"GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=", batchSSH} {
		if !slices.Contains(env, want) {
			t.Errorf("env lacks %q", want)
		}
	}
}

func TestGitEnvKeepsUserSSHCommand(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "ssh -i mykey")
	if slices.Contains(gitEnv("", []string{"fetch"}), batchSSH) {
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
	if slices.Contains(gitEnv("", []string{"fetch"}), batchSSH) {
		t.Fatal("overrode core.sshCommand")
	}
}

func TestGitEnvLocalCommandCannotFetch(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "")
	t.Setenv("GIT_SSH", "")
	env := gitEnv("", []string{"rev-parse", "HEAD"})
	for _, want := range []string{"GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=", "GIT_NO_LAZY_FETCH=1"} {
		if !slices.Contains(env, want) {
			t.Errorf("env lacks %q", want)
		}
	}
	if slices.Contains(env, batchSSH) {
		t.Error("a local command does not run ssh, so it needs no GIT_SSH_COMMAND")
	}
	for _, sub := range []string{"clone", "fetch", "ls-remote", "pull", "push"} {
		if slices.Contains(gitEnv("", []string{sub}), "GIT_NO_LAZY_FETCH=1") {
			t.Errorf("%s talks to the remote and must keep fetching", sub)
		}
	}
}
