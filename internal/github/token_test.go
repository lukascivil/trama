package github

import (
	"errors"
	"strings"
	"testing"
)

func fakeGh(answers map[string]string) Gh {
	return func(args ...string) (string, error) {
		if out, ok := answers[strings.Join(args, " ")]; ok {
			return out, nil
		}
		return "", errors.New("gh: not available")
	}
}

func env(vars map[string]string) func(string) string {
	return func(name string) string { return vars[name] }
}

func TestTokenOrder(t *testing.T) {
	gh := fakeGh(map[string]string{"auth token": "from-gh"})
	none := fakeGh(nil)
	cases := []struct {
		name string
		gh   Gh
		env  map[string]string
		want string
	}{
		{"gh wins over the environment", gh, map[string]string{"GH_TOKEN": "a", "GITHUB_TOKEN": "b"}, "from-gh"},
		{"GH_TOKEN wins over GITHUB_TOKEN", none, map[string]string{"GH_TOKEN": "a", "GITHUB_TOKEN": "b"}, "a"},
		{"GITHUB_TOKEN alone", none, map[string]string{"GITHUB_TOKEN": "b"}, "b"},
		{"nothing at all", none, nil, ""},
	}
	for _, c := range cases {
		if got := Token(c.gh, env(c.env)); got != c.want {
			t.Errorf("%s: Token = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestProtocolSSH(t *testing.T) {
	gh := fakeGh(map[string]string{"config get git_protocol": "ssh"})
	if got := Protocol(gh); got != "ssh" {
		t.Fatalf("Protocol = %q, want ssh", got)
	}
}

func TestProtocolDefaultsToHTTPS(t *testing.T) {
	for name, gh := range map[string]Gh{
		"gh missing":    fakeGh(nil),
		"gh says https": fakeGh(map[string]string{"config get git_protocol": "https"}),
		"gh says junk":  fakeGh(map[string]string{"config get git_protocol": "ftp"}),
	} {
		if got := Protocol(gh); got != "https" {
			t.Errorf("%s: Protocol = %q, want https", name, got)
		}
	}
}
