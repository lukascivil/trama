package validate

import (
	"bytes"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/schemafind"
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
	for i, tb := range c.Identity.Tables {
		where := strings.TrimSpace(fmt.Sprintf("identity.tables[%d] %s", i, tb.Value))
		if err := catalog.ValidateTable(tb.Value); err != nil {
			rep.errorf(where, "%s", err)
		}
		if len(tb.Columns) == 0 {
			rep.errorf(where, "no columns: list the table's columns, or the table answers nothing")
		}
		for _, col := range tb.Columns {
			if err := catalog.ValidateColumn(col); err != nil {
				rep.errorf(where, "%s", err)
			}
		}
		ev.check(where, tb.Evidence, false)
	}
	checkTablesMapped(rep, ev, c)
	for i, p := range c.Provides {
		where := strings.TrimSpace(fmt.Sprintf("provides[%d] %s", i, p.ID))
		checkKindID(rep, where, p.Kind, p.ID)
		checkBodies(rep, where, p)
		ev.check(where, p.Evidence, false)
	}
	for i, cs := range c.Consumes {
		where := strings.TrimSpace(fmt.Sprintf("consumes[%d] %s", i, cs.ID))
		checkKindID(rep, where, cs.Kind, cs.ID)
		for _, f := range cs.FieldsUsed {
			if err := catalog.ValidateField(f, ""); err != nil {
				rep.errorf(where, "fields_used: %s", err)
			}
		}
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

// checkTablesMapped warns when the clone has files that declare tables and
// the card lists none. A card that says "tables: []" was mapped by someone
// who looked and found no table this repository owns, so it is left alone:
// only a missing key is a question. Nothing is said about a repository that
// is not cloned, or whose files cannot be listed: that is reported elsewhere,
// and a warning must never turn into an error here.
func checkTablesMapped(rep report, ev *evidenceChecker, c *catalog.Card) {
	if c.Identity.Tables != nil || ev.notCloned {
		return
	}
	found, err := schemafind.Scan(ev.list, ev.readAll)
	if err != nil || len(found) == 0 {
		return
	}
	paths := make([]string, 0, 3)
	for _, f := range found[:min(3, len(found))] {
		paths = append(paths, f.Path)
	}
	sample := strings.Join(paths, ", ")
	if len(found) > 3 {
		sample += ", …"
	}
	files := "1 schema file"
	if len(found) > 1 {
		files = fmt.Sprintf("%d schema files", len(found))
	}
	rep.warnf("identity.tables", `%s has %s (%s) but the card lists no tables: map them, or write "tables: []" if this repository owns none`, ev.place, files, sample)
}

// checkBodies validates the request and response field maps of a provide:
// where each map is allowed, and the path and type of every field in it.
func checkBodies(rep report, where string, p catalog.Provide) {
	if p.Shape != nil {
		rep.errorf(where, "shape was renamed to response")
	}
	if p.Request != nil && p.Kind != "http" {
		rep.errorf(where, "request is only for kind: http")
	}
	if p.Response != nil && p.Kind != "http" && p.Kind != "event" {
		rep.errorf(where, "response is only for kind: http or event")
	}
	bodies := []struct {
		name   string
		fields map[string]string
	}{{"request", p.Request}, {"response", p.Response}}
	for _, body := range bodies {
		for _, path := range slices.Sorted(maps.Keys(body.fields)) {
			if strings.TrimSpace(body.fields[path]) == "" {
				rep.errorf(where, "%s: field %q has no type (use %s)", body.name, path, strings.Join(catalog.FieldTypes, ", "))
				continue
			}
			if err := catalog.ValidateField(path, body.fields[path]); err != nil {
				rep.errorf(where, "%s: %s", body.name, err)
			}
		}
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
	list      func() ([]string, error)          // lists the files read can read
	readAll   func([]string) (map[string][]byte, error)
	cache     map[string]readResult
}

type readResult struct {
	data []byte
	err  error
}

// newEvidenceChecker reads cited files the way ws.Files decides: from
// origin/<branch> when the clone has it, so a local branch, local edits or
// ignored files never change the result, and from the disk otherwise.
func newEvidenceChecker(ws *workspace.Workspace, repo workspace.Repo, rep report) *evidenceChecker {
	f := ws.Files(repo)
	return &evidenceChecker{rep: rep, notCloned: !f.Cloned, place: f.Place, read: f.Read, list: f.List, readAll: f.ReadAll, cache: map[string]readResult{}}
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
