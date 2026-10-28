# trama — Fase 1 (Esqueleto) — Plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Entregar o CLI `trama` em Go com `version`, `init`, `add`, `sync` e `status`; o plugin do Claude Code com `/trama:init`; o release `v0.0.1` instalável pelo mise; os dois repos de demo; e o workspace `trama-demo` sincronizando os dois.

**Architecture:** O repo `trama` contém o CLI (`cmd/trama`, `internal/...`) e o plugin (`.claude-plugin/`, `skills/`, `bin/`). O CLI chama o `git` da máquina. O plugin traz um shim `bin/trama` que roda, via `mise exec`, o binário com a mesma versão do plugin. Os repos de demo são independentes e não sabem da trama.

**Tech Stack:** Go (cobra, `go.yaml.in/yaml/v3`), git, mise, GoReleaser, GitHub Actions. Demo: Go stdlib; Vite + React + TypeScript + Vitest.

**Spec:** `docs/superpowers/specs/2026-10-27-trama-design.md`. Esta fase é a fase 1 da §11 e cobre §5, §6.1, `sync`/`status` da §7, `/trama:init` da §8 e a §9 (menos a tag `start`).

**Fora desta fase:** fichas completas, `map`, `validate`, `graph`, `impact`, `change`, `changes`, `setup`/`dev`/`up`/`test`, hook de sessão. O `status` ainda não lista pendências `from: unknown` nem mudanças abertas (fases 2 e 4). A tag `start` dos repos de demo vai para o plano da fase 4, logo antes da primeira mudança, para não precisar mover uma tag pública.

## Global Constraints

- Módulo `github.com/lukascivil/trama`; `go.mod` com `go 1.25` (piso); toolchain de desenvolvimento pelo `mise.toml` do repo (`go = "1.27"`).
- Dependências Go permitidas: `github.com/spf13/cobra` e `go.yaml.in/yaml/v3`. O resto é stdlib.
- Todo git roda pelo binário `git` com `GIT_TERMINAL_PROMPT=0` e `LC_ALL=C`. Nenhum código usa `reset`, `checkout -- `, `clean`, `stash` ou `push --force`.
- Nunca descartar alteração local de ninguém.
- Conteúdo dos repos (código, mensagens, skills, README) em inglês; `README.pt-BR.md` na `trama`. Licença MIT, `Copyright (c) 2026 lukascivil`.
- Repos clonados em `<workspace>/repos/<name>`. Nome de repo válido: `^[a-z0-9][a-z0-9._-]*$`.
- Versão do binário via `-ldflags "-X github.com/lukascivil/trama/internal/cli.version=<versão>"`; valor padrão `dev`. A versão em `.claude-plugin/plugin.json` é igual à tag sem o `v`.
- Plataformas: macOS e Linux.
- ⚠️ Criar repo no GitHub, fazer push, criar tag ou release são ações públicas: **pare e confirme com o usuário imediatamente antes de cada uma**.

## Review Focus

1. **Repo privado ou URL errada no `add`/`sync`** → falha rápida com mensagem clara, sem travar esperando senha; no `sync`, os outros repos continuam. (Testes: Task 2 `TestDefaultBranchFailsFastOnBadURL`, Task 6 `TestSyncContinuesAfterFailure`.)
2. **`repos/<name>` já existe e não é git, ou aponta para outro remoto** → nunca é tocado; aparece como pulado com o motivo. (Task 6 `TestSyncLeavesNonRepoDirAlone`, `TestSyncOriginMismatch`.)
3. **Branch principal diferente de `main`** (`master`, `develop`) → `add` detecta e `sync`/`status` usam a branch certa. (Task 2 `TestCloneAndDefaultBranch`, Task 5 `TestAddDetectsDefaultBranch`.)
4. **Comando rodado de dentro de uma subpasta** (`repos/x/src`, `catalog/`) → acha a raiz do workspace subindo as pastas. (Task 3 `TestFindWalksUp`, Task 5 `TestAddFromSubdirectory`.)
5. **`workspace.yaml` editado à mão** (comentários, campos extras como `hosts`, nomes inválidos como `../evil`) → `add` preserva comentários e campos; `Load` recusa nomes inseguros. (Task 3 `TestAddRepoPreservesCommentsAndUnknownFields`, `TestLoadRejectsBadNames`.)

---

### Task 1: Repo `trama` e comando `version`

**Files:**
- Create: `go.mod`, `mise.toml`, `.gitignore`, `LICENSE`, `cmd/trama/main.go`, `internal/cli/root.go`, `internal/cli/version.go`
- Test: `internal/cli/version_test.go`

**Interfaces:**
- Produces: `func Execute() int` (package `cli`); `func newRootCmd() *cobra.Command`; `var version = "dev"`; flag persistente `--dir`/`-C string` ("run as if trama was started in this directory").

- [ ] **Step 1: Iniciar o repo** (a pasta já tem `docs/` com o spec e este plano)

```bash
cd ~/Documents/GitHub/trama && git init -b main
```

- [ ] **Step 2: Ferramentas de desenvolvimento** — criar `mise.toml`:

```toml
[tools]
go = "1.27"
goreleaser = "2"
```

Run: `mise trust && mise install && go version` → Expected: `go1.27.x`.

- [ ] **Step 3: Módulo**

```bash
go mod init github.com/lukascivil/trama && go mod edit -go=1.25 && go get github.com/spf13/cobra@latest
```

- [ ] **Step 4: Teste que falha** — `internal/cli/version_test.go`:

```go
func TestVersionCommandPrintsVersion(t *testing.T) {
	version = "1.2.3"
	t.Cleanup(func() { version = "dev" })
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "1.2.3\n" {
		t.Fatalf("got %q, want %q", got, "1.2.3\n")
	}
}
```

- [ ] **Step 5:** Run `go test ./internal/cli/ -run TestVersion` → Expected: FAIL (`undefined: newRootCmd`).

- [ ] **Step 6: Implementar** `newRootCmd()` (Use `trama`, Short `Local meta-harness for many repositories`, `SilenceUsage` e `SilenceErrors` true) e o subcomando `version` que escreve `version + "\n"` em `cmd.OutOrStdout()`. `Execute()` roda o root, imprime `trama: <erro>` no stderr e devolve 1 em erro, 0 em sucesso. `cmd/trama/main.go`: `os.Exit(cli.Execute())`.

