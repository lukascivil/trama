package datafind_test

import (
	"reflect"
	"testing"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/datafind"
)

var repos = []string{"api", "web"}

// twoCards is an api that serves a route and owns a table, and a web that
// only serves a route.
func twoCards() map[string]*catalog.Card {
	return map[string]*catalog.Card{
		"api": {
			Identity: catalog.Identity{Tables: []catalog.TableFact{
				{Value: "users", Columns: []string{"id", "email", "name"}, Evidence: []string{"migrations/001.sql:3-8"}},
			}},
			Provides: []catalog.Provide{{
				Kind:     "http",
				ID:       "GET /orders/{id}",
				Response: map[string]string{"customer.email": "string", "id": "string"},
				Request:  map[string]string{"note": "string"},
				Evidence: []string{"internal/api/orders.go:42"},
			}},
		},
		"web": {
			Provides: []catalog.Provide{{
				Kind:     "http",
				ID:       "GET /me",
				Response: map[string]string{"email": "string"},
				Evidence: []string{"src/me.ts:1"},
			}},
		},
	}
}

func matches(hits []datafind.Hit) []string {
	out := make([]string, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.Match)
	}
	return out
}

func layers(hits []datafind.Hit) []string {
	out := make([]string, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.Layer)
	}
	return out
}

func TestAllOrdersByLayerThenRepo(t *testing.T) {
	hits := datafind.All(repos, twoCards())
	wantLayers := []string{"response", "response", "response", "request", "table", "table", "table"}
	if got := layers(hits); !reflect.DeepEqual(got, wantLayers) {
		t.Fatalf("layers = %v, want %v", got, wantLayers)
	}
	wantMatches := []string{"customer.email", "id", "email", "note", "users.id", "users.email", "users.name"}
	if got := matches(hits); !reflect.DeepEqual(got, wantMatches) {
		t.Fatalf("matches = %v, want %v", got, wantMatches)
	}
}

func TestAllSortsFieldPaths(t *testing.T) {
	cards := map[string]*catalog.Card{"api": {Provides: []catalog.Provide{{
		Kind: "http", ID: "GET /x",
		Response: map[string]string{"b": "string", "a": "string", "c": "string"},
		Evidence: []string{"main.go:1"},
	}}}}
	want := []string{"a", "b", "c"}
	if got := matches(datafind.All([]string{"api"}, cards)); !reflect.DeepEqual(got, want) {
		t.Fatalf("matches = %v, want %v", got, want)
	}
}

func TestAllTableHitPerColumn(t *testing.T) {
	hits := datafind.All([]string{"api"}, map[string]*catalog.Card{"api": {
		Identity: catalog.Identity{Tables: []catalog.TableFact{
			{Value: "users", Columns: []string{"id", "email", "name"}, Evidence: []string{"migrations/001.sql:3-8"}},
		}},
	}})
	if len(hits) != 3 {
		t.Fatalf("want 3 hits, got %d: %v", len(hits), hits)
	}
	want := datafind.Hit{
		Layer: "table", Repo: "api", Table: "users", Path: "email", Match: "users.email",
		Evidence: []string{"migrations/001.sql:3-8"},
	}
	if !reflect.DeepEqual(hits[1], want) {
		t.Fatalf("hit = %+v, want %+v", hits[1], want)
	}
}

func TestAllSkipsRepoWithoutCard(t *testing.T) {
	cards := twoCards()
	delete(cards, "web")
	hits := datafind.All([]string{"api", "web", "never-mapped"}, cards)
	for _, h := range hits {
		if h.Repo != "api" {
			t.Fatalf("hit from an unmapped repo: %+v", h)
		}
	}
	if len(hits) == 0 {
		t.Fatal("want the api hits, got none")
	}
}

func TestAllFieldHitCarriesTypeAndEvidence(t *testing.T) {
	hits := datafind.All(repos, twoCards())
	want := datafind.Hit{
		Layer: "response", Repo: "api", ID: "GET /orders/{id}", Path: "customer.email",
		Type: "string", Match: "customer.email", Evidence: []string{"internal/api/orders.go:42"},
	}
	if !reflect.DeepEqual(hits[0], want) {
		t.Fatalf("hit = %+v, want %+v", hits[0], want)
	}
}

func TestFindIsCaseInsensitiveSubstring(t *testing.T) {
	want := []string{"customer.email", "email", "users.email"}
	if got := matches(datafind.Find(repos, twoCards(), "EMAIL")); !reflect.DeepEqual(got, want) {
		t.Fatalf("matches = %v, want %v", got, want)
	}
}

func TestFindEmptyTermFindsNothing(t *testing.T) {
	for _, term := range []string{"", "   "} {
		if hits := datafind.Find(repos, twoCards(), term); hits != nil {
			t.Fatalf("Find(%q) = %v, want nil", term, hits)
		}
	}
}

func TestFindMatchesOnlyTheMatchField(t *testing.T) {
	// "api" is a repo name and a kind of nothing in any searchable text.
	if hits := datafind.Find(repos, twoCards(), "api"); len(hits) != 0 {
		t.Fatalf("Find(\"api\") = %v, want none", hits)
	}
}
