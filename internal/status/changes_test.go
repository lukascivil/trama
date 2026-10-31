package status_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lukascivil/trama/internal/status"
)

const openRecord = `---
id: 2026-11-03-rename-message
title: Rename message
status: open
steps:
  - {order: 1, repo: api, action: a, pr: "https://x/pull/1", state: merged}
  - {order: 2, repo: web, action: b, pr: "https://x/pull/2", state: open}
  - {order: 3, repo: api, action: c, state: pending}
---
`

const doneRecord = `---
id: 2026-11-01-done-one
title: Done
status: done
steps:
  - {order: 1, repo: api, action: a, pr: "https://x/pull/3", state: merged}
---
`

func TestStatusListsOpenChanges(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "changes"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"2026-11-03-rename-message.md": openRecord, "2026-11-01-done-one.md": doneRecord} {
		if err := os.WriteFile(filepath.Join(root, "changes", name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := status.OpenChanges(root)
	want := []string{"change 2026-11-03-rename-message: 2 step(s) not merged"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("OpenChanges = %q", got)
	}
}
