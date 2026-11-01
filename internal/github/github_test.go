package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func repoJSON(i int) map[string]any {
	return map[string]any{
		"full_name":      fmt.Sprintf("acme/r%03d", i),
		"name":           fmt.Sprintf("r%03d", i),
		"description":    "repo " + fmt.Sprint(i),
		"language":       "Go",
		"default_branch": "main",
		"clone_url":      fmt.Sprintf("https://github.com/acme/r%03d.git", i),
		"ssh_url":        fmt.Sprintf("git@github.com:acme/r%03d.git", i),
		"archived":       false,
		"pushed_at":      "2026-10-01T10:00:00Z",
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func client(srv *httptest.Server, token string) *Client {
	c := NewClient(token)
	c.BaseURL = srv.URL
	return c
}

func TestReposFollowsPagination(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs/acme/repos" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		page, size := r.URL.Query().Get("page"), 100
		var from int
		switch page {
		case "", "1":
			from = 0
			w.Header().Set("Link", fmt.Sprintf(`<%s/orgs/acme/repos?page=2&per_page=100>; rel="next"`, srv.URL))
		case "2":
			from = 100
			w.Header().Set("Link", fmt.Sprintf(`<%s/orgs/acme/repos?page=3&per_page=100>; rel="next", <%s/orgs/acme/repos?page=1>; rel="prev"`, srv.URL, srv.URL))
		default:
			from, size = 200, 50
		}
		out := make([]map[string]any, 0, size)
		for i := from; i < from+size; i++ {
			out = append(out, repoJSON(i))
		}
		writeJSON(w, out)
	}))
	defer srv.Close()

	repos, err := client(srv, "").Repos(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 250 {
		t.Fatalf("got %d repos, want 250", len(repos))
	}
	if repos[0].FullName != "acme/r000" || repos[249].FullName != "acme/r249" {
		t.Fatalf("order: first %s, last %s", repos[0].FullName, repos[249].FullName)
	}
	if r := repos[7]; r.DefaultBranch != "main" || r.CloneURL != "https://github.com/acme/r007.git" ||
		r.SSHURL != "git@github.com:acme/r007.git" || r.Language != "Go" || r.PushedAt.Year() != 2026 {
		t.Fatalf("fields: %+v", r)
	}
}

func TestReposSendsToken(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Header.Get("Authorization"))
		writeJSON(w, []map[string]any{})
	}))
	defer srv.Close()

	if _, err := client(srv, "SECRET").Repos(context.Background(), "acme"); err != nil {
		t.Fatal(err)
	}
	if _, err := client(srv, "").Repos(context.Background(), "acme"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "Bearer SECRET" || got[1] != "" {
		t.Fatalf("Authorization headers = %q", got)
	}
}

func TestReposFallsBackToUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/orgs/lucas/repos":
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]string{"message": "Not Found"})
		case "/users/lucas/repos":
			writeJSON(w, []map[string]any{repoJSON(1)})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	repos, err := client(srv, "").Repos(context.Background(), "lucas")
	if err != nil || len(repos) != 1 || repos[0].Name != "r001" {
		t.Fatalf("repos = %+v, err = %v", repos, err)
	}
}

func TestReposNullFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"full_name":"acme/x","name":"x","description":null,"language":null,"default_branch":"main","clone_url":"https://github.com/acme/x.git","ssh_url":"git@github.com:acme/x.git","archived":true,"pushed_at":null}]`)
	}))
	defer srv.Close()

	repos, err := client(srv, "").Repos(context.Background(), "acme")
	if err != nil || len(repos) != 1 {
		t.Fatalf("repos = %+v, err = %v", repos, err)
	}
	if r := repos[0]; r.Description != "" || r.Language != "" || !r.Archived || !r.PushedAt.IsZero() {
		t.Fatalf("null fields: %+v", r)
	}
}

func TestReposRefusesForeignNextLink(t *testing.T) {
	var hits atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		writeJSON(w, []map[string]any{})
	}))
	defer other.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", fmt.Sprintf(`<%s/steal>; rel="next"`, other.URL))
		writeJSON(w, []map[string]any{repoJSON(1)})
	}))
	defer srv.Close()

	_, err := client(srv, "SECRET").Repos(context.Background(), "acme")
	if err == nil {
		t.Fatal("want an error for a next link to another host")
	}
	if hits.Load() != 0 {
		t.Fatalf("the other host got %d request(s)", hits.Load())
	}
}

func TestErrorsNeverHoldTheToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		writeJSON(w, map[string]string{
			"message": "Resource protected by organization SAML enforcement",
			"echo":    r.Header.Get("Authorization"),
		})
	}))
	defer srv.Close()

	_, err := client(srv, "SECRET").Repos(context.Background(), "acme")
	if err == nil {
		t.Fatal("want an error")
	}
	if !strings.Contains(err.Error(), "Resource protected by organization SAML enforcement") {
		t.Errorf("error lacks GitHub's message: %v", err)
	}
	if strings.Contains(err.Error(), "SECRET") {
		t.Errorf("error holds the token: %v", err)
	}
	var api *APIError
	if !errors.As(err, &api) || api.Status != http.StatusForbidden {
		t.Errorf("want *APIError with status 403, got %#v", err)
	}
}

func TestOrgsListsOwnerThenOrgs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			writeJSON(w, map[string]string{"login": "me"})
		case "/user/orgs":
			writeJSON(w, []map[string]string{{"login": "acme"}, {"login": "beta"}})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	orgs, err := client(srv, "SECRET").Orgs(context.Background())
	if err != nil || strings.Join(orgs, ",") != "me,acme,beta" {
		t.Fatalf("orgs = %v, err = %v", orgs, err)
	}
}

func TestOrgsWithoutTokenIsEmpty(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer srv.Close()

	orgs, err := client(srv, "").Orgs(context.Background())
	if err != nil || orgs != nil || hits.Load() != 0 {
		t.Fatalf("orgs = %v, err = %v, requests = %d", orgs, err, hits.Load())
	}
}

func TestReposOfTheTokenOwnerIncludePrivate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/orgs/lucas/repos":
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]string{"message": "Not Found"})
		case "/user":
			writeJSON(w, map[string]string{"login": "Lucas"})
		case "/user/repos":
			if r.URL.Query().Get("affiliation") != "owner" {
				t.Errorf("affiliation = %q, want owner", r.URL.Query().Get("affiliation"))
			}
			writeJSON(w, []map[string]any{repoJSON(1), repoJSON(2)})
		default:
			t.Errorf("unexpected path %s: the public listing hides private repositories", r.URL.Path)
		}
	}))
	defer srv.Close()

	repos, err := client(srv, "SECRET").Repos(context.Background(), "lucas")
	if err != nil || len(repos) != 2 {
		t.Fatalf("repos = %+v, err = %v", repos, err)
	}
}

func TestReposOfAnotherUserStayPublic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/orgs/someone/repos":
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]string{"message": "Not Found"})
		case "/user":
			writeJSON(w, map[string]string{"login": "me"})
		case "/users/someone/repos":
			writeJSON(w, []map[string]any{repoJSON(1)})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	repos, err := client(srv, "SECRET").Repos(context.Background(), "someone")
	if err != nil || len(repos) != 1 {
		t.Fatalf("repos = %+v, err = %v", repos, err)
	}
}
