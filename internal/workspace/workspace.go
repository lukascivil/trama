// Package workspace reads and edits workspace.yaml, the list of
// repositories in a trama workspace.
package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/lukascivil/trama/internal/fsx"
	"github.com/lukascivil/trama/internal/gitx"
)

// FileName is the name of the workspace file at the workspace root.
const FileName = "workspace.yaml"

// ErrNotFound means no workspace.yaml exists in a directory or its parents.
var ErrNotFound = errors.New("not inside a trama workspace: no workspace.yaml found here or in any parent directory")

// ErrInvalidName means a repository name is not safe to use as a directory.
var ErrInvalidName = errors.New("invalid repo name")

var validName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// Repo is one repository of the workspace.
type Repo struct {
	Name   string   `yaml:"name"`
	URL    string   `yaml:"url"`
	Branch string   `yaml:"branch"`
	Hosts  []string `yaml:"hosts,omitempty"`
}

// Link is a relation a person declared: consumer uses id from provider.
type Link struct {
	Consumer string `yaml:"consumer"`
	Provider string `yaml:"provider"`
	ID       string `yaml:"id"`
}

// Workspace is a parsed workspace.yaml.
type Workspace struct {
	Version int    `yaml:"version"`
	Name    string `yaml:"name"`
	Repos   []Repo `yaml:"repos"`
	Links   []Link `yaml:"links,omitempty"`
	Root    string `yaml:"-"` // absolute directory that holds workspace.yaml

	// doc is the parsed document. AddRepo edits it in place so that Save
	// keeps comments and fields this struct does not know about.
	doc yaml.Node
}

// Find returns the closest directory, starting at start and walking up,
// that contains a trama workspace.yaml. A workspace.yaml without a top-level
// "version" key belongs to another tool and is skipped.
func Find(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if data, err := os.ReadFile(filepath.Join(dir, FileName)); err == nil && isTramaFile(data) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}
		dir = parent
	}
}

// isTramaFile reports whether a workspace.yaml looks like trama's: it has a
// top-level "version" key. Unparsable files count as trama's, so Load can
// report the error instead of the search silently walking past it.
func isTramaFile(data []byte) bool {
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		return true
	}
	_, ok := top["version"]
	return ok
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
	for i, l := range w.Links {
		if _, ok := w.Repo(l.Consumer); !ok {
			return fmt.Errorf("links[%d]: consumer %q is not a repo of the workspace", i, l.Consumer)
		}
		if _, ok := w.Repo(l.Provider); !ok {
			return fmt.Errorf("links[%d]: provider %q is not a repo of the workspace", i, l.Provider)
		}
		if l.Consumer == l.Provider {
			return fmt.Errorf("links[%d]: a repo cannot link to itself", i)
		}
		if strings.TrimSpace(l.ID) == "" {
			return fmt.Errorf("links[%d]: id is empty", i)
		}
	}
	return nil
}

func validateRepo(r Repo) error {
	if err := validateName(r.Name); err != nil {
		return err
	}
	if err := validateURL(r.Name, r.URL); err != nil {
		return err
	}
	return validateBranch(r.Name, r.Branch)
}

func validateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("%w %q: use lowercase letters, digits, '.', '_' and '-', starting with a letter or digit", ErrInvalidName, name)
	}
	return nil
}

// validateURL rejects URLs git could read as an option.
func validateURL(name, url string) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("repo %s: url is empty", name)
	}
	if strings.HasPrefix(strings.TrimSpace(url), "-") {
		return fmt.Errorf(`repo %s: url must not start with "-"`, name)
	}
	return nil
}

// validateBranch applies the parts of git's ref-name rules that matter for
// a branch name typed into workspace.yaml.
func validateBranch(name, branch string) error {
	if strings.TrimSpace(branch) == "" {
		return fmt.Errorf("repo %s: branch is empty", name)
	}
	if err := ValidBranch(branch); err != nil {
		return fmt.Errorf("repo %s: %w", name, err)
	}
	return nil
}