- [ ] **Step 7:** Run `go test ./... && go run ./cmd/trama version` → Expected: PASS e `dev`.

- [ ] **Step 8:** `.gitignore` com `/dist/` e `/trama`; `LICENSE` MIT com `Copyright (c) 2026 lukascivil`.

- [ ] **Step 9: Commit**

```bash
git add -A && git commit -m "feat: bootstrap trama CLI with version command"
```

---

### Task 2: Wrapper do git e helpers de teste

**Files:**
- Create: `internal/gitx/git.go`, `internal/testutil/git.go`
- Test: `internal/gitx/git_test.go`

**Interfaces:**
- Produces (`package gitx`):

```go
var ErrDiverged = errors.New("cannot fast-forward: local branch diverged from origin")

func Init(dir, branch string) error
func Clone(url, branch, dir string) error
func DefaultBranch(url string) (string, error)
func SameRemote(a, b string) bool

type Repo struct{ Dir string }

func (r Repo) IsRepo() bool
func (r Repo) OriginURL() (string, error)
func (r Repo) Fetch() error                                  // git fetch origin --prune
func (r Repo) CurrentBranch() (string, error)                // "" when HEAD is detached
func (r Repo) IsClean() (bool, error)                        // untracked files count as not clean
func (r Repo) FastForward(branch string) (moved bool, err error) // ErrDiverged when not possible
func (r Repo) RevParse(ref string) (string, error)           // full SHA
func (r Repo) AheadBehind(local, upstream string) (ahead, behind int, err error)
```

- Produces (`package testutil`):

```go
func Git(t testing.TB, dir string, args ...string) string // test identity; fails the test on error; trimmed stdout
type Remote struct{ URL string } // URL is the path of a bare repo
func NewRemote(t testing.TB, branch string) *Remote       // bare repo with one commit (README.md) on branch
func (r *Remote) Commit(t testing.TB, path, content string) string // commits and pushes to that branch; returns SHA
```

- [ ] **Step 1: Testes que falham** — `internal/gitx/git_test.go`:

```go
func TestCloneAndDefaultBranch(t *testing.T) {
	remote := testutil.NewRemote(t, "master")
	b, err := gitx.DefaultBranch(remote.URL)
	if err != nil || b != "master" {
		t.Fatalf("DefaultBranch = %q, %v; want master", b, err)
	}
	dir := filepath.Join(t.TempDir(), "clone")
	if err := gitx.Clone(remote.URL, "master", dir); err != nil {
		t.Fatal(err)
	}
	r := gitx.Repo{Dir: dir}
	if !r.IsRepo() { t.Fatal("not a repo") }
	if got, _ := r.CurrentBranch(); got != "master" { t.Fatalf("branch = %q", got) }
	if o, _ := r.OriginURL(); !gitx.SameRemote(o, remote.URL) { t.Fatalf("origin = %q", o) }
}

func TestDefaultBranchFailsFastOnBadURL(t *testing.T) {
	_, err := gitx.DefaultBranch(filepath.Join(t.TempDir(), "missing"))
	if err == nil { t.Fatal("want error") }
}

func TestFastForwardMovesHead(t *testing.T)      // clone; remote.Commit; Fetch; FastForward("main") == true and HEAD == new SHA; second call == false
func TestFastForwardDiverged(t *testing.T)       // local commit in clone + remote.Commit; Fetch; errors.Is(err, ErrDiverged); HEAD still the local commit
func TestIsCleanCountsUntracked(t *testing.T)    // new untracked file → IsClean() == false
func TestAheadBehind(t *testing.T)               // two remote commits + Fetch → AheadBehind("main", "origin/main") == (0, 2)

func TestSameRemote(t *testing.T) {
	cases := []struct{ a, b string; want bool }{
		{"https://github.com/a/b.git", "https://github.com/a/b", true},
		{"https://github.com/A/b/", "https://github.com/a/b.git", true},
		{"https://github.com/a/b", "https://github.com/a/c", false},
	}
	for _, c := range cases {
		if got := gitx.SameRemote(c.a, c.b); got != c.want {
			t.Errorf("SameRemote(%q, %q) = %v", c.a, c.b, got)
		}
	}
}
```

- [ ] **Step 2:** Run `go test ./internal/gitx/` → Expected: FAIL (pacotes não existem).

- [ ] **Step 3: Implementar `testutil`.** `Git` roda `git -c user.name=trama-test -c user.email=test@example.invalid -c commit.gpgsign=false -c init.defaultBranch=<branch>` (identidade só via `-c`, nada global). `NewRemote` cria um bare repo em `t.TempDir()`, um clone de trabalho, commita `README.md` e faz push.

- [ ] **Step 4: Implementar `gitx`.** Um helper `run(dir string, args ...string) (string, error)` com `exec.Command("git", ...)`, `Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")` e erro `fmt.Errorf("git %s: %s", strings.Join(args, " "), stderr)`.
  - `DefaultBranch` lê a linha `ref: refs/heads/<b>\tHEAD` de `git ls-remote --symref <url> HEAD`.
  - `FastForward` compara `HEAD` antes e depois de `git merge --ff-only origin/<branch>`. Se o merge falhar e `git merge-base --is-ancestor HEAD origin/<branch>` também falhar, devolve `ErrDiverged`.
  - `AheadBehind` usa `git rev-list --left-right --count <local>...<upstream>`.
  - `SameRemote` compara depois de tirar espaços, `/` e `.git` do final, sem diferenciar maiúsculas.

- [ ] **Step 5:** Run `go test ./internal/gitx/` → Expected: PASS.

- [ ] **Step 6: Commit** — `git add -A && git commit -m "feat: add git wrapper and test helpers"`

---

### Task 3: `workspace.yaml`

**Files:**
- Create: `internal/workspace/workspace.go`
- Test: `internal/workspace/workspace_test.go`

**Interfaces:**
- Consumes: `gitx.SameRemote`.
- Produces (`package workspace`):

