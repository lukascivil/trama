// Package changes reads and updates the change records in changes/: one
// Markdown file per change that spans repositories, with a YAML front
// matter that trama reads and keeps up to date.
package changes

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/lukascivil/trama/internal/fsx"
)

// Step is one pull request (or one step left for later) of a change.
type Step struct {
	Order  int    `yaml:"order"`
	Repo   string `yaml:"repo"`
	Action string `yaml:"action"`
	Branch string `yaml:"branch,omitempty"`
	PR     string `yaml:"pr,omitempty"`
	State  string `yaml:"state"` // pending | open | merged | closed
}

// Record is one change record.
type Record struct {
	ID        string `yaml:"id"`
	Title     string `yaml:"title"`
	Status    string `yaml:"status"` // open | done
	Continues string `yaml:"continues,omitempty"`
	Steps     []Step `yaml:"steps"`
	Path      string `yaml:"-"` // path of the file

	body string // Markdown after the front matter, kept byte for byte
}

var (
	idPattern   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-[a-z0-9]+(-[a-z0-9]+)*$`)
	stepStates  = []string{"pending", "open", "merged", "closed"}
	recordState = []string{"open", "done"}
)

// Parse reads a record. Every error names the file.
func Parse(path string, data []byte) (*Record, error) {
	r, err := parse(path, data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return r, nil
}

func parse(path string, data []byte) (*Record, error) {
	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		return nil, errors.New("no front matter: the file must start with a --- line")
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return nil, errors.New("front matter has no closing --- line")
	}
	front, body := text[4:4+end+1], text[4+end+5:]
	dec := yaml.NewDecoder(bytes.NewReader([]byte(front)))
	dec.KnownFields(true)
	r := &Record{Path: path, body: body}
	if err := dec.Decode(r); err != nil {
		return nil, err
	}
	if want := strings.TrimSuffix(filepath.Base(path), ".md"); r.ID != want {
		return nil, fmt.Errorf("id %q does not match the file name (want %q)", r.ID, want)
	}
	if !idPattern.MatchString(r.ID) {
		return nil, fmt.Errorf("id %q must look like YYYY-MM-DD-<slug>", r.ID)
	}
	if strings.TrimSpace(r.Title) == "" {
		return nil, errors.New("title is empty")
	}
	if !slices.Contains(recordState, r.Status) {
		return nil, fmt.Errorf("status %q must be open or done", r.Status)
	}
	for i, s := range r.Steps {
		switch {
		case strings.TrimSpace(s.Repo) == "":
			return nil, fmt.Errorf("steps[%d]: repo is empty", i)
		case !slices.Contains(stepStates, s.State):
			return nil, fmt.Errorf("steps[%d]: state %q must be one of %s", i, s.State, strings.Join(stepStates, ", "))
		case s.State == "pending" && s.PR != "":
			return nil, fmt.Errorf("steps[%d]: a pending step has no pr yet", i)
		case s.State != "pending" && s.PR == "":
			return nil, fmt.Errorf("steps[%d]: a %s step needs its pr", i, s.State)
		}
	}
	return r, nil
}

// Load reads every changes/*.md of the workspace at root, sorted by name.
// A broken file is reported in errs and skipped.
func Load(root string) (records []*Record, errs []error) {
	dir := filepath.Join(root, "changes")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, []error{err}
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		r, err := Parse(path, data)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		records = append(records, r)
	}
	return records, errs
}

// Save writes the front matter back and keeps the body as it was.
func (r *Record) Save() error {
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(r); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	buf.WriteString("---\n")
	buf.WriteString(r.body)
	return fsx.ReplaceFile(r.Path, buf.Bytes(), 0o644)
}

// OpenSteps returns the steps that are not merged yet.
func (r *Record) OpenSteps() []Step {
	var open []Step
	for _, s := range r.Steps {
		if s.State != "merged" {
			open = append(open, s)
		}
	}
	return open
}
