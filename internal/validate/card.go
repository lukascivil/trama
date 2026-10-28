package validate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/workspace"
)

var (
	commitSHA = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	isoDate   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	httpID    = regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS) /[^\s?#]*$`)
	runKeys   = []string{"setup", "dev", "test"}
	kinds     = []string{"http", "event", "package"}
)

// checkCard applies the rules that need only this card, workspace.yaml and
// the clone of the repository.
func checkCard(ws *workspace.Workspace, repo workspace.Repo, c *catalog.Card, rep report) {
	if c.Version != 1 {
		rep.errorf("version", "must be 1")
	}
	if c.Name != repo.Name {
		rep.errorf("name", "name is %q but the file is %s", c.Name, cardFile(repo.Name))
	}
	if strings.TrimSpace(c.Description) == "" {
		rep.errorf("description", "is empty")
	}
	if len(c.Stack) == 0 {
		rep.errorf("stack", "is empty")
	}
	if !commitSHA.MatchString(c.MappedAtCommit) {
		rep.errorf("mapped_at_commit", `must be a commit SHA (7 to 40 hex characters): run "trama stamp %s"`, repo.Name)
	}
	if !isoDate.MatchString(c.MappedAt) {
		rep.errorf("mapped_at", "must be a date (YYYY-MM-DD)")
	}
	for _, key := range sortedKeys(c.Run) {
		switch {
		case !slices.Contains(runKeys, key):
			rep.errorf("run."+key, "unknown command (use setup, dev or test)")
		case strings.TrimSpace(c.Run[key].Cmd) == "":
			rep.errorf("run."+key, "cmd is empty")
		}
	}

	ev := evidenceChecker{ws: ws, repo: repo, rep: rep}
	if _, err := os.Stat(filepath.Join(ws.Root, "repos", repo.Name)); err != nil {
		ev.notCloned = true
		rep.warnf("", "%s is not cloned: evidence not checked", repoPath(repo.Name))
	}

	for i, p := range c.Identity.Ports {
		ev.check(fmt.Sprintf("identity.ports[%d]", i), p.Evidence, false)
	}
	for i, h := range c.Identity.Hosts {
		where := fmt.Sprintf("identity.hosts[%d]", i)
		if ev.check(where, h.Evidence, true) && !slices.Contains(repo.Hosts, h.Value) {
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
		where := fmt.Sprintf("provides[%d] %s", i, p.ID)
		checkKindID(rep, where, p.Kind, p.ID)
		ev.check(where, p.Evidence, false)
	}
	for i, cs := range c.Consumes {
		where := fmt.Sprintf("consumes[%d] %s", i, cs.ID)
		checkKindID(rep, where, cs.Kind, cs.ID)
		ev.check(where, cs.Evidence, false)
	}
	for i, cf := range c.Config {
		where := fmt.Sprintf("config[%d] %s", i, cf.Env)
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
	ws        *workspace.Workspace
	repo      workspace.Repo
	rep       report
	notCloned bool
}

// check validates an evidence list and reports whether it holds only
// well-formed entries. declaredOK allows the literal "workspace.yaml".
func (e evidenceChecker) check(where string, list []string, declaredOK bool) bool {
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
		full := filepath.Join(e.ws.Root, "repos", e.repo.Name, filepath.FromSlash(ev.Path))
		data, err := os.ReadFile(full)
		if err != nil {
			e.rep.errorf(where, "evidence %s: %s does not exist in %s", raw, ev.Path, repoPath(e.repo.Name))
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
