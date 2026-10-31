// Package runner runs the commands of a repository's catalog card: setup,
// dev and test, one at a time or all repositories at once.
package runner

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/workspace"
)

// Kind is a key of run in a catalog card.
type Kind string

const (
	Setup Kind = "setup"
	Dev   Kind = "dev"
	Test  Kind = "test"
)

// Job is one command from a repository's card.
type Job struct {
	Repo  string
	Kind  Kind
	Dir   string            // absolute path of repos/<name>
	Cmd   string            // shell command from the card
	Tools map[string]string // from the card
	Port  int               // run.<kind>.port, 0 when not set
}

// ErrNoCommand means the card has no run.<kind>.
var ErrNoCommand = errors.New("no such command in the catalog card")

// JobFor reads catalog/<name>.yaml and returns its run.<kind>. It never
// guesses a command: without a card or without the key it fails.
func JobFor(ws *workspace.Workspace, name string, kind Kind) (Job, error) {
	if _, ok := ws.Repo(name); !ok {
		return Job{}, fmt.Errorf("repo %s is not in workspace.yaml", name)
	}
	card, err := catalog.Load(ws.Root, name)
	if err != nil {
		return Job{}, err
	}
	if card == nil {
		return Job{}, fmt.Errorf(`no catalog card for %s: run "/trama:map %s"`, name, name)
	}
	dir := ws.RepoDir(name)
	if _, err := os.Stat(dir); err != nil {
		return Job{}, fmt.Errorf(`repos/%s is not cloned: run "trama sync %s"`, name, name)
	}
	c, ok := card.Run[string(kind)]
	if !ok || c.Cmd == "" {
		return Job{}, &noCommandError{name: name, kind: kind}
	}
	return Job{Repo: name, Kind: kind, Dir: dir, Cmd: c.Cmd, Tools: card.Tools, Port: c.Port}, nil
}

type noCommandError struct {
	name string
	kind Kind
}

func (e *noCommandError) Error() string {
	return fmt.Sprintf("%s has no run.%s in its catalog card", e.name, e.kind)
}

func (e *noCommandError) Unwrap() error { return ErrNoCommand }

// Args returns the program and its arguments: the command through sh, inside
// "mise exec" with the card's tool versions when there are any.
func (j Job) Args() []string {
	shell := []string{"sh", "-c", j.Cmd}
	if len(j.Tools) == 0 {
		return shell
	}
	keys := make([]string, 0, len(j.Tools))
	for k := range j.Tools {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	args := []string{"mise", "exec"}
	for _, k := range keys {
		args = append(args, k+"@"+j.Tools[k])
	}
	return append(append(args, "--"), shell...)
}

// Describe is the line printed before the command runs.
func (j Job) Describe() string {
	return fmt.Sprintf("$ %s  (in repos/%s)", j.Cmd, j.Repo)
}
