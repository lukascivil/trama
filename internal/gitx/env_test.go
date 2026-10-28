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
