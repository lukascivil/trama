package validate_test

import (
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/validate"
	"github.com/lukascivil/trama/internal/workspace"
)

const webConsume = "consumes[0] GET /hello"

func editWeb(f *fx, old, new string) {
	f.t.Helper()
	if !strings.Contains(demoWeb, old) {
		f.t.Fatalf("demoWeb lacks %q", old)
	}
	f.card("trama-demo-web", strings.Replace(demoWeb, old, new, 1))
}

func relations(t *testing.T, f *fx) int {
	t.Helper()
	ws, err := workspace.Load(f.root)
	if err != nil {
		t.Fatal(err)
	}
	return validate.Relations(ws)
}

func TestValidateDemoRelation(t *testing.T) {
	f := demo(t)
	if fs := f.findings(); len(fs) != 0 {
		t.Fatalf("findings = %v", fs)
	}
	if n := relations(t, f); n != 1 {
		t.Fatalf("relations = %d, want 1", n)
	}
}

func TestValidateEnvMatchesPathAndHost(t *testing.T) {
	f := demo(t)
	editWeb(f, `value: "http://localhost:8080"`, `value: "http://localhost:8080/api"`)
	if fs := f.findings(); validate.Errors(fs) != 0 {
		t.Errorf("path in value rejected: %v", fs)
	}

	f = demo(t)
	f.write("workspace.yaml", strings.Replace(demoWorkspace, "trama-demo-api.git\n", "trama-demo-api.git\n    hosts: [api.acme.com]\n", 1))
	f.card("trama-demo-api", strings.Replace(demoAPI, "identity:\n", "identity:\n  hosts: [{ value: api.acme.com, evidence: [workspace.yaml] }]\n", 1))
	editWeb(f, `value: "http://localhost:8080"`, `value: "https://API.acme.com"`)
	if fs := f.findings(); validate.Errors(fs) != 0 {
		t.Errorf("declared host rejected: %v", fs)
	}

	f = demo(t)
	editWeb(f, `value: "http://localhost:8080"`, `value: "http://localhost:9090"`)
	if fs := f.findings(); !hasError(fs, webConsume, "via env") {
		t.Errorf("wrong port accepted: %v", fs)
	}
	if n := relations(t, f); n != 0 {
		t.Errorf("relations = %d, want 0", n)
	}
}

