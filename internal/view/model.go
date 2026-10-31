package view

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/changes"
	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/release"
	"github.com/lukascivil/trama/internal/status"
	"github.com/lukascivil/trama/internal/validate"
	"github.com/lukascivil/trama/internal/workspace"
)

// Model is everything the graph page shows.
type Model struct {
	Workspace string   `json:"workspace"`
	Repos     []Repo   `json:"repos"`
	Edges     []Edge   `json:"edges"`
	Warnings  []string `json:"warnings"`
}

// Repo is one repository of the workspace, as the page shows it. What it
// consumes is in the edges whose consumer is this repo.
type Repo struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Stack       []string         `json:"stack,omitempty"`
	URL         string           `json:"url,omitempty"` // https://github.com/<slug>, when on GitHub
	Cloned      bool             `json:"cloned"`
	Card        string           `json:"card"`    // current | stale | missing | unknown
	Release     *release.Release `json:"release"` // nil: no semver tag, not cloned, or no origin ref
	Run         []Command        `json:"run,omitempty"`
	Provides    []Provide        `json:"provides,omitempty"`
	OpenChanges []Change         `json:"openChanges,omitempty"`
	Affected    Affected         `json:"affected"`
}

// Command is how to run a repository, from its card.
type Command struct {
	Kind string `json:"kind"` // setup, dev or test
	Cmd  string `json:"cmd"`
	Port int    `json:"port,omitempty"`
}

// Provide is something a repository provides, with its evidence.
type Provide struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Evidence []Link `json:"evidence"`
}

// Change is an open change record with a step in a repository.
type Change struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url,omitempty"`
}

// check validates the workspace; tests count its runs.
var check = validate.Check

// workers bounds how many repositories are read from git at once.
const workers = 8

// Builder builds the model of the workspace at root, rereading it every
// time and keeping the validation while its inputs do not change.
type Builder struct {
	root string

	mu     sync.Mutex
	key    string
	report validate.Report
}

// NewBuilder returns a builder for the workspace at root.
func NewBuilder(root string) *Builder {
	return &Builder{root: root}
}

// Build reads the workspace and returns its model. It fails only when
// workspace.yaml cannot be loaded; any other problem becomes a warning.
func (b *Builder) Build() (*Model, error) {
	ws, err := workspace.Load(b.root)
	if err != nil {
		return nil, err
	}
	m := &Model{Workspace: ws.Name, Repos: []Repo{}, Edges: []Edge{}, Warnings: []string{}}
	sts := status.Collect(ws)
	for _, s := range sts {
		if s.Err != "" {
			m.Warnings = append(m.Warnings, s.Name+": "+s.Err)
		}
	}

	cards := map[string]*catalog.Card{}
	for _, r := range ws.Repos {
		c, err := catalog.Load(ws.Root, r.Name)
		if err != nil {
			m.Warnings = append(m.Warnings, fmt.Sprintf("catalog/%s.yaml: %v", r.Name, err))
			continue
		}
		if c != nil {
			cards[r.Name] = c
		}
	}

	report := b.validate(ws, sts)
	for _, f := range report.Findings {
		m.Warnings = append(m.Warnings, f.String())
	}
	m.Edges = edges(ws, cards, problems(report.Findings))

	releases := latestReleases(ws, sts)
	open := openChanges(ws, &m.Warnings)
	for i, r := range ws.Repos {
		repo := Repo{
			Name:        r.Name,
			URL:         repoURL(r.URL),
			Cloned:      sts[i].Cloned,
			Card:        string(sts[i].Card),
			Release:     releases[i],
			OpenChanges: open[r.Name],
			Affected:    AffectedBy(r.Name, m.Edges),
		}
		if c := cards[r.Name]; c != nil {
			repo.Description, repo.Stack = c.Description, c.Stack
			for _, kind := range []string{"setup", "dev", "test"} {
				if cmd, ok := c.Run[kind]; ok {
					repo.Run = append(repo.Run, Command{Kind: kind, Cmd: cmd.Cmd, Port: cmd.Port})
				}
			}
			for _, p := range c.Provides {
				repo.Provides = append(repo.Provides, Provide{Kind: p.Kind, ID: p.ID, Evidence: links(r.URL, c.MappedAtCommit, p.Evidence)})
			}
		}
		m.Repos = append(m.Repos, repo)
	}
	return m, nil
}

// validate runs the validation, or reuses the last report while
// workspace.yaml, the catalog and every origin/<branch> are the same.
func (b *Builder) validate(ws *workspace.Workspace, sts []status.RepoStatus) validate.Report {
	h := sha256.New()
	if data, err := os.ReadFile(filepath.Join(ws.Root, workspace.FileName)); err == nil {
		h.Write(data)
	}
	if entries, err := os.ReadDir(filepath.Join(ws.Root, "catalog")); err == nil {
		for _, e := range entries {
			data, _ := os.ReadFile(filepath.Join(ws.Root, "catalog", e.Name()))
			fmt.Fprintf(h, "\x00%s\x00%d\x00", e.Name(), len(data))
			h.Write(data)
		}
	}
	for _, s := range sts {
		fmt.Fprintf(h, "\x00%s=%s", s.Name, s.OriginSHA)
	}
	key := fmt.Sprintf("%x", h.Sum(nil))

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.key != key {
		b.report, b.key = check(ws), key
	}
	return b.report
}

