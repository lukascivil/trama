package catalog

import (
	"fmt"
	"strconv"
	"strings"
)

// Evidence is a parsed evidence reference.
type Evidence struct {
	Path       string // relative to the repo root; "" when Declared
	Start, End int    // End == Start for a single line
	Declared   bool   // the literal "workspace.yaml"
}

// ParseEvidence parses "path:line", "path:start-end" or "workspace.yaml".
func ParseEvidence(s string) (Evidence, error) {
	s = strings.TrimSpace(s)
	if s == "workspace.yaml" {
		return Evidence{Declared: true}, nil
	}
	i := strings.LastIndex(s, ":")
	if i <= 0 {
		return Evidence{}, fmt.Errorf("evidence %q: want path:line or path:start-end", s)
	}
	path, lines := s[:i], s[i+1:]
	if strings.HasPrefix(path, "/") || hasDotDot(path) {
		return Evidence{}, fmt.Errorf("evidence %q: path must be relative to the repository root, without ..", s)
	}
	from, to, isRange := strings.Cut(lines, "-")
	start, err := strconv.Atoi(from)
	if err != nil || start < 1 {
		return Evidence{}, fmt.Errorf("evidence %q: line must be a number from 1", s)
	}
	end := start
	if isRange {
		if end, err = strconv.Atoi(to); err != nil || end < start {
			return Evidence{}, fmt.Errorf("evidence %q: range must be start-end with end >= start", s)
		}
	}
	return Evidence{Path: path, Start: start, End: end}, nil
}

// ValidateContract checks the path of a provide's contract: relative to the
// repository root and without .., the rule of an evidence path, so a
// contract never points outside the repository.
func ValidateContract(path string) error {
	if p := strings.TrimSpace(path); p == "" || strings.HasPrefix(p, "/") || hasDotDot(p) {
		return fmt.Errorf("contract %q: path must be relative to the repository root, without ..", path)
	}
	return nil
}

func hasDotDot(path string) bool {
	for _, seg := range strings.Split(path, "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}
