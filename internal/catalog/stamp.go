package catalog

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/workspace"
)

// Stamp records in catalog/<name>.yaml the commit the card describes
// (origin/<branch>) and the date. It edits the YAML in place, keeping every
// other field and comment, and adds the two keys when they are missing. It
// refuses unless the clone is clean, on its main branch and at origin.
func Stamp(ws *workspace.Workspace, name string, today time.Time) (string, error) {
	r, ok := ws.Repo(name)
	if !ok {
		return "", fmt.Errorf("repo %s is not in workspace.yaml", name)
	}
	path := Path(ws.Root, name)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("no card at catalog/%s.yaml: write it first", name)
	}
	if err != nil {
		return "", err
	}
	sha, err := originCommit(ws, r)
	if err != nil {
		return "", err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return "", fmt.Errorf("%s: card is not a mapping", path)
	}
	top := doc.Content[0]
	setScalar(top, "mapped_at_commit", sha, yaml.DoubleQuotedStyle)
	setScalar(top, "mapped_at", today.Format("2006-01-02"), 0)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return "", err
	}
	return sha, os.Rename(tmp, path)
}

// originCommit returns origin/<branch> after checking that the clone is
// clean, on its main branch and exactly at origin.
func originCommit(ws *workspace.Workspace, r workspace.Repo) (string, error) {
	dir := ws.RepoDir(r.Name)
	repo := gitx.Repo{Dir: dir}
	if _, err := os.Stat(dir); err != nil || !repo.IsRepo() {
		return "", fmt.Errorf(`repos/%s is not cloned: run "trama sync %s"`, r.Name, r.Name)
	}
	branch, err := repo.CurrentBranch()
	if err != nil {
		return "", err
	}
	if branch != r.Branch {
		if branch == "" {
			branch = "a detached HEAD"
		}
		return "", fmt.Errorf("repos/%s is on %s, not %s", r.Name, branch, r.Branch)
	}
	clean, err := repo.IsClean()
	if err != nil {
		return "", err
	}
	if !clean {
		return "", fmt.Errorf("repos/%s has local changes", r.Name)
	}
	head, err := repo.RevParse("HEAD")
	if err != nil {
		return "", err
	}
	origin, err := repo.RevParse("origin/" + r.Branch)
	if err != nil || head != origin {
		return "", fmt.Errorf(`repos/%s is not at origin/%s: run "trama sync %s"`, r.Name, r.Branch, r.Name)
	}
	return origin, nil
}

// setScalar sets key to value in a mapping node, appending the key when it
// is missing.
func setScalar(m *yaml.Node, key, value string, style yaml.Style) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value, Style: style}
			return
		}
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value, Style: style})
}
