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
- If the card already exists, run `git -C repos/<name> diff --stat <mapped_at_commit> HEAD` to see what changed. Still re-check every evidence line already in the card.

## 3. Write the card

Write `catalog/<name>.yaml` following `${CLAUDE_PLUGIN_ROOT}/skills/map/card-reference.md`. For every entry under `identity`, `provides`, `consumes` and `config`:
- Open the file and read the line before you cite it as `path:line`, with the path relative to the repository root.
- If you cannot point at a line, leave the entry out.
- Leave `mapped_at_commit` and `mapped_at` out: `trama stamp` writes them in step 5.

Look for:
- **provides**: HTTP routes it serves (not CORS preflight handling), events it publishes, packages it publishes to a registry (a private package or internal-only code publishes nothing).
- **identity**: ports it listens on (cite the line that sets the address; when the code relies on a framework default, cite the line of the repository's own docs that states the port); hosts listed for it under `hosts` in `workspace.yaml` (evidence `workspace.yaml`); the package name it publishes; topics it publishes.
- **consumes**: HTTP calls to other services, events it subscribes to, internal packages it imports. Add `fields_used` when the code reads specific response fields.
- **config**: environment variables that hold another service's address, with their example value. Include them even when nothing in this repository calls that service (a CORS origin, for example): they still show which repositories know about each other.
- **run**, **tools** and **stack**: how to set it up, run it (with its port) and test it; the runtime versions from its version files; the languages and the main frameworks and build tools. Prefer what its own docs say.

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
- for each repository, the relations added, removed or changed (`git diff -- catalog/` shows them; a card that is new, or untracked in a workspace without commits, has every relation added);
- every `from: unknown` left, with its candidates and what would prove it (a config value, a declared host, a `links` entry);
- that nothing was committed: they review and commit `catalog/`, `docs/graph.md` and `AGENTS.md`.
