package runner_test

import (
	"net"
	"testing"

	"github.com/lukascivil/trama/internal/runner"
)

func TestPortFree(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if runner.PortFree(port) {
		t.Fatalf("port %d reported free while listening on 127.0.0.1", port)
	}
	l.Close()
	if !runner.PortFree(port) {
		t.Fatalf("port %d reported busy after close", port)
	}
}
