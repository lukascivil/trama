// Package openapi reads the OpenAPI 3 specs a repository commits: which
// files are specs, which routes each one declares, and the fields of each
// route's request and response body. A spec is a document with a fixed
// shape, so reading it is mechanical, and the same spec gives the same
// fields to everyone who reads it.
//
// The spec comes from a repository of the workspace, not from whoever runs
// trama, so it is read as untrusted input: $ref is only followed inside the
// file, and every body has a ceiling on how much it may expand.
package openapi

import (
	"errors"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/lukascivil/trama/internal/catalog"
)

// ErrNotSpec is returned by Parse for a YAML or JSON document that is not
// an OpenAPI 3 spec.
var ErrNotSpec = errors.New("not an OpenAPI 3 spec")

// methods are the operations of a path item that a card can provide.
var methods = []string{"get", "put", "post", "delete", "options", "head", "patch"}

var version3 = regexp.MustCompile(`^3(\.[0-9]+)*$`)

// Spec is one OpenAPI 3 document.
type Spec struct {
	Path string // relative to the repository root
	root *yaml.Node
	ops  []Operation
}

// Operation is one route a spec declares.
type Operation struct {
	ID   string // the method in upper case and the path as the spec writes it
	Line int    // the line of the method in the spec
	root *yaml.Node
	node *yaml.Node
}

// Field is one field of a body: its dotted path and its type, in the
// vocabulary of a card.
type Field struct {
	Path string
	Type string
}

// Body is what a spec says about one request or response body.
type Body struct {
	Declared bool    // the spec declares this body
	Fields   []Field // in the order of the spec, when the body was read
	Problem  string  // why a declared body was not read as fields, e.g. "is a list, not an object"
}

// Parse reads data as an OpenAPI 3 spec: a document whose top has
// openapi: 3.x and a paths mapping. JSON is read too, as YAML.
func Parse(path string, data []byte) (*Spec, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 {
		return nil, ErrNotSpec
	}
	root := deref(doc.Content[0])
	if root.Kind != yaml.MappingNode {
		return nil, ErrNotSpec
	}
	v := deref(field(root, "openapi"))
	paths := deref(field(root, "paths"))
	if v == nil || v.Kind != yaml.ScalarNode || !version3.MatchString(v.Value) || paths == nil || paths.Kind != yaml.MappingNode {
		return nil, ErrNotSpec
	}
	s := &Spec{Path: path, root: root}
	for i := 0; i+1 < len(paths.Content); i += 2 {
		route := deref(paths.Content[i]).Value
		item := deref(paths.Content[i+1])
		if item == nil || item.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j+1 < len(item.Content); j += 2 {
			key := deref(item.Content[j])
			if !slices.Contains(methods, strings.ToLower(key.Value)) {
				continue
			}
			s.ops = append(s.ops, Operation{
				ID:   strings.ToUpper(key.Value) + " " + route,
				Line: key.Line,
				root: root,
				node: deref(item.Content[j+1]),
			})
		}
	}
	return s, nil
}

// Operations returns the routes the spec declares, in its order.
func (s *Spec) Operations() []Operation {
	return s.ops
}

// Find returns the operation that serves id, matched the way trama link
// matches a route: the method's case, a trailing slash and the name of each
// path parameter do not count.
func (s *Spec) Find(id string) (Operation, bool) {
	want := catalog.NormalizeID("http", id)
	for _, op := range s.ops {
		if catalog.NormalizeID("http", op.ID) == want {
			return op, true
		}
	}
	return Operation{}, false
}

// Request returns the request body: the schema of application/json, or of
// the first +json media type, or of the first one.
func (o Operation) Request() Body {
	if o.node == nil || field(o.node, "requestBody") == nil {
		return Body{}
	}
	w := o.walker("readOnly")
	rb, err := w.follow(field(o.node, "requestBody"))
	if err != nil {
		return Body{Declared: true, Problem: err.Error()}
	}
	return w.content(rb)
}

