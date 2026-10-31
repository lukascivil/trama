package changes_test

import (
	"testing"

	"github.com/lukascivil/trama/internal/changes"
)

func TestRecordURL(t *testing.T) {
	want := "https://github.com/lukascivil/trama-demo/blob/main/changes/x.md"
	for _, origin := range []string{"https://github.com/lukascivil/trama-demo.git", "https://github.com/lukascivil/trama-demo", "git@github.com:lukascivil/trama-demo.git"} {
		if got := changes.RecordURL(origin, "main", "changes/x.md"); got != want {
			t.Errorf("RecordURL(%q) = %q", origin, got)
		}
	}
	if got := changes.RecordURL("/tmp/ws.git", "main", "changes/x.md"); got != "" {
		t.Errorf("local origin gave %q", got)
	}
}

func TestGitHubSlug(t *testing.T) {
	for _, origin := range []string{"https://github.com/owner/repo.git", "https://github.com/owner/repo/", "git@github.com:owner/repo.git", "ssh://git@github.com/owner/repo"} {
		if got := changes.GitHubSlug(origin); got != "owner/repo" {
			t.Errorf("GitHubSlug(%q) = %q, want owner/repo", origin, got)
		}
	}
	for _, origin := range []string{"https://gitlab.com/owner/repo.git", "/tmp/remote.git", ""} {
		if got := changes.GitHubSlug(origin); got != "" {
			t.Errorf("GitHubSlug(%q) = %q, want empty", origin, got)
		}
	}
}
