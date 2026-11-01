package validate_test

import (
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/validate"
)

func TestValidateCleanDemo(t *testing.T) {
	if fs := demo(t).findings(); len(fs) != 0 {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateMissingCardIsWarning(t *testing.T) {
	f := fixture(t, demoWorkspace)
	f.card("trama-demo-api", demoAPI)
	f.file("trama-demo-api", "cmd/api/main.go", 20)
	f.file("trama-demo-api", "internal/api/api.go", 45)
	got := only(t, f.findings())
	if got.Severity != validate.Warning || got.File != "catalog/trama-demo-web.yaml" || !strings.Contains(got.Message, "not mapped yet") {
		t.Fatalf("finding = %+v", got)
	}
}

func TestValidateOrphanCard(t *testing.T) {
	f := demo(t)
	f.card("old", strings.Replace(demoAPI, "name: trama-demo-api", "name: old", 1))
	fs := f.findings()
	if !hasError(fs, "", "not in workspace.yaml") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateNameMismatch(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, "name: trama-demo-api", "name: trama-demo-web", 1))
	if fs := f.findings(); !hasError(fs, "name", "catalog/trama-demo-api.yaml") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateRequiredFields(t *testing.T) {
	cases := map[string][2]string{
		"description":      {"description: Backend that serves the greeting", "description: \"\""},
		"stack":            {"stack: [go]", "stack: []"},
		"mapped_at_commit": {"mapped_at_commit: 1cb5a15", "mapped_at_commit: main"},
		"mapped_at":        {"mapped_at: 2026-10-28", "mapped_at: 28/10/2026"},
		"run.start":        {"  test: { cmd: \"go test ./...\" }", "  start: { cmd: \"go test ./...\" }"},
		"run.test":         {"  test: { cmd: \"go test ./...\" }", "  test: { cmd: \"\" }"},
		"version":          {"version: 1", "version: 2"},
	}
	for where, edit := range cases {
		f := demo(t)
		f.card("trama-demo-api", strings.Replace(demoAPI, edit[0], edit[1], 1))
		if fs := f.findings(); !hasError(fs, where, "") {
			t.Errorf("%s: findings = %v", where, fs)
		}
	}
}

func TestValidateNoEvidence(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, `evidence: ["internal/api/api.go:42"]`, `evidence: []`, 1))
	if fs := f.findings(); !hasError(fs, "provides[0] GET /hello", "no evidence") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateBadEvidenceFormat(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, `"internal/api/api.go:42"`, `"internal/api/api.go"`, 1))
	if fs := f.findings(); !hasError(fs, "provides[0] GET /hello", "want path:line") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateHTTPID(t *testing.T) {
	for _, id := range []string{"GET /orders/{id}", "POST /", "DELETE /a/b-c_d"} {
		f := demo(t)
		f.card("trama-demo-api", strings.Replace(demoAPI, `id: "GET /hello"`, `id: "`+id+`"`, 1))
		for _, fd := range f.findings() {
			if fd.Severity == validate.Error && strings.HasPrefix(fd.Where, "provides[0]") {
				t.Errorf("valid id %q rejected: %v", id, fd)
			}
		}
	}
	for _, id := range []string{"get /x", "GET orders", "GET /orders/:id", "GET /x?y=1", "FETCH /x", "GET  /x"} {
		f := demo(t)
		f.card("trama-demo-api", strings.Replace(demoAPI, `id: "GET /hello"`, `id: "`+id+`"`, 1))
		if fs := f.findings(); !hasError(fs, "provides[0] "+id, "id must look like") {
			t.Errorf("invalid id %q accepted: %v", id, fs)
		}
	}
}