```go
const FileName = "workspace.yaml"

var ErrNotFound = errors.New("not inside a trama workspace: no workspace.yaml found here or in any parent directory")

type Repo struct {
	Name   string   `yaml:"name"`
	URL    string   `yaml:"url"`
	Branch string   `yaml:"branch"`
	Hosts  []string `yaml:"hosts,omitempty"`
}

type Workspace struct {
	Version int    `yaml:"version"`
	Name    string `yaml:"name"`
	Repos   []Repo `yaml:"repos"`
	Root    string `yaml:"-"` // absolute directory that holds workspace.yaml
	// unexported: the parsed yaml.Node document, edited in place by AddRepo
}

func Find(start string) (string, error)
func Load(root string) (*Workspace, error) // parses and calls Validate
func NameFromURL(url string) string
func (w *Workspace) Validate() error
func (w *Workspace) Repo(name string) (Repo, bool)
func (w *Workspace) RepoDir(name string) string // filepath.Join(w.Root, "repos", name)
func (w *Workspace) AddRepo(r Repo) error
func (w *Workspace) Save() error
```

Regras de `Validate`: `version` precisa ser 1 (`unsupported workspace version N (expected 1)`); `name` não vazio; cada repo com nome válido (`invalid repo name "<n>"`), nomes únicos, `url` e `branch` não vazios, URLs únicas por `SameRemote`.

- [ ] **Step 1: Testes que falham:**

```go
func TestFindWalksUp(t *testing.T)            // root/workspace.yaml; Find(root/repos/x/src) == root
func TestFindNotFound(t *testing.T)           // errors.Is(err, ErrNotFound)
func TestLoadValid(t *testing.T)              // two repos; fields and Root set
func TestLoadRejectsUnknownVersion(t *testing.T) // version: 2 → error mentions "unsupported workspace version 2"
func TestLoadRejectsDuplicates(t *testing.T)  // same name twice → error

func TestLoadRejectsBadNames(t *testing.T) {
	for _, name := range []string{"../evil", "a/b", "", "Upper", ".hidden"} {
		// write a workspace with this repo name; Load must fail with "invalid repo name"
	}
}

func TestAddRepoPreservesCommentsAndUnknownFields(t *testing.T) {
	src := `# my workspace
version: 1
name: demo
custom: keep-me
repos:
  - name: api # the backend
    url: https://github.com/x/api.git
    branch: main
    hosts: [api.example.com]
`
	// write src; Load; AddRepo(Repo{Name: "web", URL: "https://github.com/x/web.git", Branch: "main"}); Save
	// raw file must contain: "# my workspace", "custom: keep-me", "# the backend", "api.example.com", "name: web"
	// Load again → 2 repos, second is web
}

func TestAddRepoToEmptyFlowList(t *testing.T)  // "repos: []" → after AddRepo+Save the file contains "- name: web" (block style) and loads 1 repo
func TestAddRepoRejectsDuplicate(t *testing.T) // same name → error; same URL differing only by ".git" → error

func TestNameFromURL(t *testing.T) {
	cases := map[string]string{
		"https://github.com/lukascivil/trama-demo-api.git": "trama-demo-api",
		"git@github.com:org/My.Repo.git":                   "my.repo",
		"/tmp/x/remote.git":                                "remote",
	}
	// NameFromURL(k) == v
}
```

- [ ] **Step 2:** Run `go test ./internal/workspace/` → Expected: FAIL.

- [ ] **Step 3: Implementar.** `Load` decodifica o arquivo num `yaml.Node` e depois esse node no struct. `AddRepo` valida, acrescenta em `w.Repos` e acrescenta um mapping node na sequência `repos` do documento; se a sequência estiver em estilo flow (`[]`), troca para block. `Save` codifica o documento com indentação 2. Comentários e campos desconhecidos sobrevivem porque o documento nunca é regerado a partir do struct.

- [ ] **Step 4:** Run `go test ./internal/workspace/` → Expected: PASS.

- [ ] **Step 5: Commit** — `git add -A && git commit -m "feat: load and edit workspace.yaml"`

---

### Task 4: `trama init`

**Files:**
- Create: `internal/scaffold/scaffold.go`, `internal/scaffold/templates/workspace.yaml.tmpl`, `mise.toml.tmpl`, `gitignore`, `AGENTS.md.tmpl`, `CLAUDE.md`, `settings.json`, `internal/cli/init.go`
- Test: `internal/scaffold/scaffold_test.go`, `internal/cli/init_test.go`

**Interfaces:**
- Consumes: `gitx.Init`, `workspace.Load` (nos testes).
- Produces (`package scaffold`):

```go
type Options struct {
	Dir          string // target directory; must exist
	Name         string // workspace name; default filepath.Base(Dir)
	TramaVersion string // "dev" means: do not pin the binary in mise.toml
}

var ErrNotEmpty = errors.New("directory is not empty")

func Init(opts Options) (created []string, err error) // slash paths relative to Dir, sorted
```

- CLI: `trama init [dir] [--name N]`. O alvo é o argumento posicional; senão `--dir`; senão a pasta atual. Imprime cada arquivo criado e `Next: trama add <url>, then trama sync`.

Comportamento: as únicas entradas aceitas antes são `.git` e `.DS_Store`; qualquer outra coisa devolve `ErrNotEmpty` com os nomes encontrados, sem criar nada. Sem `.git`, roda `gitx.Init(Dir, "main")`. Não commita. Templates embutidos com `embed`.

Conteúdo exato dos templates:

`workspace.yaml.tmpl`
```yaml
version: 1
name: {{.Name}}
repos: []
```

`mise.toml.tmpl` (versão normal / versão `dev`)
```toml
[tools]
"github:lukascivil/trama" = "{{.TramaVersion}}"
```
```toml
# trama dev build: the trama binary is not pinned. Pin it with:
# "github:lukascivil/trama" = "<version>"
[tools]
```

`gitignore` → `repos/`

`CLAUDE.md` → `@AGENTS.md`

`settings.json` (gravado em `.claude/settings.json`)
```json
{
  "extraKnownMarketplaces": {
    "trama": {
      "source": { "source": "github", "repo": "lukascivil/trama" }
    }
  },
  "enabledPlugins": {
    "trama@trama": true
  }
}
```

