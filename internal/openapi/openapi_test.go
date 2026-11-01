package openapi_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/lukascivil/trama/internal/openapi"
)

const ordersSpec = `openapi: 3.0.3
info: { title: orders, version: "1" }
paths:
  /orders:
    post:
      requestBody:
        content:
          application/json:
            schema: { $ref: "#/components/schemas/NewOrder" }
      responses:
        "201":
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Order" }
        "400":
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Error" }
  /orders/{order_id}:
    parameters: [{ name: order_id, in: path }]
    get:
      responses:
        "200":
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Order" }
    delete:
      responses:
        "204": { description: deleted }
components:
  schemas:
    NewOrder:
      type: object
      properties:
        customer: { $ref: "#/components/schemas/Customer" }
        items:
          type: array
          items:
            type: object
            properties:
              sku: { type: string }
              quantity: { type: integer }
    Order:
      allOf:
        - $ref: "#/components/schemas/NewOrder"
        - type: object
          properties:
            id: { type: string, readOnly: true }
            total: { type: number }
            tags: { type: array, items: { type: string } }
            metadata: { type: object, additionalProperties: true }
    Customer:
      type: object
      properties:
        email: { type: string }
        name: { type: [string, "null"] }
        password: { type: string, writeOnly: true }
    Error:
      type: object
      properties:
        message: { type: string }
`

