---
description: Map one repository, or all of them, of a trama workspace into catalog cards with file:line evidence for every fact. Use when the user asks to map, re-map or refresh the catalog, or when trama status says a card is missing or stale.
---

# Map repositories into the catalog

Arguments: a repository name, or `--all` ($ARGUMENTS). Work from the workspace root, the directory that holds `workspace.yaml`.

A card describes the repository's main branch, and every fact in it is proven by a line you read yourself. A card that guesses is worse than no card.

## 1. Check each repository

1. Run `trama sync <name>` (with `--all`, run `trama sync` once, with no name).
2. Run `trama status`. The repository must be on its main branch, clean and `+0 -0`. If it is not, skip it and tell the user what is in the way. Never stash, reset or switch branches yourself.

## 2. Read the repository

In `repos/<name>`:
- Read its `AGENTS.md`, `CLAUDE.md` and `README` first. They say how the team runs and tests it.
- Read the manifests and version files that exist: `go.mod`, `package.json`, `.nvmrc`, `.tool-versions`, `.python-version`, `pyproject.toml`, `pom.xml`, `build.gradle`, `Dockerfile`, `compose.yaml`, `.env.example`.
- Find the route table and the OpenAPI spec, if there is one: a committed `openapi.yaml` / `openapi.json` / `docs/*-api.yaml`, or the module that builds one at run time. Together with the router they are how you list every route the repository serves.
- If the card already exists, run `git -C repos/<name> diff --stat <mapped_at_commit> HEAD` to see what changed. Still re-check every evidence line already in the card.

## 3. Write the card

Write `catalog/<name>.yaml` following `${CLAUDE_PLUGIN_ROOT}/skills/map/card-reference.md`. For every entry under `identity`, `provides`, `consumes` and `config`:
- Open the file and read the line before you cite it as `path:line`, with the path relative to the repository root.
- If you cannot point at a line, leave the entry out.
- Leave `mapped_at_commit` and `mapped_at` out: `trama stamp` writes them in step 5.

Look for:
- **provides**: list **every** route the repository serves (not CORS preflight handling), every topic it publishes, and every package other repositories import by name: published to any registry, public or the company's private one, or a Go module with packages others can import (not only `main` and `internal/`). Skip only packages marked never to be published, such as `"private": true` in `package.json`.

  This list is what links other repositories to this one, so a route left out is a relation that stays unproven in someone else's card. Read the routes from the router (a `router.ex` scope tree, `huma.Register` calls, a framework's route table) or from the repository's OpenAPI spec, and cite the line you read. A spec's `servers:` usually points at a shared gateway, so it says nothing about who serves the route: take the paths from it, not the identity. A topic this repository publishes goes here with `kind: event` as well as in `identity.topics`.

  Add `request` and `response` to a route when you can read its body: the handler's input and output types, the DTO or the validation schema it uses, or the repository's OpenAPI spec — citing the line you read, as everywhere else. These are what answer "does this field already come from an endpoint?", so write the fields the endpoint really returns, not the columns behind it. When you could not read the body, leave the map out instead of guessing; `contract` stays the pointer to the full spec. For `kind: event`, `response` is the payload it publishes.
- **identity**: ports it listens on (cite the line that sets the address; when the code relies on a framework default, cite the line of the repository's own docs that states the port); hosts listed for it under `hosts` in `workspace.yaml` (evidence `workspace.yaml`); the package name it publishes; topics it publishes; the tables it owns.
- **identity.tables**: the tables this repository creates, with their columns. Read them where the schema is declared, and cite the line that creates the table, never one that queries it: a migration's `CREATE TABLE`, a declarative schema (`schema.prisma`, `schema.rb`, a `.sql` schema file), or the ORM entity class that maps the table. List the columns the table has, not the ones the code happens to read. A repository with no database of its own gets no `tables`.

  This is what answers "who owns this piece of data?" when no endpoint serves it yet, so a table left out is a question nobody can answer from the workspace. It is not a relation: it never goes in `provides` and links nothing.
- **consumes**: HTTP calls to services the company owns (a repository of this workspace, or one that could be added), events it subscribes to, internal packages it imports. Skip third-party APIs such as payment, email or cloud providers. Add `fields_used` when the code reads specific response fields.
- **config**: environment variables that hold another service's address, with their example value. Include them even when nothing in this repository calls that service (a CORS origin, for example): they still show which repositories know about each other. Set `points_to` only to a repository of this workspace; when that service is not a repository here yet, leave `points_to` out.
- **run**, **tools** and **stack**: how to set it up, run it (with its port) and test it; the runtime versions from its version files; the languages and the main frameworks and build tools. Prefer what its own docs say.

## 4. Link consumers to providers

Write every consume with `from: unknown` and no `via`, then run `trama link`.

Linking is mechanical, so the code does it: `trama link` gives a consume the one repository of the workspace whose card provides the same thing, and records how it knows — `via: route` for a route, `via: topic` for a topic, `via: package` for a package. Routes are matched ignoring the method's case, a trailing slash and the name of each path parameter, so the consumer's `GET /orders/{id}` and the router's `GET /orders/{order_id}` are one route. When two repositories provide the same thing, the link stays `unknown` and both go in `candidates`, because then the route does not say which one answers.

Never write `from` by hand because the names look alike, and never write `via: route`, `via: topic` or `via: package` yourself. What `trama link` leaves behind is yours to look at:

- **Nothing provides it.** Either the provider is not a repository of this workspace — a third party, or a service nobody has added yet — or the provider's card is missing that route. Open the provider's card first: an incomplete `provides` is the usual cause.
- **Two candidates.** Prove it another way, or leave it. `via: env` when a `config` entry of this card points to one of them with a value matching its host or port; `via: declared` when you add a `links` entry to `workspace.yaml`. Both are proofs you write yourself, with evidence.

When mapping several repositories, write all the cards first and run `trama link` once at the end: a provider mapped later resolves the consumes already written. Linking edits do not need a new stamp.

## 5. Stamp, validate, graph

1. Run `trama stamp <name>` for each card you wrote. It records the commit the card describes. Stamp again only when you change a card's own facts after stamping.
2. Run `trama link` if you have not already, and read its output.
3. Run `trama validate` and fix every error from the code. Never silence an error with evidence you did not read: remove the entry, or use `from: unknown`.
4. When there are no errors, run `trama graph`.

## 6. Report

Tell the user:
- for each repository, the relations added, removed or changed (`git diff -- catalog/` shows them; a card that is new, or untracked in a workspace without commits, has every relation added);
- what `trama link` resolved, and how many;
- every `from: unknown` left, split into the ones nobody provides and the ones with candidates, and what would settle each (a route missing from a provider's card, a config value, a `links` entry);
- that nothing was committed: they review and commit `catalog/`, `docs/graph.md` and `AGENTS.md`.
