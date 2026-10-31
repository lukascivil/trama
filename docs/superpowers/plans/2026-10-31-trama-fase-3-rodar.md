# trama — Fase 3 (Rodar) — Plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A trama passa a rodar os repos. `trama setup`, `trama dev`, `trama test` e `trama up` usam os comandos e as versões de runtime das fichas. O `up` sobe tudo em paralelo, com a saída de cada repo prefixada pelo nome dele. Entram também os ajustes pequenos que ficaram da fase 2, e a fase fecha com a release `v0.0.5`.

**Architecture:** Um pacote novo, `internal/runner`, monta e executa os comandos. Cada comando roda com `sh -c <cmd>` na pasta do repo; quando a ficha tem `tools`, roda através de `mise exec <tool>@<versão>... --`. O `up` coloca cada repo no próprio grupo de processos, para conseguir parar também os processos filhos (`npm` → `vite`). O `dev` roda em primeiro plano, ligado ao terminal.

**Tech Stack:** Go (stdlib `os/exec`, `os/signal`, `syscall`), mise.

**Spec:** `docs/superpowers/specs/2026-10-27-trama-design.md`. Esta é a fase 3 da §11 e cobre os comandos `setup`, `dev`, `up` e `test` da §7 e o critério 3 da §10. Base: `v0.0.4` (commit `bc68252`).

**Fora desta fase:** `impact`, `change` e `changes` (fase 4); hook de sessão (fase 5); variáveis de ambiente injetadas a partir das fichas.

## Decisões de construção (o spec deixou para quem constrói)

1. **Os comandos vêm só da ficha.** Sem ficha, ou sem `run.<tipo>`, o comando dá erro claro e não roda nada adivinhado. Ficha desatualizada (critério do `status`) gera um aviso, mas roda.
2. **Rodam no que está checado em `repos/<nome>`**, ao contrário do `validate`. Rodar é trabalho de desenvolvimento, então a branch atual é a certa.
3. **"Setup se necessário"**: um marcador em `repos/.trama/<nome>.setup` (dentro de `repos/`, que o git já ignora) guarda o commit de `HEAD` e um hash do comando de setup. O setup roda de novo quando um dos dois muda. `trama setup` sempre roda e atualiza o marcador.
4. **`trama dev <repo>` também roda o setup se necessário**, para que um clone novo funcione de primeira.
5. **`trama up` confere as portas antes.** Se alguma `run.dev.port` estiver ocupada, nada sobe e o erro diz qual porta e qual repo. Evita o caso real de outro dev server já estar na 5173.
6. **No `up`, um repo que cai não derruba os outros.** A saída mostra `<repo> exited with code N`. O `up` termina quando todos terminam ou com Ctrl+C, e sai com erro se algum falhou por conta própria.
7. **Ctrl+C no `up`**: SIGTERM para o grupo de processos de cada repo, espera de até 5 segundos e depois SIGKILL. No `dev`, o filho fica no grupo do terminal: recebe o Ctrl+C direto e pode ler o teclado (atalhos do Vite).
8. **O comando roda depois de ser mostrado**: cada execução imprime antes `$ <cmd>  (in repos/<nome>)`. O comando vem de um arquivo commitado no workspace, com a mesma confiança de um Makefile.
9. **`AGENTS.md`** ganha uma seção "Running" no template, com os comandos novos. No `trama-demo` ela entra à mão na Task 7.

## Global Constraints

- Tudo das Global Constraints das fases 1 e 2 continua valendo.
- Ferramentas na linha de comando do mise: `<chave>@<valor>` para cada entrada de `tools`, em ordem alfabética da chave (ex.: `mise exec go@1.25 node@24 -- sh -c "<cmd>"`). Sem `tools`, o comando é `sh -c "<cmd>"`, sem mise.
- Prefixo de saída do `up`: `<nome alinhado à direita com espaços até o maior nome> | <linha>`.
- Marcador de setup: `repos/.trama/<nome>.setup`, com uma linha `<sha de HEAD> <sha256 hex do cmd>`.
- Espera entre SIGTERM e SIGKILL: 5 segundos.
- ⚠️ Push, tag, release ou repo novo no GitHub: pare e confirme com o usuário imediatamente antes.

## Review Focus

