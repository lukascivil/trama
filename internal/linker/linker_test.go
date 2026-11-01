package linker_test

import (
	"os"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/linker"
	"github.com/lukascivil/trama/internal/validate"
	"github.com/lukascivil/trama/internal/workspace"
)

func TestPlanLinksTheOnlyRepoThatServesTheRoute(t *testing.T) {
	f := demo(t)
	f.unknownWebConsume()
	chs := plan(t, f)
	if len(chs) != 1 {
		t.Fatalf("changes = %v, want 1", chs)
	}
	got := chs[0]
	if got.Repo != "trama-demo-web" || got.From != "trama-demo-api" || got.Via != "route" {
		t.Fatalf("got %+v", got)
	}
}

func TestPlanOnlyListsCandidatesWhenTwoReposServeTheRoute(t *testing.T) {
	f := demo(t)
	f.unknownWebConsume()
	f.secondAPI()
	chs := plan(t, f)
	if len(chs) != 1 {
		t.Fatalf("changes = %v, want 1", chs)
	}
	got := chs[0]
	if got.From != "" {
		t.Fatalf("an ambiguous route was linked to %q", got.From)
	}
	if strings.Join(got.Candidates, ",") != "trama-demo-api,trama-demo-legacy" {
		t.Fatalf("candidates = %v", got.Candidates)
	}
}

func TestPlanLeavesAConsumeNobodyProvides(t *testing.T) {
	f := demo(t)
	f.unknownWebConsume()
	f.editWeb(`id: "GET /hello"`, `id: "GET /nowhere"`)
	if chs := plan(t, f); len(chs) != 0 {
		t.Fatalf("changes = %v, want none", chs)
	}
}

func TestPlanLinksAnEventByItsTopic(t *testing.T) {
	f := demo(t)
	f.unknownWebConsume()
	f.editAPI(`id: "GET /hello"`, `id: "greeting_fct_helloSaid_0"`)
	f.editAPI("kind: http", "kind: event")
	f.editWeb(`id: "GET /hello"`, `id: "greeting_fct_helloSaid_0"`)
	f.editWeb("kind: http", "kind: event")
	chs := plan(t, f)
	if len(chs) != 1 || chs[0].From != "trama-demo-api" || chs[0].Via != "topic" {
		t.Fatalf("got %+v", chs)
	}
}

func TestApplyWritesAValidCardAndKeepsTheRest(t *testing.T) {
	f := demo(t)
	f.unknownWebConsume()
	before := f.readCard("trama-demo-web")
	if err := linker.Apply(f.root, plan(t, f)); err != nil {
		t.Fatal(err)
	}
	after := f.readCard("trama-demo-web")
	if !strings.Contains(after, "from: trama-demo-api") || !strings.Contains(after, "via: route") {
		t.Fatalf("card not linked:\n%s", after)
	}
	if strings.Contains(after, "unknown") {
		t.Fatalf("from: unknown left behind:\n%s", after)
	}
	for _, line := range []string{"fields_used:", "VITE_API_URL", "src/api/hello.ts:10", "npm run dev"} {
		if !strings.Contains(after, line) {
			t.Errorf("%q was lost\nbefore:\n%s\nafter:\n%s", line, before, after)
		}
	}
	ws, err := workspace.Load(f.root)
	if err != nil {
		t.Fatal(err)
	}
	if fs := validate.Workspace(ws); validate.Errors(fs) != 0 {
		t.Fatalf("the linked card does not validate: %v", fs)
	}
}

func TestApplyRecordsCandidatesWithoutLinking(t *testing.T) {
	f := demo(t)
	f.unknownWebConsume()
	f.secondAPI()
	if err := linker.Apply(f.root, plan(t, f)); err != nil {
		t.Fatal(err)
	}
	after := f.readCard("trama-demo-web")
	if !strings.Contains(after, "from: unknown") {
		t.Fatalf("from was changed:\n%s", after)
	}
	if !strings.Contains(after, "trama-demo-legacy") {
		t.Fatalf("candidates were not written:\n%s", after)
	}
	ws, err := workspace.Load(f.root)
	if err != nil {
		t.Fatal(err)
	}
	if fs := validate.Workspace(ws); validate.Errors(fs) != 0 {
		t.Fatalf("the card does not validate: %v", fs)
	}
}

func TestApplyIsIdempotent(t *testing.T) {
	f := demo(t)
	f.unknownWebConsume()
	if err := linker.Apply(f.root, plan(t, f)); err != nil {
		t.Fatal(err)
	}
	once := f.readCard("trama-demo-web")
	chs := plan(t, f)
	if len(chs) != 0 {
		t.Fatalf("a second plan still wants to change %v", chs)
	}
	if err := linker.Apply(f.root, chs); err != nil {
		t.Fatal(err)
	}
	if twice := f.readCard("trama-demo-web"); twice != once {
		t.Fatalf("the card changed on a second run:\n%s\n---\n%s", once, twice)
	}
}

func plan(t *testing.T, f *fx) []linker.Change {
	t.Helper()
	ws, err := workspace.Load(f.root)
	if err != nil {
		t.Fatal(err)
	}
	chs, err := linker.Plan(ws)
	if err != nil {
		t.Fatal(err)
	}
	return chs
}

func (f *fx) readCard(name string) string {
	f.t.Helper()
	data, err := os.ReadFile(catalog.Path(f.root, name))
	if err != nil {
		f.t.Fatal(err)
	}
	return string(data)
}
