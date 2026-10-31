package runner_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/lukascivil/trama/internal/runner"
)

func TestJobForReadsCard(t *testing.T) {
	ws, _ := newRepoWS(t, runCard, true)
	j, err := runner.JobFor(ws, "api", runner.Dev)
	if err != nil {
		t.Fatal(err)
	}
	want := runner.Job{Repo: "api", Kind: runner.Dev, Dir: ws.RepoDir("api"), Cmd: "npm run dev", Tools: map[string]string{"node": "24"}, Port: 5173}
	if !reflect.DeepEqual(j, want) {
		t.Fatalf("job = %+v\nwant  %+v", j, want)
	}
}

func TestJobForMissingCommand(t *testing.T) {
	ws, _ := newRepoWS(t, runCard, true)
	_, err := runner.JobFor(ws, "api", runner.Test)
	if !errors.Is(err, runner.ErrNoCommand) || err.Error() != "api has no run.test in its catalog card" {
		t.Fatalf("err = %v", err)
	}
}

func TestJobForMissingCard(t *testing.T) {
	ws, _ := newRepoWS(t, "", true)
	if _, err := runner.JobFor(ws, "api", runner.Dev); err == nil || err.Error() != `no catalog card for api: run "/trama:map api"` {
		t.Fatalf("err = %v", err)
	}
}

func TestJobForNotCloned(t *testing.T) {
	ws, _ := newRepoWS(t, runCard, false)
	if _, err := runner.JobFor(ws, "api", runner.Dev); err == nil || err.Error() != `repos/api is not cloned: run "trama sync api"` {
		t.Fatalf("err = %v", err)
	}
}

func TestJobForUnknownRepo(t *testing.T) {
	ws, _ := newRepoWS(t, runCard, true)
	if _, err := runner.JobFor(ws, "ghost", runner.Dev); err == nil || err.Error() != "repo ghost is not in workspace.yaml" {
		t.Fatalf("err = %v", err)
	}
}

func TestArgs(t *testing.T) {
	plain := runner.Job{Cmd: "npm test"}
	if got := plain.Args(); !reflect.DeepEqual(got, []string{"sh", "-c", "npm test"}) {
		t.Fatalf("Args = %q", got)
	}
	tooled := runner.Job{Cmd: "npm test", Tools: map[string]string{"node": "24", "go": "1.25"}}
	want := []string{"mise", "exec", "go@1.25", "node@24", "--", "sh", "-c", "npm test"}
	if got := tooled.Args(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Args = %q", got)
	}
}

func TestDescribe(t *testing.T) {
	j := runner.Job{Repo: "web", Cmd: "npm run dev"}
	if got := j.Describe(); got != "$ npm run dev  (in repos/web)" {
		t.Fatalf("Describe = %q", got)
	}
}
