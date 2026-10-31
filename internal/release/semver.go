package release

import (
	"strconv"
	"strings"
)

// version is a parsed semantic version (semver 2.0); build metadata is
// dropped because it does not change the order.
type version struct {
	core [3]uint64
	pre  []string // pre-release identifiers; empty for a release
}

// parse reads a tag such as "v1.2.3", "1.2.3" or "1.0.0-rc.1+build.5".
func parse(tag string) (version, bool) {
	s := strings.TrimPrefix(tag, "v")
	if i := strings.IndexByte(s, '+'); i >= 0 {
		if !validIdents(s[i+1:], false) {
			return version{}, false
		}
		s = s[:i]
	}
	var v version
	if i := strings.IndexByte(s, '-'); i >= 0 {
		if !validIdents(s[i+1:], true) {
			return version{}, false
		}
		v.pre = strings.Split(s[i+1:], ".")
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return version{}, false
	}
	for i, p := range parts {
		if !numeric(p) {
			return version{}, false
		}
		n, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return version{}, false
		}
		v.core[i] = n
	}
	return v, true
}

// numeric reports whether s is a number without leading zeros.
func numeric(s string) bool {
	if s == "" || (len(s) > 1 && s[0] == '0') {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// validIdents checks dot-separated identifiers of [0-9A-Za-z-]; numeric
// pre-release identifiers must not have leading zeros.
func validIdents(s string, pre bool) bool {
	if s == "" {
		return false
	}
	for _, id := range strings.Split(s, ".") {
		if id == "" {
			return false
		}
		digits := true
		for _, c := range id {
			switch {
			case c >= '0' && c <= '9':
			case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '-':
				digits = false
			default:
				return false
			}
		}
		if pre && digits && !numeric(id) {
			return false
		}
	}
	return true
}

// compare orders versions by semver precedence: -1, 0 or 1.
func compare(a, b version) int {
	for i := range a.core {
		if a.core[i] != b.core[i] {
			if a.core[i] < b.core[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a.pre) == 0 && len(b.pre) == 0:
		return 0
	case len(a.pre) == 0:
		return 1
	case len(b.pre) == 0:
		return -1
	}
	for i := 0; i < len(a.pre) && i < len(b.pre); i++ {
		if c := compareIdent(a.pre[i], b.pre[i]); c != 0 {
			return c
		}
	}
	switch {
	case len(a.pre) < len(b.pre):
		return -1
	case len(a.pre) > len(b.pre):
		return 1
	}
	return 0
}

// compareIdent compares pre-release identifiers: numbers numerically and
// below text, text in ASCII order.
func compareIdent(a, b string) int {
	an, aerr := strconv.ParseUint(a, 10, 64)
	bn, berr := strconv.ParseUint(b, 10, 64)
	switch {
	case aerr == nil && berr == nil:
		if an == bn {
			return 0
		}
		if an < bn {
			return -1
		}
		return 1
	case aerr == nil:
		return -1
	case berr == nil:
		return 1
	}
	return strings.Compare(a, b)
}
