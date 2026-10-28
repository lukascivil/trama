package status_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lukascivil/trama/internal/catalog"
)

func TestStatusListsPending(t *testing.T) {
	f := newFixture(t, true)
	card := "version: 1\nname: api\nmapped_at_commit: " + f.originMain(t) + "\nconsumes:\n" +
		"  - {kind: http, id: GET /health, from: unknown, candidates: [a, b]}\n" +
		"  - {kind: event, id: orders.created, from: unknown}\n"
	if err := os.MkdirAll(filepath.Dir(catalog.Path(f.ws.Root, "api")), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(catalog.Path(f.ws.Root, "api"), []byte(card), 0o644); err != nil {
		t.Fatal(err)
	}
	s := only(t, f.ws)
	want := []string{
		"pending: consumes GET /health (candidates: a, b) from an unknown provider",
		"pending: consumes orders.created (no candidates) from an unknown provider",
	}
	if got := s.Problems(); !reflect.DeepEqual(got, want) {
		t.Fatalf("problems = %q", got)
	}
}
