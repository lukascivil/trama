package workspace_test

import (
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/workspace"
)

const withLinks = twoRepos + `links:
  - consumer: web
    provider: api
    id: "GET /hello"
`

func TestLoadLinks(t *testing.T) {
	ws, err := workspace.Load(writeWorkspace(t, withLinks))
	if err != nil {
		t.Fatal(err)
	}
	want := workspace.Link{Consumer: "web", Provider: "api", ID: "GET /hello"}
	if len(ws.Links) != 1 || ws.Links[0] != want {
		t.Fatalf("links = %+v", ws.Links)
	}
}

func TestLoadRejectsBadLinks(t *testing.T) {
	cases := map[string]string{
		"unknown consumer": "  - {consumer: ghost, provider: api, id: GET /x}\n",
		"unknown provider": "  - {consumer: web, provider: ghost, id: GET /x}\n",
		"self link":        "  - {consumer: api, provider: api, id: GET /x}\n",
		"empty id":         "  - {consumer: web, provider: api, id: \"\"}\n",
	}
	for label, item := range cases {
		_, err := workspace.Load(writeWorkspace(t, twoRepos+"links:\n"+item))
		if err == nil || !strings.Contains(err.Error(), "link") {
			t.Errorf("%s: err = %v", label, err)
		}
	}
}

func TestAddRepoKeepsLinks(t *testing.T) {
	root := writeWorkspace(t, withLinks)
	ws, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.AddRepo(workspace.Repo{Name: "new", URL: "https://github.com/x/new.git", Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	if err := ws.Save(); err != nil {
		t.Fatal(err)
	}
	again, err := workspace.Load(root)
	if err != nil || len(again.Links) != 1 || len(again.Repos) != 3 {
		t.Fatalf("reloaded = %+v, %v", again, err)
	}
}
