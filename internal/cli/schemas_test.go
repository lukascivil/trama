package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRepoFile(t *testing.T, ws, rel, content string) {
	t.Helper()
	full := filepath.Join(ws, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSchemasListsByKind(t *testing.T) {
	ws := catalogWorkspace(t, cliAPI, cliWeb)
	writeRepoFile(t, ws, "repos/api/db/init.sql", "CREATE TABLE users (id int);\n")
	writeRepoFile(t, ws, "repos/api/db/migrate/001_users.rb", "create_table :users\n")
	writeRepoFile(t, ws, "repos/api/db/schema.rb", "create_table \"users\"\n")
	out, err := run(t, "schemas", "api", "--dir", ws)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	want := "api\n" +
		"  schema     db/schema.rb\n" +
		"  migration  db/migrate/001_users.rb\n" +
		"  sql        db/init.sql\n"
	if out != want {
		t.Fatalf("output:\n%s\nwant:\n%s", out, want)
	}
}

func TestSchemasNoFiles(t *testing.T) {
	ws := catalogWorkspace(t, cliAPI, cliWeb)
	out, err := run(t, "schemas", "web", "--dir", ws)
	if err != nil || out != "web\n  no schema files found\n" {
		t.Fatalf("%v\n%q", err, out)
	}
}

func TestSchemasUnknownRepo(t *testing.T) {
	ws := catalogWorkspace(t, cliAPI, cliWeb)
	if _, err := run(t, "schemas", "nope", "--dir", ws); err == nil || err.Error() != "repo nope is not in workspace.yaml" {
		t.Fatalf("err = %v", err)
	}
}

func TestSchemasNotCloned(t *testing.T) {
	ws := catalogWorkspace(t, cliAPI, cliWeb)
	if err := os.RemoveAll(filepath.Join(ws, "repos", "web")); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "schemas", "web", "--dir", ws); err == nil || err.Error() != `repos/web is not cloned: run "trama sync web"` {
		t.Fatalf("err = %v", err)
	}
}
