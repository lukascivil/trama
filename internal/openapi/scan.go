package openapi

import (
	"path"
	"regexp"
	"sort"
	"strings"
)

var (
	// skippedDirs never hold the repository's own spec: someone else's code,
	// tests or build output. Unlike the schema scan, spec stays in: it is
	// where many repositories keep their OpenAPI.
	skippedDirs = set("node_modules", "vendor", "dist", "build", "target", "testdata", "fixtures", "test", "tests", "__tests__", "venv", "site-packages")
	// Lockfiles are large and never a spec.
	lockfiles = set("package-lock.json", "npm-shrinkwrap.json", "pnpm-lock.yaml")
	mention   = regexp.MustCompile(`["']?openapi["']?\s*:`)
)

func set(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it] = true
	}
	return m
}

// Scan returns the OpenAPI 3 specs among the files list returns, in path
// order. It goes by content, not by name, so docs/api.yaml and spec/v1.yaml
// count: a .yaml, .yml or .json file is parsed only when it mentions
// openapi:, and is a spec when Parse says so. A file that does not parse is
// left out. readAll is called once, with every candidate, and not at all
// when there is none.
func Scan(list func() ([]string, error), readAll func([]string) (map[string][]byte, error)) ([]*Spec, error) {
	paths, err := list()
	if err != nil {
		return nil, err
	}
	var candidates []string
	for _, p := range paths {
		if candidate(p) {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	sort.Strings(candidates)
	contents, err := readAll(candidates)
	if err != nil {
		return nil, err
	}
	var out []*Spec
	for _, p := range candidates {
		data := contents[p]
		if !mention.Match(data) {
			continue
		}
		if s, err := Parse(p, data); err == nil {
			out = append(out, s)
		}
	}
	return out, nil
}

// IsDocument reports whether p is a file Parse may read: YAML or JSON. A
// contract in any other format, such as proto or GraphQL, is one trama
// does not read.
func IsDocument(p string) bool {
	switch path.Ext(strings.ToLower(p)) {
	case ".yaml", ".yml", ".json":
		return true
	}
	return false
}

func candidate(p string) bool {
	lower := strings.ToLower(p)
	if !IsDocument(lower) || lockfiles[path.Base(lower)] {
		return false
	}
	for _, d := range strings.Split(path.Dir(lower), "/") {
		if skippedDirs[d] || (d != "." && strings.HasPrefix(d, ".")) {
			return false
		}
	}
	return true
}
