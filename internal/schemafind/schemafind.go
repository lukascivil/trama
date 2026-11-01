// Package schemafind lists the files of a repository that declare its
// tables. It only says which files to open: reading them and naming the
// tables is left to whoever maps the repository, because that needs to
// understand the file, and finding it does not.
package schemafind

import (
	"path"
	"regexp"
	"sort"
	"strings"
)

// File is a file that declares tables, and what kind of declaration it is:
// schema (the current state of the database in one file), migration (one
// step of it) or sql (a .sql file that creates a table).
type File struct {
	Kind string
	Path string
}

// Names are compared in lower case: EF Core keeps its migrations in
// Migrations, and nothing says a .sql file is not called INIT.SQL.
var (
	schemaNames = set("schema.prisma", "schema.rb", "structure.sql", "schema.sql")
	// Rails names each extra database's schema <name>_schema.rb or
	// <name>_structure.sql, and EF Core's model snapshot is its schema.
	schemaSuffixes = []string{"_schema.rb", "_structure.sql", "modelsnapshot.cs"}
	migrationDirs  = set("migrations", "migration", "migrate", "alembic", "flyway", "liquibase")
	// Rails migrates each extra database from db/<name>_migrate.
	migrationSuffix = "_migrate"
	skippedDirs     = set("node_modules", "vendor", "dist", "build", "target", "testdata", "fixtures", "test", "tests", "spec", "__tests__", "venv", "site-packages")
	// Files that live in a migration directory without being a migration.
	noise       = set("__init__.py", "env.py", "script.py.mako", "migration_lock.toml")
	createTable = regexp.MustCompile(`(?i)\bcreate\s+(?:[a-z]+\s+){0,2}table\b`)
	kinds       = []string{"schema", "migration", "sql"}
)

func set(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it] = true
	}
	return m
}

// Scan classifies the files list returns, schema files first, then
// migrations, then .sql files that create a table, each kind in path order.
// The schema comes first because, when there is one, it is the current
// truth, and nobody has to rebuild a table from every migration. readAll is
// called once, with every .sql file that is not already a schema or a
// migration, and not at all when there is none.
func Scan(list func() ([]string, error), readAll func([]string) (map[string][]byte, error)) ([]File, error) {
	paths, err := list()
	if err != nil {
		return nil, err
	}
	byKind := map[string][]string{}
	var loose []string // .sql files whose content decides
	for _, p := range paths {
		lower := strings.ToLower(p)
		dirs := strings.Split(path.Dir(lower), "/")
		base := path.Base(lower)
		if skipped(dirs, base) {
			continue
		}
		switch {
		case schemaNames[base] || path.Ext(base) == ".prisma" || hasSuffix(base, schemaSuffixes):
			byKind["schema"] = append(byKind["schema"], p)
		case migrationDir(dirs):
			byKind["migration"] = append(byKind["migration"], p)
		case path.Ext(base) == ".sql":
			loose = append(loose, p)
		}
	}
	if len(loose) > 0 {
		sort.Strings(loose)
		contents, err := readAll(loose)
		if err != nil {
			return nil, err
		}
		for _, p := range loose {
			if createTable.Match(contents[p]) {
				byKind["sql"] = append(byKind["sql"], p)
			}
		}
	}
	var out []File
	for _, k := range kinds {
		sort.Strings(byKind[k])
		for _, p := range byKind[k] {
			out = append(out, File{Kind: k, Path: p})
		}
	}
	return out, nil
}

// skipped reports whether a file is in a directory that never holds the
// repository's own schema — someone else's code, tests, build output, or a
// hidden directory such as .venv or .tox — or is a file that never declares
// one.
func skipped(dirs []string, base string) bool {
	for _, d := range dirs {
		if skippedDirs[d] || (d != "." && strings.HasPrefix(d, ".")) {
			return true
		}
	}
	if noise[base] || strings.HasPrefix(base, ".") {
		return true
	}
	ext := path.Ext(base)
	return ext == ".md" || ext == ".txt"
}

// migrationDir reports whether a file sits in a directory a migration tool
// keeps its steps in. Liquibase's is db/changelog, which needs its parent:
// a changelog directory alone is as likely to hold release notes.
func migrationDir(dirs []string) bool {
	for i, d := range dirs {
		if migrationDirs[d] || strings.HasSuffix(d, migrationSuffix) {
			return true
		}
		if d == "changelog" && i > 0 && dirs[i-1] == "db" {
			return true
		}
	}
	return false
}

func hasSuffix(s string, suffixes []string) bool {
	for _, suf := range suffixes {
		if strings.HasSuffix(s, suf) {
			return true
		}
	}
	return false
}
