package validate

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/workspace"
)

var (
	httpID  = regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS) /[^\s?#]*$`)
	runKeys = []string{"setup", "dev", "test"}
	kinds   = []string{"http", "event", "package"}
)

// checkCard applies the rules that need only this card, workspace.yaml and
// the clone of the repository.
func checkCard(ws *workspace.Workspace, repo workspace.Repo, c *catalog.Card, rep report) {
	if c.Version != 1 {
		rep.errorf("version", "must be 1")
	}
	if c.Name != repo.Name {
		rep.errorf("name", "%q does not match the file name %s", c.Name, cardFile(repo.Name))
	}
	if strings.TrimSpace(c.Description) == "" {
		rep.errorf("description", "is empty")
	}
	if len(c.Stack) == 0 {
		rep.errorf("stack", "is empty")
	}
	if !catalog.IsCommitSHA(c.MappedAtCommit) {
		rep.errorf("mapped_at_commit", `must be a commit SHA (7 to 40 hex characters): run "trama stamp %s"`, repo.Name)
	}
	if _, err := time.Parse("2006-01-02", c.MappedAt); err != nil {
		rep.errorf("mapped_at", "must be a date (YYYY-MM-DD)")
	}
	for _, tool := range slices.Sorted(maps.Keys(c.Tools)) {
		if err := catalog.ValidateTool(tool, c.Tools[tool]); err != nil {
			rep.errorf("tools."+tool, "use a plain tool name (e.g. node, go, python) and a version without spaces")
		}
	}
	for _, key := range sortedKeys(c.Run) {
		switch {
		case !slices.Contains(runKeys, key):
			rep.errorf("run."+key, "unknown command (use setup, dev or test)")
		case strings.TrimSpace(c.Run[key].Cmd) == "":
			rep.errorf("run."+key, "cmd is empty")
		}
	}

	ev := newEvidenceChecker(ws, repo, rep)
	if ev.notCloned {
		rep.warnf("", "%s is not cloned: evidence not checked", repoPath(repo.Name))
	}

	for i, p := range c.Identity.Ports {
		ev.check(fmt.Sprintf("identity.ports[%d]", i), p.Evidence, false)
	}
	for i, h := range c.Identity.Hosts {
		where := fmt.Sprintf("identity.hosts[%d]", i)
		declared := slices.ContainsFunc(h.Evidence, func(e string) bool { return strings.TrimSpace(e) == "workspace.yaml" })
		if ev.check(where, h.Evidence, true) && declared && !slices.Contains(repo.Hosts, h.Value) {
			rep.errorf(where, "host %s is not declared for %s in workspace.yaml", h.Value, repo.Name)
		}
	}
	if pkg := c.Identity.Package; pkg != nil {
		ev.check("identity.package", pkg.Evidence, false)
	}
	for i, tp := range c.Identity.Topics {
		ev.check(fmt.Sprintf("identity.topics[%d]", i), tp.Evidence, false)
	}
	for i, p := range c.Provides {
		where := strings.TrimSpace(fmt.Sprintf("provides[%d] %s", i, p.ID))
		checkKindID(rep, where, p.Kind, p.ID)
		ev.check(where, p.Evidence, false)
	}
	for i, cs := range c.Consumes {
		where := strings.TrimSpace(fmt.Sprintf("consumes[%d] %s", i, cs.ID))
		checkKindID(rep, where, cs.Kind, cs.ID)
		ev.check(where, cs.Evidence, false)
	}
	for i, cf := range c.Config {
		where := strings.TrimSpace(fmt.Sprintf("config[%d] %s", i, cf.Env))
		if strings.TrimSpace(cf.Env) == "" {
			rep.errorf(where, "env is empty")
		}
		ev.check(where, cf.Evidence, false)
	}
}

func checkKindID(rep report, where, kind, id string) {
	switch kind {
	case "http":
		if !httpID.MatchString(id) || strings.Contains(id, "/:") {
			rep.errorf(where, `id must look like "GET /path" with parameters as {name}`)
		}
	case "event", "package":
		if strings.TrimSpace(id) == "" {
			rep.errorf(where, "id is empty")
		}
	default:
		rep.errorf(where, "unknown kind %q (use %s)", kind, strings.Join(kinds, ", "))
	}
}

type evidenceChecker struct {
	rep       report
	notCloned bool
	place     string                            // where files are read from, for messages
	read      func(path string) ([]byte, error) // reads a file the card cites
	cache     map[string]readResult
}

type readResult struct {
	data []byte
	err  error
}

// newEvidenceChecker reads cited files from origin/<branch> when the clone
// is a git repository that has it, so a local branch, local edits or ignored
// files never change the result. A plain directory is read from disk.
func newEvidenceChecker(ws *workspace.Workspace, repo workspace.Repo, rep report) *evidenceChecker {
	e := &evidenceChecker{rep: rep, cache: map[string]readResult{}}
	dir := filepath.Join(ws.Root, "repos", repo.Name)
	if _, err := os.Stat(dir); err != nil {
		e.notCloned = true
		return e
	}
	g := gitx.Repo{Dir: dir}
	rev := "origin/" + repo.Branch
	if g.IsRepo() {
		if _, err := g.RevParse(rev); err == nil {
			e.place = repoPath(repo.Name) + " at " + rev
			e.read = func(path string) ([]byte, error) { return g.ReadFile(rev, path) }
			return e
		}
	}
	e.place = repoPath(repo.Name)
	e.read = func(path string) ([]byte, error) { return readRegular(dir, path) }
	return e
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

func (e *evidenceChecker) load(path string) ([]byte, error) {
	if r, ok := e.cache[path]; ok {
		return r.data, r.err
	}
	data, err := e.read(path)
	e.cache[path] = readResult{data, err}
	return data, err
}

// check validates an evidence list and reports whether it holds only
// well-formed entries. declaredOK allows the literal "workspace.yaml".
func (e *evidenceChecker) check(where string, list []string, declaredOK bool) bool {
	if len(list) == 0 {
		e.rep.errorf(where, "no evidence")
		return false
	}
	ok := true
	for _, raw := range list {
		ev, err := catalog.ParseEvidence(raw)
		if err != nil {
			e.rep.errorf(where, "%s", err)
			ok = false
			continue
		}
		if ev.Declared {
			if !declaredOK {
				e.rep.errorf(where, `"workspace.yaml" is only valid evidence for identity.hosts`)
				ok = false
			}
			continue
		}
		if e.notCloned {
			continue
		}
		data, err := e.load(ev.Path)
		if err != nil {
			e.rep.errorf(where, "evidence %s: %s (in %s)", raw, err, e.place)
			ok = false
			continue
		}
		if n := countLines(data); ev.End > n {
			e.rep.errorf(where, "evidence %s: line %d is past the end of %s (%d lines)", raw, ev.End, ev.Path, n)
			ok = false
		}
	}
	return ok
}

func countLines(data []byte) int {
	n := bytes.Count(data, []byte("\n"))
	if len(data) > 0 && data[len(data)-1] != '\n' {
		n++
	}
	return n
}

func sortedKeys(m map[string]catalog.Command) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
