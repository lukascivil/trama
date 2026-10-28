---
description: Map one repository, or all of them, of a trama workspace into catalog cards with file:line evidence for every fact. Use when the user asks to map, re-map or refresh the catalog, or when trama status says a card is missing or stale.
---

# Map repositories into the catalog

Arguments: a repository name, or `--all` ($ARGUMENTS). Work from the workspace root, the directory that holds `workspace.yaml`.

A card describes the repository's main branch, and every fact in it is proven by a line you read yourself. A card that guesses is worse than no card.

## 1. Check each repository

1. Run `trama sync <name>`.
2. Run `trama status`. The repository must be on its main branch, clean and `+0 -0`. If it is not, skip it and tell the user what is in the way. Never stash, reset or switch branches yourself.

## 2. Read the repository

In `repos/<name>`:
- Read its `AGENTS.md`, `CLAUDE.md` and `README` first. They say how the team runs and tests it.
- Read the manifests and version files that exist: `go.mod`, `package.json`, `.nvmrc`, `.tool-versions`, `.python-version`, `pyproject.toml`, `pom.xml`, `build.gradle`, `Dockerfile`, `compose.yaml`, `.env.example`.
- If the card already exists, run `git -C repos/<name> diff --stat <mapped_at_commit> HEAD` to see what changed. Still re-check every evidence line already in the card.

## 3. Write the card

Write `catalog/<name>.yaml` following `${CLAUDE_PLUGIN_ROOT}/skills/map/card-reference.md`. For every entry under `identity`, `provides`, `consumes` and `config`:
- Open the file and read the line before you cite it as `path:line`, with the path relative to the repository root.
- If you cannot point at a line, leave the entry out.

Look for:
- **provides**: HTTP routes it serves, events it publishes, packages it publishes.
- **identity**: ports it listens on; hosts listed for it under `hosts` in `workspace.yaml` (evidence `workspace.yaml`); the package name it publishes; topics it publishes.
- **consumes**: HTTP calls to other services, events it subscribes to, internal packages it imports. Add `fields_used` when the code reads specific response fields.
- **config**: environment variables that hold another service's address, with their example value.
- **run** and **tools**: how to set it up, run it (with its port) and test it, and the runtime versions from its version files. Prefer what its own docs say.

## 4. Link consumers to providers

Read the other cards in `catalog/`. A consume gets `from: <repo>` only with its own proof, written in `via`:
- `env`: a `config` entry of this card points to that repository, and its value matches a host or port in that repository's `identity`.
- `package`: this repository imports the package named in that repository's `identity.package`.
- `topic`: this repository subscribes to a topic listed in that repository's `identity.topics`.
- `declared`: `workspace.yaml` has a `links` entry with this consumer, that provider and this `id`.

A matching `id` alone is not proof. Without proof, write `from: unknown` and list in `candidates` the repositories whose cards provide the same `id`.

When mapping several repositories, write all cards first, then revisit every `from: unknown`: a provider mapped later may now prove the link.

## 5. Stamp, validate, graph

1. Run `trama stamp <name>` for each card you wrote. It records the commit the card describes.
2. Run `trama validate` and fix every error from the code. Never silence an error with evidence you did not read: remove the entry, or use `from: unknown`.
3. When there are no errors, run `trama graph`.

## 6. Report

Tell the user:
- for each repository, the relations added, removed or changed (`git diff -- catalog/` shows them);
- every `from: unknown` left, with its candidates and what would prove it (a config value, a declared host, a `links` entry);
- that nothing was committed: they review and commit `catalog/`, `docs/graph.md` and `AGENTS.md`.
