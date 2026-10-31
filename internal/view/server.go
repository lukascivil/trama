package view

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
)

//go:embed assets
var assets embed.FS

// csp lets the page load and fetch only from this server.
const csp = "default-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

// Handler serves the page, its files and the model of the workspace. It
// answers only GET, and only requests addressed to this server on port, so
// a page from another site cannot reach it through DNS rebinding.
func Handler(build func() (*Model, error), port int) http.Handler {
	static, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err) // the assets directory is embedded at build time
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, static, "index.html")
	})
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(static)))
	mux.HandleFunc("GET /api/workspace", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		m, err := build()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(m)
	})

	hosts := []string{"127.0.0.1:" + strconv.Itoa(port), "localhost:" + strconv.Itoa(port)}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		if !allowedHost(r.Host, hosts) {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		// A page on another site can still send requests here (an image, a
		// no-cors fetch). It cannot read the answers, but each one costs a
		// build, so refuse what the browser marks as coming from elsewhere.
		switch r.Header.Get("Sec-Fetch-Site") {
		case "", "none", "same-origin":
		default:
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func allowedHost(host string, hosts []string) bool {
	for _, h := range hosts {
		if strings.EqualFold(host, h) {
			return true
		}
	}
	return false
}
