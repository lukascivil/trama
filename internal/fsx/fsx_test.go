package fsx_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lukascivil/trama/internal/fsx"
)

func TestReplaceFileDoesNotFollowSymlink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "file.md")
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	if err := fsx.ReplaceFile(path, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(outside); string(b) != "keep" {
		t.Fatalf("outside file overwritten: %q", b)
	}
	fi, err := os.Lstat(path)
	if err != nil || !fi.Mode().IsRegular() || fi.Mode().Perm() != 0o644 {
		t.Fatalf("path = %v, %v", fi, err)
	}
	if b, _ := os.ReadFile(path); string(b) != "new" {
		t.Fatalf("content = %q", b)
	}
}

func TestReplaceFileCreates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.txt")
	if err := fsx.ReplaceFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != "data" {
		t.Fatalf("content = %q", b)
	}
}
