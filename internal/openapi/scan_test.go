package openapi_test

import (
	"slices"
	"testing"

	"github.com/lukascivil/trama/internal/openapi"
)

const tinySpec = "openapi: 3.0.0\npaths:\n  /a:\n    get: {}\n"

func scan(t *testing.T, files map[string]string) ([]string, [][]string) {
	t.Helper()
	var reads [][]string
	list := func() ([]string, error) {
		var out []string
		for p := range files {
			out = append(out, p)
		}
		return out, nil
	}
	readAll := func(paths []string) (map[string][]byte, error) {
		reads = append(reads, paths)
		out := map[string][]byte{}
		for _, p := range paths {
			out[p] = []byte(files[p])
		}
		return out, nil
	}
	specs, err := openapi.Scan(list, readAll)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, s := range specs {
		got = append(got, s.Path)
	}
	return got, reads
}

func TestScanFindsSpecsByContent(t *testing.T) {
	got, reads := scan(t, map[string]string{
		"docs/api.yaml":                tinySpec,
		"spec/v1.json":                 `{"openapi": "3.1.0", "paths": {}}`,
		"openapi.yml":                  tinySpec,
		"chart/values.yaml":            "replicas: 2\n",
		"swagger.yaml":                 "swagger: \"2.0\"\npaths: {}\n",
		"broken.yaml":                  "openapi: 3.0.0\npaths: [\n",
		"node_modules/x/openapi.yaml":  tinySpec,
		"vendor/y/openapi.yaml":        tinySpec,
		"test/openapi.yaml":            tinySpec,
		"internal/testdata/api.yaml":   tinySpec,
		".github/openapi.yaml":         tinySpec,
		"web/package-lock.json":        `{"openapi": 1}`,
		"README.md":                    "openapi: 3.0.0\n",
		"internal/api/openapi.gen.go":  "openapi: 3.0.0\n",
		"services/billing/openapi.YML": tinySpec,
	})
	want := []string{"docs/api.yaml", "openapi.yml", "services/billing/openapi.YML", "spec/v1.json"}
	if !slices.Equal(got, want) {
		t.Fatalf("specs = %v, want %v", got, want)
	}
	if len(reads) != 1 {
		t.Fatalf("readAll called %d times, want once", len(reads))
	}
	if slices.Contains(reads[0], "web/package-lock.json") || slices.Contains(reads[0], "test/openapi.yaml") {
		t.Fatalf("read a skipped file: %v", reads[0])
	}
}

func TestScanReadsNothingWithoutCandidates(t *testing.T) {
	got, reads := scan(t, map[string]string{"main.go": "package main\n", "README.md": "# x\n"})
	if len(got) != 0 || len(reads) != 0 {
		t.Fatalf("specs %v, reads %v", got, reads)
	}
}

func TestIsDocument(t *testing.T) {
	for p, want := range map[string]bool{"api.yaml": true, "a/API.JSON": true, "x.yml": true, "api.proto": false, "schema.graphql": false} {
		if openapi.IsDocument(p) != want {
			t.Errorf("IsDocument(%q) != %v", p, want)
		}
	}
}
