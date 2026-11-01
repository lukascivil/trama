package catalog_test

import (
	"testing"

	"github.com/lukascivil/trama/internal/catalog"
)

func TestNormalizeIDMatchesTheSameRouteWrittenDifferently(t *testing.T) {
	same := [][2]string{
		{"GET /resources/v2/stone-accounts/{stone_account_id}", "GET /resources/v2/stone-accounts/{id}"},
		{"GET /resources/v2/stone-accounts/{stone_account_id}", "GET /resources/v2/stone-accounts/:stone_account_id"},
		{"get /resources/v1/questionnaires", "GET /resources/v1/questionnaires"},
		{"GET /resources/v1/questionnaires/", "GET /resources/v1/questionnaires"},
		{"GET  /resources/v1/questionnaires", "GET /resources/v1/questionnaires"},
		{"PATCH /a/{x}/b/{y}", "PATCH /a/:p/b/:q"},
	}
	for _, p := range same {
		if a, b := catalog.NormalizeID("http", p[0]), catalog.NormalizeID("http", p[1]); a != b {
			t.Errorf("%q and %q should match: got %q and %q", p[0], p[1], a, b)
		}
	}
}

func TestNormalizeIDKeepsDifferentRoutesApart(t *testing.T) {
	differ := [][2]string{
		{"GET /resources/v2/stone-accounts/{id}", "POST /resources/v2/stone-accounts/{id}"},
		{"GET /resources/v1/questionnaires", "GET /resources/v2/questionnaires"},
		{"GET /a/{x}", "GET /a/{x}/b"},
		{"GET /a/b", "GET /a/{b}"},
	}
	for _, p := range differ {
		if a, b := catalog.NormalizeID("http", p[0]), catalog.NormalizeID("http", p[1]); a == b {
			t.Errorf("%q and %q should differ: both normalized to %q", p[0], p[1], a)
		}
	}
}

func TestNormalizeIDLeavesOtherKindsAlone(t *testing.T) {
	for _, kind := range []string{"event", "package"} {
		const topic = "quad_cmd_createQuestionnaire_0"
		if got := catalog.NormalizeID(kind, " "+topic+" "); got != topic {
			t.Errorf("kind %s: got %q, want %q", kind, got, topic)
		}
		if catalog.NormalizeID(kind, "a:b") == catalog.NormalizeID(kind, "a:c") {
			t.Errorf("kind %s: colons must not be treated as route parameters", kind)
		}
	}
}