func TestValidateFromWithoutViaIsError(t *testing.T) {
	f := demo(t)
	editWeb(f, "    via: env\n", "")
	if fs := f.findings(); !hasError(fs, webConsume, "from needs proof in via") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateUnknownWithCandidatesIsWarning(t *testing.T) {
	f := demo(t)
	editWeb(f, "    from: trama-demo-api\n    via: env\n", "    from: unknown\n    candidates: [trama-demo-api]\n")
	got := only(t, f.findings())
	if got.Severity != validate.Warning || got.Where != webConsume || !strings.Contains(got.Message, "candidates: trama-demo-api") {
		t.Fatalf("finding = %+v", got)
	}

	f = demo(t)
	editWeb(f, "    from: trama-demo-api\n    via: env\n", "    from: unknown\n    candidates: [ghost]\n")
	if fs := f.findings(); !hasError(fs, webConsume, `"ghost"`) {
		t.Fatalf("unknown candidate accepted: %v", fs)
	}

	f = demo(t)
	editWeb(f, "    from: trama-demo-api\n", "    from: unknown\n")
	if fs := f.findings(); !hasError(fs, webConsume, "via must be empty") {
		t.Fatalf("via with unknown accepted: %v", fs)
	}
}

func TestValidateProviderMustProvide(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, `id: "GET /hello"`, `id: "GET /bye"`, 1))
	if fs := f.findings(); !hasError(fs, webConsume, "trama-demo-api does not provide GET /hello") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateProviderNotMapped(t *testing.T) {
	f := fixture(t, demoWorkspace)
	f.card("trama-demo-web", demoWeb)
	f.file("trama-demo-web", "src/api/hello.ts", 16)
	f.file("trama-demo-web", ".env.example", 1)
	if fs := f.findings(); !hasError(fs, webConsume, "trama-demo-api is not mapped") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateBrokenProviderAddsNoError(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", demoAPI+"evidnce: []\n")
	if fs := f.findings(); validate.Errors(fs) != 1 {
		t.Fatalf("want only the parse error, got %v", fs)
	}
}

const pkgAPI = `version: 1
name: trama-demo-api
description: Shared library and events
stack: [node]
mapped_at_commit: 1cb5a15
mapped_at: 2026-10-28
identity:
  package: { value: "@acme/ui", evidence: ["package.json:2"] }
  topics: [{ value: orders.created, evidence: ["src/events.ts:3"] }]
provides:
  - kind: package
    id: "@acme/ui"
    evidence: ["package.json:2"]
  - kind: event
    id: orders.created
    evidence: ["src/events.ts:3"]
`

const pkgWeb = `version: 1
name: trama-demo-web
description: Uses the library and listens to orders
stack: [node]
mapped_at_commit: 697bd1b
mapped_at: 2026-10-28
consumes:
  - kind: package
    id: "@acme/ui"
    from: trama-demo-api
    via: package
    evidence: ["package.json:5"]
  - kind: event
    id: orders.created
    from: trama-demo-api
    via: topic
    evidence: ["src/listen.ts:1"]
`

func pkgFixture(t *testing.T, api, web string) *fx {
	t.Helper()
	f := fixture(t, demoWorkspace)
	f.card("trama-demo-api", api)
	f.card("trama-demo-web", web)
	f.file("trama-demo-api", "package.json", 5)
	f.file("trama-demo-api", "src/events.ts", 5)
	f.file("trama-demo-web", "package.json", 9)
	f.file("trama-demo-web", "src/listen.ts", 2)
	return f
}

func TestValidateViaPackageAndTopic(t *testing.T) {
	f := pkgFixture(t, pkgAPI, pkgWeb)
	if fs := f.findings(); len(fs) != 0 {
		t.Fatalf("findings = %v", fs)
	}
	if n := relations(t, f); n != 2 {
		t.Fatalf("relations = %d, want 2", n)
	}

	f = pkgFixture(t, strings.Replace(pkgAPI, `package: { value: "@acme/ui"`, `package: { value: "@acme/other"`, 1), pkgWeb)
	if fs := f.findings(); !hasError(fs, "consumes[0] @acme/ui", "via package") {
		t.Errorf("package mismatch accepted: %v", fs)
	}

	f = pkgFixture(t, strings.Replace(pkgAPI, "topics: [{ value: orders.created", "topics: [{ value: orders.updated", 1), pkgWeb)
	if fs := f.findings(); !hasError(fs, "consumes[1] orders.created", "via topic") {
		t.Errorf("topic mismatch accepted: %v", fs)
	}
}

func TestValidateViaDeclared(t *testing.T) {
	f := demo(t)
	editWeb(f, "    via: env\n", "    via: declared\n")
	if fs := f.findings(); !hasError(fs, webConsume, "via declared") {
		t.Fatalf("declared without links accepted: %v", fs)
	}
	f.write("workspace.yaml", demoWorkspace+"links:\n  - {consumer: trama-demo-web, provider: trama-demo-api, id: GET /hello}\n")
	if fs := f.findings(); validate.Errors(fs) != 0 {
		t.Fatalf("declared link rejected: %v", fs)
	}
}

func TestValidateSelfReference(t *testing.T) {
	f := demo(t)
	editWeb(f, "    from: trama-demo-api\n", "    from: trama-demo-web\n")
	if fs := f.findings(); !hasError(fs, webConsume, "itself") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateConfigPointsToUnknownRepo(t *testing.T) {
	f := demo(t)
	editWeb(f, "    points_to: trama-demo-api\n", "    points_to: ghost\n")
	if fs := f.findings(); !hasError(fs, "config[0] VITE_API_URL", `"ghost"`) {
		t.Fatalf("findings = %v", fs)
	}
}