// Response returns the body of the lowest 2xx status that has content, then
// of 2XX. A default or an error response never counts, and a route whose
// success has no content has no response body.
func (o Operation) Response() Body {
	responses := deref(field(o.node, "responses"))
	if responses == nil || responses.Kind != yaml.MappingNode {
		return Body{}
	}
	var codes []int
	var ranged *yaml.Node
	byCode := map[int]*yaml.Node{}
	for i := 0; i+1 < len(responses.Content); i += 2 {
		key := deref(responses.Content[i]).Value
		if strings.EqualFold(key, "2XX") {
			ranged = responses.Content[i+1]
			continue
		}
		if code, err := strconv.Atoi(key); err == nil && code >= 200 && code <= 299 {
			if _, dup := byCode[code]; !dup {
				codes = append(codes, code)
				byCode[code] = responses.Content[i+1]
			}
		}
	}
	slices.Sort(codes)
	var candidates []*yaml.Node
	for _, c := range codes {
		candidates = append(candidates, byCode[c])
	}
	if ranged != nil {
		candidates = append(candidates, ranged)
	}
	w := o.walker("writeOnly")
	for _, c := range candidates {
		resp, err := w.follow(c)
		if err != nil {
			return Body{Declared: true, Problem: err.Error()}
		}
		if content := deref(field(resp, "content")); content != nil && content.Kind == yaml.MappingNode && len(content.Content) > 0 {
			return w.content(resp)
		}
	}
	return Body{}
}

func (o Operation) walker(skip string) *walker {
	return &walker{root: o.root, skip: skip, seen: map[string]bool{}}
}

// The ceiling on one body. A spec of a few lines can expand without end — an
// alias that contains itself, or $ref that fan out at every level — and a
// legitimate body stays far below both.
const (
	maxSchemas = 100000
	maxDepth   = 32
)

type problem string

func (p problem) Error() string { return string(p) }

const tooLarge = problem("is too large for trama to read (more than 100000 schemas, or 32 levels deep)")

// walker turns the schema of one body into fields.
type walker struct {
	root   *yaml.Node
	skip   string // readOnly in a request, writeOnly in a response
	visits int
	fields []Field
	seen   map[string]bool
}

// content reads the body of a requestBody or response object that has
// content.
func (w *walker) content(n *yaml.Node) Body {
	content := deref(field(n, "content"))
	if content == nil || content.Kind != yaml.MappingNode || len(content.Content) == 0 {
		return Body{Declared: true, Problem: "has no schema"}
	}
	schema := field(media(content), "schema")
	if deref(schema) == nil {
		return Body{Declared: true, Problem: "has no schema"}
	}
	var g gathered
	if err := w.gather(schema, nil, 0, &g); err != nil {
		return Body{Declared: true, Problem: err.Error()}
	}
	switch k := g.kind(); k {
	case "object":
		if len(g.props) == 0 {
			return Body{Declared: true, Problem: "is an object with no properties"}
		}
	case "array":
		return Body{Declared: true, Problem: "is a list, not an object"}
	case "string", "number", "boolean":
		return Body{Declared: true, Problem: "is a " + k + ", not an object"}
	default:
		return Body{Declared: true, Problem: "has no type trama can read"}
	}
	if _, err := w.object("", &g, 0); err != nil {
		return Body{Declared: true, Problem: err.Error()}
	}
	if len(w.fields) == 0 {
		return Body{Declared: true, Problem: "has no type trama can read"}
	}
	return Body{Declared: true, Fields: w.fields}
}

// media picks application/json, then the first +json type, then the first.
func media(content *yaml.Node) *yaml.Node {
	var plusJSON, first *yaml.Node
	for i := 0; i+1 < len(content.Content); i += 2 {
		mt, _, _ := strings.Cut(deref(content.Content[i]).Value, ";")
		mt = strings.ToLower(strings.TrimSpace(mt))
		v := content.Content[i+1]
		if mt == "application/json" {
			return v
		}
		if plusJSON == nil && strings.HasSuffix(mt, "+json") {
			plusJSON = v
		}
		if first == nil {
			first = v
		}
	}
	if plusJSON != nil {
		return plusJSON
	}
	return first
}

