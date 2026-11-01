// Package linker resolves the consumes a card leaves as from: unknown. A
// consume is linked to the one repository of the workspace that provides
// the same thing; when more than one does, the link stays unknown and the
// candidates are recorded instead. The rule is mechanical, so the result
// is the same for everyone who runs it on the same catalog.
package linker

import (
	"fmt"
	"os"
	"slices"

	"go.yaml.in/yaml/v3"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/fsx"
	"github.com/lukascivil/trama/internal/workspace"
	"github.com/lukascivil/trama/internal/yamlx"
)

// Change is one consume that Apply would rewrite. From is empty when the
// provider stays unknown and only the candidates change.
type Change struct {
	Repo       string
	Index      int // position in the card's consumes
	Kind       string
	ID         string
	From       string
	Via        string
	Candidates []string
}

// Plan lists the changes that linking the catalog of ws would make. It
// never writes.
func Plan(ws *workspace.Workspace) ([]Change, error) {
	cards := map[string]*catalog.Card{}
	for _, r := range ws.Repos {
		c, err := catalog.LoadStrict(ws.Root, r.Name)
		if err != nil {
			return nil, err
		}
		if c != nil {
			cards[r.Name] = c
		}
	}
	var out []Change
	for _, r := range ws.Repos {
		c := cards[r.Name]
		if c == nil {
			continue
		}
		for i, cs := range c.Consumes {
			if cs.From != catalog.Unknown {
				continue
			}
			found := providers(ws, cards, r.Name, cs)
			switch {
			case len(found) == 1:
				via := viaFor(cs, cards[found[0]])
				if via == "" {
					continue
				}
				out = append(out, Change{Repo: r.Name, Index: i, Kind: cs.Kind, ID: cs.ID, From: found[0], Via: via})
			case len(found) > 1 && !slices.Equal(found, cs.Candidates):
				out = append(out, Change{Repo: r.Name, Index: i, Kind: cs.Kind, ID: cs.ID, Candidates: found})
			}
		}
	}
	return out, nil
}

// providers lists, in workspace order, the mapped repositories other than
// name whose card provides what cs consumes.
func providers(ws *workspace.Workspace, cards map[string]*catalog.Card, name string, cs catalog.Consume) []string {
	want := catalog.NormalizeID(cs.Kind, cs.ID)
	var out []string
	for _, r := range ws.Repos {
		pc := cards[r.Name]
		if r.Name == name || pc == nil {
			continue
		}
		if slices.ContainsFunc(pc.Provides, func(p catalog.Provide) bool {
			return p.Kind == cs.Kind && catalog.NormalizeID(p.Kind, p.ID) == want
		}) {
			out = append(out, r.Name)
		}
	}
	return out
}

// viaFor names the proof for a consume of this kind, or "" when the
// provider's card does not carry what that proof needs.
func viaFor(cs catalog.Consume, pc *catalog.Card) string {
	switch cs.Kind {
	case "http":
		return "route"
	case "event":
		if slices.ContainsFunc(pc.Identity.Topics, func(f catalog.Fact) bool { return f.Value == cs.ID }) {
			return "topic"
		}
	case "package":
		if pc.Identity.Package != nil && pc.Identity.Package.Value == cs.ID {
			return "package"
		}
	}
	return ""
}

// Apply writes the changes into the cards, leaving the rest of each file
// as it was.
func Apply(root string, changes []Change) error {
	byRepo := map[string][]Change{}
	for _, ch := range changes {
		byRepo[ch.Repo] = append(byRepo[ch.Repo], ch)
	}
	for repo, chs := range byRepo {
		path := catalog.Path(root, repo)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out, err := edit(data, chs)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if err := fsx.ReplaceFile(path, out, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// edit rewrites the consumes named by chs in the card bytes, keeping
// everything else, including comments and the order of the keys.
func edit(data []byte, chs []Change) ([]byte, error) {
	doc, err := yamlx.Parse(data)
	if err != nil {
		return nil, err
	}
	consumes := yamlx.Field(doc.Content[0], "consumes")
	if consumes == nil || consumes.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("the card has no consumes list")
	}
	for _, ch := range chs {
		if ch.Index >= len(consumes.Content) {
			return nil, fmt.Errorf("consumes[%d] is not in the card any more", ch.Index)
		}
		entry := consumes.Content[ch.Index]
		if got := yamlx.Field(entry, "id"); got == nil || got.Value != ch.ID {
			return nil, fmt.Errorf("consumes[%d] is no longer %s", ch.Index, ch.ID)
		}
		if ch.From != "" {
			yamlx.Set(entry, "from", yamlx.Scalar(ch.From))
			yamlx.Set(entry, "via", yamlx.Scalar(ch.Via), "from")
			yamlx.Remove(entry, "candidates")
			continue
		}
		yamlx.Set(entry, "candidates", yamlx.List(ch.Candidates), "via", "from")
	}
	return yamlx.Encode(doc)
}
