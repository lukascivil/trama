package validate_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/lukascivil/trama/internal/catalog"
	"go.yaml.in/yaml/v3"
)

var yamlBlock = regexp.MustCompile("(?s)```yaml\n(.*?)```")

// TestCardReferenceExamplesValidate keeps the examples in the map skill's
// card reference valid: they are what agents copy.
func TestCardReferenceExamplesValidate(t *testing.T) {
	doc, err := os.ReadFile("../../skills/map/card-reference.md")
	if err != nil {
		t.Fatal(err)
	}
	blocks := yamlBlock.FindAllSubmatch(doc, -1)
	if len(blocks) != 2 {
		t.Fatalf("want 2 example cards, found %d", len(blocks))
	}
	f := fixture(t, demoWorkspace)
	for _, b := range blocks {
		var head struct{ Name string }
		if err := yaml.Unmarshal(b[1], &head); err != nil {
			t.Fatal(err)
		}
		f.card(head.Name, string(b[1]))
		c, err := catalog.LoadStrict(f.root, head.Name)
		if err != nil {
			t.Fatal(err)
		}
		for path, lines := range citedLines(t, c) {
			f.file(head.Name, path, lines)
		}
	}
	if fs := f.findings(); len(fs) != 0 {
		t.Fatalf("examples do not validate: %v", fs)
	}
}

// citedLines maps every file a card cites to the last line it cites.
func citedLines(t *testing.T, c *catalog.Card) map[string]int {
	t.Helper()
	var all []string
	for _, p := range c.Identity.Ports {
		all = append(all, p.Evidence...)
	}
	for _, tb := range c.Identity.Tables {
		all = append(all, tb.Evidence...)
	}
	for _, p := range c.Provides {
		all = append(all, p.Evidence...)
	}
	for _, cs := range c.Consumes {
		all = append(all, cs.Evidence...)
	}
	for _, cf := range c.Config {
		all = append(all, cf.Evidence...)
	}
	out := map[string]int{}
	for _, raw := range all {
		ev, err := catalog.ParseEvidence(raw)
		if err != nil {
			t.Fatal(err)
		}
		if !ev.Declared && ev.End > out[ev.Path] {
			out[ev.Path] = ev.End
		}
	}
	return out
}