// gathered is a schema merged with every schema it composes (allOf, oneOf,
// anyOf): the search wants every field that may come, so a composition is
// the union of its parts.
type gathered struct {
	types     []string // without null, integer read as number
	props     []prop
	hasProps  bool
	items     []prop // the schemas of a list's items; name is unused
	cycle     bool   // a $ref came back to a schema already being read
	readOnly  bool
	writeOnly bool
}

type prop struct {
	name  string
	node  *yaml.Node
	stack []string // the $ref being read where the property was found
}

func (g *gathered) kind() string {
	switch {
	case len(g.types) > 1:
		return ""
	case len(g.types) == 1:
		switch t := g.types[0]; t {
		case "object", "array", "string", "number", "boolean":
			return t
		}
		return ""
	case g.hasProps:
		return "object"
	case len(g.items) > 0:
		return "array"
	case g.cycle:
		return "object"
	}
	return ""
}

func (g *gathered) flagged(key string) bool {
	return (key == "readOnly" && g.readOnly) || (key == "writeOnly" && g.writeOnly)
}

// gather adds the schema n, and what it composes, to g.
func (w *walker) gather(n *yaml.Node, stack []string, depth int, g *gathered) error {
	if depth > maxDepth {
		return tooLarge
	}
	n, stack, cycle, err := w.resolve(n, stack)
	if err != nil {
		return err
	}
	if cycle {
		g.cycle = true
		return nil
	}
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	if w.visits++; w.visits > maxSchemas {
		return tooLarge
	}
	if t := deref(field(n, "type")); t != nil {
		names := []*yaml.Node{t}
		if t.Kind == yaml.SequenceNode {
			names = t.Content
		}
		for _, name := range names {
			v := deref(name).Value
			switch v {
			case "null":
				continue
			case "integer":
				v = "number"
			}
			if !slices.Contains(g.types, v) {
				g.types = append(g.types, v)
			}
		}
	}
	g.readOnly = g.readOnly || isTrue(field(n, "readOnly"))
	g.writeOnly = g.writeOnly || isTrue(field(n, "writeOnly"))
	if props := deref(field(n, "properties")); props != nil && props.Kind == yaml.MappingNode {
		g.hasProps = true
		for i := 0; i+1 < len(props.Content); i += 2 {
			g.props = append(g.props, prop{name: deref(props.Content[i]).Value, node: props.Content[i+1], stack: stack})
		}
	}
	if items := field(n, "items"); items != nil {
		g.items = append(g.items, prop{node: items, stack: stack})
	}
	for _, key := range []string{"allOf", "oneOf", "anyOf"} {
		list := deref(field(n, key))
		if list == nil || list.Kind != yaml.SequenceNode {
			continue
		}
		for _, sub := range list.Content {
			if err := w.gather(sub, stack, depth+1, g); err != nil {
				return err
			}
		}
	}
	return nil
}

// object writes the fields of the properties in g under prefix, and reports
// whether any of them is a field a card can hold.
func (w *walker) object(prefix string, g *gathered, depth int) (bool, error) {
	if depth > maxDepth {
		return false, tooLarge
	}
	found := false
	for _, p := range g.props {
		if !validName(p.name) || isTrue(field(p.node, w.skip)) {
			continue
		}
		var sub gathered
		if err := w.gather(p.node, p.stack, depth+1, &sub); err != nil {
			return false, err
		}
		if sub.flagged(w.skip) {
			continue
		}
		ok, err := w.value(prefix+p.name, &sub, depth+1)
		if err != nil {
			return false, err
		}
		found = found || ok
	}
	return found, nil
}

