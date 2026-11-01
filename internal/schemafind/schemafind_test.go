package schemafind_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/lukascivil/trama/internal/schemafind"
)

// repo serves a fixed set of files: list returns their paths, read their
// content, and reading a file that is not there fails.
func repo(files map[string]string) (func() ([]string, error), func(string) ([]byte, error)) {
	list := func() ([]string, error) {
		out := make([]string, 0, len(files))
		for p := range files {
			out = append(out, p)
		}
		return out, nil
	}
	read := func(p string) ([]byte, error) {
		c, ok := files[p]
		if !ok {
			return nil, fmt.Errorf("%s does not exist", p)
		}
		return []byte(c), nil
	}
	return list, read
}

func scan(t *testing.T, files map[string]string) []schemafind.File {
	t.Helper()
	list, read := repo(files)
	got, err := schemafind.Scan(list, read)
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

func TestScanReadsOnlySQL(t *testing.T) {
	list := func() ([]string, error) {
		return []string{"db/schema.rb", "migrations/1.sql", "main.go", "db/q.sql"}, nil
	}
	var reads []string
	read := func(p string) ([]byte, error) {
		reads = append(reads, p)
		return []byte("SELECT 1;"), nil
	}
	if _, err := schemafind.Scan(list, read); err != nil {
		t.Fatal(err)
	}
	// Only a .sql outside a migration directory needs its content read.
	if want := []string{"db/q.sql"}; !reflect.DeepEqual(reads, want) {
		t.Fatalf("read %v, want only %v", reads, want)
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
