package validate_test

import (
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/validate"
)

const helloSpec = `openapi: 3.0.0
paths:
  /hello:
    get:
      responses:
        "200":
          content:
            application/json:
              schema: { type: object, properties: { message: { type: string } } }
`

// withContract returns the api card with contract set on GET /hello.
func withContract(contract string) string {
	return strings.Replace(demoAPI, `    id: "GET /hello"`, `    id: "GET /hello"`+"\n    contract: "+contract, 1)
}

func TestCardBehindTheSpecIsAWarning(t *testing.T) {
	f := demo(t)
	f.write("repos/trama-demo-api/openapi.yaml", strings.Replace(helloSpec, "message: { type: string }", "message: { type: string }, lang: { type: string }", 1))
	got := only(t, f.findings())
	want := `1 provide differs from the OpenAPI spec that declares it (GET /hello): run "trama bodies trama-demo-api"`
	if got.Severity != validate.Warning || got.Where != "provides" || got.Message != want {
		t.Fatalf("got %s", got)
	}
}

func TestCardWithoutTheContractTheSpecGivesIsAWarning(t *testing.T) {
	f := demo(t)
	f.write("repos/trama-demo-api/openapi.yaml", helloSpec)
	if fs := f.findings(); !hasWarning(fs, "provides", "differs from the OpenAPI spec") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestCardThatMatchesTheSpecIsQuiet(t *testing.T) {
	f := demo(t)
	f.write("repos/trama-demo-api/openapi.yaml", helloSpec)
	f.card("trama-demo-api", withContract("openapi.yaml"))
	if fs := f.findings(); len(fs) != 0 {
		t.Fatalf("findings = %v", fs)
	}
}

func TestRouteNoSpecDeclaresIsQuiet(t *testing.T) {
	f := demo(t)
	f.write("repos/trama-demo-api/openapi.yaml", strings.Replace(helloSpec, "/hello", "/other", 1))
	if fs := f.findings(); len(fs) != 0 {
		t.Fatalf("findings = %v", fs)
	}
}

func TestContractThatDoesNotExistIsAnError(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", withContract("missing.yaml"))
	got := only(t, f.findings())
	if got.Severity != validate.Error || got.Where != "provides[0] GET /hello" || got.Message != "contract missing.yaml: missing.yaml does not exist (in repos/trama-demo-api)" {
		t.Fatalf("got %s", got)
	}
}

func TestContractOutsideTheRepositoryIsAnError(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", withContract(`"../x.yaml"`))
	got := only(t, f.findings())
	if got.Severity != validate.Error || got.Message != `contract "../x.yaml": path must be relative to the repository root, without ..` {
		t.Fatalf("got %s", got)
	}
}

func TestContractInAnotherFormatIsOnlyCheckedToExist(t *testing.T) {
	f := demo(t)
	f.write("repos/trama-demo-api/api.proto", "syntax = \"proto3\";\n")
	f.card("trama-demo-api", withContract("api.proto"))
	if fs := f.findings(); len(fs) != 0 {
		t.Fatalf("findings = %v", fs)
	}
}
