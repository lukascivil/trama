package view

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

func TestAssetsEmbedded(t *testing.T) {
	for _, name := range []string{"assets/index.html", "assets/app.js", "assets/app.css", "assets/vendor/cytoscape.min.js", "assets/vendor/LICENSE.cytoscape"} {
		if _, err := fs.Stat(assets, name); err != nil {
			t.Errorf("%s is not embedded: %v", name, err)
		}
	}
}

func TestAssetsStaySelfContained(t *testing.T) {
	for _, name := range []string{"assets/index.html", "assets/app.js", "assets/app.css"} {
		data, err := fs.ReadFile(assets, name)
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ReplaceAll(string(data), `startsWith("https://")`, "")
		if strings.Contains(text, "http://") || strings.Contains(text, "https://") {
			t.Errorf("%s names an address outside the server", name)
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
	if !strings.Contains(page, `id="__________cytoscape_stylesheet"`) {
		t.Error("index.html must mark its stylesheet for Cytoscape, or Cytoscape injects a <style> the CSP blocks")
	}
}
