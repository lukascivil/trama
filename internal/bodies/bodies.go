// Package bodies writes the request and response of every route an OpenAPI 3
// spec of the repository declares. The spec owns the bodies of the routes it
// declares, as trama link owns from and via: every run rewrites them, and a
// body the spec no longer declares is removed. Bodies written by hand stay
// only on routes no committed spec declares.
//
// Which spec counts for a route: the card's contract when it names an
// OpenAPI 3 spec; without a contract, the one spec of the repository that
// declares the route, which then also becomes the contract. When two specs
// declare it, nothing is written: the route does not say which one serves it.
package bodies

import (
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/fsx"
	"github.com/lukascivil/trama/internal/openapi"
	"github.com/lukascivil/trama/internal/workspace"
	"github.com/lukascivil/trama/internal/yamlx"
)

// Change is one provide that Apply would rewrite.
type Change struct {
	Repo     string
	Index    int // position in the card's provides
	ID       string
	Spec     string // the spec the fields come from
	Line     int    // the line of the route in Spec
	Contract string // the contract to write; "" when it stays as it is
	Request  Edit
	Response Edit
}

// Edit is what happens to one body. Without Write it stays as it is; with
// Write and no Fields it is removed.
type Edit struct {
	Write  bool
	Fields []openapi.Field
}

// Summary says what the change does, e.g. "contract, response 5 fields".
func (ch Change) Summary() string {
	var parts []string
	if ch.Contract != "" {
		parts = append(parts, "contract")
	}
	for _, b := range []struct {
		name string
		edit Edit
	}{{"request", ch.Request}, {"response", ch.Response}} {
		switch {
		case !b.edit.Write:
		case len(b.edit.Fields) == 0:
			parts = append(parts, "no "+b.name)
		case len(b.edit.Fields) == 1:
			parts = append(parts, b.name+" 1 field")
		default:
			parts = append(parts, fmt.Sprintf("%s %d fields", b.name, len(b.edit.Fields)))
		}
	}
	return strings.Join(parts, ", ")
}

// Report is the plan for one card: what would change, and what was left as
// it is and why.
type Report struct {
	Repo    string
	Changes []Change
	Notes   []string
}

// Plan lists what writing the bodies of the cards of names would change,
// or of every card when names is empty. It never writes.
func Plan(ws *workspace.Workspace, names []string) ([]Report, error) {
	repos := ws.Repos
	if len(names) > 0 {
		repos = nil
		for _, n := range names {
			r, ok := ws.Repo(n)
			if !ok {
				return nil, fmt.Errorf("repo %s is not in workspace.yaml", n)
			}
			if !slices.ContainsFunc(repos, func(x workspace.Repo) bool { return x.Name == n }) {
				repos = append(repos, r)
			}
		}
	}
	var out []Report
	for _, r := range repos {
		c, err := catalog.LoadStrict(ws.Root, r.Name)
		if err != nil {
			return nil, err
		}
		if c == nil {
			if len(names) > 0 {
				return nil, fmt.Errorf("no card at catalog/%s.yaml: write it first", r.Name)
			}
			continue
		}
		chs, notes := ForCard(r.Name, c, ws.Files(r))
		if len(chs) > 0 || len(notes) > 0 {
			out = append(out, Report{Repo: r.Name, Changes: chs, Notes: notes})
		}
	}
	return out, nil
}

