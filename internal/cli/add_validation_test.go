package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddRejectsDashURLWithoutRunningGit(t *testing.T) {
	ws := newWorkspace(t)
	marker := filepath.Join(t.TempDir(), "pwned")
	_, err := run(t, "add", "--dir", ws, "--name", "d", "--", "--upload-pack=touch "+marker)
	if err == nil || !strings.Contains(err.Error(), `must not start with "-"`) {
		t.Fatalf("err = %v", err)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatal("the URL was run as a git option")
	}
}

func TestAddSuggestsNameFlagForInvalidDerivedName(t *testing.T) {
	ws := newWorkspace(t)
	_, err := run(t, "add", "--dir", ws, filepath.Join(t.TempDir(), ".hidden.git"))
	if err == nil || !strings.Contains(err.Error(), "--name") {
		t.Fatalf("err = %v, want a hint about --name", err)
	}
}
