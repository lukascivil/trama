package cli

import (
	"bytes"
	"testing"
)

func TestVersionCommandPrintsVersion(t *testing.T) {
	version = "1.2.3"
	t.Cleanup(func() { version = "dev" })
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "1.2.3\n" {
		t.Fatalf("got %q, want %q", got, "1.2.3\n")
	}
}

func TestVersionFlagPrintsVersion(t *testing.T) {
	version = "1.2.3"
	t.Cleanup(func() { version = "dev" })
	for _, arg := range []string{"--version", "-v"} {
		var out bytes.Buffer
		cmd := newRootCmd()
		cmd.SetOut(&out)
		cmd.SetArgs([]string{arg})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%s: %v", arg, err)
		}
		if got := out.String(); got != "1.2.3\n" {
			t.Fatalf("%s: got %q, want %q", arg, got, "1.2.3\n")
		}
	}
}
