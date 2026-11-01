package workspace_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/testutil"
	"github.com/lukascivil/trama/internal/workspace"
)

func apiWorkspace(root string) *workspace.Workspace {
	return &workspace.Workspace{Root: root, Repos: []workspace.Repo{{Name: "api", Branch: "main"}}}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFilesGitMode(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	remote.Commit(t, "db/schema.rb", "create_table\n")
	root := t.TempDir()
	dir := filepath.Join(root, "repos", "api")
	if err := gitx.Clone(remote.URL, "main", dir); err != nil {
		t.Fatal(err)
	}
	// Only on disk: the catalog describes origin/main, so this must not show.
	writeFile(t, filepath.Join(dir, "local.sql"), "CREATE TABLE x ();\n")

	ws := apiWorkspace(root)
	f := ws.Files(ws.Repos[0])
	if !f.Cloned || f.Place != "repos/api at origin/main" {
		t.Fatalf("files = %+v", f)
	}
	got, err := f.List()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"README.md", "db/schema.rb"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("List = %v, want %v", got, want)
	}
	if data, err := f.Read("db/schema.rb"); err != nil || string(data) != "create_table\n" {
		t.Fatalf("Read = %q, %v", data, err)
	}
}

func TestFilesDiskMode(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "repos", "api")
	// A clone with no origin/main falls back to the disk, .git and all.
	testutil.Git(t, root, "init", "-q", dir)
	writeFile(t, filepath.Join(dir, "a.txt"), "a\n")
	writeFile(t, filepath.Join(dir, "db", "schema.rb"), "create_table\n")
	writeFile(t, filepath.Join(dir, "node_modules", "x", "schema.prisma"), "model X {}\n")

	ws := apiWorkspace(root)
	f := ws.Files(ws.Repos[0])
	if !f.Cloned || f.Place != "repos/api" {
		t.Fatalf("files = %+v", f)
	}
	got, err := f.List()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a.txt", "db/schema.rb"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("List = %v, want %v (no .git, no node_modules)", got, want)
	}
}

func TestFilesDiskModeSkipsSymlinks(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "repos", "api")
	writeFile(t, filepath.Join(dir, "a.txt"), "a\n")
	if err := os.Symlink("/etc/hosts", filepath.Join(dir, "lnk.sql")); err != nil {
		t.Fatal(err)
	}
	ws := apiWorkspace(root)
	got, err := ws.Files(ws.Repos[0]).List()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("List = %v, want %v", got, want)
	}
}

func TestFilesNotCloned(t *testing.T) {
	ws := apiWorkspace(t.TempDir())
	if f := ws.Files(ws.Repos[0]); f.Cloned {
		t.Fatalf("files = %+v, want not cloned", f)
	}
}

func TestFilesReadAllGitMode(t *testing.T) {
	remote := testutil.NewRemote(t, "main")
	remote.Commit(t, "db/init.sql", "CREATE TABLE x ();\n")
	root := t.TempDir()
	if err := gitx.Clone(remote.URL, "main", filepath.Join(root, "repos", "api")); err != nil {
		t.Fatal(err)
	}
	ws := apiWorkspace(root)
	got, err := ws.Files(ws.Repos[0]).ReadAll([]string{"db/init.sql"})
	if err != nil || string(got["db/init.sql"]) != "CREATE TABLE x ();\n" {
		t.Fatalf("ReadAll = %q, %v", got, err)
	}
}

func TestFilesReadAllDiskMode(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "repos", "api", "db", "init.sql"), "CREATE TABLE x ();\n")
	ws := apiWorkspace(root)
	got, err := ws.Files(ws.Repos[0]).ReadAll([]string{"db/init.sql"})
	if err != nil || string(got["db/init.sql"]) != "CREATE TABLE x ();\n" {
		t.Fatalf("ReadAll = %q, %v", got, err)
	}
}

func TestFilesDiskModeSkipsSubmoduleCheckouts(t *testing.T) {
	// A checked-out submodule has a .git file; its files belong to another
	// repository, as git mode already treats them.
	root := t.TempDir()
	dir := filepath.Join(root, "repos", "api")
	writeFile(t, filepath.Join(dir, "a.txt"), "a\n")
	writeFile(t, filepath.Join(dir, "sub", ".git"), "gitdir: ../.git/modules/sub\n")
	writeFile(t, filepath.Join(dir, "sub", "db", "schema.rb"), "create_table\n")
	ws := apiWorkspace(root)
	got, err := ws.Files(ws.Repos[0]).List()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("List = %v, want %v", got, want)
	}
}