// value writes the field at path, whose schema is g, and reports whether
// it is one a card can hold.
func (w *walker) value(path string, g *gathered, depth int) (bool, error) {
	switch k := g.kind(); k {
	case "string", "number", "boolean":
		return w.add(path, k), nil
	case "object":
		if ok, err := w.object(path+".", g, depth); ok || err != nil {
			return ok, err
		}
		return w.add(path, "object"), nil
	case "array":
		var items gathered
		for _, it := range g.items {
			if err := w.gather(it.node, it.stack, depth+1, &items); err != nil {
				return false, err
			}
		}
		switch ik := items.kind(); ik {
		case "string", "number", "boolean":
			return w.add(path, "[]"+ik), nil
		case "object":
			if ok, err := w.object(path+"[].", &items, depth+1); ok || err != nil {
				return ok, err
			}
			return w.add(path, "[]object"), nil
		case "array":
			return w.add(path, "[]object"), nil
		}
	}
	return false, nil
}

// add records a field once, and only when the card can hold it.
func (w *walker) add(path, typ string) bool {
	if catalog.ValidateField(path, typ) != nil {
		return false
	}
	if !w.seen[path] {
		w.seen[path] = true
		w.fields = append(w.fields, Field{Path: path, Type: typ})
	}
	return true
}

// validName reports whether a property name is one segment of a field
// path: no dot, no brackets, nothing like huma's $schema.
func validName(name string) bool {
	return !strings.ContainsAny(name, ".[]") && catalog.ValidateField(name, "") == nil
}

// follow resolves a requestBody or response object, which may be a $ref to
// components.
func (w *walker) follow(n *yaml.Node) (*yaml.Node, error) {
	n, _, _, err := w.resolve(n, nil)
	return n, err
}

// resolve follows n through YAML aliases and $ref inside the file. stack
// holds the $ref being read around n: meeting one of them again is a cycle,
// which the caller reads as an object. A chain of $ref that comes back to
// itself without reaching a schema is an error.
func (w *walker) resolve(n *yaml.Node, stack []string) (*yaml.Node, []string, bool, error) {
	var chain []string
	for {
		n = deref(n)
		ref := deref(field(n, "$ref"))
		if ref == nil {
			return n, stack, false, nil
		}
		r := ref.Value
		if slices.Contains(chain, r) {
			return nil, stack, false, problem("uses $ref " + strconv.Quote(r) + ", which points back to itself")
		}
		if slices.Contains(stack, r) {
			return nil, stack, true, nil
		}
		if w.visits++; w.visits > maxSchemas {
			return nil, stack, false, tooLarge
		}
		target, err := w.pointer(r)
		if err != nil {
			return nil, stack, false, err
		}
		chain = append(chain, r)
		stack = append(stack[:len(stack):len(stack)], r)
		n = target
	}
}

// pointer finds a $ref of the form #/a/b in the file.
func (w *walker) pointer(ref string) (*yaml.Node, error) {
	if !strings.HasPrefix(ref, "#/") {
		return nil, problem("uses $ref " + strconv.Quote(ref) + ", outside the file")
	}
	n := w.root
	for _, seg := range strings.Split(ref[2:], "/") {
		seg = strings.ReplaceAll(strings.ReplaceAll(seg, "~1", "/"), "~0", "~")
		if s, err := url.PathUnescape(seg); err == nil {
			seg = s
		}
		var next *yaml.Node
		switch n = deref(n); {
		case n == nil:
		case n.Kind == yaml.MappingNode:
			next = field(n, seg)
		case n.Kind == yaml.SequenceNode:
			if i, err := strconv.Atoi(seg); err == nil && i >= 0 && i < len(n.Content) {
				next = n.Content[i]
			}
		}
		if next == nil {
			return nil, problem("uses $ref " + strconv.Quote(ref) + ", which is not in the file")
		}
		n = next
	}
	return n, nil
}

// field returns the value of key in a mapping node, or nil.
func field(m *yaml.Node, key string) *yaml.Node {
	m = deref(m)
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if deref(m.Content[i]).Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// deref follows a YAML alias to the node it names.
func deref(n *yaml.Node) *yaml.Node {
	if n != nil && n.Kind == yaml.AliasNode {
		return n.Alias
	}
	return n
}

func isTrue(n *yaml.Node) bool {
	n = deref(n)
	return n != nil && n.Kind == yaml.ScalarNode && n.Value == "true"
}
