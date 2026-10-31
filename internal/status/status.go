// Package status reports the state of every repository in a workspace
// using local git data only, so it is fast and works offline.
package status

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/changes"
	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/workspace"
)

// Card is the state of a repository's catalog card.
type Card string

const (
	CardMissing Card = "missing"
	CardCurrent Card = "current"
	CardStale   Card = "stale"
	CardUnknown Card = "unknown" // card exists but the repo is not cloned, so it cannot be checked
)

// RepoStatus describes one repository.
type RepoStatus struct {
	Name    string
	Cloned  bool
	Branch  string // current branch; "" when detached
	Main    string // main branch from workspace.yaml
	Clean   bool
	Ahead   int // local Main vs origin/Main
	Behind  int
	Card    Card
	Pending []string // consumes with from: unknown, as "<id> (candidates: a, b)"
	Err     string   // repo could not be inspected
	// OriginSHA is origin/<Main> in the clone; "" when it cannot be resolved.
	OriginSHA string
}

// workers bounds how many repositories Collect inspects at once.
const workers = 8

// Collect inspects every repository of ws, several at a time, and returns
// them in workspace order. It never touches the network.
func Collect(ws *workspace.Workspace) []RepoStatus {
	sts := make([]RepoStatus, len(ws.Repos))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i, r := range ws.Repos {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			sts[i] = collectRepo(ws, r)
		}()
	}
	wg.Wait()
	return sts
}

func collectRepo(ws *workspace.Workspace, r workspace.Repo) RepoStatus {
	s := RepoStatus{Name: r.Name, Main: r.Branch}
	var errs []string
	entry, cardErr := catalog.Load(ws.Root, r.Name)
	if cardErr != nil {
		errs = append(errs, "unreadable catalog card: "+cardErr.Error())
	}
	if entry != nil {
		s.Pending = pending(entry)
	}
	cardOrUnknown := func() Card {
		if entry == nil && cardErr == nil {
			return CardMissing
		}
		return CardUnknown
	}

	dir := ws.RepoDir(r.Name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		s.Card = cardOrUnknown()
		s.Err = strings.Join(errs, "; ")
		return s
	}
	s.Cloned = true
	repo := gitx.Repo{Dir: dir}
	if !repo.IsRepo() {
		s.Card = cardOrUnknown()
		s.Err = strings.Join(append(errs, "repos/"+r.Name+" is not a git repository"), "; ")
		return s
	}
	s.Branch, _ = repo.CurrentBranch()
	s.Clean, _ = repo.IsClean()
	if origin, err := repo.OriginURL(); err != nil || !gitx.SameRemote(origin, r.URL) {
		if err != nil {
			errs = append(errs, "repos/"+r.Name+" has no origin remote")
		} else {
			errs = append(errs, fmt.Sprintf("origin is %s, workspace.yaml says %s", origin, r.URL))
		}
		s.Card = cardOrUnknown()
		s.Err = strings.Join(errs, "; ")
		return s
	}

	originRef := "origin/" + r.Branch
	originSHA, err := repo.RevParse(originRef)
	if err != nil {
		originSHA = ""
		errs = append(errs, fmt.Sprintf(`no %s ref: run "trama sync" (if it fails, check branch in workspace.yaml)`, originRef))
	} else {
		s.OriginSHA = originSHA
		if _, err := repo.RevParse(r.Branch); err == nil {
			s.Ahead, s.Behind, _ = repo.AheadBehind(r.Branch, originRef)
		}
	}

	switch {
	case cardErr != nil:
		s.Card = CardStale
	case entry == nil:
		s.Card = CardMissing
	case originSHA == "":
		s.Card = CardUnknown
	default:
		mapped, err := repo.RevParse(entry.MappedAtCommit + "^{commit}")
		if !catalog.IsCommitSHA(entry.MappedAtCommit) || err != nil || mapped != originSHA {
			s.Card = CardStale
		} else {
			s.Card = CardCurrent
		}
	}
	s.Err = strings.Join(errs, "; ")
	return s
}

func pending(c *catalog.Card) []string {
	var out []string
	for _, cs := range c.Consumes {
		if cs.From != catalog.Unknown {
			continue
		}
		cands := "no candidates"
		if len(cs.Candidates) > 0 {
			cands = "candidates: " + strings.Join(cs.Candidates, ", ")
		}
		out = append(out, fmt.Sprintf("%s (%s)", cs.ID, cands))
	}
	return out
}

// Problems lists what needs attention, most urgent first. The current
// branch and local changes are information, not problems.
func (s RepoStatus) Problems() []string {
	var p []string
	if !s.Cloned {
		p = append(p, `not cloned: run "trama sync"`)
	}
	if s.Err != "" {
		p = append(p, s.Err)
	}
	if s.Behind > 0 {
		p = append(p, fmt.Sprintf(`%d commit(s) behind origin/%s: run "trama sync"`, s.Behind, s.Main))
	}
	switch s.Card {
	case CardMissing:
		p = append(p, fmt.Sprintf(`no catalog card: run "/trama:map %s"`, s.Name))
	case CardStale:
		p = append(p, fmt.Sprintf(`catalog card is stale: run "/trama:map %s"`, s.Name))
	}
	for _, pend := range s.Pending {
		p = append(p, "pending: consumes "+pend+" from an unknown provider")
	}
	return p
}

// Render writes a table of every repository followed by its problems. In
// quiet mode it writes only the problems, one per line, and nothing at all
// when there are none.
func Render(w io.Writer, sts []RepoStatus, quiet bool) {
	if quiet {
		for _, s := range sts {
			for _, p := range s.Problems() {
				fmt.Fprintf(w, "trama: %s: %s\n", s.Name, p)
			}
		}
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "REPO\tBRANCH\tTREE\tVS ORIGIN\tCARD")
	for _, s := range sts {
		branch, tree, sync := "-", "-", "-"
		if s.Cloned && s.Err == "" {
			branch = s.Branch
			if branch == "" {
				branch = "(detached)"
			}
			tree = "clean"
			if !s.Clean {
				tree = "dirty"
			}
			sync = fmt.Sprintf("+%d -%d", s.Ahead, s.Behind)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", s.Name, branch, tree, sync, s.Card)
	}
	tw.Flush()
	var lines []string
	for _, s := range sts {
		for _, p := range s.Problems() {
			lines = append(lines, fmt.Sprintf("- %s: %s", s.Name, p))
		}
	}
	if len(lines) > 0 {
		fmt.Fprintf(w, "\n%s\n", strings.Join(lines, "\n"))
	}
}

// OpenChanges lists one line per change record whose steps are not all
// merged, and one per record that cannot be read.
func OpenChanges(root string) []string {
	records, errs := changes.Load(root)
	var lines []string
	for _, r := range records {
		if r.Status == "open" {
			lines = append(lines, fmt.Sprintf("change %s: %d step(s) not merged", r.ID, len(r.OpenSteps())))
		}
	}
	for _, err := range errs {
		lines = append(lines, "change record: "+err.Error())
	}
	return lines
}
