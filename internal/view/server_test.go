package view_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/view"
)

const port = 7300

func handler(build func() (*view.Model, error)) http.Handler {
	if build == nil {
		build = func() (*view.Model, error) {
			return &view.Model{Workspace: "acme", Repos: []view.Repo{}, Edges: []view.Edge{}, Warnings: []string{}}, nil
		}
	}
	return view.Handler(build, port)
}

func get(t *testing.T, h http.Handler, method, target, host string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	req.Host = host
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestServerServesIndexAndAPI(t *testing.T) {
	h := handler(nil)
	page := get(t, h, "GET", "/", "127.0.0.1:7300")
	if page.Code != 200 || !strings.Contains(page.Header().Get("Content-Type"), "text/html") || !strings.Contains(page.Body.String(), "<html") {
		t.Fatalf("GET / = %d %q", page.Code, page.Header().Get("Content-Type"))
	}
	api := get(t, h, "GET", "/api/workspace", "localhost:7300")
	if api.Code != 200 || api.Header().Get("Content-Type") != "application/json" || api.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("GET /api/workspace = %d %v", api.Code, api.Header())
	}
	var m view.Model
	if err := json.Unmarshal(api.Body.Bytes(), &m); err != nil || m.Workspace != "acme" {
		t.Fatalf("body = %s (%v)", api.Body, err)
	}
	if missing := get(t, h, "GET", "/nope", "127.0.0.1:7300"); missing.Code != 404 {
		t.Fatalf("GET /nope = %d, want 404", missing.Code)
	}
}

func TestServerRejectsForeignHost(t *testing.T) {
	h := handler(nil)
	for _, host := range []string{"evil.example:7300", "localhost:7301", "127.0.0.1", "[::1]:7300"} {
		if rec := get(t, h, "GET", "/api/workspace", host); rec.Code != 403 {
			t.Errorf("Host %q: %d, want 403", host, rec.Code)
		}
	}
	if rec := get(t, h, "GET", "/api/workspace", "LOCALHOST:7300"); rec.Code != 200 {
		t.Errorf("Host LOCALHOST:7300: %d, want 200", rec.Code)
	}
}

func TestServerOnlyGET(t *testing.T) {
	if rec := get(t, handler(nil), "POST", "/api/workspace", "127.0.0.1:7300"); rec.Code != 405 {
		t.Fatalf("POST /api/workspace = %d, want 405", rec.Code)
	}
}

func TestServerSecurityHeaders(t *testing.T) {
	h := handler(nil)
	for _, target := range []string{"/", "/assets/index.html", "/api/workspace", "/nope"} {
		rec := get(t, h, "GET", target, "127.0.0.1:7300")
		if csp := rec.Header().Get("Content-Security-Policy"); !strings.HasPrefix(csp, "default-src 'self'") {
			t.Errorf("%s: CSP = %q", target, csp)
		}
		if rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("Referrer-Policy") != "no-referrer" {
			t.Errorf("%s: headers = %v", target, rec.Header())
		}
	}
	if rec := get(t, h, "GET", "/", "evil.example:7300"); !strings.HasPrefix(rec.Header().Get("Content-Security-Policy"), "default-src 'self'") {
		t.Error("the 403 answer has no CSP")
	}
}

func TestServerAPIErrorIsJSON(t *testing.T) {
	h := handler(func() (*view.Model, error) { return nil, errors.New("workspace.yaml: broken") })
	rec := get(t, h, "GET", "/api/workspace", "127.0.0.1:7300")
	var body struct{ Error string }
	if rec.Code != 500 || json.Unmarshal(rec.Body.Bytes(), &body) != nil || body.Error != "workspace.yaml: broken" {
		t.Fatalf("= %d %s", rec.Code, rec.Body)
	}
}

func TestServerDoesNotReadDisk(t *testing.T) {
	h := handler(nil)
	for _, target := range []string{"/assets/../workspace.yaml", "/assets/%2e%2e/go.mod", "/assets/../../go.mod", "/go.mod"} {
		rec := get(t, h, "GET", target, "127.0.0.1:7300")
		if rec.Code == 200 || strings.Contains(rec.Body.String(), "module github.com") {
			t.Errorf("%s = %d, served something", target, rec.Code)
		}
	}
}

func TestServerRejectsOtherSites(t *testing.T) {
	h := handler(nil)
	for _, site := range []string{"cross-site", "same-site"} {
		req := httptest.NewRequest("GET", "/api/workspace", nil)
		req.Host = "127.0.0.1:7300"
		req.Header.Set("Sec-Fetch-Site", site)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 403 {
			t.Errorf("Sec-Fetch-Site %s: %d, want 403", site, rec.Code)
		}
	}
	for _, site := range []string{"same-origin", "none", ""} {
		req := httptest.NewRequest("GET", "/api/workspace", nil)
		req.Host = "127.0.0.1:7300"
		if site != "" {
			req.Header.Set("Sec-Fetch-Site", site)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Errorf("Sec-Fetch-Site %q: %d, want 200", site, rec.Code)
		}
	}
}

func TestGuardRefusesForeignHostAndSite(t *testing.T) {
	h := view.Guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), port)
	cases := []struct {
		name, host, site string
		want             int
	}{
		{"foreign host", "evil.test", "", http.StatusForbidden},
		{"cross-site request", "127.0.0.1:7300", "cross-site", http.StatusForbidden},
		{"same machine", "127.0.0.1:7300", "", http.StatusNoContent},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodPost, "/anything", nil)
		req.Host = c.host
		if c.site != "" {
			req.Header.Set("Sec-Fetch-Site", c.site)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s: status = %d, want %d", c.name, rec.Code, c.want)
		}
		if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "default-src 'self'") ||
			rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%s: security headers missing: %v", c.name, rec.Header())
		}
	}
}
