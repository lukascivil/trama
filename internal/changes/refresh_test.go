package changes_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/changes"
)

func parsed(t *testing.T, content string) *changes.Record {
	t.Helper()
	r, err := changes.Parse(recordID+".md", []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func states(m map[string]string) changes.PRState {
	return func(url string) (string, error) {
		if s, ok := m[url]; ok {
			return s, nil
		}
		return "", errors.New("gh: not found")
	}
}

const pr1, pr2 = "https://github.com/lukascivil/trama-demo-api/pull/1", "https://github.com/lukascivil/trama-demo-web/pull/1"

func TestRefreshMarksMerged(t *testing.T) {
	r := parsed(t, record)
	changed, errs := changes.Refresh(r, states(map[string]string{pr1: "MERGED", pr2: "MERGED"}))
	if !changed || len(errs) != 0 || r.Steps[0].State != "merged" || r.Steps[1].State != "merged" || r.Steps[2].State != "pending" || r.Status != "open" {
		t.Fatalf("changed %v errs %v record %+v", changed, errs, r)
	}
}

func TestRefreshDone(t *testing.T) {
	noPending := strings.Replace(record, "  - order: 3\n    repo: trama-demo-api\n    action: Remove message from GET /hello\n    state: pending\n", "", 1)
	r := parsed(t, noPending)
	if _, errs := changes.Refresh(r, states(map[string]string{pr1: "MERGED", pr2: "MERGED"})); len(errs) != 0 || r.Status != "done" {
		t.Fatalf("errs %v status %q", errs, r.Status)
	}
}

func TestRefreshUnchanged(t *testing.T) {
	r := parsed(t, record)
	if changed, errs := changes.Refresh(r, states(map[string]string{pr1: "OPEN", pr2: "OPEN"})); changed || len(errs) != 0 {
		t.Fatalf("changed %v errs %v", changed, errs)
	}
}

func TestRefreshKeepsStateOnError(t *testing.T) {
	r := parsed(t, record)
	changed, errs := changes.Refresh(r, states(map[string]string{pr1: "CLOSED"}))
	if !changed || r.Steps[0].State != "closed" || r.Steps[1].State != "open" || len(errs) != 1 ||
		!strings.Contains(errs[0].Error(), "trama-demo-web") || !strings.Contains(errs[0].Error(), pr2) {
		t.Fatalf("changed %v errs %v steps %+v", changed, errs, r.Steps)
	}
}
