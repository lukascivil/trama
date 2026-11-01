# Catalog card reference

One card per repository, at `catalog/<name>.yaml`. `trama validate` checks every rule below.

## Fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `version` | int | yes | Always `1`. |
| `name` | string | yes | Same as the file name and the repository's `name` in `workspace.yaml`. |
| `description` | string | yes | One sentence on what the repository is for, in English. |
| `stack` | list of strings | yes | Languages, main frameworks and build tools, e.g. `[node, react, typescript, vite]`. |
| `tools` | map | no | Runtime versions from version files, e.g. `{go: "1.25"}`, `{node: "24"}`. Quote versions. |
| `mapped_at_commit` | string | yes | Written by `trama stamp` (the full SHA). Leave it out when you write the card; never type it. |
| `mapped_at` | string | yes | Written by `trama stamp` (`YYYY-MM-DD`). Leave it out when you write the card. |
| `run` | map | no | Keys `setup`, `dev`, `test`; each `{cmd: "...", port: 8080}` (`port` only for `dev`). |
| `identity` | map | no | How other repositories recognize this one; see below. |
| `provides` | list | no | What it offers to others; see below. |
| `consumes` | list | no | What it uses from others; see below. |
| `config` | list | no | Environment variables holding another service's address; see below. |

### identity

| Field | Item shape | Evidence |
|---|---|---|
| `ports` | `{value: 8080, evidence: [...]}` | the line that sets the listen address, or, for a framework default, the line of the repository's docs that states it |
| `hosts` | `{value: api.acme.com, evidence: [workspace.yaml]}` | only hosts listed for this repository under `hosts` in `workspace.yaml` |
| `package` | `{value: "@acme/ui", evidence: [...]}` | the line that declares the package name other repositories import (any registry, including the company's private one, or a Go module path); omit it only for packages marked never to be published, such as `"private": true` |
| `topics` | `{value: orders.created, evidence: [...]}` | the line that publishes to the topic; a topic this repository publishes also goes in `provides` with `kind: event`, which is what links a consumer to it |
| `tables` | `{value: users, columns: [id, email], evidence: [...]}` | the line of the migration or schema that creates the table, never one that queries it; `value` may be qualified by its schema, as in `public.users` |

A table is not a relation: it never goes in `provides`, it links nothing, and `trama link` ignores it; the graph shows it as part of the repository, never as an arrow. It is here so a repository that needs a piece of data can find who owns it. `columns` is required — a table without its columns only says who owns "users", which whoever is asking already knew. List the columns the table has, not the ones the code reads.

`tables: []` is not the same as leaving the key out. The empty list says the repository was looked at and owns no table; a missing key says nobody looked. `trama validate` warns when a clone has files that declare tables (what `trama schemas` lists) and the card has no `tables` key, and the empty list is what silences it.

### provides

`{kind, id, contract, request, response, evidence}`

- `kind`: `http`, `event` or `package`.
- `id`: for `http`, `METHOD /path` with METHOD one of `GET POST PUT PATCH DELETE HEAD OPTIONS`, path parameters as `{name}` (never `:name`) and no query string, e.g. `GET /orders/{id}`. For `event`, the topic. For `package`, the package name.
- `contract`: optional path to an OpenAPI, GraphQL or proto file. It stays a pointer: the card carries the fields so they can be searched, the file carries the full detail.
- `request`: optional body fields the endpoint accepts. Only for `kind: http`.
- `response`: optional body fields it returns, e.g. `{message: string}`; for `kind: event`, the payload it publishes. Not for `kind: package`.

`request` and `response` are a flat map of field path to type, never nested YAML:

- A path is dotted names, the way the body nests them: `id`, `customer.email`, `items[].sku`. `[]` marks a segment that is a list of objects you then descend into, so it never ends a path — a list of plain values is the type, `[]string`.
- A type is one of `string`, `number`, `boolean`, `object`, `[]string`, `[]number`, `[]boolean`, `[]object`, and nothing else. Quote the list ones (`tags: "[]string"`): YAML reads a bare `[` as the start of a list, so an unquoted `[]string` is a syntax error, not a type. Whether a field is required, its format and its allowed values belong in the `contract` file, not here.
- List the fields the endpoint really returns, not the columns the database has. Leave the map out when you could not read them, instead of guessing.

### consumes

`{kind, id, from, via, candidates, fields_used, evidence}`

- `kind` and `id`: same rules as `provides`. For `http` the two sides are matched after normalizing: the method is case-insensitive, a trailing slash is ignored and every path parameter counts as the same, so `GET /orders/{id}` and `GET /orders/{order_id}` are the same route. Write the id as this repository's own code spells it.
- `from`: the providing repository, or `unknown`.
- `via`: the proof that `from` is right. Required when `from` is a repository, empty when it is `unknown`:
  - `route`: `kind: http` and exactly one repository of the workspace provides that route. `trama link` writes this one; do not write it by hand.
  - `env`: a `config` entry of this card has `points_to: <from>` and a `value` whose host matches one of the provider's `identity.hosts`, or whose port matches one of its `identity.ports` with host `localhost`, `127.0.0.1` or `0.0.0.0`.
  - `package`: `kind: package` and `id` equal to the provider's `identity.package`.
  - `topic`: `kind: event` and `id` listed in the provider's `identity.topics`.
  - `declared`: `workspace.yaml` has `links: [{consumer: <this>, provider: <from>, id: <id>}]`.
- `candidates`: with `from: unknown`, the repositories whose cards provide the same `id`. Leave the key out when there are none.
- `fields_used`: optional response fields the code reads, in the same dotted-path form as the provider's `response`.

### config

`{env, value, points_to, evidence}`: the variable name, its example value (from `.env.example` or the code's default), and the repository it points to. `points_to` must be a repository of the workspace; leave it out until that repository is added. List it even when no `consumes` entry uses it, e.g. a CORS origin pointing to the frontend.

## Evidence

Every `identity`, `provides`, `consumes` and `config` entry has at least one evidence item:

- `path:line` or `path:start-end`, with `path` relative to the repository root, no `..` and no leading `/`. The file must exist in `repos/<name>` and the lines must be inside it.
- The literal `workspace.yaml`, only for `identity.hosts`.

Cite a line only after reading it.

## Example

The two cards of the trama demo workspace.

```yaml
version: 1
name: trama-demo-api
description: Backend that serves the greeting
stack: [go]
tools:
  go: "1.25"
mapped_at_commit: "1cb5a15"
mapped_at: 2026-10-28
run:
  dev: { cmd: "go run ./cmd/api", port: 8080 }
  test: { cmd: "go test ./..." }
identity:
  ports: [{ value: 8080, evidence: ["cmd/api/main.go:11"] }]
  tables:
    - value: greetings
      columns: [id, message, created_at]
      evidence: ["migrations/001_greetings.sql:1-6"]
provides:
  - kind: http
    id: "GET /hello"
    response: { message: string, tags: "[]string" }
    evidence: ["internal/api/api.go:42"]
```

```yaml
version: 1
name: trama-demo-web
description: Frontend that shows the greeting from the API
stack: [node, react, typescript]
tools:
  node: "24"
mapped_at_commit: "697bd1b"
mapped_at: 2026-10-28
run:
  setup: { cmd: "npm install" }
  dev: { cmd: "npm run dev", port: 5173 }
  test: { cmd: "npm test" }
consumes:
  - kind: http
    id: "GET /hello"
    from: trama-demo-api
    via: env
    fields_used: [message]
    evidence: ["src/api/hello.ts:10", "src/api/hello.ts:15"]
config:
  - env: VITE_API_URL
    value: "http://localhost:8080"
    points_to: trama-demo-api
    evidence: [".env.example:1"]
```