// cardProblems are the validation errors of one card: those of each
// consume, by index, and those of the whole file.
type cardProblems struct {
	consumes map[int][]string
	file     []string
}

// problems sorts the validation errors by card.
func problems(fs []validate.Finding) map[string]*cardProblems {
	out := map[string]*cardProblems{}
	for _, f := range fs {
		if f.Severity != validate.Error || !strings.HasPrefix(f.File, "catalog/") || !strings.HasSuffix(f.File, ".yaml") {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(f.File, "catalog/"), ".yaml")
		p := out[name]
		if p == nil {
			p = &cardProblems{consumes: map[int][]string{}}
			out[name] = p
		}
		if f.Where == "" {
			p.file = append(p.file, f.Message)
			continue
		}
		rest, ok := strings.CutPrefix(f.Where, "consumes[")
		if !ok {
			continue
		}
		num, _, ok := strings.Cut(rest, "]")
		if i, err := strconv.Atoi(num); ok && err == nil {
			p.consumes[i] = append(p.consumes[i], f.Message)
		}
	}
	return out
}

// edges lists every consume of the cards, in workspace and card order,
// then the declared links that no card cites.
func edges(ws *workspace.Workspace, cards map[string]*catalog.Card, probs map[string]*cardProblems) []Edge {
	out := []Edge{}
	for _, r := range ws.Repos {
		c := cards[r.Name]
		if c == nil {
			continue
		}
		p := probs[r.Name]
		for i, cs := range c.Consumes {
			e := Edge{Consumer: r.Name, Kind: cs.Kind, ID: cs.ID, Via: cs.Via, Evidence: links(r.URL, c.MappedAtCommit, cs.Evidence)}
			switch {
			case cs.From == catalog.Unknown:
				e.Status, e.Candidates = Pending, cs.Candidates
				out = append(out, e)
				continue
			case p != nil:
				e.Problems = append(append(e.Problems, p.file...), p.consumes[i]...)
			}
			e.Provider = cs.From
			if _, ok := ws.Repo(cs.From); !ok {
				e.Problems = append(e.Problems, fmt.Sprintf("from %q is not a repository of the workspace", cs.From))
			}
			e.Status = Proven
			if len(e.Problems) > 0 {
				e.Status = Unverified
			}
			out = append(out, e)
		}
	}
	for _, l := range ws.Links {
		cited := false
		for _, e := range out {
			if e.Consumer == l.Consumer && e.Provider == l.Provider && e.ID == l.ID {
				cited = true
				break
			}
		}
		if !cited {
			out = append(out, Edge{Consumer: l.Consumer, Provider: l.Provider, ID: l.ID, Status: Declared, Evidence: []Link{{Text: "workspace.yaml"}}})
		}
	}
	return out
}

// links turns evidence into links to the lines on GitHub, at the commit
// the card was mapped at, for a repository on GitHub.
func links(repoURL, commit string, evidence []string) []Link {
	slug := changes.GitHubSlug(repoURL)
	out := []Link{}
	for _, raw := range evidence {
		l := Link{Text: raw}
		if ev, err := catalog.ParseEvidence(raw); err == nil && !ev.Declared && slug != "" && commit != "" {
			l.URL = fmt.Sprintf("https://github.com/%s/blob/%s/%s#L%d", slug, commit, ev.Path, ev.Start)
			if ev.End != ev.Start {
				l.URL += fmt.Sprintf("-L%d", ev.End)
			}
		}
		out = append(out, l)
	}
	return out
}

// repoURL is the GitHub page of a repository, or "".
func repoURL(url string) string {
	if slug := changes.GitHubSlug(url); slug != "" {
		return "https://github.com/" + slug
	}
	return ""
}

// latestReleases reads the latest release of every repository whose
// origin/<branch> exists, several at a time; failures become nil.
func latestReleases(ws *workspace.Workspace, sts []status.RepoStatus) []*release.Release {
	out := make([]*release.Release, len(ws.Repos))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i, r := range ws.Repos {
		if sts[i].OriginSHA == "" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			out[i], _ = release.Latest(gitx.Repo{Dir: ws.RepoDir(r.Name)}, "origin/"+r.Branch)
		}()
	}
	wg.Wait()
	return out
}

// openChanges maps each repository to the open change records with a step
// in it that is not merged.
func openChanges(ws *workspace.Workspace, warnings *[]string) map[string][]Change {
	records, errs := changes.Load(ws.Root)
	for _, err := range errs {
		*warnings = append(*warnings, "change record: "+err.Error())
	}
	var origin, branch string
	wsRepo := gitx.Repo{Dir: ws.Root}
	if o, err := wsRepo.OriginURL(); err == nil {
		origin = o
		// The current branch, not OriginHead: that may ask the network.
		if branch, _ = wsRepo.CurrentBranch(); branch == "" {
			branch = "main"
		}
	}
	out := map[string][]Change{}
	for _, rec := range records {
		if rec.Status != "open" {
			continue
		}
		seen := map[string]bool{}
		for _, s := range rec.OpenSteps() {
			if seen[s.Repo] {
				continue
			}
			seen[s.Repo] = true
			out[s.Repo] = append(out[s.Repo], Change{ID: rec.ID, Title: rec.Title, URL: changes.RecordURL(origin, branch, "changes/"+rec.ID+".md")})
		}
	}
	for _, list := range out {
		sort.SliceStable(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	}
	return out
}
