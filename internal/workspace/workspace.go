// Package workspace reads and edits workspace.yaml, the list of
// repositories in a trama workspace.
package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/lukascivil/trama/internal/gitx"
)

// FileName is the name of the workspace file at the workspace root.
const FileName = "workspace.yaml"

// ErrNotFound means no workspace.yaml exists in a directory or its parents.
var ErrNotFound = errors.New("not inside a trama workspace: no workspace.yaml found here or in any parent directory")

var validName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// Repo is one repository of the workspace.
type Repo struct {
	Name   string   `yaml:"name"`
	URL    string   `yaml:"url"`
	Branch string   `yaml:"branch"`
	Hosts  []string `yaml:"hosts,omitempty"`
}

// Workspace is a parsed workspace.yaml.
type Workspace struct {
	Version int    `yaml:"version"`
	Name    string `yaml:"name"`
	Repos   []Repo `yaml:"repos"`
	Root    string `yaml:"-"` // absolute directory that holds workspace.yaml

	// doc is the parsed document. AddRepo edits it in place so that Save
	// keeps comments and fields this struct does not know about.
	doc yaml.Node
}

// Find returns the closest directory, starting at start and walking up,
// that contains workspace.yaml.
func Find(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, FileName)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}
		dir = parent
	}
}

// Load parses root/workspace.yaml and validates it.
func Load(root string) (*Workspace, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(abs, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	w := &Workspace{}
	if err := yaml.Unmarshal(data, &w.doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(w.doc.Content) > 0 {
		if err := w.doc.Decode(w); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	w.Root = abs
	if err := w.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return w, nil
}

// Validate checks the version, the workspace name and every repository.
func (w *Workspace) Validate() error {
	if w.Version != 1 {
		return fmt.Errorf("unsupported workspace version %d (expected 1)", w.Version)
	}
	if strings.TrimSpace(w.Name) == "" {
		return errors.New("workspace name is empty")
	}
	for i, r := range w.Repos {
		if err := validateRepo(r); err != nil {
			return err
		}
		for _, prev := range w.Repos[:i] {
			if prev.Name == r.Name {
				return fmt.Errorf("repo %s is listed twice", r.Name)
			}
			if gitx.SameRemote(prev.URL, r.URL) {
				return fmt.Errorf("repos %s and %s point to the same url", prev.Name, r.Name)
			}
		}
	}
	return nil
}

func validateRepo(r Repo) error {
	if !validName.MatchString(r.Name) {
		return fmt.Errorf("invalid repo name %q: use lowercase letters, digits, '.', '_' and '-', starting with a letter or digit", r.Name)
	}
	if strings.TrimSpace(r.URL) == "" {
		return fmt.Errorf("repo %s: url is empty", r.Name)
	}
	if strings.TrimSpace(r.Branch) == "" {
		return fmt.Errorf("repo %s: branch is empty", r.Name)
	}
	return nil
}

// Repo returns the repository called name.
func (w *Workspace) Repo(name string) (Repo, bool) {
	for _, r := range w.Repos {
		if r.Name == name {
			return r, true
		}
	}
	return Repo{}, false
}

// RepoDir returns the directory a repository is cloned into.
func (w *Workspace) RepoDir(name string) string {
	return filepath.Join(w.Root, "repos", name)
}

// AddRepo appends r to the workspace. Call Save to write it.
func (w *Workspace) AddRepo(r Repo) error {
	if err := validateRepo(r); err != nil {
		return err
	}
	for _, existing := range w.Repos {
		if existing.Name == r.Name {
			return fmt.Errorf("repo %s already in workspace", r.Name)
		}
		if gitx.SameRemote(existing.URL, r.URL) {
			return fmt.Errorf("%s is already in the workspace as %s", r.URL, existing.Name)
		}
	}
	seq, err := w.reposNode()
	if err != nil {
		return err
	}
	var item yaml.Node
	if err := item.Encode(r); err != nil {
		return err
	}
	seq.Content = append(seq.Content, &item)
	w.Repos = append(w.Repos, r)
	return nil
}

// reposNode returns the sequence node under the "repos" key, creating it
// when missing and switching a flow list ("repos: []") to block style.
func (w *Workspace) reposNode() (*yaml.Node, error) {
	if len(w.doc.Content) == 0 {
		w.doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
	}
	top := w.doc.Content[0]
	if top.Kind != yaml.MappingNode {
		return nil, errors.New("workspace.yaml is not a mapping")
	}
	for i := 0; i+1 < len(top.Content); i += 2 {
		if top.Content[i].Value != "repos" {
			continue
		}
		seq := top.Content[i+1]
		if seq.Kind == yaml.ScalarNode && seq.Tag == "!!null" {
			*seq = yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		}
		if seq.Kind != yaml.SequenceNode {
			return nil, errors.New("workspace.yaml: repos is not a list")
		}
		seq.Style &^= yaml.FlowStyle
		return seq, nil
	}
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	top.Content = append(top.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "repos"}, seq)
	return seq, nil
}

// Save writes the workspace back to Root/workspace.yaml.
func (w *Workspace) Save() error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&w.doc); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	path := filepath.Join(w.Root, FileName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// NameFromURL derives a repository name from its clone URL: the last path
// segment, lowercased, without ".git".
func NameFromURL(url string) string {
	u := strings.TrimSuffix(strings.TrimSpace(url), "/")
	if i := strings.LastIndexAny(u, "/:"); i >= 0 {
		u = u[i+1:]
	}
	return strings.ToLower(strings.TrimSuffix(u, ".git"))
}
