// Package release finds a repository's latest release from its git tags.
package release

import (
	"github.com/lukascivil/trama/internal/gitx"
)

// Release is the highest semver tag on a branch.
type Release struct {
	Version      string `json:"version"`      // the tag name, e.g. v1.2.3
	Date         string `json:"date"`         // strict ISO 8601
	CommitsAfter int    `json:"commitsAfter"` // commits on the branch that are not in the tag
}

// Latest returns the highest semver tag reachable from ref, or nil when
// there is none. It fails only when git does.
func Latest(repo gitx.Repo, ref string) (*Release, error) {
	tags, err := repo.TagsMerged(ref)
	if err != nil {
		return nil, err
	}
	var best *gitx.Tag
	var bestV version
	for i, tg := range tags {
		v, ok := parse(tg.Name)
		if !ok {
			continue
		}
		// Equal precedence (v1.0.0 and 1.0.0): the smaller name wins, so
		// the result does not depend on the order git lists them in.
		if c := compare(v, bestV); best == nil || c > 0 || (c == 0 && tg.Name < best.Name) {
			best, bestV = &tags[i], v
		}
	}
	if best == nil {
		return nil, nil
	}
	n, err := repo.CountCommits("refs/tags/"+best.Name, ref)
	if err != nil {
		return nil, err
	}
	return &Release{Version: best.Name, Date: best.Date, CommitsAfter: n}, nil
}