`AGENTS.md.tmpl`
```markdown
# {{.Name}} — trama workspace

This repository is a [trama](https://github.com/lukascivil/trama) workspace. It holds no application code: it lists the repositories that work together, maps how they relate, and records changes that span them.

## Layout

- `workspace.yaml` — the repositories in this workspace.
- `catalog/<repo>.yaml` — one card per repository: stack, how to run it, what it provides and consumes, with `file:line` evidence.
- `docs/graph.md` — dependency graph generated from the catalog.
- `changes/` — one record per cross-repository change: why, plan, pull requests.
- `repos/` — the cloned repositories (ignored by git). Each keeps its own history and rules.

## Repositories

<!-- trama:begin -->
_No repositories mapped yet._
<!-- trama:end -->

## Rules for agents

- Before editing a repository, read its own `AGENTS.md` or `CLAUDE.md`.
- If a change touches anything listed under `provides`, `consumes` or `config` in the catalog, run `/trama:impact` first.
- Changes that span repositories go through `/trama:change`.
- Never commit code from `repos/` to this workspace repository.
- If `trama status` reports a catalog card as stale, do not trust it: run `/trama:map` first.
```

- [ ] **Step 1: Testes que falham** — `internal/scaffold/scaffold_test.go`:

```go
func TestInitCreatesLayout(t *testing.T) {
	dir := t.TempDir()
	created, err := scaffold.Init(scaffold.Options{Dir: dir, TramaVersion: "0.0.1"})
	if err != nil { t.Fatal(err) }
	want := []string{".claude/settings.json", ".gitignore", "AGENTS.md", "CLAUDE.md",
		"catalog/.gitkeep", "changes/.gitkeep", "docs/.gitkeep", "mise.toml", "workspace.yaml"}
	// created == want; dir/.git exists
	// workspace.Load(dir) succeeds with Name == filepath.Base(dir) and 0 repos
	// mise.toml contains `"github:lukascivil/trama" = "0.0.1"`
	// .claude/settings.json parses; enabledPlugins["trama@trama"] == true
}

func TestInitDevVersionDoesNotPin(t *testing.T)   // TramaVersion "dev" → no line of mise.toml starts with `"github:lukascivil/trama"`
func TestInitAllowsExistingGit(t *testing.T)      // dir with `git init` → no error
func TestInitRefusesNonEmpty(t *testing.T)        // README.md present → errors.Is(err, ErrNotEmpty), message contains "README.md", no workspace.yaml created
```

`internal/cli/init_test.go`:

```go
func TestInitCommandPinsBinaryVersion(t *testing.T) // version = "0.0.1"; args {"init", dir, "--name", "demo"} → workspace name "demo", mise.toml pins 0.0.1
```

- [ ] **Step 2:** Run `go test ./internal/scaffold/ ./internal/cli/` → Expected: FAIL.
- [ ] **Step 3:** Implementar `scaffold.Init` e o comando `init`.
- [ ] **Step 4:** Run `go test ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `git add -A && git commit -m "feat: add trama init"`

---

### Task 5: `trama add`

**Files:**
- Create: `internal/cli/workspace.go`, `internal/cli/add.go`
- Test: `internal/cli/add_test.go`

**Interfaces:**
- Consumes: `workspace.Find/Load/NameFromURL/AddRepo/Save`, `gitx.DefaultBranch`.
- Produces: `func loadWorkspace(cmd *cobra.Command) (*workspace.Workspace, error)`. Parte de `--dir` (ou `os.Getwd()`), chama `Find` e `Load`. Usado por `add`, `sync` e `status`.
- CLI: `trama add <url> [--name N] [--branch B]`. Sem `--branch`, usa `DefaultBranch(url)`; se falhar, o erro é `could not read the default branch of <url>: <err>` e nada é gravado. Sem `--name`, usa `NameFromURL`. Saída: `added <name> (<url>, branch <branch>)` e na linha seguinte `run "trama sync" to clone it`. Não clona.

- [ ] **Step 1: Testes que falham:**

```go
func TestAddDetectsDefaultBranch(t *testing.T)  // remote on "develop"; init ws; {"add", remote.URL, "--dir", ws} → repo "remote" with Branch "develop"
func TestAddFailsOnUnreachableURL(t *testing.T) // bad path → error; workspace.yaml bytes unchanged
func TestAddRejectsDuplicate(t *testing.T)      // adding the same URL twice → second call errors
func TestAddFromSubdirectory(t *testing.T)      // "--dir", ws+"/catalog" → repo added to ws/workspace.yaml
```

- [ ] **Step 2:** Run `go test ./internal/cli/ -run TestAdd` → Expected: FAIL.
- [ ] **Step 3:** Implementar `loadWorkspace` e `add`.
- [ ] **Step 4:** Run `go test ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `git add -A && git commit -m "feat: add trama add"`

---

### Task 6: `trama sync`

**Files:**
- Create: `internal/syncer/syncer.go`, `internal/cli/sync.go`
- Test: `internal/syncer/syncer_test.go`, `internal/cli/sync_test.go`

**Interfaces:**
- Consumes: `workspace.Workspace`, `gitx.*`.
- Produces (`package syncer`):

```go
type Outcome string

const (
	Cloned          Outcome = "cloned"
	Updated         Outcome = "updated"
	UpToDate        Outcome = "up-to-date"
	SkippedDirty    Outcome = "skipped-dirty"
	SkippedBranch   Outcome = "skipped-branch"
	SkippedDiverged Outcome = "skipped-diverged"
	SkippedNotRepo  Outcome = "skipped-not-a-repo"
	SkippedOrigin   Outcome = "skipped-origin-mismatch"
	Failed          Outcome = "failed"
)

type Result struct {
	Repo    string
	Outcome Outcome
	Detail  string // human-readable reason; empty for cloned and up-to-date
}

// Sync processes repos in workspace order. only filters by name; an unknown
// name returns an error before anything runs. Per-repo failures go in Result.
func Sync(ws *workspace.Workspace, only []string) ([]Result, error)
```

Decisão por repo (nesta ordem):

