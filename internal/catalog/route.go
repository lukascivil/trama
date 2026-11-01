package catalog

import "strings"

// NormalizeID returns the form of an id used to match what one repository
// consumes against what another provides. For kind http it uppercases the
// method and replaces every path parameter with {}, so the same route
// matches however each side spells its parameters: the consumer's
// "GET /stone-accounts/{stone_account_id}" and the router's
// "GET /stone-accounts/:id" are the same route. Other kinds, such as a
// topic or a package, are matched by their exact id.
func NormalizeID(kind, id string) string {
	s := strings.TrimSpace(id)
	if kind != "http" {
		return s
	}
	method, path, ok := strings.Cut(s, " ")
	if !ok {
		return s
	}
	segments := strings.Split(strings.TrimSpace(path), "/")
	for i, seg := range segments {
		if isRouteParam(seg) {
			segments[i] = "{}"
		}
	}
	path = strings.Join(segments, "/")
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	return strings.ToUpper(method) + " " + path
}

// isRouteParam reports whether one path segment names a parameter, in any
// of the spellings the routers in use write: {name}, :name and *rest.
func isRouteParam(seg string) bool {
	if len(seg) < 2 {
		return false
	}
	switch {
	case seg[0] == ':', seg[0] == '*':
		return true
	case seg[0] == '{' && seg[len(seg)-1] == '}':
		return len(seg) > 2
	}
	return false
}
