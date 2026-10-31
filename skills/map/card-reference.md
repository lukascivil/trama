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
| `topics` | `{value: orders.created, evidence: [...]}` | the line that publishes to the topic |

### provides

`{kind, id, contract, shape, evidence}`

- `kind`: `http`, `event` or `package`.
- `id`: for `http`, `METHOD /path` with METHOD one of `GET POST PUT PATCH DELETE HEAD OPTIONS`, path parameters as `{name}` (never `:name`) and no query string, e.g. `GET /orders/{id}`. For `event`, the topic. For `package`, the package name.
- `contract`: optional path to an OpenAPI, GraphQL or proto file.
- `shape`: optional response fields, e.g. `{message: string}`.

### consumes

`{kind, id, from, via, candidates, fields_used, evidence}`

- `kind` and `id`: same rules as `provides`, and equal to the provider's entry.
- `from`: the providing repository, or `unknown`.
- `via`: the proof that `from` is right. Required when `from` is a repository, empty when it is `unknown`:
  - `env`: a `config` entry of this card has `points_to: <from>` and a `value` whose host matches one of the provider's `identity.hosts`, or whose port matches one of its `identity.ports` with host `localhost`, `127.0.0.1` or `0.0.0.0`.
  - `package`: `kind: package` and `id` equal to the provider's `identity.package`.
  - `topic`: `kind: event` and `id` listed in the provider's `identity.topics`.
  - `declared`: `workspace.yaml` has `links: [{consumer: <this>, provider: <from>, id: <id>}]`.
- `candidates`: with `from: unknown`, the repositories whose cards provide the same `id`. Leave the key out when there are none.
- `fields_used`: optional response fields the code reads.

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
provides:
  - kind: http
    id: "GET /hello"
    shape: { message: string }
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
