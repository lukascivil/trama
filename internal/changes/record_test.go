package changes_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/changes"
)

const recordID = "2026-11-03-rename-message"

const record = `---
id: 2026-11-03-rename-message
title: Rename message to greeting in GET /hello
status: open
steps:
  - order: 1
    repo: trama-demo-api
    action: Add greeting to GET /hello, keeping message
    branch: trama/rename-message
    pr: https://github.com/lukascivil/trama-demo-api/pull/1
    state: open
  - order: 2
    repo: trama-demo-web
    action: Read greeting instead of message
    branch: trama/rename-message
    pr: https://github.com/lukascivil/trama-demo-web/pull/1
    state: open
  - order: 3
    repo: trama-demo-api
    action: Remove message from GET /hello
    state: pending
---

## Why

The frontend should say "greeting".
`

func writeRecord(t *testing.T, root, id, content string) string {
	t.Helper()
	path := filepath.Join(root, "changes", id+".md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseRecord(t *testing.T) {
	r, err := changes.Parse(filepath.Join("changes", recordID+".md"), []byte(record))
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != recordID || r.Title != "Rename message to greeting in GET /hello" || r.Status != "open" || len(r.Steps) != 3 {
		t.Fatalf("record = %+v", r)
	}
	if r.Steps[2].State != "pending" || r.Steps[0].PR != "https://github.com/lukascivil/trama-demo-api/pull/1" {
		t.Fatalf("steps = %+v", r.Steps)
	}
}

func TestParseRejects(t *testing.T) {
	cases := map[string]string{
		"no front matter": "## Why\n",
		"unknown state":   strings.Replace(record, "state: pending", "state: doing", 1),
		"id differs":      strings.Replace(record, "id: "+recordID, "id: 2026-11-03-other", 1),
		"pending with pr": strings.Replace(record, "    state: pending", "    pr: https://x/pull/9\n    state: pending", 1),
		"open without pr": strings.Replace(record, "    pr: https://github.com/lukascivil/trama-demo-web/pull/1\n", "", 1),
		"unknown field":   strings.Replace(record, "status: open", "status: open\nstatsu: x", 1),
	}
	for label, content := range cases {
		path := filepath.Join("changes", recordID+".md")
		if _, err := changes.Parse(path, []byte(content)); err == nil || !strings.Contains(err.Error(), recordID+".md") {
			t.Errorf("%s: err = %v", label, err)
		}
	}
}

func TestSaveKeepsBody(t *testing.T) {
	root := t.TempDir()
	path := writeRecord(t, root, recordID, record)
	data, _ := os.ReadFile(path)
	r, err := changes.Parse(path, data)
	if err != nil {
		t.Fatal(err)
	}
	r.Steps[0].State = "merged"
	if err := r.Save(); err != nil {
		t.Fatal(err)
	}
	saved, _ := os.ReadFile(path)
	if !strings.HasSuffix(string(saved), "\n## Why\n\nThe frontend should say \"greeting\".\n") {
		t.Fatalf("body changed:\n%s", saved)
	}
	again, err := changes.Parse(path, saved)
	if err != nil || again.Steps[0].State != "merged" {
		t.Fatalf("reparse = %+v, %v", again, err)
	}
}

func TestLoadSkipsBroken(t *testing.T) {
	root := t.TempDir()
	writeRecord(t, root, recordID, record)
	writeRecord(t, root, "2026-11-04-second", strings.ReplaceAll(record, recordID, "2026-11-04-second"))
	writeRecord(t, root, "2026-11-05-broken", "no front matter\n")
	if err := os.WriteFile(filepath.Join(root, "changes", ".gitkeep"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	records, errs := changes.Load(root)
	if len(records) != 2 || len(errs) != 1 || !strings.Contains(errs[0].Error(), "2026-11-05-broken.md") {
		t.Fatalf("records %d, errs %v", len(records), errs)
	}
	if records[0].ID != recordID {
		t.Fatalf("not sorted: %s first", records[0].ID)
	}
}

func TestOpenSteps(t *testing.T) {
	r, _ := changes.Parse(recordID+".md", []byte(strings.Replace(record, "pull/1\n    state: open", "pull/1\n    state: merged", 1)))
	if got := r.OpenSteps(); len(got) != 2 {
		t.Fatalf("open steps = %+v", got)
	}
}