1. **Porta já ocupada por outro processo** (outro dev server na 5173) → `up` recusa antes de subir qualquer coisa e diz porta e repo. (Task 5 `TestUpRefusesBusyPort`.)
2. **Ctrl+C no `up`** → para também os netos (`sh` → `sleep`, `npm` → `vite`), sem processo órfão. (Task 5 `TestUpStopsProcessTrees`.)
3. **Um repo cai logo de cara** (`mise.toml` não confiável, comando errado) → aparece com o código de saída, e os outros continuam. (Task 5 `TestUpKeepsOthersWhenOneExits`.)
4. **Ficha sem `run.dev`, sem ficha ou desatualizada** → erro claro ou aviso; nunca roda comando adivinhado. (Task 2 `TestJobForMissingCommand`, `TestJobForMissingCard`; Task 4 `TestDevWarnsStaleCard`.)
5. **O repo muda depois do setup** (`trama sync` trouxe commits) → o próximo `dev`/`up` roda o setup de novo; sem mudança, pula. (Task 2 `TestNeedsSetupAfterHeadMoves`.)

---

### Task 1: Ajustes da fase 2

**Files:**
- Create: `internal/fsx/fsx.go`, `internal/fsx/fsx_test.go`
- Modify: `internal/catalog/catalog.go`, `internal/catalog/stamp.go`, `internal/workspace/workspace.go`, `internal/graph/graph.go`, `internal/validate/card.go`, `internal/validate/validate.go`, `internal/status/status.go`, `skills/init/SKILL.md`, `README.md`, `README.pt-BR.md`
- Test: `internal/catalog/card_test.go`, `internal/validate/leftovers_test.go`, `internal/workspace/workspace_test.go`

**Interfaces:**
- Produces:

```go
// package fsx
// ReplaceFile writes data to a new temp file next to path (os.CreateTemp)
// and renames it over path, so a symlink at path is replaced, not followed.
func ReplaceFile(path string, data []byte, perm os.FileMode) error

// package catalog
func IsCommitSHA(s string) bool // ^[0-9a-f]{7,40}$; replaces the regexes in validate and status
```

- `workspace.Save`, `catalog.Stamp` e `graph.Write` passam a usar `fsx.ReplaceFile`, e o `replaceFile` do graph sai.
- `LoadStrict` dá erro `more than one YAML document` quando há um segundo documento.
- `validate` dá erro em `catalog/<x>.yml`: `use the .yaml extension`.
- `mapped_at` precisa passar em `time.Parse("2006-01-02", …)`.
- A mensagem de nome trocado vira `"<name>" does not match the file name catalog/<file>.yaml`.
- `Where` sem espaço no fim quando o `id` está vazio.
- No `skills/init/SKILL.md`, o passo 5 aponta para `/trama:add-repo <url>` e `trama status`.
- README (en e pt-BR), em limitações conhecidas: `trama stamp` regrava a ficha sem linhas em branco e com mapas compactos.

- [ ] **Step 1: Testes que falham:**

```go
func TestReplaceFileDoesNotFollowSymlink(t *testing.T) // path is a symlink to outside.txt → after ReplaceFile, outside.txt unchanged, path is a regular file with data, mode 0644
func TestReplaceFileCreates(t *testing.T)              // new path → file with data
func TestLoadStrictRejectsSecondDocument(t *testing.T) // webCard + "---\nname: x\n" → error contains "more than one YAML document"
func TestValidateYmlCard(t *testing.T)                 // catalog/trama-demo-api.yml present → Error "use the .yaml extension"
func TestValidateImpossibleDate(t *testing.T)          // mapped_at: 2026-13-45 → Error at "mapped_at"
func TestValidateNameMessage(t *testing.T)             // name mismatch → message `"trama-demo-web" does not match the file name catalog/trama-demo-api.yaml`
func TestSaveDoesNotFollowSymlink(t *testing.T)        // workspace.yaml.tmp pre-created as a symlink to outside → AddRepo+Save leaves outside unchanged
func TestIsCommitSHA(t *testing.T)                     // "1cb5a15", 40 hex → true; "main", "1CB5A15", "abc", 41 hex → false
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/...` → Expected: FAIL nos testes novos.
- [ ] **Step 3:** Implementar e trocar os usos.
- [ ] **Step 4:** Run `mise exec -- go test -race ./... && claude plugin validate .` → Expected: PASS.
- [ ] **Step 5: Commit** (conferir `git status`) — `fix: phase 2 leftovers (safe writes, stricter cards, init next steps)`

---

### Task 2: `runner`: jobs, comandos, marcador de setup e portas

**Files:**
- Create: `internal/runner/job.go`, `internal/runner/setup.go`, `internal/runner/port.go`
- Test: `internal/runner/job_test.go`, `internal/runner/setup_test.go`, `internal/runner/port_test.go`

**Interfaces:**
- Consumes: `catalog.Card`/`catalog.Load`, `workspace.Workspace`, `gitx.Repo`.
- Produces (`package runner`):

