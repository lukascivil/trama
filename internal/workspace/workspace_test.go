package workspace_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/workspace"
)

func writeWorkspace(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, workspace.FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const twoRepos = `version: 1
name: demo
repos:
  - name: api
    url: https://github.com/x/api.git
    branch: main
  - name: web
    url: https://github.com/x/web.git
    branch: develop
    hosts: [web.example.com]
`

func TestFindWalksUp(t *testing.T) {
	root := writeWorkspace(t, twoRepos)
	deep := filepath.Join(root, "repos", "x", "src")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := workspace.Find(deep)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(root)
	if gotReal, _ := filepath.EvalSymlinks(got); gotReal != want {
		t.Fatalf("Find = %q, want %q", got, root)
	}
}

func TestFindNotFound(t *testing.T) {
	if _, err := workspace.Find(t.TempDir()); !errors.Is(err, workspace.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestLoadValid(t *testing.T) {
	root := writeWorkspace(t, twoRepos)
	ws, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != "demo" || ws.Version != 1 || len(ws.Repos) != 2 {
		t.Fatalf("got %+v", ws)
	}
	web := ws.Repos[1]
	if web.Name != "web" || web.Branch != "develop" || len(web.Hosts) != 1 || web.Hosts[0] != "web.example.com" {
		t.Fatalf("web = %+v", web)
	}
	if ws.RepoDir("api") != filepath.Join(ws.Root, "repos", "api") {
		t.Fatalf("RepoDir = %q", ws.RepoDir("api"))
	}
	if r, ok := ws.Repo("web"); !ok || r.URL != "https://github.com/x/web.git" {
		t.Fatalf("Repo(web) = %+v, %v", r, ok)
	}
	if _, ok := ws.Repo("nope"); ok {
		t.Fatal("Repo(nope) found")
	}
}

func TestLoadRejectsUnknownVersion(t *testing.T) {
	root := writeWorkspace(t, "version: 2\nname: demo\nrepos: []\n")
	_, err := workspace.Load(root)
	if err == nil || !strings.Contains(err.Error(), "unsupported workspace version 2") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadRejectsDuplicates(t *testing.T) {
	root := writeWorkspace(t, `version: 1
name: demo
repos:
  - {name: api, url: https://github.com/x/api.git, branch: main}
  - {name: api, url: https://github.com/x/other.git, branch: main}
`)
	if _, err := workspace.Load(root); err == nil {
		t.Fatal("want error for duplicate names")
	}
}

func TestLoadRejectsBadNames(t *testing.T) {
	for _, name := range []string{"../evil", "a/b", "", "Upper", ".hidden"} {
		root := writeWorkspace(t, "version: 1\nname: demo\nrepos:\n  - name: \""+name+"\"\n    url: https://github.com/x/y.git\n    branch: main\n")
		_, err := workspace.Load(root)
		if err == nil || !strings.Contains(err.Error(), "invalid repo name") {
			t.Errorf("name %q: err = %v, want invalid repo name", name, err)
		}
	}
}

func TestAddRepoPreservesCommentsAndUnknownFields(t *testing.T) {
	src := `# my workspace
version: 1
name: demo
custom: keep-me
repos:
  - name: api # the backend
    url: https://github.com/x/api.git
    branch: main
    hosts: [api.example.com]
`
	root := writeWorkspace(t, src)
	ws, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.AddRepo(workspace.Repo{Name: "web", URL: "https://github.com/x/web.git", Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.Save(); err != nil {
		t.Fatal(err)
	}
	raw := readFile(t, filepath.Join(root, workspace.FileName))
	for _, want := range []string{"# my workspace", "custom: keep-me", "# the backend", "api.example.com", "name: web"} {
		if !strings.Contains(raw, want) {
			t.Errorf("saved file lost %q:\n%s", want, raw)
		}
	}
	again, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Repos) != 2 || again.Repos[1].Name != "web" {
		t.Fatalf("reloaded repos = %+v", again.Repos)
	}
}

func TestAddRepoToEmptyFlowList(t *testing.T) {
	root := writeWorkspace(t, "version: 1\nname: demo\nrepos: []\n")
	ws, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.AddRepo(workspace.Repo{Name: "web", URL: "https://github.com/x/web.git", Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.Save(); err != nil {
		t.Fatal(err)
	}
	raw := readFile(t, filepath.Join(root, workspace.FileName))
	if !strings.Contains(raw, "- name: web") {
		t.Fatalf("want block style list:\n%s", raw)
	}
	again, err := workspace.Load(root)
	if err != nil || len(again.Repos) != 1 {
		t.Fatalf("reload = %+v, %v", again, err)
	}
}

func TestAddRepoRejectsDuplicate(t *testing.T) {
	root := writeWorkspace(t, twoRepos)
	ws, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.AddRepo(workspace.Repo{Name: "api", URL: "https://github.com/x/new.git", Branch: "main"}); err == nil {
		t.Error("same name accepted")
	}
	if err := ws.AddRepo(workspace.Repo{Name: "api2", URL: "https://github.com/x/api", Branch: "main"}); err == nil {
		t.Error("same URL without .git accepted")
	}
	if err := ws.AddRepo(workspace.Repo{Name: "../evil", URL: "https://github.com/x/evil.git", Branch: "main"}); err == nil {
		t.Error("invalid name accepted")
	}
	if len(ws.Repos) != 2 {
		t.Fatalf("rejected repos were added: %+v", ws.Repos)
	}
}

func TestNameFromURL(t *testing.T) {
	cases := map[string]string{
		"https://github.com/lukascivil/trama-demo-api.git": "trama-demo-api",
		"git@github.com:org/My.Repo.git":                   "my.repo",
		"/tmp/x/remote.git":                                "remote",
		"https://github.com/org/web/":                      "web",
	}
	for in, want := range cases {
		if got := workspace.NameFromURL(in); got != want {
			t.Errorf("NameFromURL(%q) = %q, want %q", in, got, want)
		}
	}
}
