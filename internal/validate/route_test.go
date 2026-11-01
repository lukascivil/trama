package validate_test

import (
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/validate"
)

func TestValidateViaRouteNeedsNoEnv(t *testing.T) {
	f := demo(t)
	editWeb(f, "via: env", "via: route")
	if fs := f.findings(); validate.Errors(fs) != 0 {
		t.Fatalf("via route rejected: %v", fs)
	}
	if n := relations(t, f); n != 1 {
		t.Fatalf("relations = %d, want 1", n)
	}
}

func TestValidateViaRouteMatchesDifferentParameterNames(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, `id: "GET /hello"`, `id: "GET /hello/{name}"`, 1))
	editWeb(f, "via: env", "via: route")
	editWeb2(f, `id: "GET /hello"`, `id: "GET /hello/{user_name}"`)
	if fs := f.findings(); validate.Errors(fs) != 0 {
		t.Fatalf("the same route with another parameter name was rejected: %v", fs)
	}
}

func TestValidateViaRouteRefusesWhenTwoReposProvideIt(t *testing.T) {
	f := demo(t)
	f.write("workspace.yaml", demoWorkspace+`  - name: trama-demo-legacy
    url: https://github.com/lukascivil/trama-demo-legacy.git
    branch: main
`)
	f.card("trama-demo-legacy", strings.Replace(demoAPI, "name: trama-demo-api", "name: trama-demo-legacy", 1))
	f.file("trama-demo-legacy", "cmd/api/main.go", 20)
	f.file("trama-demo-legacy", "internal/api/api.go", 45)
	editWeb(f, "via: env", "via: route")
	fs := f.findings()
	if !hasError(fs, webConsume, "via route") {
		t.Fatalf("an ambiguous route was accepted: %v", fs)
	}
	if n := relations(t, f); n != 0 {
		t.Fatalf("relations = %d, want 0", n)
	}
}

func TestValidateViaRouteNeedsKindHTTP(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, "kind: http", "kind: event", 1))
	editWeb(f, "via: env", "via: route")
	editWeb2(f, "kind: http", "kind: event")
	if fs := f.findings(); !hasError(fs, webConsume, "via route") {
		t.Fatalf("via route accepted for an event: %v", fs)
	}
}

func TestValidateProvidesMatchIgnoresTrailingSlash(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, `id: "GET /hello"`, `id: "GET /hello/"`, 1))
	if fs := f.findings(); validate.Errors(fs) != 0 {
		t.Fatalf("the same route written differently was rejected: %v", fs)
	}
}
