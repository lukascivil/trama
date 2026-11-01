package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lukascivil/trama/internal/pick"
	"github.com/lukascivil/trama/internal/runner"
	"github.com/lukascivil/trama/internal/workspace"
)

const pickToken = "SECRET-TOKEN"

// fakeGitHub serves the three repositories of the organization "acme" and
// records the Authorization header of every request.
func fakeGitHub(t *testing.T) *[]string {
	t.Helper()
	var mu sync.Mutex
	var auth []string
	repo := func(name string) map[string]any {
		return map[string]any{
			"full_name": "acme/" + name, "name": name, "description": "the " + name, "language": "Go",
			"default_branch": "main", "clone_url": "https://github.com/acme/" + name + ".git",
			"ssh_url": "git@github.com:acme/" + name + ".git", "archived": false, "pushed_at": "2026-10-01T10:00:00Z",
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		auth = append(auth, r.Header.Get("Authorization"))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/orgs/acme/repos":
			_ = json.NewEncoder(w).Encode([]map[string]any{repo("api"), repo("web"), repo(".github")})
		case "/user":
			fmt.Fprint(w, `{"login":"me"}`)
		case "/user/orgs":
			fmt.Fprint(w, `[{"login":"acme"}]`)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"Not Found"}`)
		}
	}))
	t.Cleanup(srv.Close)

	oldAPI, oldGh, oldIdle := githubAPI, ghRun, pickIdle
	githubAPI = srv.URL
	ghRun = func(args ...string) (string, error) {
		if strings.Join(args, " ") == "auth token" {
			return pickToken, nil
		}
		return "", errors.New("gh: no answer")
	}
	pickIdle = time.Minute
	t.Cleanup(func() { githubAPI, ghRun, pickIdle = oldAPI, oldGh, oldIdle })
	return &auth
}

var pickServingLine = regexp.MustCompile(`serving (http://127\.0\.0\.1:\d+/)`)

// servePick runs "trama add --pick ..." until it ends, with stdout and stderr
// apart, and returns the URL it printed.
func servePick(t *testing.T, args ...string) (url string, stdout, stderr *syncBuffer, cancel context.CancelFunc, done <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	stdout, stderr = &syncBuffer{}, &syncBuffer{}
	cmd := newRootCmd()
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)
	errc := make(chan error, 1)
	go func() { errc <- cmd.ExecuteContext(ctx) }()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if m := pickServingLine.FindStringSubmatch(stderr.String()); m != nil {
			return m[1], stdout, stderr, cancel, errc
		}
		select {
		case err := <-errc:
			t.Fatalf("trama stopped before serving: %v\n%s", err, stderr)
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatalf("no serving line in:\n%s", stderr)
	return
}

func pageToken(t *testing.T, url string) string {
	t.Helper()
	res, err := http.Get(url + "api/repos?org=acme")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out struct {
		Token         string `json:"token"`
		Authenticated bool   `json:"authenticated"`
		Repos         []any  `json:"repos"`
	}
	if res.StatusCode != 200 || json.NewDecoder(res.Body).Decode(&out) != nil || out.Token == "" || len(out.Repos) != 3 {
		t.Fatalf("GET api/repos = %d, %+v", res.StatusCode, out)
	}
	return out.Token
}

func postPage(t *testing.T, url, path, token, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trama-Token", token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	return res
}

func ended(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("trama did not end")
		return nil
	}
}

func workspaceFile(t *testing.T, ws string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(ws, workspace.FileName))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestPickAddsChosenRepos(t *testing.T) {
	fakeGitHub(t)
	noBrowser(t)
	ws := newWorkspace(t)
	url, stdout, stderr, _, done := servePick(t, "add", "--pick", "acme", "--no-open", "--port", "0", "--dir", ws)

	token := pageToken(t, url)
	if res := postPage(t, url, "api/add", token, `{"org":"acme","ids":["acme/api","acme/.github","acme/web"]}`); res.StatusCode != 200 {
		t.Fatalf("POST api/add = %d", res.StatusCode)
	}
	if err := ended(t, done); err != nil {
		t.Fatalf("trama: %v\n%s", err, stderr)
	}
	if stdout.String() != "api\nweb\n" {
		t.Fatalf("stdout = %q, want the two names", stdout)
	}
	if !strings.Contains(stderr.String(), "refused acme/.github:") {
		t.Fatalf("stderr lacks the refusal:\n%s", stderr)
	}
	w, err := workspace.Load(ws)
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := w.Repo("web"); !ok || r.URL != "https://github.com/acme/web.git" || r.Branch != "main" {
		t.Fatalf("web = %+v, %v", r, ok)
	}
	if _, ok := w.Repo("api"); !ok {
		t.Fatal("api is not in the workspace")
	}
}

func TestPickCancelLeavesWorkspaceAlone(t *testing.T) {
	fakeGitHub(t)
	noBrowser(t)
	ws := newWorkspace(t)
	before := workspaceFile(t, ws)
	url, stdout, _, _, done := servePick(t, "add", "--pick", "acme", "--no-open", "--port", "0", "--dir", ws)

	postPage(t, url, "api/cancel", pageToken(t, url), "")
	if err := ended(t, done); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "" || workspaceFile(t, ws) != before {
		t.Fatalf("stdout = %q, workspace changed = %v", stdout, workspaceFile(t, ws) != before)
	}
}

func TestPickInterruptExits130(t *testing.T) {
	fakeGitHub(t)
	noBrowser(t)
	ws := newWorkspace(t)
	before := workspaceFile(t, ws)
	_, _, _, cancel, done := servePick(t, "add", "--pick", "acme", "--no-open", "--port", "0", "--dir", ws)

	cancel()
	if err := ended(t, done); !errors.Is(err, runner.ErrInterrupted) {
		t.Fatalf("err = %v, want runner.ErrInterrupted", err)
	}
	if code, quiet := exitCode(runner.ErrInterrupted); code != 130 || !quiet {
		t.Fatalf("exitCode = %d, %v", code, quiet)
	}
	if workspaceFile(t, ws) != before {
		t.Fatal("workspace.yaml changed")
	}
}

func TestPickIdleStops(t *testing.T) {
	fakeGitHub(t)
	noBrowser(t)
	pickIdle = 100 * time.Millisecond
	ws := newWorkspace(t)
	before := workspaceFile(t, ws)
	_, _, _, _, done := servePick(t, "add", "--pick", "acme", "--no-open", "--port", "0", "--dir", ws)

	err := ended(t, done)
	if !errors.Is(err, pick.ErrIdle) {
		t.Fatalf("err = %v, want pick.ErrIdle", err)
	}
	if workspaceFile(t, ws) != before {
		t.Fatal("workspace.yaml changed")
	}
}

func TestPickPrintsServingLineToStderrOnly(t *testing.T) {
	fakeGitHub(t)
	noBrowser(t)
	ws := newWorkspace(t)
	url, stdout, _, cancel, done := servePick(t, "add", "--pick", "acme", "--no-open", "--port", "0", "--dir", ws)
	if strings.Contains(stdout.String(), url) || stdout.String() != "" {
		t.Fatalf("stdout = %q", stdout)
	}
	cancel()
	ended(t, done)
}

func TestPickNeverPrintsTheToken(t *testing.T) {
	auth := fakeGitHub(t)
	noBrowser(t)
	ws := newWorkspace(t)
	url, stdout, stderr, _, done := servePick(t, "add", "--pick", "acme", "--no-open", "--port", "0", "--dir", ws)

	token := pageToken(t, url)
	postPage(t, url, "api/add", token, `{"org":"acme","ids":["acme/api"]}`)
	ended(t, done)

	if strings.Contains(stdout.String()+stderr.String(), pickToken) {
		t.Fatalf("the token is in the output:\n%s\n%s", stdout, stderr)
	}
	if strings.Contains(workspaceFile(t, ws), pickToken) {
		t.Fatal("the token is in workspace.yaml")
	}
	if len(*auth) == 0 || (*auth)[0] != "Bearer "+pickToken {
		t.Fatalf("GitHub saw Authorization %q, want the gh token", *auth)
	}
}

func TestPickWithoutTokenIsAnonymous(t *testing.T) {
	auth := fakeGitHub(t)
	ghRun = func(...string) (string, error) { return "", errors.New("gh: not installed") }
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	noBrowser(t)
	ws := newWorkspace(t)
	url, _, _, cancel, done := servePick(t, "add", "--pick", "acme", "--no-open", "--port", "0", "--dir", ws)

	res, err := http.Get(url + "api/repos?org=acme")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out struct {
		Authenticated bool `json:"authenticated"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil || out.Authenticated {
		t.Fatalf("authenticated = %v, err = %v", out.Authenticated, err)
	}
	for _, h := range *auth {
		if h != "" {
			t.Fatalf("GitHub saw Authorization %q without a token", h)
		}
	}
	cancel()
	ended(t, done)
}

func TestPickOpensTheBrowser(t *testing.T) {
	fakeGitHub(t)
	opened := make(chan string, 1)
	old := openURL
	openURL = func(url string) error { opened <- url; return nil }
	t.Cleanup(func() { openURL = old })
	ws := newWorkspace(t)
	url, _, _, cancel, done := servePick(t, "add", "--pick", "acme", "--port", "0", "--dir", ws)
	select {
	case got := <-opened:
		if got != url {
			t.Fatalf("opened %s, want %s", got, url)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the browser was not opened")
	}
	cancel()
	ended(t, done)
}

func TestPickFlagRules(t *testing.T) {
	ws := newWorkspace(t)
	for name, c := range map[string]struct {
		args []string
		want string
	}{
		"name with pick":     {[]string{"add", "--pick", "--name", "x", "--dir", ws}, "--pick"},
		"branch with pick":   {[]string{"add", "--pick", "--branch", "x", "--dir", ws}, "--pick"},
		"no-open without it": {[]string{"add", "https://example.com/a.git", "--no-open", "--dir", ws}, "need --pick"},
		"port without it":    {[]string{"add", "https://example.com/a.git", "--port", "1", "--dir", ws}, "need --pick"},
		"no url, no pick":    {[]string{"add", "--dir", ws}, "accepts 1 arg(s), received 0"},
		"two args with pick": {[]string{"add", "--pick", "a", "b", "--dir", ws}, "accepts at most 1 arg(s)"},
	} {
		var out bytes.Buffer
		cmd := newRootCmd()
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(c.args)
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want it to contain %q", name, err, c.want)
		}
	}
}
