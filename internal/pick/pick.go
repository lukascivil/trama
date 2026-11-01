// Package pick lets a person choose repositories of a GitHub organization in a
// page on their own machine and adds the chosen ones to the workspace. The
// page only chooses: this package lists, validates and writes.
package pick

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lukascivil/trama/internal/github"
	"github.com/lukascivil/trama/internal/workspace"
)

// ErrIdle is what Wait returns when nobody asked the picker for anything for
// the whole idle time.
var ErrIdle = errors.New("no request for too long")

// Options is what a Picker works from.
type Options struct {
	Workspace *workspace.Workspace
	// Org is the organization to list. Empty lets the page choose from Orgs.
	Org string
	// Orgs lists the accounts the person can pick from; nil means none.
	Orgs func(context.Context) ([]string, error)
	// List returns the repositories of an organization.
	List func(ctx context.Context, org string) ([]github.Repo, error)
	// Protocol is "ssh" to clone with the ssh URL; anything else uses https.
	Protocol string
	// Authenticated is false when the listing is anonymous, so the page can
	// say that private repositories are missing.
	Authenticated bool
}

// Picker holds one choice session.
type Picker struct {
	o     Options
	token string // proves a write comes from the page this picker served

	mu         sync.Mutex // guards everything below and the workspace
	cache      map[string][]github.Repo
	orgs       []string
	orgsLoaded bool
	added      []string

	done     chan struct{}
	finish   sync.Once
	lastSeen atomic.Int64 // unix nanoseconds of the last request
}

// New returns a Picker for o.
func New(o Options) *Picker {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err) // the system has no randomness
	}
	p := &Picker{o: o, token: hex.EncodeToString(b[:]), cache: map[string][]github.Repo{}, done: make(chan struct{})}
	p.touch()
	return p
}

func (p *Picker) touch() { p.lastSeen.Store(time.Now().UnixNano()) }

func (p *Picker) end() { p.finish.Do(func() { close(p.done) }) }

// Added returns the names of the repositories added so far, in order.
func (p *Picker) Added() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.added...)
}

// Wait blocks until the person adds something or cancels (nil), nothing asks
// the picker for idle (ErrIdle), or ctx ends (its error).
func (p *Picker) Wait(ctx context.Context, idle time.Duration) error {
	for {
		remaining := idle - time.Since(time.Unix(0, p.lastSeen.Load()))
		if remaining <= 0 {
			return ErrIdle
		}
		timer := time.NewTimer(remaining)
		select {
		case <-p.done:
			timer.Stop()
			return nil
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// AddedRepo is a repository that went into workspace.yaml.
type AddedRepo struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Branch string `json:"branch"`
}

// Refused is a chosen repository that did not go in, and why.
type Refused struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// AddResult is what Apply did with a choice.
type AddResult struct {
	Added   []AddedRepo `json:"added"`
	Refused []Refused   `json:"refused"`
}

// Apply adds the repositories of org whose ids (owner/name) are given. Every
// id is resolved against the list this picker loaded itself, and each repo
// goes through the checks of "trama add". The file is written once, with the
// ones that passed. Adding at least one ends the session.
func (p *Picker) Apply(ctx context.Context, org string, ids []string) (AddResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	repos, err := p.listLocked(ctx, org)
	if err != nil {
		return AddResult{}, err
	}
	byID := make(map[string]github.Repo, len(repos))
	for _, r := range repos {
		byID[r.FullName] = r
	}
	ws := p.o.Workspace
	res := AddResult{Added: []AddedRepo{}, Refused: []Refused{}}
	for _, id := range ids {
		r, ok := byID[id]
		if !ok {
			res.Refused = append(res.Refused, Refused{id, "not in the list"})
			continue
		}
		url := p.cloneURL(r)
		repo := workspace.Repo{Name: workspace.NameFromURL(url), URL: url, Branch: r.DefaultBranch}
		if err := ws.AddRepo(repo); err != nil {
			res.Refused = append(res.Refused, Refused{id, err.Error()})
			continue
		}
		res.Added = append(res.Added, AddedRepo{repo.Name, repo.URL, repo.Branch})
	}
	if len(res.Added) == 0 {
		return res, nil
	}
	if err := ws.Save(); err != nil {
		// Drop what AddRepo put in memory, so a retry starts from the file.
		if fresh, loadErr := workspace.Load(ws.Root); loadErr == nil {
			p.o.Workspace = fresh
		}
		return AddResult{}, fmt.Errorf("saving workspace.yaml: %w", err)
	}
	for _, a := range res.Added {
		p.added = append(p.added, a.Name)
	}
	p.end()
	return res, nil
}

func (p *Picker) cloneURL(r github.Repo) string {
	if p.o.Protocol == "ssh" {
		return r.SSHURL
	}
	return r.CloneURL
}

// listLocked returns the repositories of org, asking GitHub once per org.
func (p *Picker) listLocked(ctx context.Context, org string) ([]github.Repo, error) {
	if repos, ok := p.cache[org]; ok {
		return repos, nil
	}
	repos, err := p.o.List(ctx, org)
	if err != nil {
		return nil, err
	}
	p.cache[org] = repos
	return repos, nil
}

// orgsLocked returns the accounts to choose from, asking GitHub once.
func (p *Picker) orgsLocked(ctx context.Context) ([]string, error) {
	if p.o.Orgs == nil {
		return nil, nil
	}
	if !p.orgsLoaded {
		orgs, err := p.o.Orgs(ctx)
		if err != nil {
			return nil, err
		}
		p.orgs, p.orgsLoaded = orgs, true
	}
	return p.orgs, nil
}
