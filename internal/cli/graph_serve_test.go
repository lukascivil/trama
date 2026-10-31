package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lukascivil/trama/internal/runner"
)

// syncBuffer is an output that the command and the test use at once.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

var servingLine = regexp.MustCompile(`serving (http://127\.0\.0\.1:\d+/) \(Ctrl\+C to stop\)`)

// serve runs trama with args until the test cancels it. It returns the URL
// the command printed, the cancel function and the command's result.
func serve(t *testing.T, args ...string) (string, context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	out := &syncBuffer{}
	cmd := newRootCmd()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs(args)
	done := make(chan error, 1)
	go func() { done <- cmd.ExecuteContext(ctx) }()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if m := servingLine.FindStringSubmatch(out.String()); m != nil {
			return m[1], cancel, done
		}
		select {
		case err := <-done:
			t.Fatalf("trama stopped before serving: %v\n%s", err, out)
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatalf("no serving line in:\n%s", out)
	return "", nil, nil
}

func noBrowser(t *testing.T) *[]string {
	t.Helper()
	var opened []string
	old := openURL
	openURL = func(url string) error { opened = append(opened, url); return nil }
	t.Cleanup(func() { openURL = old })
	return &opened
}

func fetchModel(t *testing.T, url string) map[string]any {
	t.Helper()
	res, err := http.Get(url + "api/workspace")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var m map[string]any
	if res.StatusCode != 200 || json.NewDecoder(res.Body).Decode(&m) != nil {
		t.Fatalf("GET api/workspace = %d", res.StatusCode)
	}
	return m
}

func stopped(t *testing.T, cancel context.CancelFunc, done <-chan error) error {
	t.Helper()
	cancel()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("trama did not stop after the context ended")
		return nil
	}
}

func TestGraphServeServesAndStops(t *testing.T) {
	noBrowser(t)
	ws := catalogWorkspace(t, cliAPI, cliWeb)
	url, cancel, done := serve(t, "graph", "--serve", "--no-open", "--port", "0", "--dir", ws)
	if m := fetchModel(t, url); m["workspace"] != "demo" {
		t.Fatalf("workspace = %v, want demo", m["workspace"])
	}
	if err := stopped(t, cancel, done); !errors.Is(err, runner.ErrInterrupted) {
		t.Fatalf("err = %v, want ErrInterrupted", err)
	}
	if _, err := os.Stat(filepath.Join(ws, "docs", "graph.md")); !os.IsNotExist(err) {
		t.Fatalf("--serve wrote docs/graph.md (%v)", err)
	}
}

func TestGraphServeWithValidationErrors(t *testing.T) {
	noBrowser(t)
	ws := catalogWorkspace(t, cliAPI, strings.Replace(cliWeb, `"app.ts:1"`, `"app.ts:99"`, 1))
	url, cancel, done := serve(t, "graph", "--serve", "--no-open", "--port", "0", "--dir", ws)
	warnings, _ := fetchModel(t, url)["warnings"].([]any)
	if len(warnings) == 0 || !strings.Contains(strings.Join(func() []string {
		var s []string
		for _, w := range warnings {
			s = append(s, w.(string))
		}
		return s
	}(), "\n"), "line 99") {
		t.Fatalf("warnings = %v, want the broken evidence", warnings)
	}
	stopped(t, cancel, done)
}

func TestGraphServeBusyPort(t *testing.T) {
	noBrowser(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	ws := catalogWorkspace(t, cliAPI, cliWeb)
	_, err = run(t, "graph", "--serve", "--no-open", "--port", strconv.Itoa(port), "--dir", ws)
	if err == nil || !strings.Contains(err.Error(), "is in use") {
		t.Fatalf("err = %v, want a busy port error", err)
	}
}

func TestGraphServeDefaultPortFallback(t *testing.T) {
	noBrowser(t)
	ln, err := net.Listen("tcp", "127.0.0.1:7300")
	if err != nil {
		t.Skipf("cannot hold port 7300 for the test: %v", err)
	}
	defer ln.Close()
	ws := catalogWorkspace(t, cliAPI, cliWeb)
	url, cancel, done := serve(t, "graph", "--serve", "--no-open", "--dir", ws)
	if strings.Contains(url, ":7300/") {
		t.Fatalf("url = %s, want another port while 7300 is busy", url)
	}
	fetchModel(t, url)
	stopped(t, cancel, done)
}

func TestGraphServeOpensBrowser(t *testing.T) {
	opened := noBrowser(t)
	ws := catalogWorkspace(t, cliAPI, cliWeb)
	url, cancel, done := serve(t, "graph", "--serve", "--port", "0", "--dir", ws)
	stopped(t, cancel, done)
	if len(*opened) != 1 || (*opened)[0] != url {
		t.Fatalf("opened = %v, want [%s]", *opened, url)
	}
	*opened = nil
	_, cancel, done = serve(t, "graph", "--serve", "--no-open", "--port", "0", "--dir", ws)
	stopped(t, cancel, done)
	if len(*opened) != 0 {
		t.Fatalf("opened = %v with --no-open", *opened)
	}
}

func TestGraphPortNeedsServe(t *testing.T) {
	ws := catalogWorkspace(t, cliAPI, cliWeb)
	for _, args := range [][]string{{"--port", "8000"}, {"--no-open"}} {
		_, err := run(t, append(append([]string{"graph"}, args...), "--dir", ws)...)
		if err == nil || err.Error() != "--port and --no-open need --serve" {
			t.Fatalf("graph %v: err = %v", args, err)
		}
	}
}