```go
type Kind string

const (
	Setup Kind = "setup"
	Dev   Kind = "dev"
	Test  Kind = "test"
)

// Job is one command from a repository's card.
type Job struct {
	Repo  string
	Kind  Kind
	Dir   string            // absolute path of repos/<name>
	Cmd   string            // shell command from the card
	Tools map[string]string // from the card
	Port  int               // run.<kind>.port, 0 when not set
}

var ErrNoCommand = errors.New("no such command in the catalog card")

// JobFor reads catalog/<name>.yaml and returns its run.<kind>. Errors:
// `repo <name> is not in workspace.yaml`; `no catalog card for <name>: run "/trama:map <name>"`;
// `repos/<name> is not cloned: run "trama sync <name>"`; ErrNoCommand wrapped as
// `<name> has no run.<kind> in its catalog card`.
func JobFor(ws *workspace.Workspace, name string, kind Kind) (Job, error)

// Args returns the program and arguments: ["sh", "-c", cmd] or
// ["mise", "exec", "go@1.25", "node@24", "--", "sh", "-c", cmd].
func (j Job) Args() []string

// Describe is the line printed before running: "$ <cmd>  (in repos/<name>)".
func (j Job) Describe() string

func NeedsSetup(ws *workspace.Workspace, j Job) (bool, error) // j.Kind == Setup; marker missing or "<HEAD> <sha256(cmd)>" differs → true
func MarkSetup(ws *workspace.Workspace, j Job) error          // writes the marker with fsx.ReplaceFile
func PortFree(port int) bool                                  // net.Listen("tcp", ":<port>") succeeds
```

- [ ] **Step 1: Testes que falham** (workspace temporário com `repos/<nome>` clonado de um remoto do `testutil`, e ficha escrita à mão):

```go
func TestJobForReadsCard(t *testing.T)        // run.dev {cmd: "npm run dev", port: 5173}, tools {node: "24"} → Job{Repo, Dev, Dir abs, "npm run dev", {node: 24}, 5173}
func TestJobForMissingCommand(t *testing.T)   // card without run.test → errors.Is(err, ErrNoCommand), message "api has no run.test in its catalog card"
func TestJobForMissingCard(t *testing.T)      // no card → `no catalog card for api: run "/trama:map api"`
func TestJobForNotCloned(t *testing.T)        // card but no clone → `repos/api is not cloned: run "trama sync api"`
func TestArgs(t *testing.T)                   // no tools → ["sh","-c","npm test"]; tools {node:"24", go:"1.25"} → ["mise","exec","go@1.25","node@24","--","sh","-c","npm test"]
func TestDescribe(t *testing.T)               // "$ npm run dev  (in repos/web)"
func TestNeedsSetupFirstTime(t *testing.T)    // no marker → true; after MarkSetup → false
func TestNeedsSetupAfterHeadMoves(t *testing.T) // MarkSetup; remote.Commit + trama sync (syncer.Sync) → true
func TestNeedsSetupWhenCommandChanges(t *testing.T) // MarkSetup with "npm install"; job with "npm ci" → true
func TestPortFree(t *testing.T)               // listener on a random port → false for it; after Close → true
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/runner/` → Expected: FAIL.
- [ ] **Step 3:** Implementar.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `feat: runner jobs, setup marker and port check`

---

### Task 3: `trama setup` e `trama test`

**Files:**
- Create: `internal/runner/run.go`, `internal/cli/setup.go`, `internal/cli/test.go`
- Test: `internal/runner/run_test.go`, `internal/cli/run_cmds_test.go`

**Interfaces:**
- Consumes: Task 2.
- Produces:

```go
// Run runs j in the foreground: stdin, stdout and stderr connected, child in
// trama's process group so the terminal's Ctrl+C reaches it. While it runs,
// trama ignores SIGINT and waits for the child. Returns the child's exit
// error (an *exec.ExitError carries the code).
func Run(ctx context.Context, j Job, stdin io.Reader, stdout, stderr io.Writer) error
```

- CLI `trama setup [repo...]`: sem argumentos, todos os repos que têm `run.setup`, em sequência. Para cada um: imprime `Describe()`, roda e, se der certo, chama `MarkSetup`. Repos sem `run.setup` aparecem como `<repo>: no run.setup, skipped`. Termina com uma linha por repo, `<repo>: ok` ou `<repo>: failed (exit N)`, e devolve `setup failed for N repo(s)`.
- CLI `trama test [repo...]`: igual, com `run.test`, sem marcador. Resumo `<repo>: passed` ou `<repo>: failed (exit N)`; erro `tests failed for N repo(s)`.
- Os dois avisam `warning: the catalog card of <repo> is stale` quando o `status` diria que está desatualizada.

