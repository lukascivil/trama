package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/lukascivil/trama/internal/fsx"
	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/workspace"
)

// markerPath is where the last successful setup of a repository is
// recorded. It lives in repos/, which git already ignores.
func markerPath(ws *workspace.Workspace, repo string) string {
	return filepath.Join(ws.Root, "repos", ".trama", repo+".setup")
}

// marker is "<HEAD sha> <sha256 of the setup command>".
func marker(j Job) string {
	head, err := gitx.Repo{Dir: j.Dir}.RevParse("HEAD")
	if err != nil {
		head = "none"
	}
	sum := sha256.Sum256([]byte(j.Cmd))
	return head + " " + hex.EncodeToString(sum[:])
}

// NeedsSetup reports whether the setup job j has not run since HEAD or the
// command last changed.
func NeedsSetup(ws *workspace.Workspace, j Job) (bool, error) {
	data, err := os.ReadFile(markerPath(ws, j.Repo))
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(data)) != marker(j), nil
}

// MarkSetup records that the setup job j just succeeded.
func MarkSetup(ws *workspace.Workspace, j Job) error {
	path := markerPath(ws, j.Repo)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return fsx.ReplaceFile(path, []byte(marker(j)+"\n"), 0o644)
}
