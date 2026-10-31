package cli

import (
	"strings"
	"testing"
)

func TestUsesListsProvidersAndConsumers(t *testing.T) {
	ws := catalogWorkspace(t, cliAPI, strings.Replace(cliWeb, "via: env,", "via: env, fields_used: [message],", 1))
	out, err := run(t, "uses", "GET /hello", "--dir", ws)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, want := range []string{"GET /hello\n", "  provided by api  main.go:2\n", "  consumed by web  via env, fields: message  app.ts:1\n"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
}

func TestUsesShowsPending(t *testing.T) {
	ws := catalogWorkspace(t, cliAPI, strings.Replace(cliWeb, "from: api, via: env,", "from: unknown, candidates: [api],", 1))
	out, err := run(t, "uses", "GET /hello", "--dir", ws)
	if err != nil || !strings.Contains(out, "  pending in  web  candidates: api  app.ts:1\n") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestUsesUnknownID(t *testing.T) {
	ws := catalogWorkspace(t, cliAPI, cliWeb)
	if _, err := run(t, "uses", "GET /nope", "--dir", ws); err == nil || err.Error() != `no catalog card provides or consumes "GET /nope"` {
		t.Fatalf("err = %v", err)
	}
}
