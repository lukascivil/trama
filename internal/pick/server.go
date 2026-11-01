package pick

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"io/fs"
	"mime"
	"net/http"
	"time"

	"github.com/lukascivil/trama/internal/view"
	"github.com/lukascivil/trama/internal/workspace"
)

//go:embed assets
var assets embed.FS

// maxBody bounds what a POST may send.
const maxBody = 1 << 20

type repoView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	PushedAt    string `json:"pushedAt"`
	Archived    bool   `json:"archived"`
	Blocked     string `json:"blocked"`
}

type reposBody struct {
	Token         string     `json:"token"`
	Org           string     `json:"org"`
	Orgs          []string   `json:"orgs"`
	Authenticated bool       `json:"authenticated"`
	Error         string     `json:"error"`
	Repos         []repoView `json:"repos"`
}

// Handler serves the page and its API. It shares the guard of the graph page
// (only this machine, only this port, nothing from another site), and the
// writes also need the token that GET /api/repos hands to the page.
func (p *Picker) Handler(port int) http.Handler {
	static, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err) // the assets directory is embedded at build time
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, static, "index.html")
	})
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(static)))
	mux.HandleFunc("GET /api/repos", p.handleRepos)
	mux.HandleFunc("POST /api/add", p.write(p.handleAdd))
	mux.HandleFunc("POST /api/cancel", p.write(func(w http.ResponseWriter, _ *http.Request) {
		p.end()
		w.WriteHeader(http.StatusNoContent)
	}))
	return view.Guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.touch()
		mux.ServeHTTP(w, r)
	}), port)
}

// write refuses a POST that is not JSON from the page this picker served.
func (p *Picker) write(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || media != "application/json" ||
			subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Trama-Token")), []byte(p.token)) != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (p *Picker) handleRepos(w http.ResponseWriter, r *http.Request) {
	org := r.URL.Query().Get("org")
	if org == "" {
		org = p.o.Org
	}
	out := reposBody{Token: p.token, Org: org, Orgs: []string{}, Authenticated: p.o.Authenticated, Repos: []repoView{}}

	p.mu.Lock()
	defer p.mu.Unlock()
	orgs, err := p.orgsLocked(r.Context())
	if err != nil && org == "" {
		out.Error = err.Error()
	}
	if orgs != nil {
		out.Orgs = orgs
	}
	if org != "" {
		repos, err := p.listLocked(r.Context(), org)
		if err != nil {
			out.Error = err.Error()
		}
		for _, repo := range repos {
			v := repoView{
				ID: repo.FullName, Name: repo.Name, Description: repo.Description,
				Language: repo.Language, Archived: repo.Archived,
			}
			if !repo.PushedAt.IsZero() {
				v.PushedAt = repo.PushedAt.UTC().Format(time.RFC3339)
			}
			url := p.cloneURL(repo)
			if err := p.o.Workspace.CheckNew(workspace.NameFromURL(url), url); err != nil {
				v.Blocked = err.Error()
			}
			out.Repos = append(out.Repos, v)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Picker) handleAdd(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Org string   `json:"org"`
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	if in.Org == "" {
		in.Org = p.o.Org
	}
	res, err := p.Apply(r.Context(), in.Org, in.IDs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}
