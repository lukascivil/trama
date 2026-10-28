// Package scaffold creates the files of a new trama workspace.
package scaffold

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"go.yaml.in/yaml/v3"

	"github.com/lukascivil/trama/internal/gitx"
)

//go:embed templates/*
var templates embed.FS

// ErrNotEmpty means the target directory already has files in it.
var ErrNotEmpty = errors.New("directory is not empty")

// Options configures Init.
type Options struct {
	Dir          string // target directory; created when missing
	Name         string // workspace name; default filepath.Base(Dir)
	TramaVersion string // "dev" means: do not pin the binary in mise.toml
}

// allowedEntries may exist in the target directory before Init runs. A
// .claude directory is also allowed when it has no settings.json.
var allowedEntries = map[string]bool{".git": true, ".DS_Store": true}

// files maps each created path to the template it comes from. Templates
// ending in .tmpl go through text/template; the rest are copied as is.
var files = map[string]string{
	"workspace.yaml":        "workspace.yaml.tmpl",
	"mise.toml":             "mise.toml.tmpl",
	".gitignore":            "gitignore",
	"AGENTS.md":             "AGENTS.md.tmpl",
	"CLAUDE.md":             "CLAUDE.md",
	".claude/settings.json": "settings.json",
	"catalog/.gitkeep":      "",
	"docs/.gitkeep":         "",
	"changes/.gitkeep":      "",
}

var plainScalar = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._-]*$`)

// Init creates the workspace layout in opts.Dir and returns the created
// paths, relative to the directory and sorted. It initializes a git
// repository when there is none and never commits.
func Init(opts Options) ([]string, error) {
	dir, err := filepath.Abs(opts.Dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var blocking []string
	for _, e := range entries {
		switch {
		case allowedEntries[e.Name()]:
		case e.Name() == ".claude" && e.IsDir():
			// Claude Code may already have written settings.local.json here.
			if _, err := os.Stat(filepath.Join(dir, ".claude", "settings.json")); err == nil {
				blocking = append(blocking, ".claude/settings.json")
			}
		default:
			blocking = append(blocking, e.Name())
		}
	}
	if len(blocking) > 0 {
		return nil, fmt.Errorf("%w: %s contains %s", ErrNotEmpty, dir, strings.Join(blocking, ", "))
	}

	name := opts.Name
	if name == "" {
		name = filepath.Base(dir)
	}
	data := struct {
		Name, NameYAML, TramaVersion string
		Pin                          bool
	}{name, yamlScalar(name), opts.TramaVersion, opts.TramaVersion != "dev"}

	rendered := make(map[string][]byte, len(files))
	for path, src := range files {
		content, err := render(src, data)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", path, err)
		}
		rendered[path] = content
	}

	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		if err := gitx.Init(dir, "main"); err != nil {
			return nil, err
		}
	}

	created := make([]string, 0, len(rendered))
	for path, content := range rendered {
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(full, content, 0o644); err != nil {
			return nil, err
		}
		created = append(created, path)
	}
	sort.Strings(created)
	return created, nil
}

func render(src string, data any) ([]byte, error) {
	if src == "" {
		return nil, nil
	}
	raw, err := templates.ReadFile("templates/" + src)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(src, ".tmpl") {
		return raw, nil
	}
	tmpl, err := template.New(src).Parse(string(raw))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// yamlScalar returns s as a YAML scalar, quoting it unless it is plainly safe
// and reads back as the same string (so "null", "true" or "123" get quotes).
func yamlScalar(s string) string {
	var back any
	if plainScalar.MatchString(s) && !strings.HasSuffix(s, " ") && yaml.Unmarshal([]byte(s), &back) == nil && back == s {
		return s
	}
	quoted, _ := json.Marshal(s)
	return string(quoted)
}
