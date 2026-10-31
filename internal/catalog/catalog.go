// Package catalog reads the catalog cards (catalog/<repo>.yaml) of a
// workspace: what each repository is, how to run it, and what it provides
// and consumes, with file:line evidence.
package catalog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Unknown is the value of Consume.From when no provider is proven.
const Unknown = "unknown"

var (
	commitSHA   = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	toolName    = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
	toolVersion = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)
)

// ValidateTool checks a tools entry: a plain mise tool name (no backend
// prefix such as npm: or github:, which would install arbitrary software)
// and a version without spaces or shell characters.
func ValidateTool(name, version string) error {
	if !toolName.MatchString(name) || !toolVersion.MatchString(version) {
		return fmt.Errorf("tools.%s: use a plain tool name (e.g. node, go, python) and a version without spaces", name)
	}
	return nil
}

// IsCommitSHA reports whether s looks like a commit SHA (7 to 40 lowercase
// hex characters). Branch names and other refs would never look stale.
func IsCommitSHA(s string) bool {
	return commitSHA.MatchString(s)
}

// Card is one catalog card.
type Card struct {
	Version        int                `yaml:"version"`
	Name           string             `yaml:"name"`
	Description    string             `yaml:"description"`
	Stack          []string           `yaml:"stack"`
	Tools          map[string]string  `yaml:"tools,omitempty"`
	MappedAtCommit string             `yaml:"mapped_at_commit"`
	MappedAt       string             `yaml:"mapped_at"`
	Run            map[string]Command `yaml:"run,omitempty"` // keys: setup, dev, test
	Identity       Identity           `yaml:"identity,omitempty"`
	Provides       []Provide          `yaml:"provides,omitempty"`
	Consumes       []Consume          `yaml:"consumes,omitempty"`
	Config         []ConfigEntry      `yaml:"config,omitempty"`
}

// Command is how to run one thing in the repository.
type Command struct {
	Cmd  string `yaml:"cmd"`
	Port int    `yaml:"port,omitempty"`
}

// Fact is a value with the evidence that proves it.
type Fact struct {
	Value    string   `yaml:"value"`
	Evidence []string `yaml:"evidence"`
}

// PortFact is a port with the evidence that proves it.
type PortFact struct {
	Value    int      `yaml:"value"`
	Evidence []string `yaml:"evidence"`
}

// Identity is how other repositories can recognize this one.
type Identity struct {
	Ports   []PortFact `yaml:"ports,omitempty"`
	Hosts   []Fact     `yaml:"hosts,omitempty"`
	Package *Fact      `yaml:"package,omitempty"`
	Topics  []Fact     `yaml:"topics,omitempty"`
}

// Provide is something the repository offers to others.
type Provide struct {
	Kind     string         `yaml:"kind"`
	ID       string         `yaml:"id"`
	Contract string         `yaml:"contract,omitempty"`
	Shape    map[string]any `yaml:"shape,omitempty"`
	Evidence []string       `yaml:"evidence"`
}

// Consume is something the repository uses from another one.
type Consume struct {
	Kind       string   `yaml:"kind"`
	ID         string   `yaml:"id"`
	From       string   `yaml:"from"`
	Via        string   `yaml:"via,omitempty"`
	Candidates []string `yaml:"candidates,omitempty"`
	FieldsUsed []string `yaml:"fields_used,omitempty"`
	Evidence   []string `yaml:"evidence"`
}

// ConfigEntry is an environment variable that points to another repository.
type ConfigEntry struct {
	Env      string   `yaml:"env"`
	Value    string   `yaml:"value,omitempty"`
	PointsTo string   `yaml:"points_to,omitempty"`
	Evidence []string `yaml:"evidence"`
}

// Path returns where the card of repo name lives.
func Path(root, name string) string {
	return filepath.Join(root, "catalog", name+".yaml")
}

// Load reads the card of repo name, ignoring fields Card does not know. It
// returns (nil, nil) when the card does not exist.
func Load(root, name string) (*Card, error) {
	data, path, err := read(root, name)
	if data == nil || err != nil {
		return nil, err
	}
	var c Card
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &c, nil
}

// LoadStrict reads the card of repo name and fails on any field Card does
// not know, so typos surface with their line number. It returns (nil, nil)
// when the card does not exist.
func LoadStrict(root, name string) (*Card, error) {
	data, path, err := read(root, name)
	if data == nil || err != nil {
		return nil, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var c Card
	if err := dec.Decode(&c); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%s: empty card", path)
		}
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for {
		var extra yaml.Node
		err := dec.Decode(&extra)
		if errors.Is(err, io.EOF) {
			return &c, nil
		}
		if err != nil || !emptyDocument(&extra) {
			return nil, fmt.Errorf("%s: more than one YAML document", path)
		}
	}
}

// emptyDocument reports whether n is a document with nothing in it, as a
// trailing "---" produces.
func emptyDocument(n *yaml.Node) bool {
	if len(n.Content) == 0 {
		return true
	}
	return len(n.Content) == 1 && n.Content[0].Kind == yaml.ScalarNode && n.Content[0].Tag == "!!null" && n.Content[0].Value == ""
}

func read(root, name string) ([]byte, string, error) {
	path := Path(root, name)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, path, nil
	}
	if err != nil {
		return nil, path, err
	}
	if data == nil {
		data = []byte{}
	}
	return data, path, nil
}

// Names returns the names of the cards in root/catalog, sorted.
func Names(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "catalog"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			names = append(names, strings.TrimSuffix(e.Name(), ".yaml"))
		}
	}
	sort.Strings(names)
	return names, nil
}
