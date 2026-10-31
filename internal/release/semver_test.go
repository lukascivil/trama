package release

import "testing"

func TestParseSemver(t *testing.T) {
	for _, tag := range []string{"v1.2.3", "1.2.3", "1.0.0-rc.1", "1.0.0+build.5", "v0.10.0-alpha.beta+exp.sha.5114f85"} {
		if _, ok := parse(tag); !ok {
			t.Errorf("parse(%q) refused a semver tag", tag)
		}
	}
	for _, tag := range []string{"start", "v1.2", "release-2024", "01.2.3", "1.02.3", "1.2.3-", "1.2.3-01", "v1.2.3.4", "V1.2.3", "1.2.3+", ""} {
		if _, ok := parse(tag); ok {
			t.Errorf("parse(%q) accepted a tag that is not semver", tag)
		}
	}
}

func TestCompareSemver(t *testing.T) {
	ordered := []string{
		"1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta",
		"1.0.0-beta.2", "1.0.0-beta.11", "1.0.0-rc.1", "1.0.0",
	}
	for i := 0; i+1 < len(ordered); i++ {
		a, _ := parse(ordered[i])
		b, _ := parse(ordered[i+1])
		if compare(a, b) >= 0 || compare(b, a) <= 0 {
			t.Errorf("%s should sort before %s", ordered[i], ordered[i+1])
		}
	}
	small, _ := parse("0.9.0")
	big, _ := parse("0.10.0")
	if compare(small, big) >= 0 {
		t.Error("0.9.0 should sort before 0.10.0")
	}
	plain, _ := parse("1.0.0")
	build, _ := parse("v1.0.0+build.7")
	if compare(plain, build) != 0 {
		t.Error("build metadata must not change the order")
	}
}
