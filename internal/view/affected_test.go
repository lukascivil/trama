package view_test

import (
	"reflect"
	"testing"

	"github.com/lukascivil/trama/internal/view"
)

func edge(consumer, provider, id string, status view.EdgeStatus) view.Edge {
	return view.Edge{Consumer: consumer, Provider: provider, Kind: "http", ID: id, Status: status}
}

// sketch is the example of the approved sketch: a gateway in front of
// orders-api, two direct consumers more, and two apps behind the gateway.
func sketch() []view.Edge {
	return []view.Edge{
		edge("web-app", "api-gateway", "GET /orders/{id}", view.Proven),
		edge("mobile-app", "api-gateway", "GET /orders/{id}", view.Proven),
		edge("api-gateway", "orders-api", "GET /orders/{id}", view.Proven),
		edge("payments-api", "orders-api", "PATCH /orders/{id}/status", view.Proven),
		{Consumer: "notifications-worker", Provider: "orders-api", Kind: "event", ID: "orders.created", Status: view.Proven},
	}
}

func hits(hs []view.Hit) []string {
	var out []string
	for _, h := range hs {
		out = append(out, h.Repo+" "+h.ID)
	}
	return out
}

func TestAffectedDirect(t *testing.T) {
	got := view.AffectedBy("orders-api", sketch())
	want := []string{"api-gateway GET /orders/{id}", "payments-api PATCH /orders/{id}/status", "notifications-worker orders.created"}
	if !reflect.DeepEqual(hits(got.Direct), want) {
		t.Fatalf("direct = %v, want %v", hits(got.Direct), want)
	}
}

func TestAffectedIndirectThroughGateway(t *testing.T) {
	got := view.AffectedBy("orders-api", sketch())
	if len(got.Indirect) != 2 {
		t.Fatalf("indirect = %+v, want web-app and mobile-app", got.Indirect)
	}
	for i, name := range []string{"web-app", "mobile-app"} {
		h := got.Indirect[i]
		if h.Repo != name || !reflect.DeepEqual(h.Path, []string{"api-gateway"}) {
			t.Fatalf("indirect[%d] = %+v, want %s via api-gateway", i, h, name)
		}
	}
	far := append(sketch(), edge("crawler", "web-app", "GET /", view.Proven))
	got = view.AffectedBy("orders-api", far)
	last := got.Indirect[len(got.Indirect)-1]
	if last.Repo != "crawler" || !reflect.DeepEqual(last.Path, []string{"web-app", "api-gateway"}) {
		t.Fatalf("three steps away: %+v, want crawler via web-app, api-gateway", last)
	}
}

func TestAffectedCycleAndSelf(t *testing.T) {
	edges := []view.Edge{
		edge("a", "b", "GET /b", view.Proven),
		edge("b", "a", "GET /a", view.Proven),
		edge("a", "a", "GET /self", view.Proven),
	}
	got := view.AffectedBy("a", edges)
	if !reflect.DeepEqual(hits(got.Direct), []string{"b GET /a"}) || len(got.Indirect) != 0 || len(got.Possible) != 0 {
		t.Fatalf("got %+v, want only b as direct", got)
	}
}

func TestAffectedUnverifiedIsOnlyPossible(t *testing.T) {
	edges := []view.Edge{
		edge("c", "orders-api", "GET /orders", view.Unverified),
		edge("d", "c", "GET /c", view.Proven),
	}
	got := view.AffectedBy("orders-api", edges)
	if len(got.Direct) != 0 || len(got.Indirect) != 0 {
		t.Fatalf("got %+v, want nothing direct or indirect", got)
	}
	if len(got.Possible) != 1 || got.Possible[0].Repo != "c" || got.Possible[0].Reason != "unverified" {
		t.Fatalf("possible = %+v, want c (unverified)", got.Possible)
	}
}

func TestAffectedPendingCandidates(t *testing.T) {
	edges := []view.Edge{{Consumer: "x", Kind: "http", ID: "GET /orders", Status: view.Pending, Candidates: []string{"orders-api", "legacy"}}}
	got := view.AffectedBy("orders-api", edges)
	if len(got.Possible) != 1 || got.Possible[0].Repo != "x" || got.Possible[0].Reason != "unknown" || got.Possible[0].ID != "GET /orders" {
		t.Fatalf("possible = %+v, want x (unknown)", got.Possible)
	}
	if other := view.AffectedBy("payments-api", edges); len(other.Possible) != 0 {
		t.Fatalf("a repo that is not a candidate: possible = %+v", other.Possible)
	}
}

func TestAffectedDeclaredIsDirect(t *testing.T) {
	edges := []view.Edge{{Consumer: "web", Provider: "api", ID: "GET /x", Status: view.Declared, Evidence: []view.Link{{Text: "workspace.yaml"}}}}
	got := view.AffectedBy("api", edges)
	if len(got.Direct) != 1 || got.Direct[0].Repo != "web" || got.Direct[0].Evidence[0].Text != "workspace.yaml" {
		t.Fatalf("direct = %+v, want web with workspace.yaml", got.Direct)
	}
}