// ForCard plans the bodies of one card, reading the specs through files.
// The repository is scanned for specs only when a route has no contract.
func ForCard(name string, c *catalog.Card, files workspace.Files) ([]Change, []string) {
	if !slices.ContainsFunc(c.Provides, func(p catalog.Provide) bool { return p.Kind == "http" }) {
		return nil, nil
	}
	if !files.Cloned {
		return nil, []string{fmt.Sprintf(`%s is not cloned: run "trama sync %s"`, files.Place, name)}
	}
	r := &reader{files: files, specs: map[string]*openapi.Spec{}, problems: map[string]string{}}
	var (
		changes []Change
		notes   []string
		used    []*openapi.Spec // specs that declare a route of the card
		inNone  int             // routes without contract that no spec declares
		noted   = map[string]bool{}
	)
	for i, p := range c.Provides {
		if p.Kind != "http" {
			continue
		}
		var spec *openapi.Spec
		contract := ""
		if p.Contract != "" {
			s, problem := r.contract(p.Contract)
			if problem != "" && !noted[problem] {
				noted[problem] = true
				notes = append(notes, problem)
			}
			if s == nil {
				continue
			}
			if _, ok := s.Find(p.ID); !ok {
				notes = append(notes, fmt.Sprintf("%s: not in contract %s", p.ID, p.Contract))
				continue
			}
			spec = s
		} else {
			specs, err := r.scan()
			if err != nil {
				return changes, append(notes, fmt.Sprintf("%s: %s", files.Place, err))
			}
			var found []*openapi.Spec
			for _, s := range specs {
				if _, ok := s.Find(p.ID); ok {
					found = append(found, s)
				}
			}
			switch len(found) {
			case 0:
				inNone++
				continue
			case 1:
				spec, contract = found[0], found[0].Path
			default:
				var paths []string
				for _, s := range found {
					paths = append(paths, s.Path)
				}
				notes = append(notes, fmt.Sprintf("%s: declared by %s: set contract to the one that serves it", p.ID, and(paths)))
				continue
			}
		}
		if !slices.Contains(used, spec) {
			used = append(used, spec)
		}
		op, _ := spec.Find(p.ID)
		ch := Change{Repo: name, Index: i, ID: p.ID, Spec: spec.Path, Line: op.Line, Contract: contract}
		var problem string
		if ch.Request, problem = edit(p.Request, op.Request()); problem != "" {
			notes = append(notes, fmt.Sprintf("%s: request %s: left as it is", p.ID, problem))
		}
		if ch.Response, problem = edit(p.Response, op.Response()); problem != "" {
			notes = append(notes, fmt.Sprintf("%s: response %s: left as it is", p.ID, problem))
		}
		if ch.Contract != "" || ch.Request.Write || ch.Response.Write {
			changes = append(changes, ch)
		}
	}
	if inNone > 0 && len(r.scanned) > 0 {
		if inNone == 1 {
			notes = append(notes, "1 route is in no OpenAPI spec of the repository")
		} else {
			notes = append(notes, fmt.Sprintf("%d routes are in no OpenAPI spec of the repository", inNone))
		}
	}
	return changes, append(notes, unlisted(c, used)...)
}

// unlisted says, for each spec that declares a route of the card, the
// routes it declares that the card does not list. The router is where the
// routes come from, and a spec can be ahead of it or behind it, so this is
// only a note for whoever maps the repository to check.
func unlisted(c *catalog.Card, specs []*openapi.Spec) []string {
	listed := map[string]bool{}
	for _, p := range c.Provides {
		if p.Kind == "http" {
			listed[catalog.NormalizeID("http", p.ID)] = true
		}
	}
	slices.SortFunc(specs, func(a, b *openapi.Spec) int { return strings.Compare(a.Path, b.Path) })
	var notes []string
	for _, s := range specs {
		var missing []string
		for _, op := range s.Operations() {
			if !listed[catalog.NormalizeID("http", op.ID)] {
				missing = append(missing, op.ID)
			}
		}
		switch n := len(missing); {
		case n == 0:
		case n == 1:
			notes = append(notes, fmt.Sprintf("%s declares 1 route the card does not list: %s", s.Path, missing[0]))
		default:
			sample := strings.Join(missing[:min(5, n)], ", ")
			if n > 5 {
				sample += ", …"
			}
			notes = append(notes, fmt.Sprintf("%s declares %d routes the card does not list: %s", s.Path, n, sample))
		}
	}
	return notes
}

// edit compares a body of the card with what the spec says about it, and
// returns the problem when the spec declares a body trama cannot read.
func edit(card map[string]string, spec openapi.Body) (Edit, string) {
	switch {
	case !spec.Declared:
		return Edit{Write: card != nil}, ""
	case spec.Problem != "":
		return Edit{}, spec.Problem
	case same(card, spec.Fields):
		return Edit{}, ""
	}
	return Edit{Write: true, Fields: spec.Fields}, ""
}

func same(card map[string]string, fields []openapi.Field) bool {
	if card == nil || len(card) != len(fields) {
		return false
	}
	for _, f := range fields {
		if t, ok := card[f.Path]; !ok || t != f.Type {
			return false
		}
	}
	return true
}

