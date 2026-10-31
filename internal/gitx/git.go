// Package gitx runs the git binary on the user's machine.
//
// Every command runs with prompts disabled (no terminal prompt, no askpass
// helper, ssh in batch mode unless the user configured ssh) so a private
// or missing remote fails fast instead of waiting for credentials. URLs are
// always passed after "--" so they are never read as options. Nothing here
// discards local work: there is no reset, clean, stash or forced checkout.
package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ErrDiverged means the local branch has commits origin does not have.
var ErrDiverged = errors.New("cannot fast-forward: local branch diverged from origin")

// ErrNoUpstream means origin has no branch with the requested name.
var ErrNoUpstream = errors.New("origin has no such branch")

// gitEnv is the environment of every git command run in dir.
func gitEnv(dir string) []string {
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=", "LC_ALL=C")
	if os.Getenv("GIT_SSH_COMMAND") == "" && os.Getenv("GIT_SSH") == "" && !sshCommandConfigured(dir) {
		env = append(env, "GIT_SSH_COMMAND=ssh -o BatchMode=yes")
	}
	return env
}

// sshCommandConfigured reports whether git config sets core.sshCommand,
// which GIT_SSH_COMMAND would override.
func sshCommandConfigured(dir string) bool {
	cmd := exec.Command("git", "config", "--get", "core.sshCommand")
	cmd.Dir = dir
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) != ""
}

func run(dir string, args ...string) (string, error) {
	out, err := runRaw(dir, args...)
	return strings.TrimSpace(string(out)), err
}

// runRaw runs git and returns stdout untouched.
func runRaw(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv(dir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.Bytes(), nil
}

// Init creates a repository in dir whose first branch is branch.
func Init(dir, branch string) error {
	_, err := run(dir, "init", "-q", "-b", branch)
	return err
}

// Clone clones branch of url into dir.
func Clone(url, branch, dir string) error {
	_, err := run("", "clone", "-q", "--branch", branch, "--", url, dir)
	return err
}

// DefaultBranch asks the remote which branch its HEAD points to.
func DefaultBranch(url string) (string, error) {
	out, err := run("", "ls-remote", "--symref", "--", url, "HEAD")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		ref, target, ok := strings.Cut(line, "\t")
		if ok && target == "HEAD" && strings.HasPrefix(ref, "ref: refs/heads/") {
			return strings.TrimPrefix(ref, "ref: refs/heads/"), nil
		}
	}
	return "", fmt.Errorf("remote %s did not report a default branch", url)
}

// SameRemote reports whether two remote URLs name the same repository,
// ignoring case, surrounding spaces, a trailing slash and a ".git" suffix.
func SameRemote(a, b string) bool {
	return normalizeRemote(a) == normalizeRemote(b)
}

func normalizeRemote(u string) string {
	u = strings.ToLower(strings.TrimSpace(u))
	u = strings.TrimSuffix(u, "/")
	return strings.TrimSuffix(u, ".git")
}

// Repo is a local clone.
type Repo struct{ Dir string }

// IsRepo reports whether Dir is the top level of a git working tree.
func (r Repo) IsRepo() bool {
	out, err := run(r.Dir, "rev-parse", "--show-cdup")
	return err == nil && out == ""
}

// OriginURL returns the URL of the origin remote as configured, before any
// url.<base>.insteadOf rewriting, so it compares with workspace.yaml.
func (r Repo) OriginURL() (string, error) {
	return run(r.Dir, "config", "--get", "remote.origin.url")
}

// Fetch updates the remote-tracking refs of origin.
func (r Repo) Fetch() error {
	_, err := run(r.Dir, "fetch", "-q", "--prune", "origin")
	return err
}

// CurrentBranch returns the checked-out branch, or "" when HEAD is detached.
func (r Repo) CurrentBranch() (string, error) {
	out, err := run(r.Dir, "symbolic-ref", "-q", "--short", "HEAD")
	if err != nil {
		if _, headErr := run(r.Dir, "rev-parse", "--verify", "-q", "HEAD"); headErr == nil {
			return "", nil
		}
		return "", err
	}
	return out, nil
}

// IsClean reports whether the working tree has no changes. Untracked files
// count as changes.
func (r Repo) IsClean() (bool, error) {
	out, err := run(r.Dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out == "", nil
}

// FastForward advances the checked-out branch to origin/<branch>. It
// reports whether HEAD moved and returns ErrDiverged when a fast-forward is
// not possible.
func (r Repo) FastForward(branch string) (bool, error) {
	upstream := "origin/" + branch
	if _, err := r.RevParse(upstream); err != nil {
		return false, fmt.Errorf("%w: %s", ErrNoUpstream, upstream)
	}
	before, err := r.RevParse("HEAD")
	if err != nil {
		return false, err
	}
	if _, err := run(r.Dir, "merge", "-q", "--ff-only", upstream); err != nil {
		if _, ancestorErr := run(r.Dir, "merge-base", "--is-ancestor", "HEAD", upstream); ancestorErr != nil {
			return false, ErrDiverged
		}
		return false, err
	}
	after, err := r.RevParse("HEAD")
	if err != nil {
		return false, err
	}
	return before != after, nil
}

// RevParse resolves ref to a full commit SHA.
func (r Repo) RevParse(ref string) (string, error) {
	return run(r.Dir, "rev-parse", "--verify", "-q", ref)
}

// AheadBehind counts the commits local has that upstream lacks (ahead) and
// the commits upstream has that local lacks (behind).
func (r Repo) AheadBehind(local, upstream string) (ahead, behind int, err error) {
	out, err := run(r.Dir, "rev-list", "--left-right", "--count", local+"..."+upstream)
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(out)
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output %q", out)
	}
	if ahead, err = strconv.Atoi(fields[0]); err != nil {
		return 0, 0, err
	}
	if behind, err = strconv.Atoi(fields[1]); err != nil {
		return 0, 0, err
	}
	return ahead, behind, nil
}

// ReadFile returns the content of path at rev, without touching the working
// tree. It refuses anything that is not a regular file there (symlinks,
// directories, submodules), so it never reads outside the repository.
func (r Repo) ReadFile(rev, path string) ([]byte, error) {
	out, err := run(r.Dir, "ls-tree", "-z", rev, "--", path)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, fmt.Errorf("%s does not exist at %s", path, rev)
	}
	meta, _, _ := strings.Cut(out, "\t")
	f := strings.Fields(meta)
	if len(f) != 3 || f[1] != "blob" || (f[0] != "100644" && f[0] != "100755") {
		return nil, fmt.Errorf("%s is not a regular file at %s", path, rev)
	}
	return runRaw(r.Dir, "cat-file", "blob", f[2])
}