func TestValidateKind(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, "kind: http", "kind: grpc", 1))
	if fs := f.findings(); !hasError(fs, "provides[0] GET /hello", `unknown kind "grpc"`) {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateEvidenceMissingFile(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, `"internal/api/api.go:42"`, `"src/gone.go:3"`, 1))
	if fs := f.findings(); !hasError(fs, "provides[0] GET /hello", "src/gone.go") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateEvidenceLineOutOfRange(t *testing.T) {
	f := demo(t)
	f.file("trama-demo-api", "f.go", 3)
	f.card("trama-demo-api", strings.Replace(demoAPI, `"internal/api/api.go:42"`, `"f.go:2-4"`, 1))
	fs := f.findings()
	if !hasError(fs, "provides[0] GET /hello", "line 4") || !hasError(fs, "provides[0] GET /hello", "3 lines") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateNotClonedIsWarning(t *testing.T) {
	f := fixture(t, demoWorkspace)
	f.card("trama-demo-api", demoAPI)
	f.card("trama-demo-web", demoWeb)
	fs := f.findings()
	if validate.Errors(fs) != 0 {
		t.Fatalf("errors without clones: %v", fs)
	}
	warnings := 0
	for _, fd := range fs {
		if fd.Severity == validate.Warning && strings.Contains(fd.Message, "is not cloned") {
			warnings++
		}
	}
	if warnings != 2 {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateDeclaredHost(t *testing.T) {
	declared := strings.Replace(demoWorkspace, "    url: https://github.com/lukascivil/trama-demo-api.git\n", "    url: https://github.com/lukascivil/trama-demo-api.git\n    hosts: [api.acme.com]\n", 1)
	host := "identity:\n  hosts: [{ value: api.acme.com, evidence: [workspace.yaml] }]\n  ports:"
	cardWithHost := strings.Replace(demoAPI, "identity:\n  ports:", host, 1)

	f := demo(t)
	f.write("workspace.yaml", declared)
	f.card("trama-demo-api", cardWithHost)
	if fs := f.findings(); validate.Errors(fs) != 0 {
		t.Fatalf("declared host rejected: %v", fs)
	}

	f = demo(t)
	f.card("trama-demo-api", cardWithHost)
	if fs := f.findings(); !hasError(fs, "identity.hosts[0]", "not declared") {
		t.Fatalf("undeclared host accepted: %v", fs)
	}

	f = demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, `"internal/api/api.go:42"`, `"workspace.yaml"`, 1))
	if fs := f.findings(); !hasError(fs, "provides[0] GET /hello", "only valid evidence for identity.hosts") {
		t.Fatalf("workspace.yaml under provides accepted: %v", fs)
	}
}

func TestValidateBrokenCard(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", demoAPI+"evidnce: []\n")
	fs := f.findings()
	if validate.Errors(fs) != 1 || !strings.Contains(fs[0].Message, "evidnce") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestFindingString(t *testing.T) {
	f := validate.Finding{Severity: validate.Error, File: "catalog/a.yaml", Where: "provides[0] GET /x", Message: "no evidence"}
	if got := f.String(); got != "error: catalog/a.yaml: provides[0] GET /x: no evidence" {
		t.Fatalf("String() = %q", got)
	}
	f.Where = ""
	if got := f.String(); got != "error: catalog/a.yaml: no evidence" {
		t.Fatalf("String() = %q", got)
	}
}

func TestCardResponseFieldTypeUnknown(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, "response: { message: string }", "response: { message: str }", 1))
	fs := f.findings()
	if !hasError(fs, "provides[0] GET /hello", `"str"`) || !hasError(fs, "provides[0] GET /hello", "[]object") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestCardResponseFieldPathInvalid(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, "response: { message: string }", `response: { "tags[]": string }`, 1))
	if fs := f.findings(); !hasError(fs, "provides[0] GET /hello", "tags[]") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestCardResponseFieldEmptyType(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, "response: { message: string }", "response: { message: }", 1))
	if fs := f.findings(); !hasError(fs, "provides[0] GET /hello", "message") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestCardRequestOnlyForHTTP(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.NewReplacer(
		"kind: http", "kind: event",
		`id: "GET /hello"`, "id: orders.created",
		"response:", "request:",
	).Replace(demoAPI))
	if fs := f.findings(); !hasError(fs, "provides[0] orders.created", "request is only for kind: http") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestCardResponseNotForPackage(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.NewReplacer(
		"kind: http", "kind: package",
		`id: "GET /hello"`, `id: "@acme/ui"`,
	).Replace(demoAPI))
	if fs := f.findings(); !hasError(fs, "provides[0] @acme/ui", "response is only for kind: http or event") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestCardShapeRenamed(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, "response:", "shape:", 1))
	if fs := f.findings(); !hasError(fs, "provides[0] GET /hello", "shape was renamed to response") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestCardFieldsUsedPathInvalid(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-web", strings.Replace(demoWeb, "fields_used: [message]", `fields_used: ["a..b"]`, 1))
	if fs := f.findings(); !hasError(fs, "consumes[0] GET /hello", "a..b") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestCardBodiesOK(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, "response: { message: string }",
		`request: { user_id: string }
    response: { id: string, "customer.email": string }`, 1))
	if fs := f.findings(); len(fs) != 0 {
		t.Fatalf("findings = %v", fs)
	}
}
