package pick

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/github"
	"github.com/lukascivil/trama/internal/workspace"
)

// newWorkspace writes an empty workspace.yaml in a temp directory and loads it.
func newWorkspace(t *testing.T) *workspace.Workspace {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, workspace.FileName), []byte("version: 1\nname: demo\nrepos: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func ghRepo(org, name string) github.Repo {
	return github.Repo{
		FullName:      org + "/" + name,
		Name:          name,
		Description:   "the " + name,
		Language:      "Go",
		DefaultBranch: "main",
		CloneURL:      fmt.Sprintf("https://github.com/%s/%s.git", org, name),
		SSHURL:        fmt.Sprintf("git@github.com:%s/%s.git", org, name),
	}
}

// newPicker serves the given repos per organization.
func newPicker(ws *workspace.Workspace, protocol string, lists map[string][]github.Repo) *Picker {
	return New(Options{
		Workspace:     ws,
		Org:           "acme",
		Protocol:      protocol,
		Authenticated: true,
		Orgs:          func(context.Context) ([]string, error) { return []string{"me", "acme", "beta"}, nil },
		List: func(_ context.Context, org string) ([]github.Repo, error) {
			return lists[org], nil
		},
	})
}

func reload(t *testing.T, ws *workspace.Workspace) *workspace.Workspace {
	t.Helper()
	fresh, err := workspace.Load(ws.Root)
	if err != nil {
		t.Fatal(err)
	}
	return fresh
}

func TestAddWritesOnceAndKeepsOrder(t *testing.T) {
	for _, c := range []struct{ protocol, wantURL string }{
		{"https", "https://github.com/acme/web.git"},
		{"ssh", "git@github.com:acme/web.git"},
	} {
		ws := newWorkspace(t)
		api, web, jobs := ghRepo("acme", "api"), ghRepo("acme", "web"), ghRepo("acme", "jobs")
		web.DefaultBranch = "develop"
		p := newPicker(ws, c.protocol, map[string][]github.Repo{"acme": {api, web, jobs}})

		res, err := p.Apply(context.Background(), "acme", []string{"acme/web", "acme/api", "acme/jobs"})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Added) != 3 || len(res.Refused) != 0 {
			t.Fatalf("%s: result = %+v", c.protocol, res)
		}
		got := reload(t, ws)
		var names []string
		for _, r := range got.Repos {
			names = append(names, r.Name)
		}
		if strings.Join(names, ",") != "web,api,jobs" {
			t.Fatalf("%s: repos in the file = %v, want web,api,jobs", c.protocol, names)
		}
		if r, _ := got.Repo("web"); r.URL != c.wantURL || r.Branch != "develop" {
			t.Fatalf("%s: web = %+v", c.protocol, r)
		}
		if strings.Join(p.Added(), ",") != "web,api,jobs" {
			t.Fatalf("%s: Added() = %v", c.protocol, p.Added())
		}
	}
}

func TestAddRefusesWithReason(t *testing.T) {
	ws := newWorkspace(t)
	if err := ws.AddRepo(workspace.Repo{Name: "api", URL: "https://github.com/acme/api.git", Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.Save(); err != nil {
		t.Fatal(err)
	}
	p := newPicker(ws, "https", map[string][]github.Repo{"acme": {ghRepo("acme", "api"), ghRepo("acme", ".github"), ghRepo("acme", "web")}})

	res, err := p.Apply(context.Background(), "acme", []string{"acme/api", "acme/.github", "acme/web"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 1 || res.Added[0].Name != "web" {
		t.Fatalf("added = %+v", res.Added)
	}
	reasons := map[string]string{}
	for _, r := range res.Refused {
		reasons[r.ID] = r.Reason
	}
	if !strings.Contains(reasons["acme/api"], "already in") {
		t.Errorf("api reason = %q", reasons["acme/api"])
	}
	if !strings.Contains(reasons["acme/.github"], "invalid repo name") {
		t.Errorf(".github reason = %q", reasons["acme/.github"])
	}
	if _, ok := reload(t, ws).Repo("web"); !ok {
		t.Error("web did not reach the file")
	}
}

func TestAddSameNameFromTwoOrgs(t *testing.T) {
	ws := newWorkspace(t)
	p := newPicker(ws, "https", map[string][]github.Repo{
		"acme": {ghRepo("acme", "api")},
		"beta": {ghRepo("beta", "api")},
	})
	first, err := p.Apply(context.Background(), "acme", []string{"acme/api"})
	if err != nil || len(first.Added) != 1 {
		t.Fatalf("first = %+v, %v", first, err)
	}
	second, err := p.Apply(context.Background(), "beta", []string{"beta/api"})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Added) != 0 || len(second.Refused) != 1 || !strings.Contains(second.Refused[0].Reason, "already in") {
		t.Fatalf("second = %+v", second)
	}
}

func TestAddDoubleSubmit(t *testing.T) {
	ws := newWorkspace(t)
	p := newPicker(ws, "https", map[string][]github.Repo{"acme": {ghRepo("acme", "api"), ghRepo("acme", "web")}})
	ids := []string{"acme/api", "acme/web"}
	if _, err := p.Apply(context.Background(), "acme", ids); err != nil {
		t.Fatal(err)
	}
	again, err := p.Apply(context.Background(), "acme", ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Added) != 0 || len(again.Refused) != 2 {
		t.Fatalf("second submit = %+v", again)
	}
	if n := len(reload(t, ws).Repos); n != 2 {
		t.Fatalf("file has %d repos, want 2", n)
	}
	if len(p.Added()) != 2 {
		t.Fatalf("Added() = %v", p.Added())
	}
}

func TestAddIgnoresIdsNotInTheList(t *testing.T) {
	ws := newWorkspace(t)
	p := newPicker(ws, "https", map[string][]github.Repo{"acme": {ghRepo("acme", "api")}})
	res, err := p.Apply(context.Background(), "acme", []string{"evil/payload"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 0 || len(res.Refused) != 1 || !strings.Contains(res.Refused[0].Reason, "not in the list") {
		t.Fatalf("result = %+v", res)
	}
	if n := len(reload(t, ws).Repos); n != 0 {
		t.Fatalf("file has %d repos, want 0", n)
	}
}

func TestSaveFailureAddsNothing(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a read-only directory does not stop root")
	}
	ws := newWorkspace(t)
	p := newPicker(ws, "https", map[string][]github.Repo{"acme": {ghRepo("acme", "api")}})
	if err := os.Chmod(ws.Root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ws.Root, 0o755) })

	if _, err := p.Apply(context.Background(), "acme", []string{"acme/api"}); err == nil {
		t.Fatal("want an error when the workspace cannot be saved")
	}
	if len(p.Added()) != 0 {
		t.Fatalf("Added() = %v, want none", p.Added())
	}
	if err := os.Chmod(ws.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := p.Apply(context.Background(), "acme", []string{"acme/api"})
	if err != nil || len(res.Added) != 1 {
		t.Fatalf("retry = %+v, %v", res, err)
	}
}