- [ ] **Step 1: Testes que falham** (comandos falsos: `cmd: "echo hello"`, `cmd: "exit 3"`, sem `tools`):

```go
func TestRunPassesOutputAndExitCode(t *testing.T) // "echo hello; exit 3" → stdout "hello\n", ExitCode 3
func TestRunUsesRepoDir(t *testing.T)             // "pwd" → output == job.Dir (EvalSymlinks on both)
func TestSetupCommandMarks(t *testing.T)          // setup ok → output has "$ echo setup  (in repos/api)" and "api: ok"; runner.NeedsSetup then false
func TestSetupCommandSkipsMissing(t *testing.T)   // repo without run.setup → "web: no run.setup, skipped", no error
func TestTestCommandReportsFailure(t *testing.T)  // run.test "exit 1" → err "tests failed for 1 repo(s)", output "api: failed (exit 1)"
func TestTestCommandPasses(t *testing.T)          // run.test "true" → "api: passed", no error
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/runner/ ./internal/cli/` → Expected: FAIL.
- [ ] **Step 3:** Implementar e registrar os comandos.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `feat: trama setup and trama test`

---

### Task 4: `trama dev`

**Files:**
- Create: `internal/cli/dev.go`
- Test: `internal/cli/dev_test.go`

**Interfaces:**
- Consumes: `runner.JobFor`, `runner.NeedsSetup`, `runner.MarkSetup`, `runner.Run`.
- CLI `trama dev <repo>`:
  1. Avisa se a ficha estiver desatualizada.
  2. Roda o setup se `NeedsSetup` disser que sim, e marca.
  3. Imprime `Describe()` do dev e roda com `runner.Run` em primeiro plano.
  4. Devolve o erro do processo. Se o setup falhar, para antes do dev.

- [ ] **Step 1: Testes que falham:**

```go
func TestDevRunsSetupOnce(t *testing.T)   // run.setup "echo S", run.dev "echo D": first call output contains "S" then "D"; second call only "D"
func TestDevStopsWhenSetupFails(t *testing.T) // setup "exit 2" → err, output has no "$ echo D"
func TestDevWarnsStaleCard(t *testing.T)  // mapped_at_commit of an old commit → output starts with "warning: the catalog card of api is stale"
func TestDevNeedsDevCommand(t *testing.T) // no run.dev → err "api has no run.dev in its catalog card"
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/cli/ -run TestDev` → Expected: FAIL.
- [ ] **Step 3:** Implementar e registrar.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `feat: trama dev`

---

### Task 5: `trama up`

**Files:**
- Create: `internal/runner/up.go`, `internal/runner/prefix.go`, `internal/cli/up.go`
- Test: `internal/runner/up_test.go`, `internal/runner/prefix_test.go`, `internal/cli/up_test.go`

**Interfaces:**
- Consumes: Tasks 2–4.
- Produces:

```go
// NewPrefixWriter writes each complete line as "<label> | <line>"; mu
// serializes writers that share out. Close flushes a last partial line.
func NewPrefixWriter(out io.Writer, label string, mu *sync.Mutex) io.WriteCloser

// Result is how one job ended.
type Result struct {
	Repo     string
	ExitCode int  // -1 when it could not start
	Stopped  bool // ended because Up stopped it
	Err      error
}

// Up starts every job at once, each in its own process group with stdin
// closed, and prefixes its output with the repo name padded to the
// longest one. It returns when all jobs ended. When ctx is cancelled it
// sends SIGTERM to every group, waits up to grace, then SIGKILL.
func Up(ctx context.Context, jobs []Job, out io.Writer, grace time.Duration) []Result

// BusyPorts lists "<repo>: port <n> is already in use" for every job whose
// Port is set and not free.
func BusyPorts(jobs []Job) []string
```

- CLI `trama up [repo...]`. Sem argumentos, todos os repos que têm `run.dev`; repos sem `run.dev` aparecem como `<repo>: no run.dev, skipped`. Ordem:
  1. Avisos de ficha desatualizada.
  2. Setup se necessário, em sequência, com a saída prefixada.
  3. `BusyPorts`: se houver alguma, devolve `cannot start: <linhas>` sem subir nada.
  4. Imprime o `Describe()` de cada job e chama `Up`, com contexto cancelado em SIGINT ou SIGTERM e espera de 5 segundos.
  5. Imprime `<repo> exited with code N` para cada job que terminou sozinho. Devolve erro se algum terminou com código diferente de 0 sem ter sido parado.

