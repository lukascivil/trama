// Package syncer clones and updates the repositories of a workspace
// without ever discarding local work.
package syncer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/workspace"
)

// Outcome is what Sync did with one repository.
type Outcome string

const (
	Cloned          Outcome = "cloned"
	Updated         Outcome = "updated"
	UpToDate        Outcome = "up-to-date"
	SkippedDirty    Outcome = "skipped-dirty"
	SkippedBranch   Outcome = "skipped-branch"
	SkippedDiverged Outcome = "skipped-diverged"
	SkippedNotRepo  Outcome = "skipped-not-a-repo"
	SkippedOrigin   Outcome = "skipped-origin-mismatch"
	Failed          Outcome = "failed"
)

// Result is the outcome for one repository.
type Result struct {
	Repo    string
	Outcome Outcome
	Detail  string // human-readable reason; empty for cloned and up-to-date
}

// Sync processes repos in workspace order. only filters by name; an unknown
// name returns an error before anything runs. Per-repo failures go in Result.
func Sync(ws *workspace.Workspace, only []string) ([]Result, error) {
	repos := ws.Repos
	if len(only) > 0 {
		repos = nil
		for _, name := range only {
			r, ok := ws.Repo(name)
			if !ok {
				return nil, fmt.Errorf("unknown repo %q", name)
			}
			repos = append(repos, r)
		}
	}
	results := make([]Result, 0, len(repos))
	for _, r := range repos {
		outcome, detail := syncRepo(ws, r)
		results = append(results, Result{Repo: r.Name, Outcome: outcome, Detail: detail})
	}
	return results, nil
}

func syncRepo(ws *workspace.Workspace, r workspace.Repo) (Outcome, string) {
	dir := ws.RepoDir(r.Name)
	rel := filepath.ToSlash(filepath.Join("repos", r.Name))
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
			return Failed, err.Error()
		}
		if err := gitx.Clone(r.URL, r.Branch, dir); err != nil {
			return Failed, err.Error()
		}
		return Cloned, ""
	}

	repo := gitx.Repo{Dir: dir}
	if !repo.IsRepo() {
		return SkippedNotRepo, rel + " exists but is not a git repository; left untouched"
	}
	origin, err := repo.OriginURL()
	if err != nil {
		return SkippedOrigin, rel + " has no origin remote; left untouched"
	}
	if !gitx.SameRemote(origin, r.URL) {
		return SkippedOrigin, fmt.Sprintf("origin is %s, workspace.yaml says %s; left untouched", origin, r.URL)
	}
	if err := repo.Fetch(); err != nil {
		return Failed, err.Error()
	}
	branch, err := repo.CurrentBranch()
	if err != nil {
		return Failed, err.Error()
	}
	if branch != r.Branch {
		if branch == "" {
			branch = "a detached HEAD"
		} else {
			branch = "branch " + branch
		}
		return SkippedBranch, fmt.Sprintf("on %s; fetched, not advanced", branch)
	}
	clean, err := repo.IsClean()
	if err != nil {
		return Failed, err.Error()
	}
	if !clean {
		return SkippedDirty, "has local changes; fetched, not advanced"
	}
	before, err := repo.RevParse("HEAD")
	if err != nil {
		return Failed, err.Error()
	}
	moved, err := repo.FastForward(r.Branch)
	if errors.Is(err, gitx.ErrDiverged) {
		return SkippedDiverged, fmt.Sprintf("diverged from origin/%s; fetched, not advanced", r.Branch)
	}
	if err != nil {
		return Failed, err.Error()
	}
	if !moved {
		return UpToDate, ""
	}
	after, err := repo.RevParse("HEAD")
	if err != nil {
		return Failed, err.Error()
	}
	return Updated, short(before) + ".." + short(after)
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
