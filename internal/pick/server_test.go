package pick

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lukascivil/trama/internal/github"
	"github.com/lukascivil/trama/internal/workspace"
)

const port = 7300

type reposResponse struct {
	Token         string   `json:"token"`
	Org           string   `json:"org"`
	Orgs          []string `json:"orgs"`
	Authenticated bool     `json:"authenticated"`
	Error         string   `json:"error"`
	Repos         []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Blocked     string `json:"blocked"`
	} `json:"repos"`
}

func do(t *testing.T, h http.Handler, method, target, host, body string, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Host = host
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

const local = "127.0.0.1:7300"

func getRepos(t *testing.T, h http.Handler, target string) reposResponse {
	t.Helper()
	rec := do(t, h, http.MethodGet, target, local, "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d: %s", target, rec.Code, rec.Body)
	}
	var out reposResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func post(t *testing.T, h http.Handler, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	return do(t, h, http.MethodPost, path, local, body, map[string]string{"Content-Type": "application/json", "X-Trama-Token": token})
}

func fileBytes(t *testing.T, ws *workspace.Workspace) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(ws.Root, workspace.FileName))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestReposNeedsOrgOrOrgs(t *testing.T) {
	p := newPicker(newWorkspace(t), "https", map[string][]github.Repo{"acme": {ghRepo("acme", "api")}})
	p.o.Org = ""
	h := p.Handler(port)

	out := getRepos(t, h, "/api/repos")
	if strings.Join(out.Orgs, ",") != "me,acme,beta" || len(out.Repos) != 0 || out.Token == "" || !out.Authenticated {
		t.Fatalf("response = %+v", out)
	}
	if got := getRepos(t, h, "/api/repos?org=acme"); got.Org != "acme" || len(got.Repos) != 1 || got.Repos[0].ID != "acme/api" {
		t.Fatalf("response for acme = %+v", got)
	}
}

func TestReposUsesTheOrgOfTheOptions(t *testing.T) {
	p := newPicker(newWorkspace(t), "https", map[string][]github.Repo{"acme": {ghRepo("acme", "api")}})
	if out := getRepos(t, p.Handler(port), "/api/repos"); out.Org != "acme" || len(out.Repos) != 1 {
		t.Fatalf("response = %+v", out)
	}
}

