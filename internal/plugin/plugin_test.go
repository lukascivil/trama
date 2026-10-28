// Package plugin_test checks the Claude Code plugin files at the repo root:
// the manifests and the bin/trama shim.
package plugin_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const repoRoot = "../.."

var shim = filepath.Join(repoRoot, "bin", "trama")

type manifest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type marketplace struct {
	Name    string `json:"name"`
	Plugins []struct {
		Name   string `json:"name"`
		Source string `json:"source"`
	} `json:"plugins"`
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

func pluginVersion(t *testing.T) string {
	t.Helper()
	var m manifest
	readJSON(t, filepath.Join(repoRoot, ".claude-plugin", "plugin.json"), &m)
	return m.Version
}

// runShim runs bin/trama through sh with the given environment additions.
func runShim(t *testing.T, env []string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command("sh", append([]string{shim}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err := cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		code = exitErr.ExitCode()
	default:
		t.Fatal(err)
	}
	return out.String(), errOut.String(), code
}

func TestManifestsAgree(t *testing.T) {
	var m manifest
	readJSON(t, filepath.Join(repoRoot, ".claude-plugin", "plugin.json"), &m)
	if m.Name != "trama" {
		t.Fatalf("plugin name = %q", m.Name)
	}
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(m.Version) {
		t.Fatalf("plugin version = %q, want x.y.z", m.Version)
	}
	var mp marketplace
	readJSON(t, filepath.Join(repoRoot, ".claude-plugin", "marketplace.json"), &mp)
	if mp.Name != "trama" || len(mp.Plugins) != 1 || mp.Plugins[0].Name != "trama" || mp.Plugins[0].Source != "./" {
		t.Fatalf("marketplace = %+v", mp)
	}
}

func TestShimUsesTramaBin(t *testing.T) {
	out, errOut, code := runShim(t, []string{"TRAMA_BIN=/bin/echo"}, "a", "b")
	if code != 0 || out != "a b\n" {
		t.Fatalf("code %d, stdout %q, stderr %q", code, out, errOut)
	}
}

func TestShimRunsPluginVersionThroughMise(t *testing.T) {
	fake := t.TempDir()
	if err := os.WriteFile(filepath.Join(fake, "mise"), []byte("#!/bin/sh\necho \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, errOut, code := runShim(t, []string{"PATH=" + fake + ":/usr/bin:/bin", "TRAMA_BIN=", "TRAMA_SHIM_ACTIVE="}, "status")
	want := "exec github:lukascivil/trama@" + pluginVersion(t) + " -- trama status\n"
	if code != 0 || out != want {
		t.Fatalf("code %d, stdout %q, want %q, stderr %q", code, out, want, errOut)
	}
}

func TestShimStopsRecursion(t *testing.T) {
	_, errOut, code := runShim(t, []string{"TRAMA_BIN=", "TRAMA_SHIM_ACTIVE=1"}, "version")
	if code != 127 || !strings.Contains(errOut, "mise did not provide") {
		t.Fatalf("code %d, stderr %q", code, errOut)
	}
}

func TestShimRequiresMise(t *testing.T) {
	for _, dir := range []string{"/usr/bin", "/bin"} {
		if _, err := os.Stat(filepath.Join(dir, "mise")); err == nil {
			t.Skip("mise is installed in " + dir)
		}
	}
	_, errOut, code := runShim(t, []string{"PATH=/usr/bin:/bin", "TRAMA_BIN=", "TRAMA_SHIM_ACTIVE="}, "version")
	if code != 127 || !strings.Contains(errOut, "mise is required") {
		t.Fatalf("code %d, stderr %q", code, errOut)
	}
}

func TestShimRejectsMissingVersion(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"bin", ".claude-plugin"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	src, err := os.ReadFile(shim)
	if err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(root, "bin", "trama")
	if err := os.WriteFile(copied, src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".claude-plugin", "plugin.json"), []byte(`{"name":"trama"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	fake := t.TempDir()
	marker := filepath.Join(t.TempDir(), "mise-called")
	if err := os.WriteFile(filepath.Join(fake, "mise"), []byte("#!/bin/sh\ntouch '"+marker+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", copied, "version")
	cmd.Env = append(os.Environ(), "PATH="+fake+":/usr/bin:/bin", "TRAMA_BIN=", "TRAMA_SHIM_ACTIVE=")
	var errOut bytes.Buffer
	cmd.Stderr = &errOut
	err = cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 || !strings.Contains(errOut.String(), "could not read the version") {
		t.Fatalf("err = %v, stderr %q", err, errOut.String())
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatal("mise was called without a version")
	}
}