// ValidBranch applies the parts of git's ref-name rules that matter for a
// branch name trama is given (in workspace.yaml or on the command line).
func ValidBranch(branch string) error {
	bad := branch == "" || branch == "@" ||
		strings.HasPrefix(branch, "-") || strings.HasPrefix(branch, "/") ||
		strings.HasSuffix(branch, "/") || strings.HasSuffix(branch, ".") || strings.HasSuffix(branch, ".lock") ||
		strings.Contains(branch, "..") || strings.Contains(branch, "@{") || strings.Contains(branch, "//") ||
		strings.ContainsAny(branch, " ~^:?*[\\")
	for _, c := range branch {
		bad = bad || c < 0x20 || c == 0x7f
	}
	if bad {
		return fmt.Errorf("invalid branch name %q", branch)
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

// CheckNew reports whether a repository called name at url could be added:
// both are valid and neither is in the workspace yet. It does no I/O.
func (w *Workspace) CheckNew(name, url string) error {
	if err := validateName(name); err != nil {
		return err
	}
	if err := validateURL(name, url); err != nil {
		return err
	}
	for _, existing := range w.Repos {
		if existing.Name == name {
			return fmt.Errorf("repo %s already in workspace", name)
		}
		if gitx.SameRemote(existing.URL, url) {
			return fmt.Errorf("%s is already in the workspace as %s", url, existing.Name)
		}
	}
	return nil
}

// AddRepo appends r to the workspace. Call Save to write it.
func (w *Workspace) AddRepo(r Repo) error {
	if err := w.CheckNew(r.Name, r.URL); err != nil {
		return err
	}
	if err := validateBranch(r.Name, r.Branch); err != nil {
		return err
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
	return fsx.ReplaceFile(filepath.Join(w.Root, FileName), buf.Bytes(), 0o644)
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

// LocalURL returns url as an absolute path when it is a local path, so the
// workspace means the same repository from any directory. Remote URLs
// ("scheme://..." or scp-like "host:path") and values starting with "-"
// (which validation rejects) are returned unchanged.
func LocalURL(url, cwd string) string {
	u := strings.TrimSpace(url)
	if u == "" || strings.HasPrefix(u, "-") || strings.Contains(u, "://") || isSCPLike(u) {
		return url
	}
	if filepath.IsAbs(u) {
		return filepath.Clean(u)
	}
	return filepath.Join(cwd, u)
}

func isSCPLike(u string) bool {
	colon := strings.Index(u, ":")
	slash := strings.Index(u, "/")
	return colon > 0 && (slash < 0 || colon < slash)
}

// Files is how the committed files of a repository are listed and read: at
// origin/<branch> when the clone is a git repository that has it, so local
// edits and ignored files never count, and from the disk otherwise. List and
// Read are nil when the repository is not cloned.
type Files struct {
	Place   string // where the files come from, for messages
	Cloned  bool
	List    func() ([]string, error)
	Read    func(path string) ([]byte, error)
	ReadAll func(paths []string) (map[string][]byte, error) // Read for many files at once
}

// Files returns how to list and read the files of r.
func (ws *Workspace) Files(r Repo) Files {
	dir := ws.RepoDir(r.Name)
	place := filepath.ToSlash(filepath.Join("repos", r.Name))
	if _, err := os.Stat(dir); err != nil {
		return Files{Place: place}
	}
	g := gitx.Repo{Dir: dir}
	rev := "origin/" + r.Branch
	if g.IsRepo() {
		if _, err := g.RevParse(rev); err == nil {
			return Files{
				Place:   place + " at " + rev,
				Cloned:  true,
				List:    func() ([]string, error) { return g.ListFiles(rev) },
				Read:    func(path string) ([]byte, error) { return g.ReadFile(rev, path) },
				ReadAll: func(paths []string) (map[string][]byte, error) { return g.ReadFiles(rev, paths) },
			}
		}
	}
	return Files{
		Place:   place,
		Cloned:  true,
		List:    func() ([]string, error) { return listRegular(dir) },
		Read:    func(path string) ([]byte, error) { return readRegular(dir, path) },
		ReadAll: func(paths []string) (map[string][]byte, error) { return readAllRegular(dir, paths) },
	}
}

// readAllRegular reads paths from disk one by one: there is no process to
// spare here, unlike in git.
func readAllRegular(dir string, paths []string) (map[string][]byte, error) {
	out := make(map[string][]byte, len(paths))
	for _, p := range paths {
		data, err := readRegular(dir, p)
		if err != nil {
			return nil, err
		}
		out[p] = data
	}
	return out, nil
}

// listRegular lists the regular files under dir as slash paths relative to
// it, skipping .git, node_modules and checked-out submodules. Symlinks are
// never followed.
func listRegular(dir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == dir {
				return nil
			}
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			// A checked-out submodule has a .git of its own: its files are
			// another repository's, as git mode already treats them.
			if _, err := os.Lstat(filepath.Join(path, ".git")); err == nil {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	return out, err
}

// readRegular reads dir/path from disk, refusing anything but a regular file.
func readRegular(dir, path string) ([]byte, error) {
	full := filepath.Join(dir, filepath.FromSlash(path))
	fi, err := os.Lstat(full)
	if err != nil {
		return nil, fmt.Errorf("%s does not exist", path)
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	return os.ReadFile(full)
}