func TestBlockedMarksWorkspaceRepos(t *testing.T) {
	ws := newWorkspace(t)
	if err := ws.AddRepo(workspace.Repo{Name: "api", URL: "https://github.com/acme/api.git", Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	p := newPicker(ws, "https", map[string][]github.Repo{"acme": {ghRepo("acme", "api"), ghRepo("acme", "web"), ghRepo("acme", ".github")}})
	blocked := map[string]string{}
	for _, r := range getRepos(t, p.Handler(port), "/api/repos").Repos {
		blocked[r.ID] = r.Blocked
	}
	if !strings.Contains(blocked["acme/api"], "already in") || blocked["acme/web"] != "" || !strings.Contains(blocked["acme/.github"], "invalid repo name") {
		t.Fatalf("blocked = %v", blocked)
	}
}

func TestPostIsRefusedWithoutTheRightToken(t *testing.T) {
	ws := newWorkspace(t)
	before := fileBytes(t, ws)
	p := newPicker(ws, "https", map[string][]github.Repo{"acme": {ghRepo("acme", "api")}})
	h := p.Handler(port)
	token := getRepos(t, h, "/api/repos").Token
	body := `{"org":"acme","ids":["acme/api"]}`

	cases := map[string]*httptest.ResponseRecorder{
		"no token":        do(t, h, http.MethodPost, "/api/add", local, body, map[string]string{"Content-Type": "application/json"}),
		"wrong token":     post(t, h, "/api/add", "nope", body),
		"form type":       do(t, h, http.MethodPost, "/api/add", local, body, map[string]string{"Content-Type": "text/plain", "X-Trama-Token": token}),
		"foreign host":    do(t, h, http.MethodPost, "/api/add", "evil.test", body, map[string]string{"Content-Type": "application/json", "X-Trama-Token": token}),
		"cross-site":      do(t, h, http.MethodPost, "/api/add", local, body, map[string]string{"Content-Type": "application/json", "X-Trama-Token": token, "Sec-Fetch-Site": "cross-site"}),
		"cancel no token": do(t, h, http.MethodPost, "/api/cancel", local, "", map[string]string{"Content-Type": "application/json"}),
	}
	for name, rec := range cases {
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d, want 403", name, rec.Code)
		}
	}
	if fileBytes(t, ws) != before {
		t.Fatalf("workspace.yaml changed:\n%s", fileBytes(t, ws))
	}
	select {
	case <-p.done:
		t.Fatal("a refused request ended the picker")
	default:
	}
}

func TestAddOverHTTP(t *testing.T) {
	ws := newWorkspace(t)
	p := newPicker(ws, "https", map[string][]github.Repo{"acme": {ghRepo("acme", "api"), ghRepo("acme", ".github")}})
	h := p.Handler(port)
	token := getRepos(t, h, "/api/repos").Token

	rec := post(t, h, "/api/add", token, `{"org":"acme","ids":["acme/api","acme/.github"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	var res struct {
		Added   []struct{ Name, URL, Branch string } `json:"added"`
		Refused []struct{ ID, Reason string }        `json:"refused"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 1 || res.Added[0].Name != "api" || res.Added[0].Branch != "main" || len(res.Refused) != 1 || res.Refused[0].ID != "acme/.github" {
		t.Fatalf("result = %+v", res)
	}
}

func TestListErrorReachesThePage(t *testing.T) {
	p := New(Options{
		Workspace:     newWorkspace(t),
		Org:           "acme",
		Authenticated: true,
		List: func(context.Context, string) ([]github.Repo, error) {
			return nil, &github.APIError{Status: 403, Message: "Resource protected by organization SAML enforcement"}
		},
	})
	rec := do(t, p.Handler(port), http.MethodGet, "/api/repos", local, "", nil)
	var out reposResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Error, "SAML enforcement") || len(out.Repos) != 0 {
		t.Fatalf("response = %+v", out)
	}
}

func TestDescriptionIsJSONText(t *testing.T) {
	r := ghRepo("acme", "api")
	r.Description = `<script>alert(1)</script>`
	p := newPicker(newWorkspace(t), "https", map[string][]github.Repo{"acme": {r}})
	rec := do(t, p.Handler(port), http.MethodGet, "/api/repos", local, "", nil)
	if strings.Contains(rec.Body.String(), "<script>") {
		t.Fatalf("the body carries a raw tag: %s", rec.Body)
	}
	var out reposResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.Repos[0].Description != r.Description {
		t.Fatalf("description = %q, err = %v", out.Repos[0].Description, err)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q", ct)
	}
}

func waitResult(p *Picker, idle time.Duration) <-chan error {
	out := make(chan error, 1)
	go func() { out <- p.Wait(context.Background(), idle) }()
	return out
}

func TestWaitEndsOnAddAndOnCancel(t *testing.T) {
	for _, path := range []string{"/api/add", "/api/cancel"} {
		p := newPicker(newWorkspace(t), "https", map[string][]github.Repo{"acme": {ghRepo("acme", "api")}})
		h := p.Handler(port)
		token := getRepos(t, h, "/api/repos").Token
		res := waitResult(p, time.Minute)

		body := ""
		if path == "/api/add" {
			body = `{"org":"acme","ids":["acme/api"]}`
		}
		if rec := post(t, h, path, token, body); rec.Code >= 300 {
			t.Fatalf("%s = %d: %s", path, rec.Code, rec.Body)
		}
		select {
		case err := <-res:
			if err != nil {
				t.Fatalf("%s: Wait = %v", path, err)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%s: Wait did not return", path)
		}
	}
}

func TestWaitStaysOpenWhenEverythingRefused(t *testing.T) {
	p := newPicker(newWorkspace(t), "https", map[string][]github.Repo{"acme": {ghRepo("acme", ".github")}})
	h := p.Handler(port)
	token := getRepos(t, h, "/api/repos").Token
	res := waitResult(p, time.Minute)

	post(t, h, "/api/add", token, `{"org":"acme","ids":["acme/.github"]}`)
	select {
	case err := <-res:
		t.Fatalf("Wait returned %v although nothing was added", err)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestWaitIdle(t *testing.T) {
	p := newPicker(newWorkspace(t), "https", nil)
	select {
	case err := <-waitResult(p, 50*time.Millisecond):
		if !errors.Is(err, ErrIdle) {
			t.Fatalf("Wait = %v, want ErrIdle", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return")
	}
}

func TestRequestsKeepThePickerAlive(t *testing.T) {
	p := newPicker(newWorkspace(t), "https", map[string][]github.Repo{"acme": {ghRepo("acme", "api")}})
	h := p.Handler(port)
	res := waitResult(p, 200*time.Millisecond)
	for i := 0; i < 4; i++ {
		time.Sleep(80 * time.Millisecond)
		getRepos(t, h, "/api/repos")
	}
	select {
	case err := <-res:
		t.Fatalf("Wait returned %v although requests kept coming", err)
	default:
	}
}

func TestWaitStopsWithTheContext(t *testing.T) {
	p := newPicker(newWorkspace(t), "https", nil)
	ctx, cancel := context.WithCancel(context.Background())
	out := make(chan error, 1)
	go func() { out <- p.Wait(ctx, time.Minute) }()
	cancel()
	select {
	case err := <-out:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Wait = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return")
	}
}

func TestPageIsServed(t *testing.T) {
	h := newPicker(newWorkspace(t), "https", nil).Handler(port)
	if rec := do(t, h, http.MethodGet, "/", local, "", nil); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<html") {
		t.Fatalf("GET / = %d: %s", rec.Code, rec.Body)
	}
}
