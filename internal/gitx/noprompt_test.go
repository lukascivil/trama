package gitx_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/testutil"
)

func isolateGitConfig(t *testing.T, global string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", global)
}

func TestDefaultBranchNeverRunsAskpass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="private"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	marker := filepath.Join(t.TempDir(), "asked")
	stub := filepath.Join(t.TempDir(), "askpass")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\ntouch '"+marker+"'\necho x\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	isolateGitConfig(t, os.DevNull)
	t.Setenv("GIT_ASKPASS", stub)
	if _, err := gitx.DefaultBranch(srv.URL + "/private.git"); err == nil {
		t.Fatal("want error for a private remote")
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the askpass helper was called")
	}
}

func TestDefaultBranchDoesNotTreatURLAsOption(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "pwned")
	if _, err := gitx.DefaultBranch("--upload-pack=touch " + marker); err == nil {
		t.Fatal("want error")
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the URL was run as a git option")
	}
}

func TestOriginURLIgnoresInsteadOf(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	cfg := filepath.Join(t.TempDir(), "gitconfig")
	rule := "[url \"" + filepath.Dir(remote.URL) + "/\"]\n\tinsteadOf = https://example.invalid/org/\n"
	if err := os.WriteFile(cfg, []byte(rule), 0o644); err != nil {
		t.Fatal(err)
	}
	isolateGitConfig(t, cfg)
	url := "https://example.invalid/org/remote.git"
	dir := filepath.Join(t.TempDir(), "clone")
	if err := gitx.Clone(url, "main", dir); err != nil {
		t.Fatal(err)
	}
	if got, err := (gitx.Repo{Dir: dir}).OriginURL(); err != nil || got != url {
		t.Fatalf("OriginURL = %q, %v; want %q", got, err, url)
	}
}

func TestFastForwardMissingUpstreamIsNotDivergence(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	r := cloneOf(t, remote, "main")
	if _, err := r.FastForward("gone"); err == nil || errors.Is(err, gitx.ErrDiverged) {
		t.Fatalf("FastForward(gone) err = %v; want an error that is not ErrDiverged", err)
	}
}
