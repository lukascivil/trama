package cli

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpRefusesBusyPort(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port
	ws := runWorkspace(t, fmt.Sprintf("run:\n  dev: { cmd: \"touch ran\", port: %d }\n", port))
	_, err = run(t, "up", "--dir", ws)
	want := fmt.Sprintf("cannot start: api: port %d is already in use", port)
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
	if _, statErr := os.Stat(filepath.Join(ws, "repos", "api", "ran")); statErr == nil {
		t.Fatal("the dev command ran although its port was busy")
	}
}

func TestUpRunsAndReportsExit(t *testing.T) {
	ws := runWorkspace(t, "run:\n  dev: { cmd: \"echo hi; exit 5\" }\n")
	out, err := run(t, "up", "--dir", ws)
	if err == nil || !contains(out, "api | hi", "api exited with code 5") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func contains(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