```
pasta não existe        → cria repos/ e clona                         → Cloned | Failed
existe e não é git      → não toca                                    → SkippedNotRepo  "repos/<n> exists but is not a git repository; left untouched"
origin ≠ url (SameRemote) → não toca                                  → SkippedOrigin   "origin is <a>, workspace.yaml says <b>; left untouched"
Fetch falhou            →                                             → Failed
branch atual ≠ principal → já buscou, não avança                      → SkippedBranch   "on branch <x>; fetched, not advanced"
alterações locais       → já buscou, não avança                       → SkippedDirty    "has local changes; fetched, not advanced"
FastForward             → ErrDiverged → SkippedDiverged "diverged from origin/<b>; fetched, not advanced"
                          moveu → Updated "<sha7 antigo>..<sha7 novo>"; não moveu → UpToDate
```

- CLI: `trama sync [repo...]`. Imprime `fmt.Fprintf(out, "%-24s %-24s %s\n", r.Repo, r.Outcome, r.Detail)` para cada resultado. Depois de imprimir todos, devolve o erro `sync failed for N repo(s)` se algum for `Failed`. Pulados não são erro.

- [ ] **Step 1: Testes que falham** (`syncer_test.go`, cada um com remotos do `testutil`):

```go
func TestSyncClonesMissing(t *testing.T)          // 2 repos → both Cloned; each on its branch
func TestSyncUpdatesCleanRepo(t *testing.T)       // clone; remote.Commit → Updated and HEAD == new SHA; again → UpToDate
func TestSyncNeverTouchesLocalChanges(t *testing.T) // edit a tracked file; remote.Commit → SkippedDirty; file keeps the edit; HEAD unchanged; origin/main == new SHA
func TestSyncSkipsOtherBranch(t *testing.T)       // checkout -b feature; remote.Commit → SkippedBranch; still on feature; origin/main updated
func TestSyncDiverged(t *testing.T)               // local commit + remote commit → SkippedDiverged; HEAD == local commit
func TestSyncLeavesNonRepoDirAlone(t *testing.T)  // repos/x/file.txt without git → SkippedNotRepo; file intact
func TestSyncOriginMismatch(t *testing.T)         // repos/x cloned from remote A, workspace says B → SkippedOrigin
func TestSyncContinuesAfterFailure(t *testing.T)  // [bad URL, good URL] → [Failed, Cloned]
func TestSyncOnlyFilter(t *testing.T)             // only {"b"} → only b; only {"zzz"} → error and nothing cloned
```

`internal/cli/sync_test.go`:

```go
func TestSyncCommandFailsWhenARepoFails(t *testing.T) // workspace with a bad URL → Execute returns error "sync failed for 1 repo(s)"; output lists the repo as failed
```

- [ ] **Step 2:** Run `go test ./internal/syncer/ ./internal/cli/` → Expected: FAIL.
- [ ] **Step 3:** Implementar `Sync` e o comando.
- [ ] **Step 4:** Run `go test ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `git add -A && git commit -m "feat: add trama sync"`

---

### Task 7: `trama status`

**Files:**
- Create: `internal/catalog/catalog.go`, `internal/status/status.go`, `internal/cli/status.go`
- Test: `internal/catalog/catalog_test.go`, `internal/status/status_test.go`, `internal/cli/e2e_test.go`

**Interfaces:**
- Consumes: `workspace.Workspace`, `gitx.Repo`, `syncer.Sync` (nos testes).
- Produces (`package catalog`). A ficha completa chega na fase 2; agora só o necessário:

```go
type Entry struct {
	Version        int    `yaml:"version"`
	Name           string `yaml:"name"`
	MappedAtCommit string `yaml:"mapped_at_commit"`
}

func Path(root, name string) string          // <root>/catalog/<name>.yaml
func Load(root, name string) (*Entry, error) // (nil, nil) when the file does not exist; unknown fields ignored
```

- Produces (`package status`):

```go
type Card string

const (
	CardMissing Card = "missing"
	CardCurrent Card = "current"
	CardStale   Card = "stale"
	CardUnknown Card = "unknown" // card exists but the repo is not cloned, so it cannot be checked
)

type RepoStatus struct {
	Name   string
	Cloned bool
	Branch string // current branch; "" when detached
	Main   string // main branch from workspace.yaml
	Clean  bool
	Ahead  int // local Main vs origin/Main
	Behind int
	Card   Card
	Err    string // repo could not be inspected
}

func Collect(ws *workspace.Workspace) []RepoStatus // local git only, never the network
func (s RepoStatus) Problems() []string
func Render(w io.Writer, sts []RepoStatus, quiet bool)
```

Regras:
- Ficha `stale` quando `mapped_at_commit` não resolve (`RevParse(sha + "^{commit}")` falha) ou é diferente de `RevParse("origin/" + Main)`. Igual → `current`. Sem arquivo → `missing`.
- `Problems()` devolve, nesta ordem e com este texto:
  - `not cloned: run "trama sync"`
  - o próprio `Err`
  - `<N> commit(s) behind origin/<main>: run "trama sync"`
  - `no catalog card: run "/trama:map <name>"`
  - `catalog card is stale: run "/trama:map <name>"`
- Branch atual e alterações locais são informação, não problema.
- `Render` sem `quiet`: uma linha por repo com nome, branch, `clean`/`dirty`, `+<ahead> -<behind>` e ficha; depois cada problema como `- <name>: <problema>`. Com `quiet`: só as linhas `trama: <name>: <problema>`, e nada se não houver problema.
- CLI: `trama status [--quiet]`. Sai com 0 mesmo havendo problemas; só sai com erro quando o workspace não carrega.

- [ ] **Step 1: Testes que falham:**

```go
// catalog_test.go
func TestLoadMissingReturnsNil(t *testing.T)
func TestLoadIgnoresUnknownFields(t *testing.T) // card shaped like spec §6.2 (provides, consumes, identity...) → MappedAtCommit parsed

