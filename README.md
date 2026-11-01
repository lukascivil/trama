# trama

[Português](README.pt-BR.md) · [Website](https://lukascivil.github.io/trama/)

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

Only the first workspace needs those two lines. A workspace already created carries the marketplace and the plugin in its own `.claude/settings.json`, so whoever clones it opens Claude Code and has trama, with nothing to install.

## Update

Update the plugin in Claude Code. That is the whole update: the binary is not a second thing to keep in step, because the plugin's version is what decides which binary runs, and mise fetches it on the next command. `trama version` says which one you are on.

## Create a workspace

Open Claude Code in an empty directory and run `/trama:init`. It creates:

```
workspace.yaml        the repositories in this workspace
AGENTS.md, CLAUDE.md  instructions for agents
.claude/settings.json enables the trama plugin for everyone who clones the workspace
catalog/ docs/ changes/
repos/                the clones (ignored by git)
```

## Map the repositories

In Claude Code, from the workspace:

- `/trama:add-repo <url>` adds a repository, clones it and maps it. With no URL (or with an organization name) it opens a page on your machine that lists the organization's repositories on GitHub, with a filter and checkboxes, and then adds, clones and maps the ones you tick.
- `/trama:map <repo>` (or `--all`) reads a repository and writes its catalog card, `catalog/<repo>.yaml`: stack, how to run it, what it provides and consumes, and the `file:line` evidence for each fact. A consumer is linked to a provider by `trama link`, which is mechanical: the relation goes to the one repository of the workspace that serves that route, publishes that topic or owns that package, and routes match however each side spells a path parameter. When two repositories serve it, the relation stays `from: unknown` with both as candidates, for you to settle with a config value matching one provider's host or port, or a link declared in `workspace.yaml`.

Then `trama link` resolves the relations and `trama graph` writes `docs/graph.md` and the repository table in `AGENTS.md`. `/trama:map` never commits: review and commit `catalog/`, `docs/graph.md` and `AGENTS.md` yourself.

`trama graph --serve` opens an interactive graph of the workspace in your browser: every repository with its latest release (the highest semver tag) and every relation with its proof. Click a repository to see who may be affected if you release something there: direct consumers, with the `file:line` that proves it; indirect ones, through other repositories; and possible ones, from relations without proof. The search goes further than repositories and ids: type a field or a column, such as `email`, and it answers who holds that piece of data — first the endpoints that already return or accept it, then the tables that store it. A repository's panel shows the fields of each route it serves and the tables it owns. It also has filters and a focus on one repository's neighbors. Everything runs on your machine: the server listens only on `127.0.0.1` and the page loads nothing from the internet.

## Change across repositories

- `/trama:impact <change>` tells, without changing anything, which repositories and lines a change affects, whether it breaks them, and the safe order to ship it (expand → migrate → contract).
- `/trama:change <change>` writes a record in `changes/`, asks you to approve it, creates a `trama/<slug>` branch in each repository from its up-to-date main branch, makes the change, runs each repository's tests and opens linked pull requests in merge order. It never pushes to a main branch and never merges.
- The records in `changes/` are the only thing trama commits to the workspace itself, straight to its main branch (no pull request); it refuses when other workspace commits are waiting to be pushed, and stops when the push is rejected.
- After the merges, `trama changes` updates the record from the pull requests, and `/trama:map --all` refreshes the cards.

## Commands

Run them from anywhere inside the workspace (in your own shell, through `mise exec -- trama ...` or with mise activated):

| Command | What it does |
|---|---|
| `trama add <url>` | Adds a repository to `workspace.yaml`. Detects its default branch. Does not clone. |
| `trama add --pick [<org>]` | Opens a page on `127.0.0.1` that lists the repositories of a GitHub organization, with a filter and checkboxes, and adds the ones you tick to `workspace.yaml`. Prints the added names, one per line. The token comes from `gh`, or `GH_TOKEN` / `GITHUB_TOKEN`, and never leaves the process. Does not clone. |
| `trama sync [repo...]` | Clones missing repositories. Fetches the others and fast-forwards them only when they are clean and on their main branch. Never discards local changes. |
| `trama status [--quiet]` | Shows branch, local changes, distance from origin, catalog card state and unproven relations for every repository. Offline. `--quiet` prints only problems. |
| `trama validate` | Checks every catalog card: required fields, evidence that exists in the clone, and proof for every relation. |
| `trama stamp <repo>` | Records in the card the commit it describes (`origin/<branch>`). Refuses unless the clone is clean and up to date. |
| `trama link [--dry-run]` | Links every `from: unknown` to the one repository of the workspace that provides it and writes the proof in `via`. Leaves the candidates when more than one does. |
| `trama graph` | Writes `docs/graph.md` (Mermaid) and the table between the markers in `AGENTS.md`. Refuses an invalid catalog. |
| `trama graph --serve [--port <n>] [--no-open]` | Serves an interactive graph of the workspace on `127.0.0.1` (port 7300, or a free one) and opens the browser, until Ctrl+C. It only reads; validation errors show on the page. |
| `trama setup [repo...]` | Runs `run.setup` from each card (e.g. `npm install`) and remembers it, so `dev`, `test` and `up` only repeat it after new commits, a new command or new tool versions. |
| `trama dev <repo>` | Runs `run.dev` in the foreground, after the setup when needed, with the card's runtime versions through `mise exec`. |
| `trama test [repo...]` | Runs `run.test` of each repository, after its setup when needed, and prints `passed` or `failed` for each. |
| `trama up [repo...]` | Runs `run.dev` of every repository at once, each line prefixed with its name. Refuses to start when a dev port is already in use. Ctrl+C stops every process, children included. |
| `trama uses <id>` | Lists the repositories that provide or consume an id (e.g. `"GET /hello"`), with their evidence. |
| `trama data <term>` | Lists what holds a piece of data (e.g. `email`), with their evidence: first the endpoints whose `response` or `request` has a matching field, then the tables whose columns do. The order is the answer — an endpoint that already returns it can be called today; a table means the data exists and its owner has not exposed it yet. |
| `trama schemas <repo>` | Lists the files of a repository that declare tables: schema files (`schema.prisma`, `schema.rb`, `structure.sql`) first, then migrations, then `.sql` files that create a table. It only finds the files; `/trama:map` reads them and writes `identity.tables`. |
| `trama branch <repo> <name>` | Creates a branch from the up-to-date main branch of a repository; refuses a repository with local changes or on another branch. `--main` switches back. |
| `trama pr <repo> --title T --body-file F [--draft]` | Pushes the current branch and opens a pull request with `gh`. Refuses the main branch. |
| `trama changes [--no-push]` | Updates every open change record from its pull requests, then commits and pushes the records. `trama changes save <id>` commits and pushes one record. |
| `trama init [dir]` | Creates the workspace files (what `/trama:init` runs). |
| `trama version`, `trama --version` | Prints the version. |

## Development

```bash
go test ./...
```

To try the plugin from this checkout with a local build:

```bash
go build -o /tmp/trama-dev ./cmd/trama
TRAMA_BIN=/tmp/trama-dev claude --plugin-dir .
```

The website lives in `site/` (Astro). Every push to `main` that changes it publishes it to GitHub Pages:

```bash
cd site && mise exec -- npm install && mise exec -- npm run dev
```

## Releasing

The plugin is served from the default branch and runs the binary with its own version, so the binaries must exist before `main` changes:

1. Set the new version in `.claude-plugin/plugin.json` and commit it.
2. Push the tag `v<version>`. The release workflow runs the tests, refuses a tag that does not match the plugin version, and publishes the binaries with GoReleaser.
3. When the release exists, push `main`.

Workspaces need nothing: each one runs the binary that matches the plugin its user has.

## Known limitations

- `trama add` rewrites `workspace.yaml` without its blank lines. Comments and unknown fields are kept.
- `trama stamp` rewrites the card the same way: blank lines go and inline maps are compacted (`{ cmd: x }` becomes `{cmd: x}`).

## License

[MIT](LICENSE)
