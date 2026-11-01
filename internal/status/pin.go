package status

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// tramaPin matches a mise.toml line that pins the trama binary, in either
// quoting style.
var tramaPin = regexp.MustCompile(`^["']github:lukascivil/trama["']\s*=`)

// StalePin reports a workspace mise.toml that still pins the trama
// binary. The plugin runs the binary whose version matches the plugin
// itself, so a version pinned here never applies to a trama command run
// through Claude, and can only disagree with it.
func StalePin(root string) []string {
	data, err := os.ReadFile(filepath.Join(root, "mise.toml"))
	if err != nil {
		return nil
	}
	for _, line := range strings.Split(string(data), "\n") {
		if tramaPin.MatchString(strings.TrimSpace(line)) {
			return []string{`mise.toml pins the trama binary: the plugin already runs the version that matches it, so that line can go`}
		}
	}
	return nil
}
