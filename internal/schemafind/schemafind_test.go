package schemafind_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/lukascivil/trama/internal/schemafind"
)

// repo serves a fixed set of files: list returns their paths, readAll
// their content, and reading a file that is not there fails.
func repo(files map[string]string) (func() ([]string, error), func([]string) (map[string][]byte, error)) {
	list := func() ([]string, error) {
		out := make([]string, 0, len(files))
		for p := range files {
			out = append(out, p)
		}
		return out, nil
	}
	readAll := func(paths []string) (map[string][]byte, error) {
		out := map[string][]byte{}
		for _, p := range paths {
			c, ok := files[p]
			if !ok {
				return nil, fmt.Errorf("%s does not exist", p)
			}
			out[p] = []byte(c)
		}
		return out, nil
	}
	return list, readAll
}

func scan(t *testing.T, files map[string]string) []schemafind.File {
	t.Helper()
	list, readAll := repo(files)
	got, err := schemafind.Scan(list, readAll)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestScanKinds(t *testing.T) {
	got := scan(t, map[string]string{
		"db/init.sql":                    "CREATE TABLE users (id int);",
		"db/migrate/001_create_users.rb": "create_table :users",
		"db/schema.rb":                   "create_table \"users\"",
		"README.md":                      "# api",
		"main.go":                        "package main",
	})
	want := []schemafind.File{
		{Kind: "schema", Path: "db/schema.rb"},
		{Kind: "migration", Path: "db/migrate/001_create_users.rb"},
		{Kind: "sql", Path: "db/init.sql"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scan = %v, want %v", got, want)
	}
}

func TestScanPrecedence(t *testing.T) {
	got := scan(t, map[string]string{
		"prisma/schema.prisma":                     "model User {}",
		"prisma/migrations/20240101/migration.sql": "CREATE TABLE \"User\" ();",
	})
	want := []schemafind.File{
		{Kind: "schema", Path: "prisma/schema.prisma"},
		{Kind: "migration", Path: "prisma/migrations/20240101/migration.sql"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scan = %v, want %v", got, want)
	}
}

func TestScanSkipsQueriesWithoutCreateTable(t *testing.T) {
	// sqlc keeps its queries in .sql files: they declare no table.
	if got := scan(t, map[string]string{"db/query.sql": "-- name: GetUser :one\nSELECT * FROM users WHERE id = $1;"}); len(got) != 0 {
		t.Fatalf("Scan = %v, want nothing", got)
	}
}

func TestScanCreateTableIsCaseInsensitive(t *testing.T) {
	got := scan(t, map[string]string{
		"a.sql": "create table users (id int);",
		"b.sql": "CREATE UNLOGGED TABLE events (id int);",
		"c.sql": "Create   Table orders (id int);",
	})
	want := []schemafind.File{{Kind: "sql", Path: "a.sql"}, {Kind: "sql", Path: "b.sql"}, {Kind: "sql", Path: "c.sql"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scan = %v, want %v", got, want)
	}
}

func TestScanSkipsExcludedDirs(t *testing.T) {
	files := map[string]string{}
	for _, dir := range []string{"node_modules/x", "vendor/y", "dist", "build", "target", "testdata", "fixtures", "test", "tests", "spec", "__tests__"} {
		files[dir+"/schema.prisma"] = "model X {}"
		files[dir+"/migrations/1.sql"] = "CREATE TABLE x ();"
	}
	if got := scan(t, files); len(got) != 0 {
		t.Fatalf("Scan = %v, want nothing", got)
	}
}

func TestScanSkipsNoiseInMigrationDirs(t *testing.T) {
	got := scan(t, map[string]string{
		"migrations/.keep":               "",
		"migrations/README.md":           "how to migrate",
		"migrations/notes.txt":           "todo",
		"app/migrations/__init__.py":     "",
		"app/migrations/0001_initial.py": "migrations.CreateModel(name='User')",
	})
	want := []schemafind.File{{Kind: "migration", Path: "app/migrations/0001_initial.py"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scan = %v, want %v", got, want)
	}
}

func TestScanReadsOnlySQLInOneBatch(t *testing.T) {
	list := func() ([]string, error) {
		return []string{"db/schema.rb", "migrations/1.sql", "main.go", "db/q.sql", "reports/a.sql"}, nil
	}
	var calls [][]string
	readAll := func(paths []string) (map[string][]byte, error) {
		calls = append(calls, paths)
		return map[string][]byte{}, nil
	}
	if _, err := schemafind.Scan(list, readAll); err != nil {
		t.Fatal(err)
	}
	// One read for every loose .sql, however many there are: reading them
	// one by one is what made a repository of dbt models take minutes.
	if want := [][]string{{"db/q.sql", "reports/a.sql"}}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("readAll calls = %v, want %v", calls, want)
	}
}

func TestScanNoSQLNoRead(t *testing.T) {
	readAll := func([]string) (map[string][]byte, error) {
		t.Fatal("readAll called with no loose .sql to read")
		return nil, nil
	}
	if _, err := schemafind.Scan(func() ([]string, error) { return []string{"db/schema.rb", "main.go"}, nil }, readAll); err != nil {
		t.Fatal(err)
	}
}

func TestScanManyMigrations(t *testing.T) {
	files := map[string]string{}
	for i := 500; i >= 1; i-- {
		files[fmt.Sprintf("db/migrate/%04d_step.rb", i)] = ""
	}
	got := scan(t, files)
	if len(got) != 500 {
		t.Fatalf("got %d files, want all 500", len(got))
	}
	if got[0].Path != "db/migrate/0001_step.rb" || got[499].Path != "db/migrate/0500_step.rb" {
		t.Fatalf("not in path order: first %s, last %s", got[0].Path, got[499].Path)
	}
}

func TestScanListError(t *testing.T) {
	boom := errors.New("boom")
	_, err := schemafind.Scan(func() ([]string, error) { return nil, boom }, nil)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
}

func TestScanEFCore(t *testing.T) {
	// EF Core keeps them in Migrations, capitalized, as C#; the model
	// snapshot is its current schema.
	got := scan(t, map[string]string{
		"Data/Migrations/20240101_Init.cs":             "migrationBuilder.CreateTable(",
		"Data/Migrations/AppDbContextModelSnapshot.cs": "modelBuilder.Entity(",
	})
	want := []schemafind.File{
		{Kind: "schema", Path: "Data/Migrations/AppDbContextModelSnapshot.cs"},
		{Kind: "migration", Path: "Data/Migrations/20240101_Init.cs"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scan = %v, want %v", got, want)
	}
}

func TestScanLiquibase(t *testing.T) {
	got := scan(t, map[string]string{
		"src/main/resources/db/changelog/db.changelog-master.yaml": "createTable:",
		"docs/changelog/v1.yaml":                                   "release notes",
	})
	want := []schemafind.File{{Kind: "migration", Path: "src/main/resources/db/changelog/db.changelog-master.yaml"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scan = %v, want %v (only the db/changelog one)", got, want)
	}
}

func TestScanUppercaseSQL(t *testing.T) {
	got := scan(t, map[string]string{"db/INIT.SQL": "CREATE TABLE x ();"})
	if want := []schemafind.File{{Kind: "sql", Path: "db/INIT.SQL"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Scan = %v, want %v", got, want)
	}
}

func TestScanRailsMultipleDatabases(t *testing.T) {
	got := scan(t, map[string]string{
		"db/animals_schema.rb":           "create_table",
		"db/animals_migrate/001_dogs.rb": "create_table :dogs",
		"db/animals_structure.sql":       "CREATE TABLE dogs ();",
	})
	want := []schemafind.File{
		{Kind: "schema", Path: "db/animals_schema.rb"},
		{Kind: "schema", Path: "db/animals_structure.sql"},
		{Kind: "migration", Path: "db/animals_migrate/001_dogs.rb"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scan = %v, want %v", got, want)
	}
}

func TestScanSkipsVirtualenvsAndHiddenDirs(t *testing.T) {
	got := scan(t, map[string]string{
		".venv/lib/python3.12/site-packages/django/contrib/auth/migrations/0001_initial.py": "x",
		"venv/lib/python3.12/site-packages/x/migrations/0001.py":                            "x",
		"env/lib/python3.12/site-packages/y/migrations/0001.py":                             "x",
		".tox/py312/schema.sql": "CREATE TABLE x ();",
	})
	if len(got) != 0 {
		t.Fatalf("Scan = %v, want nothing", got)
	}
}

func TestScanSkipsMigrationToolFiles(t *testing.T) {
	got := scan(t, map[string]string{
		"alembic/env.py":                        "context.configure(",
		"alembic/script.py.mako":                "${up_revision}",
		"alembic.ini":                           "[alembic]",
		"prisma/migrations/migration_lock.toml": "provider = \"postgresql\"",
		"alembic/versions/3f2a_init.py":         "op.create_table(",
	})
	want := []schemafind.File{{Kind: "migration", Path: "alembic/versions/3f2a_init.py"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scan = %v, want %v", got, want)
	}
}

func TestScanSkipsNestedTestDirs(t *testing.T) {
	// Maven keeps test resources under src/test: a copy for tests, not the
	// repository's own schema.
	if got := scan(t, map[string]string{"src/test/resources/db/migration/V1__init.sql": "CREATE TABLE x ();"}); len(got) != 0 {
		t.Fatalf("Scan = %v, want nothing", got)
	}
}
