# trama

[Português](README.pt-BR.md)

Companies have many repositories that depend on each other — a frontend that calls a backend's API, services that share events — but each repository has a life of its own: its own tools, its own way to run locally, its own docs and team. Moving everything into a monorepo is expensive and often not possible. AI coding agents work inside one repository at a time, so they cannot tell that `front-app` depends on `backend-app`, answer "what breaks if I change this?", or plan a change that spans both.

**trama** is a local meta-harness: a layer above Claude Code that gathers a company's repositories into a **workspace** (a repository of its own, with no application code), maps how they relate with `file:line` evidence, knows how to run them, and coordinates changes across them. Nothing leaves your machine except what already goes to GitHub. The name is Portuguese for the weft of a fabric: every repository is a thread, trama is what ties them together.

## Status

Pre-release. Phase 1 of 5 is done: workspaces, `add`, `sync` and `status`. Catalog cards, the dependency graph, impact analysis and cross-repository changes come next.

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

## Commands

Run them from anywhere inside the workspace (in your own shell, through `mise exec -- trama ...` or with mise activated):

| Command | What it does |
|---|---|
| `trama add <url>` | Adds a repository to `workspace.yaml`. Detects its default branch. Does not clone. |
| `trama sync [repo...]` | Clones missing repositories. Fetches the others and fast-forwards them only when they are clean and on their main branch. Never discards local changes. |
| `trama status [--quiet]` | Shows branch, local changes, distance from origin and catalog card state for every repository. Offline. `--quiet` prints only problems. |
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

Set the same version in `.claude-plugin/plugin.json` and in the tag (`v<version>`), then push the tag. The release workflow refuses a tag that does not match the plugin version and publishes the binaries with GoReleaser.

## License

[MIT](LICENSE)
