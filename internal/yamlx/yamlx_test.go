package yamlx_test

import (
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/yamlx"
)

func TestSetAfterTheRightKey(t *testing.T) {
	doc, err := yamlx.Parse([]byte("kind: http\nid: GET /a\nevidence: [x]\n"))
	if err != nil {
		t.Fatal(err)
	}
	yamlx.Set(doc.Content[0], "contract", yamlx.Scalar("api.yaml"), "id")
	out, _ := yamlx.Encode(doc)
	if want := "kind: http\nid: GET /a\ncontract: api.yaml\nevidence: [x]\n"; string(out) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSetAfterTheLastKey(t *testing.T) {
	doc, err := yamlx.Parse([]byte("kind: http\nid: GET /a\ncontract: api.yaml\n"))
	if err != nil {
		t.Fatal(err)
	}
	yamlx.Set(doc.Content[0], "request", yamlx.Scalar("x"), "contract", "id")
	out, _ := yamlx.Encode(doc)
	if want := "kind: http\nid: GET /a\ncontract: api.yaml\nrequest: x\n"; string(out) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSetFallsBackToTheNextKeyAndThenTheEnd(t *testing.T) {
	doc, err := yamlx.Parse([]byte("kind: http\nid: GET /a\nevidence: [x]\n"))
	if err != nil {
		t.Fatal(err)
	}
	m := doc.Content[0]
	yamlx.Set(m, "request", yamlx.Scalar("x"), "contract", "id")
	yamlx.Set(m, "note", yamlx.Scalar("y"), "missing")
	out, _ := yamlx.Encode(doc)
	if want := "kind: http\nid: GET /a\nrequest: x\nevidence: [x]\nnote: y\n"; string(out) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSetReplacesInPlaceAndKeepsComments(t *testing.T) {
	doc, err := yamlx.Parse([]byte("# card\nfrom: unknown # not yet\nid: GET /a\n"))
	if err != nil {
		t.Fatal(err)
	}
	yamlx.Set(doc.Content[0], "from", yamlx.Scalar("api"), "id")
	out, _ := yamlx.Encode(doc)
	if want := "# card\nfrom: api\nid: GET /a\n"; string(out) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
	if yamlx.Field(doc.Content[0], "from").Value != "api" {
		t.Fatal("Field does not see the new value")
	}
}

func TestRemove(t *testing.T) {
	doc, err := yamlx.Parse([]byte("a: 1\nb: 2\nc: 3\n"))
	if err != nil {
		t.Fatal(err)
	}
	yamlx.Remove(doc.Content[0], "b")
	yamlx.Remove(doc.Content[0], "missing")
	out, _ := yamlx.Encode(doc)
	if string(out) != "a: 1\nc: 3\n" {
		t.Fatalf("got:\n%s", out)
	}
	if yamlx.Field(doc.Content[0], "b") != nil {
		t.Fatal("b is still there")
	}
}

func TestList(t *testing.T) {
	doc, err := yamlx.Parse([]byte("a: 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	yamlx.Set(doc.Content[0], "candidates", yamlx.List([]string{"api", "legacy"}))
	out, _ := yamlx.Encode(doc)
	if string(out) != "a: 1\ncandidates: [api, legacy]\n" {
		t.Fatalf("got:\n%s", out)
	}
}

func TestParseRefusesWhatIsNotAMapping(t *testing.T) {
	for _, src := range []string{"", "- a\n- b\n", "just text\n"} {
		if _, err := yamlx.Parse([]byte(src)); err == nil || !strings.Contains(err.Error(), "not a YAML mapping") {
			t.Errorf("%q: err = %v", src, err)
		}
	}
	if _, err := yamlx.Parse([]byte("a: [\n")); err == nil {
		t.Error("broken YAML parsed")
	}
}
