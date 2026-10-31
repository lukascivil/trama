package validate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/testutil"
	"github.com/lukascivil/trama/internal/validate"
)

// gitFixture is the demo with trama-demo-api as a real clone whose
// origin/main has cmd/api/main.go (20 lines) and internal/api/api.go (45).
func gitFixture(t *testing.T) (*fx, string) {
	t.Helper()
	remote := testutil.NewRemote(t, "main")
	var lines20, lines45 strings.Builder
	for i := 1; i <= 45; i++ {
		if i <= 20 {
			lines20.WriteString("x\n")
		}
		lines45.WriteString("x\n")
	}
	remote.Commit(t, "cmd/api/main.go", lines20.String())
	remote.Commit(t, "internal/api/api.go", lines45.String())
	f := fixture(t, strings.Replace(demoWorkspace, "https://github.com/lukascivil/trama-demo-api.git", remote.URL, 1))
	f.card("trama-demo-api", demoAPI)
	f.card("trama-demo-web", demoWeb)
	f.file("trama-demo-web", "src/api/hello.ts", 16)
	f.file("trama-demo-web", ".env.example", 1)
	clone := filepath.Join(f.root, "repos", "trama-demo-api")
	if err := gitx.Clone(remote.URL, "main", clone); err != nil {
		t.Fatal(err)
	}
	return f, clone
}

func TestValidateReadsEvidenceFromOriginBranch(t *testing.T) {
	f, clone := gitFixture(t)
	testutil.Git(t, clone, "checkout", "-q", "-b", "feature")
	testutil.Git(t, clone, "rm", "-q", "internal/api/api.go")
	testutil.Git(t, clone, "commit", "-q", "-m", "drop")
	if fs := f.findings(); validate.Errors(fs) != 0 {
		t.Fatalf("a local branch changed the result: %v", fs)
	}
}

func TestValidateRejectsEvidenceOnlyInWorkingTree(t *testing.T) {
	f, clone := gitFixture(t)
	if err := os.WriteFile(filepath.Join(clone, "local.env"), []byte("A=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.card("trama-demo-api", strings.Replace(demoAPI, `"internal/api/api.go:42"`, `"local.env:1"`, 1))
	if fs := f.findings(); !hasError(fs, "provides[0] GET /hello", "local.env") {
		t.Fatalf("evidence only in the working tree accepted: %v", fs)
	}
}

func TestValidateEvidenceSymlinkRejected(t *testing.T) {
	f := demo(t)
	link := filepath.Join(f.root, "repos", "trama-demo-api", "hosts")
	if err := os.Symlink("/etc/hosts", link); err != nil {
		t.Fatal(err)
	}
	f.card("trama-demo-api", strings.Replace(demoAPI, `"internal/api/api.go:42"`, `"hosts:1"`, 1))
	if fs := f.findings(); !hasError(fs, "provides[0] GET /hello", "not a regular file") {
		t.Fatalf("symlink evidence accepted: %v", fs)
	}
}

const billingWorkspace = demoWorkspace + `  - name: billing
    url: https://github.com/lukascivil/billing.git
    branch: main
`

const billingCard = `version: 1
name: billing
description: Billing service
stack: [go]
mapped_at_commit: 1cb5a15
mapped_at: 2026-10-28
identity:
  ports: [{ value: 8080, evidence: ["main.go:1"] }]
provides:
  - { kind: http, id: "GET /hello", evidence: ["main.go:2"] }
`

func TestValidateEnvAmbiguousProvider(t *testing.T) {
	f := demo(t)
	f.write("workspace.yaml", billingWorkspace)
	f.card("billing", billingCard)
	f.file("billing", "main.go", 2)
	fs := f.findings()
	if !hasError(fs, webConsume, "trama-demo-api") || !hasError(fs, webConsume, "billing") {
		t.Fatalf("ambiguous env proof accepted: %v", fs)
	}
}

func TestValidateUnknownMentionsMappedProvider(t *testing.T) {
	f := demo(t)
	editWeb(f, "    from: trama-demo-api\n    via: env\n", "    from: unknown\n")
	for _, fd := range f.findings() {
		if fd.Severity == validate.Warning && fd.Where == webConsume && strings.Contains(fd.Message, "trama-demo-api provides GET /hello") {
			return
		}
	}
	t.Fatalf("no warning that a mapped repository provides the pending id: %v", f.findings())
}

func TestValidateHostFromCode(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, "identity:\n", "identity:\n  hosts: [{ value: api.acme.com, evidence: [\"cmd/api/main.go:3\"] }]\n", 1))
	if fs := f.findings(); validate.Errors(fs) != 0 {
		t.Fatalf("host proven by code rejected: %v", fs)
	}
}
