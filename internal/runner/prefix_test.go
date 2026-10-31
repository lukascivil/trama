package runner_test

import (
	"bytes"
	"sync"
	"testing"

	"github.com/lukascivil/trama/internal/runner"
)

func TestPrefixWriter(t *testing.T) {
	var out bytes.Buffer
	var mu sync.Mutex
	w := runner.NewPrefixWriter(&out, "web", &mu)
	w.Write([]byte("a\nb"))
	w.Write([]byte("c\n"))
	if got := out.String(); got != "web | a\nweb | bc\n" {
		t.Fatalf("got %q", got)
	}
	w.Write([]byte("x"))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "web | a\nweb | bc\nweb | x\n" {
		t.Fatalf("after Close got %q", got)
	}
}
