package changes_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/lukascivil/trama/internal/changes"
)

// TestRecordReferenceExampleParses keeps the example in the change skill's
// record reference valid: it is what agents copy.
func TestRecordReferenceExampleParses(t *testing.T) {
	doc, err := os.ReadFile("../../skills/change/record-reference.md")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile("(?s)````markdown\n(.*?)````").FindSubmatch(doc)
	if m == nil {
		t.Fatal("no ````markdown example block")
	}
	r, err := changes.Parse(recordID+".md", m[1])
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Steps) != 3 || r.Steps[2].State != "pending" {
		t.Fatalf("example = %+v", r)
	}
}