- [ ] **Step 1: Testes que falham** (jobs falsos em pastas temporárias, sem `tools`):

```go
func TestPrefixWriter(t *testing.T)            // writes "a\nb" then "c\n" → "web | a\nweb | bc\n"; Close flushes "x" as "web | x\n"
func TestUpPrefixesOutput(t *testing.T)        // jobs api "echo one", web-app "echo two" → output contains "    api | one" and "web-app | two"; both ExitCode 0
func TestUpKeepsOthersWhenOneExits(t *testing.T) // job a "exit 4", job b "echo up; sleep 30"; cancel after "b | up" appears → a: ExitCode 4, Stopped false; b: Stopped true
func TestUpStopsProcessTrees(t *testing.T)     // job "sh -c 'echo $$ > pidfile; sleep 60' & wait": cancel → Up returns within 7s; the pid in pidfile no longer exists (syscall.Kill(pid, 0) fails)
func TestUpRefusesBusyPort(t *testing.T)       // CLI: listener on port P; card run.dev.port P → err "cannot start: api: port P is already in use"; the dev command never ran (marker file absent)
func TestBusyPorts(t *testing.T)               // one busy, one free, one 0 → exactly one line
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/runner/ ./internal/cli/ -run 'Prefix|Up|Busy'` → Expected: FAIL.
- [ ] **Step 3:** Implementar. `SysProcAttr{Setpgid: true}`, `Stdin` nil, SIGTERM e SIGKILL com `syscall.Kill(-pid, …)`.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `feat: trama up`

---

### Task 6: `AGENTS.md` e README

**Files:**
- Modify: `internal/scaffold/templates/AGENTS.md.tmpl`, `README.md`, `README.pt-BR.md`
- Test: `internal/scaffold/scaffold_test.go`

Seção nova no template, depois de "Repositories" e antes de "Rules for agents" (texto exato):

```markdown
## Running

- `trama dev <repo>` runs one repository with the commands and runtime versions from its catalog card, after its setup when needed.
- `trama up` runs every repository at once, each line prefixed with its name. Ctrl+C stops them all.
- `trama test <repo>` runs a repository's tests. Run it in every repository a change touches.
- `trama setup <repo>` runs a repository's setup again (e.g. after changing its dependencies).
```

- README (en e pt-BR): linhas novas na tabela de comandos para `setup`, `dev`, `test` e `up`; Status passa a "Phases 1 to 3 of 5 are done".

- [ ] **Step 1:** Teste que falha: em `TestInitCreatesLayout`, afirmar que o `AGENTS.md` contém `## Running` e `trama up`.
- [ ] **Step 2:** Run `mise exec -- go test ./internal/scaffold/` → Expected: FAIL.
- [ ] **Step 3:** Template e READMEs.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `docs: running commands in AGENTS.md template and README`

---

### Task 7: Aceite (critério 3) e release `v0.0.5`

**Files:**
- Modify: `.claude-plugin/plugin.json` (`0.0.5`); no `~/Documents/GitHub/trama-demo`: `mise.toml` (pin `0.0.5`) e `AGENTS.md` (seção "Running")

- [ ] **Step 1: Aceite com build de desenvolvimento**, no `trama-demo` (`$SP/devbin/trama` = build local):
  - `trama test` → `trama-demo-api: passed` e `trama-demo-web: passed`.
  - `trama up` em segundo plano → `curl -s localhost:8080/hello` responde `{"message":"Hello, World"}`, e o navegador embutido em `http://localhost:5173` mostra `Hello, World`.
  - SIGINT no processo do `trama up` → os dois param: portas 8080 e 5173 livres em até 7 segundos e nenhum processo `vite` ou `api` restante.
  - Se a 5173 estiver ocupada por outro projeto do usuário, conferir que o `up` recusa com a mensagem da porta. Não parar o processo do usuário: pedir para ele parar, ou registrar e seguir só com o `curl`.
- [ ] **Step 2:** `plugin.json` em `0.0.5`. Run `mise exec -- go test -race ./... && mise exec -- goreleaser check` → PASS. Commit `chore: release 0.0.5`.
- [ ] **Step 3: ⚠️ Confirmar com o usuário.** Com o sim, seguir a ordem do README: tag `v0.0.5` → release → `mise exec github:lukascivil/trama@0.0.5 -- trama version` → push da `main` → CI verde.
- [ ] **Step 4:** No `trama-demo`: pin `0.0.5`, `mise install`, seção "Running" no `AGENTS.md` (fora dos marcadores) e `mise exec -- trama test` passando. Commit e push, dentro da mesma confirmação.
