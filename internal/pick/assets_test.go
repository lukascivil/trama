package pick

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

func TestAssetsEmbedded(t *testing.T) {
	for _, name := range []string{"assets/index.html", "assets/pick.js", "assets/pick.css"} {
		if _, err := fs.Stat(assets, name); err != nil {
			t.Errorf("%s is not embedded: %v", name, err)
		}
	}
}

// pick.js is built from web/ and carries React, which names a few addresses:
// XML namespaces and error-page links. None of them is ever requested; the CSP
// would stop it.
var allowedAddress = regexp.MustCompile(`^https?://(www\.w3\.org/|react\.dev/errors/)`)

func TestAssetsStaySelfContained(t *testing.T) {
	address := regexp.MustCompile(`https?://[^"'` + "`" + ` )\\]*`)
	for _, name := range []string{"assets/index.html", "assets/pick.css", "assets/pick.js"} {
		data, err := fs.ReadFile(assets, name)
		if err != nil {
			t.Fatal(err)
		}
		for _, url := range address.FindAllString(string(data), -1) {
			if name == "assets/pick.js" && allowedAddress.MatchString(url) {
				continue
			}
			t.Errorf("%s names an address outside the server: %s", name, url)
		}
	}
}

// The page may ask its own server for the list, to add and to cancel, and
// nothing else.
func TestScriptRequestsOnlyThePickerAPI(t *testing.T) {
	data, err := fs.ReadFile(assets, "assets/pick.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(data)
	seen := map[string]bool{}
	for _, path := range regexp.MustCompile(`/api/[A-Za-z0-9_/-]*`).FindAllString(js, -1) {
		seen[path] = true
		if path != "/api/repos" && path != "/api/add" && path != "/api/cancel" {
			t.Errorf("pick.js names the endpoint %s", path)
		}
	}
	for _, want := range []string{"/api/repos", "/api/add", "/api/cancel"} {
		if !seen[want] {
			t.Errorf("pick.js never names %s", want)
		}
	}
	for _, api := range []string{"XMLHttpRequest", "sendBeacon", "WebSocket", "EventSource", "importScripts"} {
		if strings.Contains(js, api) {
			t.Errorf("pick.js uses %s", api)
		}
	}
}

func TestIndexHasNoInlineCode(t *testing.T) {
	data, err := fs.ReadFile(assets, "assets/index.html")
	if err != nil {
		t.Fatal(err)
	}
	page := string(data)
	for _, bad := range []*regexp.Regexp{
		regexp.MustCompile(`<style`),
		regexp.MustCompile(`\sstyle=`),
		regexp.MustCompile(`<script(?:\s[^>]*)?>[^<]`),
		regexp.MustCompile(`<script>`),
		regexp.MustCompile(`\son[a-z]+=`),
	} {
		if bad.MatchString(page) {
			t.Errorf("index.html has inline code the CSP blocks: %s", bad)
		}
	}
}