// status_test.go
func TestStatusNotCloned(t *testing.T)               // Cloned false; Problems()[0] == `not cloned: run "trama sync"`
func TestStatusCardMissing(t *testing.T)             // synced, no card → CardMissing
func TestStatusCardCurrent(t *testing.T)             // card with origin/main SHA → CardCurrent; Problems() empty
func TestStatusCardStaleAfterRemoteMoves(t *testing.T) // card at old SHA; remote.Commit; syncer.Sync → CardStale
func TestStatusCardStaleForUnknownCommit(t *testing.T) // mapped_at_commit "deadbeef" → CardStale
func TestStatusBehind(t *testing.T)                  // dirty clone + remote.Commit + Sync (fetches, no ff) → Behind == 1
func TestStatusInfoFields(t *testing.T)              // feature branch + untracked file → Branch "feature", Clean false, no problem for either
func TestStatusDoesNotUseNetwork(t *testing.T)       // after Sync, os.RemoveAll(remote bare dir) → Collect still has Err == ""
func TestRenderQuietHealthyPrintsNothing(t *testing.T)
func TestRenderQuietPrintsProblems(t *testing.T)     // output == "trama: api: no catalog card: run \"/trama:map api\"\n"

// e2e_test.go (package cli)
func TestInitAddSyncStatus(t *testing.T) // init → add 2 remotes → sync (both cloned) → status shows both repos and "no catalog card" for each
```

- [ ] **Step 2:** Run `go test ./internal/catalog/ ./internal/status/ ./internal/cli/` → Expected: FAIL.
- [ ] **Step 3:** Implementar `catalog`, `status` e o comando.
- [ ] **Step 4:** Run `go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `git add -A && git commit -m "feat: add trama status"`

---

### Task 8: Plugin do Claude Code

**Files:**
- Create: `.claude-plugin/plugin.json`, `.claude-plugin/marketplace.json`, `bin/trama` (executável), `skills/init/SKILL.md`
- Test: `internal/plugin/plugin_test.go` (pacote só de testes)

**Interfaces:**
- Consumes: os comandos `trama version` e `trama init` (Task 1 e 4).
- Produces: `/trama:init`; o comando `trama` no PATH do Claude enquanto o plugin está ativo.

`.claude-plugin/plugin.json`
```json
{
  "name": "trama",
  "version": "0.0.1",
  "description": "Local meta-harness: map, run and change many repositories as one workspace.",
  "author": { "name": "lukascivil" },
  "homepage": "https://github.com/lukascivil/trama",
  "repository": "https://github.com/lukascivil/trama",
  "license": "MIT",
  "keywords": ["polyrepo", "meta-repo", "workspace", "dependencies"]
}
```

`.claude-plugin/marketplace.json`
```json
{
  "name": "trama",
  "owner": { "name": "lukascivil" },
  "plugins": [
    {
      "name": "trama",
      "source": "./",
      "description": "Local meta-harness: map, run and change many repositories as one workspace."
    }
  ]
}
```

`bin/trama` (texto exato)
```sh
#!/bin/sh
# Runs the trama binary whose version matches this plugin.
# TRAMA_BIN points to another binary (local development).
set -eu
if [ -n "${TRAMA_BIN:-}" ]; then
  exec "$TRAMA_BIN" "$@"
fi
if [ -n "${TRAMA_SHIM_ACTIVE:-}" ]; then
  echo "trama: mise did not provide the trama binary (github:lukascivil/trama)" >&2
  exit 127
fi
if ! command -v mise >/dev/null 2>&1; then
  echo "trama: mise is required (https://mise.jdx.dev)" >&2
  exit 127
fi
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
version=$(sed -n 's/^[[:space:]]*"version":[[:space:]]*"\([^"]*\)".*/\1/p' "$root/.claude-plugin/plugin.json" | head -n 1)
export TRAMA_SHIM_ACTIVE=1
exec mise exec "github:lukascivil/trama@$version" -- trama "$@"
```

`skills/init/SKILL.md` (texto exato)
```markdown
---
description: Create a trama workspace in the current directory. Use when the user asks to start or initialize a trama workspace.
disable-model-invocation: true
---

# Initialize a trama workspace

Create the workspace in the session's current directory. Do not create a GitHub repository and do not commit.

1. Run `trama version`. If it fails, show the error and stop. The usual cause is that `mise` is not installed: point the user to https://mise.jdx.dev.
2. Run `ls -A`. The directory must be empty or contain only `.git` (and `.DS_Store`). Otherwise stop and tell the user which entries are in the way. Never delete or move them.
3. Run `trama init`. If the user gave a workspace name ($ARGUMENTS), add `--name <name>`.
4. Run `mise trust`, then `mise install`.
5. Tell the user what was created and what comes next:
   - `trama add <url>` for each repository, then `trama sync` to clone them.
   - `trama status` to see the state of every repository.
```

- [ ] **Step 1: Testes que falham** — `internal/plugin/plugin_test.go` (roda o shim com `sh ../../bin/trama`):

```go
func TestManifestsAgree(t *testing.T)            // both JSONs parse; plugin name "trama"; marketplace plugins[0].name "trama", source "./"; version matches ^\d+\.\d+\.\d+$
func TestShimUsesTramaBin(t *testing.T)          // TRAMA_BIN=/bin/echo, args {"a", "b"} → stdout "a b\n"
func TestShimRunsPluginVersionThroughMise(t *testing.T) // fake `mise` (`#!/bin/sh` + `echo "$@"`) first on PATH; args {"status"} → stdout "exec github:lukascivil/trama@<plugin version> -- trama status\n"
func TestShimStopsRecursion(t *testing.T)        // TRAMA_SHIM_ACTIVE=1 → exit 127; stderr contains "mise did not provide"
func TestShimRequiresMise(t *testing.T)          // PATH="/usr/bin:/bin" (skip if mise is found there) → exit 127; stderr contains "mise is required"
```

- [ ] **Step 2:** Run `go test ./internal/plugin/` → Expected: FAIL.
- [ ] **Step 3:** Criar os quatro arquivos com o conteúdo acima; `chmod +x bin/trama`.
- [ ] **Step 4:** Run `go test ./internal/plugin/ && claude plugin validate .` → Expected: PASS e validação sem erros.
- [ ] **Step 5: Teste local da skill.** Numa pasta vazia temporária, com `TRAMA_BIN` apontando para `go build -o /tmp/trama-dev ./cmd/trama`:

```bash
claude --plugin-dir ~/Documents/GitHub/trama
```

Rodar `/trama:init` → Expected: arquivos da Task 4 criados e `mise.toml` sem pin (build `dev`). Sem sessão interativa disponível, peça ao usuário para rodar este passo.

- [ ] **Step 6: Commit** — `git add -A && git commit -m "feat: add Claude Code plugin with /trama:init"`

---

### Task 9: Release, CI e README

**Files:**
- Create: `.goreleaser.yaml`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `README.md`, `README.pt-BR.md`

`.goreleaser.yaml`
```yaml
version: 2
project_name: trama
builds:
  - main: ./cmd/trama
    binary: trama
    env: [CGO_ENABLED=0]
    goos: [darwin, linux]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X github.com/lukascivil/trama/internal/cli.version={{ .Version }}
