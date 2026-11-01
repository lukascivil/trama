package view

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

func TestAssetsEmbedded(t *testing.T) {
	for _, name := range []string{"assets/index.html", "assets/app.js", "assets/app.css", "assets/icon.svg"} {
		if _, err := fs.Stat(assets, name); err != nil {
			t.Errorf("%s is not embedded: %v", name, err)
		}
	}
}

// app.js is built from web/ and carries React and React Flow, which name a
// few addresses: XML namespaces, error-page links and the React Flow
// attribution link. None of them is ever requested; the CSP would stop it.
var allowedAddress = regexp.MustCompile(`^https?://(www\.w3\.org/|react\.dev/errors/|reactflow\.dev|\$\{)`)

func TestAssetsStaySelfContained(t *testing.T) {
	address := regexp.MustCompile(`https?://[^"'` + "`" + ` )\\]*`)
	for _, name := range []string{"assets/index.html", "assets/app.css", "assets/app.js"} {
		data, err := fs.ReadFile(assets, name)
		if err != nil {
			t.Fatal(err)
		}
		// The page links to an https evidence URL only after checking the prefix.
		text := strings.NewReplacer(`startsWith("https://")`, "", "startsWith(`https://`)", "").Replace(string(data))
		for _, url := range address.FindAllString(text, -1) {
			if name == "assets/app.js" && allowedAddress.MatchString(url) {
				continue
			}
			t.Errorf("%s names an address outside the server: %s", name, url)
		}
	}
}

// The page may ask its own server for the model and nothing else.
func TestScriptRequestsOnlyTheWorkspace(t *testing.T) {
	data, err := fs.ReadFile(assets, "assets/app.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(data)
	if n := strings.Count(js, "fetch("); n != 1 || !strings.Contains(js, "fetch(`/api/workspace`") {
		t.Errorf("app.js must call fetch once, for /api/workspace; it calls it %d times", n)
	}
	for _, api := range []string{"XMLHttpRequest", "sendBeacon", "WebSocket", "EventSource", "importScripts"} {
		if strings.Contains(js, api) {
			t.Errorf("app.js uses %s", api)
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
