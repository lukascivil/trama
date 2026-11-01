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

var (
	schemaNames   = set("schema.prisma", "schema.rb", "structure.sql", "schema.sql")
	migrationDirs = set("migrations", "migration", "migrate", "alembic", "flyway", "liquibase")
	skippedDirs   = set("node_modules", "vendor", "dist", "build", "target", "testdata", "fixtures", "test", "tests", "spec", "__tests__")
	noise         = set("__init__.py")
	createTable   = regexp.MustCompile(`(?i)\bcreate\s+(?:[a-z]+\s+){0,2}table\b`)
	kinds         = []string{"schema", "migration", "sql"}
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
// truth, and nobody has to rebuild a table from every migration. read is
// only called for a .sql file that is not already a schema or a migration.
func Scan(list func() ([]string, error), read func(string) ([]byte, error)) ([]File, error) {
	paths, err := list()
	if err != nil {
		return nil, err
	}
	byKind := map[string][]string{}
	for _, p := range paths {
		dirs := strings.Split(path.Dir(p), "/")
		base := path.Base(p)
		if skipped(dirs, base) {
			continue
		}
		switch {
		case schemaNames[base] || path.Ext(base) == ".prisma":
			byKind["schema"] = append(byKind["schema"], p)
		case anyIn(dirs, migrationDirs):
			byKind["migration"] = append(byKind["migration"], p)
		case path.Ext(base) == ".sql":
			data, err := read(p)
			if err != nil {
				return nil, err
			}
			if createTable.Match(data) {
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
// repository's own schema, or is a file that never declares one.
func skipped(dirs []string, base string) bool {
	if anyIn(dirs, skippedDirs) || noise[base] || strings.HasPrefix(base, ".") {
		return true
	}
	ext := path.Ext(base)
	return ext == ".md" || ext == ".txt"
}

func anyIn(items []string, m map[string]bool) bool {
	for _, it := range items {
		if m[it] {
			return true
		}
	}
	return false
}