archives:
  - formats: [tar.gz]
checksum:
  name_template: checksums.txt
changelog:
  use: github
```

- `ci.yml`: em `push` para `main` e em `pull_request`; matriz `ubuntu-latest` e `macos-latest`; `actions/checkout@v4`, `actions/setup-go@v5` com `go-version-file: go.mod`; `go vet ./...` e `go test -race ./...`.
- `release.yml`: em `push` de tags `v*`; `permissions: contents: write`; checkout com `fetch-depth: 0`; setup-go; passo `Check plugin version matches tag` com `test "v$(jq -r .version .claude-plugin/plugin.json)" = "$GITHUB_REF_NAME"`; `goreleaser/goreleaser-action@v6` com `version: "~> v2"`, `args: release --clean` e `GITHUB_TOKEN`.
- `README.md`, em inglês, com estas seções:
  - What trama is (2 parágrafos tirados do §1 do spec);
  - Status: pre-release, phase 1;
  - Requirements: git, gh, mise, Claude Code;
  - Install the plugin: `/plugin marketplace add lukascivil/trama` e `/plugin install trama@trama`;
  - Create a workspace: `/trama:init`;
  - Commands: `trama add`, `sync` e `status`;
  - Development: `claude --plugin-dir .` e `TRAMA_BIN`;
  - Releasing: atualizar `plugin.json` e criar a tag com a mesma versão;
  - License.
- `README.pt-BR.md`: a mesma coisa em português.

- [ ] **Step 1:** Run `goreleaser check` → Expected: sem erros.
- [ ] **Step 2:** Run `goreleaser release --snapshot --clean` → Expected: `dist/` com 4 arquivos `trama_*_{darwin,linux}_{amd64,arm64}.tar.gz` e `checksums.txt`.
- [ ] **Step 3: Commit** — `git add -A && git commit -m "build: add release pipeline, CI and README"`
- [ ] **Step 4: ⚠️ Confirmar com o usuário** antes de publicar `lukascivil/trama` (público) e criar a tag `v0.0.1`. Com o sim:

```bash
gh repo create lukascivil/trama --public --source . --push
```
```bash
git tag v0.0.1 && git push origin v0.0.1
```

- [ ] **Step 5:** Run `gh run watch --exit-status $(gh run list --workflow release.yml --limit 1 --json databaseId -q '.[0].databaseId')` → Expected: sucesso. Depois `gh release view v0.0.1` lista os 4 `.tar.gz` e o `checksums.txt`.
- [ ] **Step 6:** Run `mise exec github:lukascivil/trama@0.0.1 -- trama version` → Expected: `0.0.1`.

---

### Task 10: `trama-demo-api`

**Files** (novo repo em `~/Documents/GitHub/trama-demo-api`):
- Create: `go.mod` (`module github.com/lukascivil/trama-demo-api`, `go 1.25`), `cmd/api/main.go`, `internal/api/api.go`, `CLAUDE.md`, `README.md`, `LICENSE`
- Test: `internal/api/api_test.go`

**Interfaces:**

```go
package api

const DefaultWebOrigin = "http://localhost:5173"

type HelloResponse struct {
	Message string `json:"message"`
}

func Hello(w http.ResponseWriter, r *http.Request)             // 200, Content-Type application/json, {"message":"Hello, World"}
func WithCORS(next http.Handler, origin string) http.Handler   // Access-Control-Allow-Origin: origin; Allow-Methods "GET, OPTIONS"; Allow-Headers "Content-Type"; OPTIONS → 204 without calling next
func NewServer(webOrigin string) http.Handler                  // mux "GET /hello" → Hello, wrapped by WithCORS
```

`main.go`: `const addr = ":8080"`; origem a partir de `WEB_ORIGIN` (padrão `DefaultWebOrigin`); loga `listening on http://localhost:8080`.

`CLAUDE.md` (texto exato)
```markdown
# trama-demo-api

Demo backend for trama. Go, standard library only.

- Run: `go run ./cmd/api` (listens on http://localhost:8080)
- Test: `go test ./...`
- `GET /hello` returns `{"message": "Hello, World"}`.
- The allowed browser origin comes from `WEB_ORIGIN` (default `http://localhost:5173`).
```

- [ ] **Step 1:** `mkdir ~/Documents/GitHub/trama-demo-api && cd $_ && git init -b main && go mod init github.com/lukascivil/trama-demo-api && go mod edit -go=1.25`
- [ ] **Step 2: Testes que falham:**

```go
func TestHelloReturnsGreeting(t *testing.T) {
	srv := httptest.NewServer(api.NewServer(api.DefaultWebOrigin))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/hello")
	// err nil; status 200; Content-Type "application/json"
	// body decodes to HelloResponse{Message: "Hello, World"}
	// header Access-Control-Allow-Origin == "http://localhost:5173"
}
func TestHelloRejectsPost(t *testing.T)            // POST /hello → 405
func TestPreflightReturnsNoContent(t *testing.T)   // OPTIONS /hello → 204 with Access-Control-Allow-Origin
```

- [ ] **Step 3:** Run `go test ./...` → Expected: FAIL.
- [ ] **Step 4:** Implementar `api.go` e `main.go`.
- [ ] **Step 5:** Run `go test ./...` → Expected: PASS. Depois `go run ./cmd/api &` e `curl -s localhost:8080/hello` → `{"message":"Hello, World"}`; parar o servidor.
- [ ] **Step 6:** `CLAUDE.md`, `README.md` (o que é, como rodar e testar) e `LICENSE` MIT. Commit: `git add -A && git commit -m "feat: hello world API"`.
- [ ] **Step 7: ⚠️ Confirmar com o usuário**, depois: `gh repo create lukascivil/trama-demo-api --public --source . --push`

---

### Task 11: `trama-demo-web`

**Files** (novo repo em `~/Documents/GitHub/trama-demo-web`):
- Create (via template Vite `react-ts`, depois ajustados): `src/api/hello.ts`, `src/App.tsx`, `src/vite-env.d.ts`, `.nvmrc`, `.env.example`, `CLAUDE.md`, `README.md`, `LICENSE`
- Test: `src/api/hello.test.ts`

**Interfaces:**

```ts
export const DEFAULT_API_URL = "http://localhost:8080";
// baseUrl defaults to import.meta.env.VITE_API_URL ?? DEFAULT_API_URL; a trailing "/" is ignored.
// Calls `${base}/hello`; on a non-2xx status throws Error(`GET /hello failed with status ${status}`).
export async function fetchHello(baseUrl?: string, fetchFn: typeof fetch = fetch): Promise<string>;
```

- `App.tsx` mostra `Loading…`, depois `<h1>{message}</h1>`, ou `Could not reach the API: {error}`.
- `vite-env.d.ts` declara `interface ImportMetaEnv { readonly VITE_API_URL?: string }`.
- `.nvmrc` → `24`; `.env.example` → `VITE_API_URL=http://localhost:8080`.

