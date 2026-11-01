package view

import (
	"slices"
	"strings"

	"github.com/lukascivil/trama/internal/catalog"
)

var (
	appStack   = []string{"reactnative", "expo", "flutter", "swift", "swiftui", "kotlin", "android", "ios", "dart", "xamarin"}
	webStack   = []string{"react", "next", "nextjs", "vue", "nuxt", "angular", "svelte", "sveltekit", "astro", "remix", "solid", "gatsby", "html"}
	infraStack = []string{"terraform", "helm", "kubernetes", "k8s", "pulumi", "ansible", "cloudformation", "cdk"}
)

// roleOf says what kind of repository a card describes: app, web, infra,
// api, lib or worker, or "" when nothing in the card tells. It is a hint for
// the chip on the node, read from the stack and from what the card provides
// and consumes; the card has no field for it, so a repository that looks
// like something else can still be wrong.
func roleOf(c *catalog.Card) string {
	stack := make([]string, 0, len(c.Stack))
	for _, s := range c.Stack {
		stack = append(stack, strings.NewReplacer(".", "", "-", "", " ", "", "_", "").Replace(strings.ToLower(s)))
	}
	has := func(names []string) bool {
		return slices.ContainsFunc(stack, func(s string) bool { return slices.Contains(names, s) })
	}
	provides := func(kind string) bool {
		return slices.ContainsFunc(c.Provides, func(p catalog.Provide) bool { return p.Kind == kind })
	}
	consumesEvents := slices.ContainsFunc(c.Consumes, func(cs catalog.Consume) bool { return cs.Kind == "event" })

	switch {
	case has(appStack):
		return "app"
	case has(webStack):
		return "web"
	case has(infraStack):
		return "infra"
	case provides("http"):
		return "api"
	case provides("package"):
		return "lib"
	case provides("event") || consumesEvents:
		return "worker"
	}
	return ""
}
