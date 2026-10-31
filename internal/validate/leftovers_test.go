package validate_test

import (
	"strings"
	"testing"
)

func TestValidateYmlCard(t *testing.T) {
	f := demo(t)
	f.write("catalog/trama-demo-api.yml", demoAPI)
	if fs := f.findings(); !hasError(fs, "", "use the .yaml extension") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateImpossibleDate(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, "mapped_at: 2026-10-28", "mapped_at: 2026-13-45", 1))
	if fs := f.findings(); !hasError(fs, "mapped_at", "") {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateNameMessage(t *testing.T) {
	f := demo(t)
	f.card("trama-demo-api", strings.Replace(demoAPI, "name: trama-demo-api", "name: trama-demo-web", 1))
	want := `"trama-demo-web" does not match the file name catalog/trama-demo-api.yaml`
	if fs := f.findings(); !hasError(fs, "name", want) {
		t.Fatalf("findings = %v", fs)
	}
}

func TestValidateTools(t *testing.T) {
	cases := map[string]string{
		`"npm:pkg": "1"`:   `tools.npm:pkg`,
		`node: "24 && rm"`: `tools.node`,
		`"-x": "1"`:        `tools.-x`,
	}
	for entry, where := range cases {
		f := demo(t)
		f.card("trama-demo-api", strings.Replace(demoAPI, `go: "1.25"`, entry, 1))
		if fs := f.findings(); !hasError(fs, where, "") {
			t.Errorf("%s accepted: %v", entry, fs)
		}
	}
}