`CLAUDE.md` (texto exato)
```markdown
# trama-demo-web

Demo frontend for trama. Vite + React + TypeScript.

- Setup: `npm install`
- Run: `npm run dev` (http://localhost:5173)
- Test: `npm test`
- The API base URL comes from `VITE_API_URL` (see `.env.example`; default `http://localhost:8080`).
```

- [ ] **Step 1:** `cd ~/Documents/GitHub && npm create vite@latest trama-demo-web -- --template react-ts`. Se perguntar algo, recuse instalar ou iniciar. Depois `cd trama-demo-web && git init -b main && npm install && npm install -D vitest`, e em `package.json` adicionar `"test": "vitest run"`.
- [ ] **Step 2:** Remover o conteúdo de exemplo do template (logos, contador, estilos de demonstração).
- [ ] **Step 3: Testes que falham** — `src/api/hello.test.ts`:

```ts
import { describe, expect, it, vi } from "vitest";
import { fetchHello } from "./hello";

const ok = (body: unknown) => new Response(JSON.stringify(body), { status: 200 });

describe("fetchHello", () => {
  it("returns the message from GET /hello", async () => {
    const fetchFn = vi.fn().mockResolvedValue(ok({ message: "Hello, World" }));
    await expect(fetchHello("http://api.test", fetchFn)).resolves.toBe("Hello, World");
    expect(fetchFn).toHaveBeenCalledWith("http://api.test/hello");
  });

  it("ignores a trailing slash in the base url", async () => {
    const fetchFn = vi.fn().mockResolvedValue(ok({ message: "Hello, World" }));
    await fetchHello("http://api.test/", fetchFn);
    expect(fetchFn).toHaveBeenCalledWith("http://api.test/hello");
  });

  it("throws when the API answers with an error status", async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response("", { status: 500 }));
    await expect(fetchHello("http://api.test", fetchFn)).rejects.toThrow("GET /hello failed with status 500");
  });
});
```

- [ ] **Step 4:** Run `npm test` → Expected: FAIL (`./hello` não existe).
- [ ] **Step 5:** Implementar `hello.ts`, `App.tsx` e `vite-env.d.ts`; criar `.nvmrc` e `.env.example`.
- [ ] **Step 6:** Run `npm test && npm run build` → Expected: 3 testes passando e build sem erro.
- [ ] **Step 7: Verificação visual:** com a API rodando (`go run ./cmd/api` em `trama-demo-api`) e `npm run dev`, abrir `http://localhost:5173` → a página mostra `Hello, World`. Parar os dois.
- [ ] **Step 8:** `CLAUDE.md`, `README.md` e `LICENSE` MIT. Commit: `git add -A && git commit -m "feat: hello world web"`.
- [ ] **Step 9: ⚠️ Confirmar com o usuário**, depois: `gh repo create lukascivil/trama-demo-web --public --source . --push`

---

### Task 12: Workspace `trama-demo` (aceite da fase)

**Files:** novo repo em `~/Documents/GitHub/trama-demo`, gerado pela própria trama.

**Interfaces:**
- Consumes: release `v0.0.1` (Task 9), plugin (Task 8), repos publicados (Tasks 10 e 11).

- [ ] **Step 1:** `mkdir ~/Documents/GitHub/trama-demo`. Numa sessão do Claude Code aberta nessa pasta com `claude --plugin-dir ~/Documents/GitHub/trama` (sem `TRAMA_BIN`), rodar `/trama:init`. Se não houver sessão interativa disponível, peça ao usuário para rodar. Expected: arquivos da Task 4 e `mise.toml` com `"github:lukascivil/trama" = "0.0.1"`.
- [ ] **Step 2:** Run `cd ~/Documents/GitHub/trama-demo && mise exec -- trama version` → Expected: `0.0.1`.
- [ ] **Step 3:** Run `mise exec -- trama add https://github.com/lukascivil/trama-demo-api.git && mise exec -- trama add https://github.com/lukascivil/trama-demo-web.git` → Expected: duas linhas `added ... branch main`.
- [ ] **Step 4:** Run `mise exec -- trama sync` → Expected: os dois `cloned`. Rodar de novo → os dois `up-to-date`.
- [ ] **Step 5:** Run `mise exec -- trama status` → Expected: os dois em `main`, `clean`, `+0 -0`, ficha `missing`, e `no catalog card` nos problemas. Run `mise exec -- trama status --quiet` → só as duas linhas `trama: ...: no catalog card ...`.
- [ ] **Step 6:** Run `git status --short` → Expected: `repos/` não aparece. Commit: `git add -A && git commit -m "chore: initialize trama-demo workspace"`.
- [ ] **Step 7: ⚠️ Confirmar com o usuário**, depois: `gh repo create lukascivil/trama-demo --public --source . --push`
- [ ] **Step 8:** Em `~/Documents/GitHub/trama`, rodar `go test -race ./...` → Expected: PASS. Resumir para o usuário o que ficou publicado e o que vem na fase 2 (`/trama:add-repo`, `/trama:map`, `trama validate`, `trama graph`).