func parse(t *testing.T, src string) *openapi.Spec {
	t.Helper()
	s, err := openapi.Parse("openapi.yaml", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func op(t *testing.T, s *openapi.Spec, id string) openapi.Operation {
	t.Helper()
	o, ok := s.Find(id)
	if !ok {
		t.Fatalf("%s not found", id)
	}
	return o
}

func fields(b openapi.Body) string {
	var parts []string
	for _, f := range b.Fields {
		parts = append(parts, f.Path+": "+f.Type)
	}
	return strings.Join(parts, ", ")
}

func TestParseRefusesWhatIsNotASpec(t *testing.T) {
	docs := map[string]string{
		"helm":         "apiVersion: v2\nname: chart\n",
		"package.json": `{"name": "web", "scripts": {"openapi": "gen"}}`,
		"swagger 2":    "swagger: \"2.0\"\npaths: {}\n",
		"no paths":     "openapi: 3.0.0\ninfo: {}\n",
		"paths list":   "openapi: 3.0.0\npaths: []\n",
		"list":         "- openapi: 3.0.0\n",
		"empty":        "",
		"version 4":    "openapi: 4.0.0\npaths: {}\n",
	}
	for name, src := range docs {
		if _, err := openapi.Parse("x.yaml", []byte(src)); !errors.Is(err, openapi.ErrNotSpec) {
			t.Errorf("%s: err = %v, want ErrNotSpec", name, err)
		}
	}
}

func TestParseReportsBrokenYAML(t *testing.T) {
	_, err := openapi.Parse("x.yaml", []byte("openapi: 3.0.0\npaths: [\n"))
	if err == nil || errors.Is(err, openapi.ErrNotSpec) {
		t.Fatalf("err = %v, want a YAML error", err)
	}
}

func TestOperationsInSpecOrderWithTheirLine(t *testing.T) {
	s := parse(t, ordersSpec)
	var got []string
	for _, o := range s.Operations() {
		got = append(got, fmt.Sprintf("%s:%d", o.ID, o.Line))
	}
	want := []string{"POST /orders:5", "GET /orders/{order_id}:21", "DELETE /orders/{order_id}:27"}
	if !slices.Equal(got, want) {
		t.Fatalf("operations = %v, want %v", got, want)
	}
}

func TestFindMatchesLikeLink(t *testing.T) {
	s := parse(t, ordersSpec)
	for _, id := range []string{"get /orders/{id}", "GET /orders/:id/", "DELETE /orders/{x}"} {
		if _, ok := s.Find(id); !ok {
			t.Errorf("%s not found", id)
		}
	}
	if _, ok := s.Find("PUT /orders/{id}"); ok {
		t.Error("PUT found")
	}
}

func TestRequestAndResponseFields(t *testing.T) {
	s := parse(t, ordersSpec)
	post := op(t, s, "POST /orders")
	req := post.Request()
	if want := "customer.email: string, customer.name: string, customer.password: string, items[].sku: string, items[].quantity: number"; !req.Declared || fields(req) != want {
		t.Fatalf("request = %+v\nwant %s", req, want)
	}
	resp := post.Response()
	want := "customer.email: string, customer.name: string, items[].sku: string, items[].quantity: number, id: string, total: number, tags: []string, metadata: object"
	if fields(resp) != want {
		t.Fatalf("response = %s\nwant       %s", fields(resp), want)
	}
	if b := op(t, s, "GET /orders/{id}").Request(); b.Declared {
		t.Fatalf("GET has a request: %+v", b)
	}
	if b := op(t, s, "DELETE /orders/{id}").Response(); b.Declared {
		t.Fatalf("204 without content has a response: %+v", b)
	}
}

func TestReadOnlyAndWriteOnly(t *testing.T) {
	s := parse(t, ordersSpec)
	post := op(t, s, "POST /orders")
	if strings.Contains(fields(post.Response()), "password") {
		t.Error("writeOnly field in the response")
	}
	s = parse(t, strings.Replace(ordersSpec, "schema: { $ref: \"#/components/schemas/NewOrder\" }", "schema: { $ref: \"#/components/schemas/Order\" }", 1))
	req := op(t, s, "POST /orders").Request()
	if strings.Contains(fields(req), "id: string") {
		t.Errorf("readOnly field in the request: %s", fields(req))
	}
	if !strings.Contains(fields(req), "customer.password: string") {
		t.Errorf("writeOnly field missing from the request: %s", fields(req))
	}
}

// body builds a spec whose only route returns schema, with extra under
// components.
func body(t *testing.T, schema, components string) openapi.Body {
	t.Helper()
	src := "openapi: 3.1.0\npaths:\n  /x:\n    get:\n      responses:\n        \"200\":\n          content:\n            application/json:\n              schema: " + schema + "\n"
	if components != "" {
		src += "components:\n" + components
	}
	return op(t, parse(t, src), "GET /x").Response()
}

func TestShapes(t *testing.T) {
	cases := []struct {
		name, schema, components, want string
	}{
		{"free map", `{type: object, properties: {meta: {type: object}}}`, "", "meta: object"},
		{"list of lists", `{type: object, properties: {grid: {type: array, items: {type: array, items: {type: number}}}}}`, "", "grid: []object"},
		{"list of free objects", `{type: object, properties: {rows: {type: array, items: {type: object}}}}`, "", "rows: []object"},
		{"nullable", `{type: object, properties: {a: {type: string, nullable: true}, b: {type: [integer, "null"]}}}`, "", "a: string, b: number"},
		{"no type, properties", `{properties: {a: {type: boolean}}}`, "", "a: boolean"},
		{"mixed types out", `{type: object, properties: {a: {type: [string, number]}, b: {}, c: {type: string}}}`, "", "c: string"},
		{"oneOf union", `{oneOf: [{properties: {a: {type: string}}}, {properties: {a: {type: string}, b: {type: number}}}]}`, "", "a: string, b: number"},
		{"anyOf union", `{anyOf: [{properties: {a: {type: string}}}, {properties: {b: {type: string}}}]}`, "", "a: string, b: string"},
		{"bad names out", `{type: object, properties: {$schema: {type: string}, a.b: {type: string}, "x[]": {type: string}, ok-name: {type: string}}}`, "", "ok-name: string"},
		{"chained ref", `{$ref: "#/components/schemas/A"}`, "  schemas:\n    A: {$ref: \"#/components/schemas/B\"}\n    B: {type: object, properties: {b: {type: string}}}\n", "b: string"},
		{"escaped pointer", `{$ref: "#/components/schemas/a~1b"}`, "  schemas:\n    a/b: {type: object, properties: {c: {type: string}}}\n", "c: string"},
		{"ref cycle", `{$ref: "#/components/schemas/Node"}`, "  schemas:\n    Node: {type: object, properties: {name: {type: string}, parent: {$ref: \"#/components/schemas/Node\"}, children: {type: array, items: {$ref: \"#/components/schemas/Node\"}}}}\n", "name: string, parent: object, children: []object"},
	}
	for _, c := range cases {
		b := body(t, c.schema, c.components)
		if b.Problem != "" || fields(b) != c.want {
			t.Errorf("%s: got %q (problem %q), want %q", c.name, fields(b), b.Problem, c.want)
		}
	}
}

func TestProblems(t *testing.T) {
	cases := []struct {
		name, schema, components, want string
	}{
		{"list", `{type: array, items: {type: object, properties: {a: {type: string}}}}`, "", "is a list, not an object"},
		{"string", `{type: string}`, "", "is a string, not an object"},
		{"integer", `{type: integer}`, "", "is a number, not an object"},
		{"no properties", `{type: object}`, "", "is an object with no properties"},
		{"no type", `{}`, "", "has no type trama can read"},
		{"outside", `{$ref: "./schemas/order.yaml"}`, "", `uses $ref "./schemas/order.yaml", outside the file`},
		{"missing", `{$ref: "#/components/schemas/Nope"}`, "  schemas: {}\n", `uses $ref "#/components/schemas/Nope", which is not in the file`},
		{"self", `{$ref: "#/components/schemas/A"}`, "  schemas:\n    A: {$ref: \"#/components/schemas/A\"}\n", `uses $ref "#/components/schemas/A", which points back to itself`},
		{"nested outside", `{type: object, properties: {a: {$ref: "other.yaml#/A"}}}`, "", `uses $ref "other.yaml#/A", outside the file`},
	}
	for _, c := range cases {
		b := body(t, c.schema, c.components)
		if !b.Declared || b.Problem != c.want || len(b.Fields) != 0 {
			t.Errorf("%s: got %+v, want problem %q", c.name, b, c.want)
		}
	}
}

func TestNoSchema(t *testing.T) {
	s := parse(t, "openapi: 3.0.0\npaths:\n  /x:\n    post:\n      requestBody:\n        content:\n          application/json: {}\n")
	if b := op(t, s, "POST /x").Request(); b.Problem != "has no schema" {
		t.Fatalf("request = %+v", b)
	}
}

func TestComponentsRequestBodiesAndResponses(t *testing.T) {
	s := parse(t, `openapi: 3.0.0
paths:
  /x:
    post:
      requestBody: { $ref: "#/components/requestBodies/In" }
      responses:
        "200": { $ref: "#/components/responses/Out" }
components:
  requestBodies:
    In: { content: { application/json: { schema: { properties: { a: { type: string } } } } } }
  responses:
    Out: { content: { application/json: { schema: { properties: { b: { type: number } } } } } }
`)
	o := op(t, s, "POST /x")
	if fields(o.Request()) != "a: string" || fields(o.Response()) != "b: number" {
		t.Fatalf("request %+v, response %+v", o.Request(), o.Response())
	}
}

func TestStatusAndMediaChoice(t *testing.T) {
	s := parse(t, `openapi: 3.0.0
paths:
  /x:
    get:
      responses:
        default: { content: { application/json: { schema: { properties: { d: { type: string } } } } } }
        "202": { content: { application/json: { schema: { properties: { late: { type: string } } } } } }
        "201": { description: no content }
        2XX: { content: { application/json: { schema: { properties: { any: { type: string } } } } } }
        "200":
          content:
            text/plain: { schema: { properties: { text: { type: string } } } }
            application/problem+json: { schema: { properties: { plus: { type: string } } } }
    put:
      responses:
        2XX: { content: { application/json: { schema: { properties: { ranged: { type: string } } } } } }
        default: { content: { application/json: { schema: { properties: { d: { type: string } } } } } }
    post:
      requestBody:
        content:
          text/plain: { schema: { properties: { text: { type: string } } } }
          application/json; charset=utf-8: { schema: { properties: { json: { type: string } } } }
      responses:
        "500": { content: { application/json: { schema: { properties: { e: { type: string } } } } } }
`)
	if got := fields(op(t, s, "GET /x").Response()); got != "plus: string" {
		t.Errorf("GET response = %q, want the 200 +json body", got)
	}
	if got := fields(op(t, s, "PUT /x").Response()); got != "ranged: string" {
		t.Errorf("PUT response = %q, want the 2XX body", got)
	}
	post := op(t, s, "POST /x")
	if got := fields(post.Request()); got != "json: string" {
		t.Errorf("POST request = %q, want the application/json body", got)
	}
	if b := post.Response(); b.Declared {
		t.Errorf("an error status counted as the response: %+v", b)
	}
}

func TestMinifiedJSON(t *testing.T) {
	src := `{"openapi":"3.0.0","paths":{"/a":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"type":"object","properties":{"id":{"type":"integer"}}}}}}}}}}}`
	s, err := openapi.Parse("api.json", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	o := op(t, s, "GET /a")
	if fields(o.Response()) != "id: number" || o.Line != 1 {
		t.Fatalf("response %+v line %d", o.Response(), o.Line)
	}
}

func TestAliasThatContainsItself(t *testing.T) {
	inProperties := "openapi: 3.0.0\npaths:\n  /x:\n    get:\n      responses:\n        \"200\":\n          content:\n            application/json:\n              schema: &a\n                type: object\n                properties:\n                  next: *a\n"
	inAllOf := "openapi: 3.0.0\npaths:\n  /x:\n    get:\n      responses:\n        \"200\":\n          content:\n            application/json:\n              schema: &a\n                properties: { a: { type: string } }\n                allOf: [*a]\n"
	for name, src := range map[string]string{"properties": inProperties, "allOf": inAllOf} {
		b := op(t, parse(t, src), "GET /x").Response()
		if !strings.HasPrefix(b.Problem, "is too large for trama to read") {
			t.Errorf("%s: got %+v", name, b)
		}
	}
}

func TestRefFanOutHitsTheCeiling(t *testing.T) {
	var b strings.Builder
	b.WriteString("openapi: 3.0.0\npaths:\n  /x:\n    get:\n      responses:\n        \"200\":\n          content:\n            application/json:\n              schema: { $ref: \"#/components/schemas/L0\" }\ncomponents:\n  schemas:\n")
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&b, "    L%d: { type: object, properties: { a: { $ref: \"#/components/schemas/L%d\" }, b: { $ref: \"#/components/schemas/L%d\" } } }\n", i, i+1, i+1)
	}
	b.WriteString("    L30: { type: object, properties: { leaf: { type: string } } }\n")
	got := op(t, parse(t, b.String()), "GET /x").Response()
	if !strings.HasPrefix(got.Problem, "is too large for trama to read") {
		t.Fatalf("got %d fields, problem %q", len(got.Fields), got.Problem)
	}
}

func TestALargeLegitimateBodyIsReadWhole(t *testing.T) {
	var b strings.Builder
	b.WriteString("openapi: 3.0.0\npaths:\n  /x:\n    get:\n      responses:\n        \"200\":\n          content:\n            application/json:\n              schema:\n                type: object\n                properties:\n")
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&b, "                  f%d: { $ref: \"#/components/schemas/Leaf\" }\n", i)
	}
	b.WriteString("components:\n  schemas:\n    Leaf: { type: object, properties: { v: { type: string } } }\n")
	got := op(t, parse(t, b.String()), "GET /x").Response()
	if got.Problem != "" || len(got.Fields) != 1000 || got.Fields[999].Path != "f999.v" {
		t.Fatalf("got %d fields, problem %q", len(got.Fields), got.Problem)
	}
}
