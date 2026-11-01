package github

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// Gh runs the GitHub CLI with args and returns what it printed without the
// final line break. Tests replace it.
type Gh func(args ...string) (string, error)

// ExecGh runs the gh binary on the PATH. Its error output is dropped, so a
// failure never carries anything gh printed.
func ExecGh(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimRight(out.String(), "\r\n"), nil
}

// Token returns the token to use for the API: the one gh holds, else GH_TOKEN,
// else GITHUB_TOKEN, else "" (anonymous).
func Token(gh Gh, getenv func(string) string) string {
	if out, err := gh("auth", "token"); err == nil && out != "" {
		return out
	}
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if v := getenv(name); v != "" {
			return v
		}
	}
	return ""
}

// Protocol returns "ssh" when gh is set to clone over ssh, else "https".
func Protocol(gh Gh) string {
	if out, err := gh("config", "get", "git_protocol"); err == nil && out == "ssh" {
		return "ssh"
	}
	return "https"
}
