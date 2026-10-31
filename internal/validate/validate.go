// Package validate checks the catalog cards of a workspace: required
// fields, evidence that exists on disk, and proof for every relation.
package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/workspace"
)

// Severity tells whether a finding blocks the catalog.
type Severity string

const (
	Error   Severity = "error"
	Warning Severity = "warning"
)

// Finding is one problem in the catalog.
type Finding struct {
	Severity Severity
	File     string // slash path relative to the workspace root
	Where    string // e.g. `consumes[0] GET /hello`; "" for the whole file
	Message  string
}

func (f Finding) String() string {
	if f.Where == "" {
		return fmt.Sprintf("%s: %s: %s", f.Severity, f.File, f.Message)
	}
	return fmt.Sprintf("%s: %s: %s: %s", f.Severity, f.File, f.Where, f.Message)
}

// Errors counts the findings with Error severity.
func Errors(fs []Finding) int {
	n := 0
	for _, f := range fs {
		if f.Severity == Error {
			n++
		}
	}
	return n
}

// cardFile is the slash path of a card relative to the workspace root.
func cardFile(name string) string {
	return "catalog/" + name + ".yaml"
}

// report collects findings for one card file.
type report struct {
	file string
	out  *[]Finding
}

func (r report) errorf(where, format string, args ...any) {
	*r.out = append(*r.out, Finding{Severity: Error, File: r.file, Where: where, Message: fmt.Sprintf(format, args...)})
}

func (r report) warnf(where, format string, args ...any) {
	*r.out = append(*r.out, Finding{Severity: Warning, File: r.file, Where: where, Message: fmt.Sprintf(format, args...)})
}

// loaded is the result of reading every card of a workspace.
type loaded struct {
	cards  map[string]*catalog.Card // cards that parsed
	broken map[string]bool          // cards that exist but did not parse
}

// Report is the result of checking a workspace's catalog.
type Report struct {
	Findings  []Finding
	Cards     int // cards that parsed
	Relations int // consumes with a proven provider
}

// Check checks every card against workspace.yaml, the other cards and the
// clones in repos/. Findings are sorted by file, then by order found.
func Check(ws *workspace.Workspace) Report {
	var fs []Finding
	l := load(ws, &fs)
	for _, r := range ws.Repos {
		if c := l.cards[r.Name]; c != nil {
			checkCard(ws, r, c, report{file: cardFile(r.Name), out: &fs})
		}
	}
	relations := checkLinks(ws, l, &fs)
	sort.SliceStable(fs, func(i, j int) bool { return fs[i].File < fs[j].File })
	return Report{Findings: fs, Cards: len(l.cards), Relations: relations}
}

// Workspace returns the findings of Check.
func Workspace(ws *workspace.Workspace) []Finding {
	return Check(ws).Findings
}

// Relations returns how many consumes have a proven provider.
func Relations(ws *workspace.Workspace) int {
	return Check(ws).Relations
}

func load(ws *workspace.Workspace, fs *[]Finding) loaded {
	l := loaded{cards: map[string]*catalog.Card{}, broken: map[string]bool{}}
	names, err := catalog.Names(ws.Root)
	if err != nil {
		*fs = append(*fs, Finding{Severity: Error, File: "catalog/", Message: err.Error()})
	}
	if entries, err := os.ReadDir(filepath.Join(ws.Root, "catalog")); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".yml") {
				report{file: "catalog/" + e.Name(), out: fs}.errorf("", "use the .yaml extension")
			}
		}
	}
	exists := map[string]bool{}
	for _, n := range names {
		exists[n] = true
		if _, ok := ws.Repo(n); !ok {
			report{file: cardFile(n), out: fs}.errorf("", "card for a repository that is not in workspace.yaml")
		}
	}
	for _, r := range ws.Repos {
		rep := report{file: cardFile(r.Name), out: fs}
		if !exists[r.Name] {
			rep.warnf("", `not mapped yet: run "/trama:map %s"`, r.Name)
			continue
		}
		c, err := catalog.LoadStrict(ws.Root, r.Name)
		if err != nil {
			l.broken[r.Name] = true
			rep.errorf("", "%s", strings.TrimPrefix(err.Error(), catalog.Path(ws.Root, r.Name)+": "))
			continue
		}
		l.cards[r.Name] = c
	}
	return l
}

// repoPath is the clone directory of name, relative to the workspace root.
func repoPath(name string) string {
	return filepath.ToSlash(filepath.Join("repos", name))
}
