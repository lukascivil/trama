package workspace_test

import (
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/workspace"
)

func repoYAML(url, branch string) string {
	return "version: 1\nname: demo\nrepos:\n  - name: api\n    url: \"" + url + "\"\n    branch: \"" + branch + "\"\n"
}

func TestLoadRejectsDashURL(t *testing.T) {
	_, err := workspace.Load(writeWorkspace(t, repoYAML("--upload-pack=touch x", "main")))
	if err == nil || !strings.Contains(err.Error(), `must not start with "-"`) {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadRejectsBadBranch(t *testing.T) {
	for _, b := range []string{"-x", "a..b", "has space", "x~1", "end/", "x.lock", "a:b", "@{u}"} {
		_, err := workspace.Load(writeWorkspace(t, repoYAML("https://github.com/x/api.git", b)))
		if err == nil || !strings.Contains(err.Error(), "invalid branch name") {
			t.Errorf("branch %q: err = %v", b, err)
		}
	}
}

func TestLoadAcceptsSlashBranch(t *testing.T) {
	for _, b := range []string{"release/1.0", "feature/x-y_z"} {
		if _, err := workspace.Load(writeWorkspace(t, repoYAML("https://github.com/x/api.git", b))); err != nil {
			t.Errorf("branch %q: %v", b, err)
		}
	}
}

func TestCheckNew(t *testing.T) {
	ws, err := workspace.Load(writeWorkspace(t, twoRepos))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][2]string{
		"duplicate name": {"api", "https://github.com/x/new.git"},
		"duplicate url":  {"api2", "https://github.com/x/api"},
		"dash url":       {"ok", "-x"},
		"invalid name":   {"Bad", "https://github.com/x/bad.git"},
	}
	for label, c := range cases {
		if err := ws.CheckNew(c[0], c[1]); err == nil {
			t.Errorf("%s: CheckNew(%q, %q) accepted", label, c[0], c[1])
		}
	}
	if err := ws.CheckNew("new", "https://github.com/x/new.git"); err != nil {
		t.Errorf("valid repo rejected: %v", err)
	}
}