// and joins "a.yaml and b.yaml", or "a.yaml, b.yaml and c.yaml".
func and(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

// reader reads the specs of one repository, each at most once.
type reader struct {
	files    workspace.Files
	specs    map[string]*openapi.Spec // by path
	problems map[string]string        // by contract
	scanned  []*openapi.Spec
	didScan  bool
	scanErr  error
}

func (r *reader) scan() ([]*openapi.Spec, error) {
	if !r.didScan {
		r.didScan = true
		r.scanned, r.scanErr = openapi.Scan(r.files.List, r.files.ReadAll)
		for _, s := range r.scanned {
			r.specs[s.Path] = s
		}
	}
	return r.scanned, r.scanErr
}

// contract reads the spec a contract names. It returns no spec and no
// problem for a contract trama does not read, such as proto or GraphQL.
func (r *reader) contract(contract string) (*openapi.Spec, string) {
	if err := catalog.ValidateContract(contract); err != nil {
		return nil, err.Error()
	}
	if !openapi.IsDocument(contract) {
		return nil, ""
	}
	p := path.Clean(strings.TrimSpace(contract))
	if s, ok := r.specs[p]; ok {
		return s, ""
	}
	if problem, ok := r.problems[p]; ok {
		return nil, problem
	}
	s, problem := r.read(contract, p)
	if problem != "" {
		r.problems[p] = problem
		return nil, problem
	}
	r.specs[p] = s
	return s, ""
}

func (r *reader) read(contract, p string) (*openapi.Spec, string) {
	data, err := r.files.Read(p)
	if err != nil {
		return nil, fmt.Sprintf("contract %s: %s", contract, err)
	}
	s, err := openapi.Parse(p, data)
	if errors.Is(err, openapi.ErrNotSpec) {
		return nil, fmt.Sprintf("contract %s is not an OpenAPI 3 spec", contract)
	}
	if err != nil {
		return nil, fmt.Sprintf("contract %s is not YAML: %s", contract, err)
	}
	return s, ""
}

// Apply writes the changes into the cards, leaving the rest of each file
// as it was.
func Apply(root string, changes []Change) error {
	var repos []string
	byRepo := map[string][]Change{}
	for _, ch := range changes {
		if _, ok := byRepo[ch.Repo]; !ok {
			repos = append(repos, ch.Repo)
		}
		byRepo[ch.Repo] = append(byRepo[ch.Repo], ch)
	}
	for _, repo := range repos {
		file := catalog.Path(root, repo)
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		out, err := rewrite(data, byRepo[repo])
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		if err := fsx.ReplaceFile(file, out, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// rewrite applies chs to the card bytes, keeping everything else, including
// comments and the order of the keys: kind, id, contract, request,
// response, evidence.
func rewrite(data []byte, chs []Change) ([]byte, error) {
	doc, err := yamlx.Parse(data)
	if err != nil {
		return nil, err
	}
	provides := yamlx.Field(doc.Content[0], "provides")
	if provides == nil || provides.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("the card has no provides list")
	}
	for _, ch := range chs {
		if ch.Index >= len(provides.Content) {
			return nil, fmt.Errorf("provides[%d] is not in the card any more", ch.Index)
		}
		entry := provides.Content[ch.Index]
		if got := yamlx.Field(entry, "id"); entry.Kind != yaml.MappingNode || got == nil || got.Value != ch.ID {
			return nil, fmt.Errorf("provides[%d] is no longer %s", ch.Index, ch.ID)
		}
		if ch.Contract != "" {
			yamlx.Set(entry, "contract", yamlx.Scalar(ch.Contract), "id", "kind")
		}
		setBody(entry, "request", ch.Request, "contract", "id", "kind")
		setBody(entry, "response", ch.Response, "request", "contract", "id", "kind")
		if len(ch.Request.Fields) > 0 || len(ch.Response.Fields) > 0 {
			block(provides, entry)
		}
	}
	return yamlx.Encode(doc)
}

func setBody(entry *yaml.Node, key string, e Edit, after ...string) {
	switch {
	case !e.Write:
	case len(e.Fields) == 0:
		yamlx.Remove(entry, key)
	default:
		m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, f := range e.Fields {
			v := yamlx.Scalar(f.Type)
			if strings.HasPrefix(f.Type, "[") {
				v.Style = yaml.DoubleQuotedStyle // a bare [ starts a YAML list
			}
			m.Content = append(m.Content, yamlx.Scalar(f.Path), v)
		}
		yamlx.Set(entry, key, m, after...)
	}
}

// block turns an entry written on one line into a block, one field per
// line, so the bodies it gains read as a list. The comment at the end of
// its line moves to the end of its last line.
func block(provides, entry *yaml.Node) {
	if entry.Style&yaml.FlowStyle == 0 {
		return
	}
	provides.Style &^= yaml.FlowStyle
	entry.Style &^= yaml.FlowStyle
	if entry.LineComment != "" && len(entry.Content) > 0 {
		last := entry.Content[len(entry.Content)-1]
		if last.LineComment == "" {
			last.LineComment = entry.LineComment
		} else {
			last.LineComment += " " + entry.LineComment
		}
		entry.LineComment = ""
	}
}
