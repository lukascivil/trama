package status_test

import (
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/status"
	"github.com/lukascivil/trama/internal/testutil"
)

func TestStatusReportsOriginMismatch(t *testing.T) {
	f := newFixture(t, false)
	f.writeCard(t, "abc1234")
	other := testutil.NewRemote(t, "main")
	if err := gitx.Clone(other.URL, "main", f.ws.RepoDir("api")); err != nil {
		t.Fatal(err)
	}
	s := only(t, f.ws)
	if !strings.Contains(s.Err, "workspace.yaml says") || s.Card != status.CardUnknown {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusCardStaleForSymbolicRef(t *testing.T) {
	for _, ref := range []string{"HEAD", "main", "origin/main"} {
		f := newFixture(t, true)
		f.writeCard(t, ref)
		if s := only(t, f.ws); s.Card != status.CardStale {
			t.Errorf("mapped_at_commit %q: card = %q, want stale", ref, s.Card)
		}
	}
}
