package view

import (
	"testing"

	"github.com/lukascivil/trama/internal/catalog"
)

func TestRoleOf(t *testing.T) {
	http := []catalog.Provide{{Kind: "http", ID: "GET /x"}}
	event := []catalog.Provide{{Kind: "event", ID: "orders.created"}}
	pkg := []catalog.Provide{{Kind: "package", ID: "@acme/kit"}}
	for _, tc := range []struct {
		name string
		card catalog.Card
		want string
	}{
		{"a React app that also serves routes is web", catalog.Card{Stack: []string{"node", "Next.js", "react"}, Provides: http}, "web"},
		{"a mobile stack wins over react", catalog.Card{Stack: []string{"react-native", "expo"}}, "app"},
		{"a Go service with routes is an api", catalog.Card{Stack: []string{"go", "postgres"}, Provides: http}, "api"},
		{"terraform that publishes topics is infra, not a worker", catalog.Card{Stack: []string{"terraform"}, Provides: event}, "infra"},
		{"only a package is a lib", catalog.Card{Stack: []string{"typescript"}, Provides: pkg}, "lib"},
		{"routes and a package are still an api", catalog.Card{Provides: append(append([]catalog.Provide{}, pkg...), http...)}, "api"},
		{"publishing events is a worker", catalog.Card{Stack: []string{"node", "kafka"}, Provides: event}, "worker"},
		{"consuming events with nothing provided is a worker", catalog.Card{Consumes: []catalog.Consume{{Kind: "event", ID: "orders.created"}}}, "worker"},
		{"nothing to go on", catalog.Card{Stack: []string{"yaml"}}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := roleOf(&tc.card); got != tc.want {
				t.Errorf("roleOf = %q, want %q", got, tc.want)
			}
		})
	}
}
