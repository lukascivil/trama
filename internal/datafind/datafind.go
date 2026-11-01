// Package datafind searches the catalog cards of a workspace for a piece of
// data: a field an endpoint accepts or returns, or a column of a table a
// repository owns. It answers in the order the answers are useful, so the
// caller never has to decide which layer comes first.
package datafind

import (
	"maps"
	"slices"
	"strings"

	"github.com/lukascivil/trama/internal/catalog"
)

// Hit is one place a piece of data was found.
type Hit struct {
	Layer    string // response, request or table
	Repo     string
	ID       string // the route or topic; empty in a table hit
	Table    string // the table; empty in a field hit
	Path     string // the field path, or the column name
	Type     string // the field type; empty in a table hit
	Match    string // the text a term is compared against
	Evidence []string
}

// All returns every searchable field and column, grouped by layer in the
// order the answers are useful — a field that already comes from an endpoint
// can be called today, a column only means the data has an owner — and, in
// each layer, in the order of repos. A repo with no card is skipped.
func All(repos []string, cards map[string]*catalog.Card) []Hit {
	var out []Hit
	for _, layer := range []string{"response", "request"} {
		for _, name := range repos {
			card := cards[name]
			if card == nil {
				continue
			}
			for _, p := range card.Provides {
				fields := p.Response
				if layer == "request" {
					fields = p.Request
				}
				for _, path := range slices.Sorted(maps.Keys(fields)) {
					out = append(out, Hit{
						Layer: layer, Repo: name, ID: p.ID, Path: path,
						Type: fields[path], Match: path, Evidence: p.Evidence,
					})
				}
			}
		}
	}
	for _, name := range repos {
		card := cards[name]
		if card == nil {
			continue
		}
		for _, tb := range card.Identity.Tables {
			for _, col := range tb.Columns {
				out = append(out, Hit{
					Layer: "table", Repo: name, Table: tb.Value, Path: col,
					Match: tb.Value + "." + col, Evidence: tb.Evidence,
				})
			}
		}
	}
	return out
}

// Find returns the hits of All whose Match holds term, ignoring case. An
// empty term finds nothing: it is a question nobody asked.
func Find(repos []string, cards map[string]*catalog.Card, term string) []Hit {
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return nil
	}
	var out []Hit
	for _, h := range All(repos, cards) {
		if strings.Contains(strings.ToLower(h.Match), term) {
			out = append(out, h)
		}
	}
	return out
}
