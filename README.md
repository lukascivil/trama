# trama

[Português](README.pt-BR.md)

Companies have many repositories that depend on each other — a frontend that calls a backend's API, services that share events — but each repository has a life of its own: its own tools, its own way to run locally, its own docs and team. Moving everything into a monorepo is expensive and often not possible. AI coding agents work inside one repository at a time, so they cannot tell that `front-app` depends on `backend-app`, answer "what breaks if I change this?", or plan a change that spans both.

**trama** is a local meta-harness: a layer above Claude Code that gathers a company's repositories into a **workspace** (a repository of its own, with no application code), maps how they relate with `file:line` evidence, knows how to run them, and coordinates changes across them. Nothing leaves your machine except what already goes to GitHub. The name is Portuguese for the weft of a fabric: every repository is a thread, trama is what ties them together.

## Status

Pre-release. Phases 1 to 4 of 5 are done: workspaces, `add`, `sync`, `status`, catalog cards with evidence, validation, the dependency graph, running the repositories (`setup`, `dev`, `test`, `up`), impact analysis and changes across repositories. A session-start check and the teammate flow come next.

## Requirements

- [git](https://git-scm.com) and the [GitHub CLI](https://cli.github.com) (`gh`), signed in
- [mise](https://mise.jdx.dev)
- [Claude Code](https://code.claude.com)

## Install the plugin

In Claude Code:

```
/plugin marketplace add lukascivil/trama
/plugin install trama@trama
```

The plugin puts a `trama` command on Claude's `PATH` that runs, through mise, the trama binary with the same version as the plugin.

## Create a workspace

Open Claude Code in an empty directory and run `/trama:init`. It creates:

```
workspace.yaml        the repositories in this workspace
mise.toml             pins the trama binary
AGENTS.md, CLAUDE.md  instructions for agents
.claude/settings.json enables the trama plugin for everyone who clones the workspace
catalog/ docs/ changes/
repos/                the clones (ignored by git)
```

## Map the repositories

In Claude Code, from the workspace:

- `/trama:add-repo <url>` adds a repository, clones it and maps it.
- `/trama:map <repo>` (or `--all`) reads a repository and writes its catalog card, `catalog/<repo>.yaml`: stack, how to run it, what it provides and consumes, and the `file:line` evidence for each fact. A consumer is linked to a provider only with proof (a config value matching the provider's host or port, a package, a topic, or a link declared in `workspace.yaml`); otherwise the relation stays `from: unknown` with its candidates.

Then `trama graph` writes `docs/graph.md` and the repository table in `AGENTS.md`. The skills never commit: review and commit `catalog/`, `docs/graph.md` and `AGENTS.md` yourself.

## Change across repositories

- `/trama:impact <change>` tells, without changing anything, which repositories and lines a change affects, whether it breaks them, and the safe order to ship it (expand → migrate → contract).
- `/trama:change <change>` writes a record in `changes/`, asks you to approve it, creates a `trama/<slug>` branch in each repository from its up-to-date main branch, makes the change, runs each repository's tests and opens linked pull requests in merge order. It never pushes to a main branch and never merges.
- After the merges, `trama changes` updates the record from the pull requests, and `/trama:map --all` refreshes the cards.

## Commands

Run them from anywhere inside the workspace (in your own shell, through `mise exec -- trama ...` or with mise activated):

| Command | What it does |
|---|---|
| `trama add <url>` | Adds a repository to `workspace.yaml`. Detects its default branch. Does not clone. |
| `trama sync [repo...]` | Clones missing repositories. Fetches the others and fast-forwards them only when they are clean and on their main branch. Never discards local changes. |
| `trama status [--quiet]` | Shows branch, local changes, distance from origin, catalog card state and unproven relations for every repository. Offline. `--quiet` prints only problems. |
| `trama validate` | Checks every catalog card: required fields, evidence that exists in the clone, and proof for every relation. |
| `trama stamp <repo>` | Records in the card the commit it describes (`origin/<branch>`). Refuses unless the clone is clean and up to date. |
| `trama graph` | Writes `docs/graph.md` (Mermaid) and the table between the markers in `AGENTS.md`. Refuses an invalid catalog. |
| `trama setup [repo...]` | Runs `run.setup` from each card (e.g. `npm install`) and remembers it, so `dev`, `test` and `up` only repeat it after new commits, a new command or new tool versions. |
| `trama dev <repo>` | Runs `run.dev` in the foreground, after the setup when needed, with the card's runtime versions through `mise exec`. |
| `trama test [repo...]` | Runs `run.test` of each repository, after its setup when needed, and prints `passed` or `failed` for each. |
| `trama up [repo...]` | Runs `run.dev` of every repository at once, each line prefixed with its name. Refuses to start when a dev port is already in use. Ctrl+C stops every process, children included. |
| `trama uses <id>` | Lists the repositories that provide or consume an id (e.g. `"GET /hello"`), with their evidence. |
| `trama branch <repo> <name>` | Creates a branch from the up-to-date main branch of a repository; refuses a repository with local changes. `--main` switches back. |
| `trama pr <repo> --title T --body-file F` | Pushes the current branch and opens a pull request with `gh`. Refuses the main branch. |
| `trama changes [--no-push]` | Updates every open change record from its pull requests, then commits and pushes the records. `trama changes save <id>` commits and pushes one record. |
| `trama init [dir]` | Creates the workspace files (what `/trama:init` runs). |
| `trama version` | Prints the version. |

## Development

```bash
go test ./...
```

To try the plugin from this checkout with a local build:

```bash
go build -o /tmp/trama-dev ./cmd/trama
TRAMA_BIN=/tmp/trama-dev claude --plugin-dir .
```

## Releasing

The plugin is served from the default branch and runs the binary with its own version, so the binaries must exist before `main` changes:

1. Set the new version in `.claude-plugin/plugin.json` and commit it.
2. Push the tag `v<version>`. The release workflow runs the tests, refuses a tag that does not match the plugin version, and publishes the binaries with GoReleaser.
3. When the release exists, push `main`.
4. Bump the `github:lukascivil/trama` pin in each workspace's `mise.toml`.

## Known limitations

- `trama add` rewrites `workspace.yaml` without its blank lines. Comments and unknown fields are kept.
- `trama stamp` rewrites the card the same way: blank lines go and inline maps are compacted (`{ cmd: x }` becomes `{cmd: x}`).

## License

[MIT](LICENSE)
