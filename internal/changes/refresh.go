package changes

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// PRState returns OPEN, MERGED or CLOSED for a pull request URL.
type PRState func(url string) (string, error)

// GHState asks the GitHub CLI for the state of a pull request.
func GHState(url string) (string, error) {
	cmd := exec.Command("gh", "pr", "view", url, "--json", "state", "-q", ".state")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GH_PROMPT_DISABLED=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("gh pr view: %s", msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

var fromGitHub = map[string]string{"OPEN": "open", "MERGED": "merged", "CLOSED": "closed"}

// Refresh sets each step that has a pull request to its current state and
// the record to done once every step is merged. A step whose state cannot
// be read keeps its state and adds an error.
func Refresh(r *Record, state PRState) (changed bool, errs []error) {
	for i := range r.Steps {
		s := &r.Steps[i]
		if s.PR == "" {
			continue
		}
		got, err := state(s.PR)
		if err == nil {
			if mapped, ok := fromGitHub[got]; ok {
				if mapped != s.State {
					s.State = mapped
					changed = true
				}
				continue
			}
			err = fmt.Errorf("unexpected state %q", got)
		}
		errs = append(errs, fmt.Errorf("%s: %s %s: %w", r.ID, s.Repo, s.PR, err))
	}
	status := "open"
	if len(r.Steps) > 0 && len(r.OpenSteps()) == 0 {
		status = "done"
	}
	if status != r.Status {
		r.Status = status
		changed = true
	}
	return changed, errs
}
