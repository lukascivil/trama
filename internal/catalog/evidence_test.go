package catalog_test

import (
	"testing"

	"github.com/lukascivil/trama/internal/catalog"
)

func TestParseEvidence(t *testing.T) {
	ok := map[string]catalog.Evidence{
		"src/api/hello.ts:5":        {Path: "src/api/hello.ts", Start: 5, End: 5},
		"internal/api/api.go:40-44": {Path: "internal/api/api.go", Start: 40, End: 44},
		"workspace.yaml":            {Declared: true},
	}
	for in, want := range ok {
		got, err := catalog.ParseEvidence(in)
		if err != nil || got != want {
			t.Errorf("ParseEvidence(%q) = %+v, %v; want %+v", in, got, err, want)
		}
	}
	for _, in := range []string{"src/a.go", "src/a.go:0", "src/a.go:9-3", "/etc/passwd:1", "../x.go:1", "a/../../x.go:1", "src/a.go:x", ""} {
		if _, err := catalog.ParseEvidence(in); err == nil {
			t.Errorf("ParseEvidence(%q) accepted", in)
		}
	}
}
