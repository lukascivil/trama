# trama — Fase 4 (Mudança) — Plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A trama passa a planejar e executar mudanças que atravessam repos. `/trama:impact` diz o que quebra e em que ordem publicar; `/trama:change` escreve o registro em `changes/`, cria as branches, implementa, testa e abre os PRs ligados entre si; `trama changes` acompanha os PRs até o merge. A fase fecha com a release `v0.0.6` e a mudança "renomear `message` para `greeting`" feita de verdade nos repos de demo.

**Architecture:** As partes arriscadas ficam no CLI, determinísticas e com travas:
- `trama uses <id>` lista quem fornece e quem consome um `id`;
- `trama branch` cria a branch a partir da `origin/<principal>` e recusa repo sujo;
- `trama pr` recusa a branch principal, envia só a branch atual e abre o PR com `gh`;
- `trama changes` lê e atualiza os registros e faz o commit e o push deles na `main` do workspace.

As skills ficam com o julgamento: entender o pedido, confirmar no código, escrever o plano e implementar. O registro é Markdown com front matter YAML: a máquina lê os passos e as pessoas leem o resto.

**Tech Stack:** Go (cobra, `go.yaml.in/yaml/v3`), git, GitHub CLI (`gh`), skills do Claude Code.

**Spec:** `docs/superpowers/specs/2026-10-27-trama-design.md`. Esta é a fase 4 da §11 e cobre §6.3, `trama changes` da §7, `/trama:impact` e `/trama:change` da §8 e os critérios 4 e 5 da §10. Base: `v0.0.5` (commit `5d4e6bc`).

**Fora desta fase:** hook de início de sessão e o fluxo do segundo dev (fase 5, critérios 6 e 7). O `trama changes` que marca os passos como mergeados entra aqui, porque a `change` precisa dele.

## Decisões de construção (o spec deixou para quem constrói)

1. **Formato do registro**: `changes/<id>.md` com front matter YAML (`id`, `title`, `status`, `continues`, `steps`) e as seções da §6.3 em Markdown. O CLI reescreve só o front matter; o texto das pessoas nunca é tocado.
2. **Estados**:
   - passos: `pending` (ainda sem PR, como a remoção do campo antigo), `open`, `merged` e `closed`;
   - registro: `open` até todos os passos estarem `merged`, e então `done`.
3. **Comandos novos no CLI**, para o que precisa ser exato e seguro:
   - `trama uses`;
   - `trama branch`;
   - `trama pr`;
   - `trama changes` e `trama changes save`.
4. **`trama pr` nunca envia a branch principal e nunca faz merge.** Recusa se a branch atual for a principal do repo ou se o repo tiver alterações sem commit.
5. **Commit dos registros**: `trama changes save <id>` commita só aquele arquivo e envia para a `main` do workspace (decisão sua). Ele recusa se o workspace estiver em outra branch. Sem remoto, commita e avisa; com push recusado (main protegida), para e avisa. `trama changes` faz o mesmo com os registros que mudou, e `--no-push` só commita.
6. **Links entre PRs**: os PRs são abertos um a um; depois que todos existem, a skill reescreve o corpo de cada um (`gh pr edit`) com a lista completa, a ordem de merge e o link do registro.
7. **`trama status`** passa a listar as mudanças com passos não mergeados (§7, adiado da fase 1).
8. **Pendência da fase 3**: repos repetidos nos argumentos rodam uma vez só, e duas fichas com a mesma porta de dev são recusadas pelo `up` antes de subir.
9. **Demo real**:
   - antes da primeira mudança, os repos de demo recebem a tag `start` (adiada desde a fase 1);
   - os PRs são abertos nos repos públicos, mas os merges ficam com você;
   - tudo isso só com a sua confirmação.

## Global Constraints

- Tudo das Global Constraints das fases 1 a 3 continua valendo.
- Nome do registro: `changes/AAAA-MM-DD-<slug>.md`; `id` = nome do arquivo sem `.md`; slug: `^[a-z0-9]+(-[a-z0-9]+)*$`.
- Branch das mudanças: `trama/<slug>`.
- Estados de passo: `pending`, `open`, `merged`, `closed`. Estados de registro: `open`, `done`.
- `gh pr view <url> --json state -q .state` responde `OPEN`, `MERGED` ou `CLOSED`.
- Mensagem de commit do registro: `change: <title> (<id>)`.
- Nunca: push na branch principal de um repo de código, merge de PR, `stash`, `reset`, `checkout --`, `push --force`.
- ⚠️ Abrir PR em repo público, criar tag, publicar release ou fazer push: pare e confirme com o usuário imediatamente antes.

## Review Focus

1. **Repo com alterações locais ou em outra branch quando a mudança começa** → `trama branch` recusa e nada é guardado ou descartado. (Task 4 `TestBranchRefusesDirty`.)
2. **`trama pr` na branch principal** → recusa sem enviar nada. (Task 4 `TestPRRefusesMainBranch`.)
3. **Workspace em outra branch, sem remoto ou com push recusado** → `trama changes save` diz o que aconteceu e o commit local fica feito quando possível. (Task 5 `TestSaveRefusesOtherBranch`, `TestSaveWithoutRemote`, `TestSaveReportsRejectedPush`.)
4. **Registro editado à mão e quebrado** (sem front matter, estado desconhecido, `id` diferente do arquivo) → erro com o nome do arquivo; os outros registros seguem. (Task 2 `TestParseRejects`; Task 5 `TestChangesSkipsBrokenRecord`.)
5. **`gh` ausente, sem login ou com erro num PR** → o passo mantém o estado, o erro aparece com o repo e o PR, e os outros passos seguem. (Task 5 `TestChangesKeepsStateOnGhError`.)

---

### Task 1: Pendência da fase 3 e validação de branch exportada

**Files:**
- Modify: `internal/cli/runcmds.go` (`targets`), `internal/runner/up.go` (`BusyPorts`), `internal/workspace/workspace.go`
- Test: `internal/cli/run_cmds_test.go`, `internal/runner/up_test.go`, `internal/workspace/validate_test.go`

**Interfaces:**
- Produces: `func ValidBranch(name string) error` (package `workspace`), com as mesmas regras de `validateBranch` e erro `invalid branch name "<name>"`. `validateBranch` passa a chamar essa função.
- `targets` remove nomes repetidos e mantém a ordem.
- `BusyPorts` acrescenta `<repoA> and <repoB>: both use port <n>` quando dois jobs declaram a mesma porta.

- [ ] **Step 1: Testes que falham:**

```go
func TestTestRunsRepeatedRepoOnce(t *testing.T)  // run "test", "api", "api" with test cmd "echo X >> count" → repos/api/count has 1 line
func TestBusyPortsSharedPort(t *testing.T)        // jobs a and b both Port 4321 (free) → ["a and b: both use port 4321"]
func TestValidBranch(t *testing.T)                // "trama/rename-message" ok; "-x", "a..b", "a b" → error
```

- [ ] **Step 2:** Run `mise exec -- go test ./...` → Expected: FAIL nos três.
- [ ] **Step 3:** Implementar.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** (conferir `git status`) — `fix: dedupe repo arguments, shared dev ports, export ValidBranch`

---

### Task 2: Pacote `changes`: ler, gravar e atualizar registros

**Files:**
- Create: `internal/changes/record.go`, `internal/changes/refresh.go`
- Test: `internal/changes/record_test.go`, `internal/changes/refresh_test.go`

**Interfaces:**
- Produces (`package changes`):

```go
type Step struct {
	Order  int    `yaml:"order"`
	Repo   string `yaml:"repo"`
	Action string `yaml:"action"`
	Branch string `yaml:"branch,omitempty"`
	PR     string `yaml:"pr,omitempty"`
	State  string `yaml:"state"` // pending | open | merged | closed
}

type Record struct {
	ID        string `yaml:"id"`
	Title     string `yaml:"title"`
	Status    string `yaml:"status"` // open | done
	Continues string `yaml:"continues,omitempty"`
	Steps     []Step `yaml:"steps"`
	Path      string `yaml:"-"` // absolute path of the file
	// unexported: the Markdown body after the front matter, kept byte for byte
}

func Parse(path string, data []byte) (*Record, error)
func Load(root string) (records []*Record, errs []error) // changes/*.md sorted by name; one error per broken file, naming it
func (r *Record) Save() error                             // re-encodes the front matter, keeps the body, fsx.ReplaceFile
func (r *Record) OpenSteps() []Step                        // steps whose State is not "merged"

// PRState returns OPEN, MERGED or CLOSED for a pull request URL.
type PRState func(url string) (string, error)

// GHState asks gh: gh pr view <url> --json state -q .state
func GHState(url string) (string, error)

// Refresh sets each step with a PR to open/merged/closed from state, then
// Status to done when every step is merged. It returns whether anything
// changed and one error per step whose state could not be read (that step
// keeps its state).
func Refresh(r *Record, state PRState) (changed bool, errs []error)
```

Regras do `Parse`, nesta ordem, com o nome do arquivo em cada erro:
1. O arquivo começa com `---\n`, e o front matter termina na próxima linha `---`.
2. `id` igual ao nome do arquivo sem `.md`, no formato `AAAA-MM-DD-<slug>`.
3. `title` não vazio.
4. `status` em `open` ou `done`.
5. Cada passo com `repo` não vazio e `state` em `pending`, `open`, `merged` ou `closed`.
6. Um passo `pending` não tem `pr`; um passo `open`, `merged` ou `closed` tem.

- [ ] **Step 1: Testes que falham** (registro de exemplo do critério 5 como fixture):

```go
func TestParseRecord(t *testing.T)          // fixture → ID "2026-11-03-rename-message", 3 steps, steps[2].State "pending", body starts with "\n## Why"
func TestParseRejects(t *testing.T)         // table: no front matter; state "doing"; id "x" in file 2026-11-03-rename-message.md; pending step with pr; open step without pr → each an error containing the file name
func TestSaveKeepsBody(t *testing.T)        // Parse, set Steps[0].State = "merged", Save → body bytes identical, Parse again shows "merged"
func TestLoadSkipsBroken(t *testing.T)      // two good files + one broken → 2 records, 1 error naming the broken file
func TestOpenSteps(t *testing.T)            // merged, open, pending → 2 open steps
func TestRefreshMarksMerged(t *testing.T)   // fake state: pr1 MERGED, pr2 MERGED; pending step stays pending; Status stays "open"; changed true
func TestRefreshDone(t *testing.T)          // all steps with PRs MERGED and none pending → Status "done"
func TestRefreshUnchanged(t *testing.T)     // states already match → changed false
func TestRefreshKeepsStateOnError(t *testing.T) // fake returns error for pr2 → steps[1].State unchanged, 1 error mentioning the repo and the URL
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/changes/` → Expected: FAIL.
- [ ] **Step 3:** Implementar. `GHState` roda `gh` com o mesmo ambiente sem prompts do `gitx` (`GIT_TERMINAL_PROMPT=0`); ele não é testado contra o GitHub, e os testes usam o `PRState` falso.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `feat: change records`

---

### Task 3: `trama uses <id>`

**Files:**
- Create: `internal/cli/uses.go`
- Test: `internal/cli/uses_test.go`

**Interfaces:**
- Consumes: `catalog.Load` (tolerante), `workspace.Workspace`.
- CLI `trama uses <id>`: procura `id` em `provides` e `consumes` de todas as fichas, na ordem do workspace. Saída:

```
GET /hello
  provided by trama-demo-api  internal/api/api.go:42
  consumed by trama-demo-web  via env, fields: message  src/api/hello.ts:10, src/api/hello.ts:15
  pending in  other-app       candidates: a, b  src/x.ts:3
```

  Sem nenhuma ocorrência, erro `no catalog card provides or consumes "<id>"`. Repos sem ficha entram no fim como `not mapped: <repos>`.

- [ ] **Step 1: Testes que falham** (reusar `catalogWorkspace` de `validate_test.go`):

```go
func TestUsesListsProvidersAndConsumers(t *testing.T) // "uses", "GET /hello" → output has "provided by api  main.go:2" and "consumed by web  via env" 
func TestUsesUnknownID(t *testing.T)                  // "GET /nope" → err `no catalog card provides or consumes "GET /nope"`
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/cli/ -run TestUses` → Expected: FAIL.
- [ ] **Step 3:** Implementar e registrar.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `feat: trama uses`

---

### Task 4: `trama branch` e `trama pr`

**Files:**
- Modify: `internal/gitx/git.go`
- Create: `internal/cli/branch.go`, `internal/cli/pr.go`
- Test: `internal/gitx/branch_test.go`, `internal/cli/branch_test.go`, `internal/cli/pr_test.go`

**Interfaces:**
- Produces (`package gitx`):

```go
func (r Repo) BranchExists(branch string) bool
func (r Repo) SwitchNew(branch, start string) error // git switch --no-track -c <branch> <start>
func (r Repo) Switch(branch string) error           // git switch <branch>
func (r Repo) Push(branch string) error             // git push -u origin <branch>; error when branch is ""
```

- CLI `trama branch <repo> <name>`:
  1. Valida `name` com `workspace.ValidBranch`.
  2. Recusa `repos/<repo> has local changes; commit or move them first` e `branch <name> already exists in repos/<repo>`.
  3. `Fetch`, depois `SwitchNew(name, "origin/<principal>")`.
  4. Imprime `repos/<repo> is on <name>, from origin/<principal> <sha7>`.
- CLI `trama branch <repo> --main`: recusa repo sujo, faz `Switch(<principal>)` e imprime `repos/<repo> is back on <principal>`.
- CLI `trama pr <repo> --title T --body-file F [--draft]`:
  1. Recusa a branch principal (`refusing to open a pull request from <principal>`), o HEAD destacado e alterações locais.
  2. `Push(<atual>)`.
  3. Roda `gh pr create --base <principal> --head <atual> --title T --body-file F [--draft]` na pasta do repo.
  4. Imprime só a URL do PR.

- [ ] **Step 1: Testes que falham** (remotos do `testutil`; um `gh` falso no `PATH`, `#!/bin/sh` que grava os argumentos em `$GH_LOG` e imprime `https://github.com/example/api/pull/7`):

```go
func TestBranchCreatesFromOrigin(t *testing.T) // remote.Commit after sync (local main behind) → new branch starts at the new origin/main SHA
func TestBranchRefusesDirty(t *testing.T)      // tracked edit → error; still on main; the edit is still there
func TestBranchRefusesExisting(t *testing.T)
func TestBranchBackToMain(t *testing.T)        // after branch, "--main" → on main
func TestPRRefusesMainBranch(t *testing.T)     // on main → error; GH_LOG empty; remote has no new branch
func TestPRPushesAndOpens(t *testing.T)        // on trama/x with a commit → output "https://github.com/example/api/pull/7\n"; remote has branch trama/x at that commit; GH_LOG has "pr create --base main --head trama/x --title T --body-file"
func TestPRRefusesDirty(t *testing.T)
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/gitx/ ./internal/cli/ -run 'Branch|PR'` → Expected: FAIL.
- [ ] **Step 3:** Implementar e registrar.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `feat: trama branch and trama pr`

---

### Task 5: `trama changes`, `trama changes save` e o `status`

**Files:**
- Modify: `internal/gitx/git.go`, `internal/status/status.go`, `internal/cli/status.go`
- Create: `internal/cli/changes.go`
- Test: `internal/cli/changes_test.go`, `internal/status/changes_test.go`

**Interfaces:**
- Produces (`package gitx`):

```go
func (r Repo) CommitPaths(msg string, paths ...string) (bool, error) // git add -- paths; false when nothing changed in them; git commit -m msg -- paths
func (r Repo) OriginHead() (string, error)                            // branch of refs/remotes/origin/HEAD, else "main"
func (r Repo) PushHead(branch string) error                           // git push origin HEAD:<branch>
```

- `func RecordURL(origin, branch, path string) string` (package `changes`): para remotos do GitHub (`https://github.com/o/r(.git)` ou `git@github.com:o/r.git`), devolve `https://github.com/o/r/blob/<branch>/<path>`; senão `""`.
- CLI `trama changes save <id>` faz commit e push de `changes/<id>.md`, nesta ordem:
  1. O workspace precisa estar na branch de `OriginHead()`, senão `workspace is on <x>; change records go to <main>`.
  2. `CommitPaths("change: <title> (<id>)", "changes/<id>.md")`.
  3. Sem remoto `origin`: `committed locally; the workspace has no origin remote`.
  4. Push recusado: `push to <main> was rejected (protected branch?): <git error>`, e o commit local fica.
  5. Imprime `saved <id>` e, quando houver, `record: <RecordURL>`.
- CLI `trama changes [--no-push]`:
  1. `Load`; para cada registro com status `open`, `Refresh(r, changes.GHState)` e, se mudou, `Save`.
  2. Imprime uma linha por registro: `<id>: <n> open step(s)` ou `<id>: done`. Erros por arquivo ou passo saem como `warning: …`.
  3. Commit de todos os registros alterados num só commit (`change: update status`) e push, salvo com `--no-push`.
  4. Para testar, a função de estado vem de `stateFunc`, uma variável do pacote `cli` trocada nos testes.
- `status`: `status.OpenChanges(root string) []string`, com uma linha `change <id>: <n> step(s) not merged` por registro aberto. O CLI `status` imprime essas linhas depois dos repos (no `--quiet`, com o prefixo `trama: `).

- [ ] **Step 1: Testes que falham** (workspace criado com `trama init` e um bare remote como `origin`):

```go
func TestSaveCommitsAndPushes(t *testing.T)        // write a valid record → "changes", "save", id → remote main has commit "change: <title> (<id>)" touching only that file
func TestSaveRefusesOtherBranch(t *testing.T)      // workspace on branch x → error "workspace is on x; change records go to main"; no commit
func TestSaveWithoutRemote(t *testing.T)           // no origin → commit exists locally; output has "committed locally"
func TestSaveReportsRejectedPush(t *testing.T)     // remote with a pre-receive hook `exit 1` → error contains "was rejected"; local commit exists
func TestChangesMarksMerged(t *testing.T)          // stateFunc returns MERGED for every PR → record shows merged steps; output "<id>: 1 open step(s)" (the pending one); remote has "change: update status"
func TestChangesSkipsBrokenRecord(t *testing.T)    // one broken + one good → good refreshed; output has "warning:" naming the broken file
func TestChangesKeepsStateOnGhError(t *testing.T)  // stateFunc errors → state unchanged; output "warning: … trama-demo-api …"; no commit
func TestRecordURL(t *testing.T)                   // https and ssh GitHub remotes → blob URL; a local path → ""
func TestStatusListsOpenChanges(t *testing.T)      // open record with 2 unmerged steps → "change <id>: 2 step(s) not merged"; done record → nothing
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/cli/ ./internal/status/ ./internal/changes/ -run 'Save|Changes|RecordURL|OpenChanges'` → Expected: FAIL.
- [ ] **Step 3:** Implementar e registrar.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `feat: trama changes and open changes in status`

---

### Task 6: Skills `/trama:impact` e `/trama:change`

**Files:**
- Create: `skills/impact/SKILL.md`, `skills/change/SKILL.md`, `skills/change/record-reference.md`
- Modify: `README.md`, `README.pt-BR.md`
- Test: `internal/changes/reference_test.go`

`skills/impact/SKILL.md` (texto exato)
```markdown
---
description: Tell what a change would break across the repositories of a trama workspace and in which order to ship it. Read-only. Use when the user asks what a change affects, what would break, or before planning a change that touches an API, event or package another repository uses.
---

# Impact of a change

Arguments: the change, in the user's words ($ARGUMENTS). Work from the workspace root. Change nothing.

1. Run `trama status`. If a card involved is missing or stale, say so first and suggest `/trama:map <repo>`: an answer from an outdated card can be wrong.
2. Find what the change touches: the `id` (`GET /hello`, a topic, a package) and the repository that provides it. Read the cards in `catalog/` and the provider's code when the request names code rather than an `id`.
3. Run `trama uses "<id>"` for each `id`. It lists the provider and every consumer with their evidence.
4. Confirm in the code of every consumer: open each evidence line, then search the repository for what the change touches (a field, a parameter, a status code). List every place with `path:line`. Count `fields_used` as a hint, not as the full answer.
5. Decide for each consumer whether the change breaks it:
   - additive (a new optional field, a new route): compatible;
   - removing or renaming something it reads, changing a type or a required input: breaking.
6. Answer with:
   - the affected repositories and every `path:line`;
   - breaking or compatible, and why;
   - the safe order. For a breaking change, expand → migrate → contract: the provider adds the new form and keeps the old one; each consumer moves to the new form; the old form is removed only after every consumer shipped;
   - pending relations (`from: unknown`) that might also use the `id`: they are not proven, so name them as a risk.
```

`skills/change/SKILL.md` (texto exato)
```markdown
---
description: Plan and carry out a change that spans repositories of a trama workspace: record, branches, code, tests and linked pull requests, in a safe order. Use when the user asks to make a change that touches more than one repository.
---

# Change across repositories

Arguments: the change, in the user's words ($ARGUMENTS). Work from the workspace root.

Never push to a repository's main branch, never merge a pull request, never stash, reset or discard anyone's changes, and never skip a repository's tests.

## 1. Impact and plan

1. Look in `changes/` for records with steps that are not `merged` and touch the same `id`. If one exists, tell the user and offer to continue it: the new record then sets `continues` to its `id`.
2. Follow the `/trama:impact` skill for the change.
3. Write `changes/<date>-<slug>.md` following `${CLAUDE_PLUGIN_ROOT}/skills/change/record-reference.md`: today's date, a short slug, one step per repository and order, expand → migrate → contract. Steps you will do now get `state: pending` for the moment; steps left for later (removing the old form) stay `pending`.
4. Show the plan to the user and wait for their approval. Then run `trama changes save <id>`, which commits the record to the workspace's main branch and prints its link. If it fails (another branch, a rejected push), stop and tell the user.

## 2. Branches, code and tests

For each step you do now, in order:

1. Run `trama branch <repo> trama/<slug>`. If it refuses (local changes, existing branch), stop and tell the user.
2. Read the repository's `AGENTS.md` or `CLAUDE.md` and follow its rules. Make the step's change and update its tests.
3. Run `trama test <repo>`. If it fails, fix it within the step's scope; if you cannot, stop and tell the user.
4. Commit in the repository, following its commit message rules.

## 3. Pull requests

1. For each step, write a body file and run `trama pr <repo> --title "<title>" --body-file <file>`. It prints the pull request URL.
2. When every pull request exists, rewrite each body with `gh pr edit <url> --body-file <file>` so that all of them carry:
   - one line on the change and a link to the record;
   - every pull request of the change, in merge order, marking this one;
   - the pending steps left for later.
3. In the record, set each step's `branch`, `pr` and `state: open`, fill the Execution section, and run `trama changes save <id>`.
4. Run `trama branch <repo> --main` for every repository you changed. The `trama/<slug>` branches stay.

## 4. Report

Tell the user: the record link, each pull request in merge order, the tests that ran, and the pending steps. Remind them that after the merges, `trama sync`, `trama changes` and `/trama:map --all` bring the workspace up to date.
```

`skills/change/record-reference.md`: o formato do registro, com:
- o front matter, campo a campo, com as regras do `Parse` da Task 2;
- as seções `## Why`, `## Impact`, `## Plan`, `## Execution`, `## Pending`, mais `## Continues` quando houver;
- um exemplo completo e válido: o registro "Rename message to greeting in GET /hello" da demo, com 3 passos (api `open`, web `open`, remoção no api `pending`).

- [ ] **Step 1: Teste que falha:** `TestRecordReferenceExampleParses`. Extrai o bloco ```` ```markdown ```` do `record-reference.md`, grava como `changes/2026-11-03-rename-message.md` e afirma que `Parse` não dá erro e há 3 passos. Run → FAIL (arquivo não existe).
- [ ] **Step 2:** Escrever as skills, a referência e as linhas do README: comandos `uses`, `branch`, `pr` e `changes`, e uma seção "Change across repositories" (en e pt-BR). Status do README: "Phases 1 to 4 of 5".
- [ ] **Step 3:** Run `mise exec -- go test -race ./... && claude plugin validate .` → Expected: PASS.
- [ ] **Step 4: Commit** — `feat: /trama:impact and /trama:change skills`

---

### Task 7: Aceite com remotos locais (critérios 4 e 5, sem tocar o GitHub)

**Files:** nenhum no repo; tudo em `$SP` (scratchpad).

- [ ] **Step 1: Montar o cenário:**
  1. `git clone --bare` dos dois repos de demo para `$SP/remotes/` e um bare vazio `$SP/remotes/workspace.git`.
  2. Workspace `$SP/change-ws`: `trama init`, `git remote add origin $SP/remotes/workspace.git`, `trama add` dos dois bares com `--name trama-demo-api` e `--name trama-demo-web`, `trama sync`.
  3. Copiar as fichas de `~/Documents/GitHub/trama-demo/catalog/`, rodar `trama validate` e esperar `ok`.
  4. Commit inicial e push.
- [ ] **Step 2:** `gh` falso em `$SP/devbin/gh`:
  - `pr create` grava os argumentos e imprime `https://github.com/example/<pasta do repo>/pull/<n>`;
  - `pr edit` grava os argumentos;
  - `pr view <url> --json state -q .state` imprime o conteúdo de `$SP/gh-state` (padrão `OPEN`).
- [ ] **Step 3: Critério 4.** Despachar um agente novo, limitado a `skills/impact/SKILL.md`, com `$SP/devbin` primeiro no `PATH`, pedindo: "renomear message para greeting no GET /hello". Conferir na resposta:
  - `trama-demo-web` aparece com `src/api/hello.ts:<linha de body.message>`;
  - a mudança é dada como quebrando;
  - a ordem é api (adiciona `greeting`, mantém `message`) → web → remoção de `message`.
- [ ] **Step 4: Critério 5.** Despachar outro agente novo, limitado a `skills/change/SKILL.md` e à referência, com o mesmo pedido e a instrução "the user approves the plan as written". Conferir:
  - **Registro:** `changes/<id>.md` passa no `Parse` e está no bare do workspace, com 2 commits `change: …`.
  - **Branches:** `trama/<slug>` existe nos dois bares; o api adiciona `greeting` e mantém `message`; o web lê `greeting`; nenhum commit novo na `main` dos bares de código.
  - **Testes:** `trama test` passou nos dois.
  - **PRs:** o log do `gh` falso tem 2 `pr create` e 2 `pr edit`, e cada corpo final cita os dois PRs e o registro.
  - **Estado final:** o registro tem 2 passos `open` com PR e 1 `pending` (remoção de `message`); os dois repos voltaram para `main`.
- [ ] **Step 5:** `echo MERGED > $SP/gh-state && trama changes` → os dois passos ficam `merged`, sobra `1 open step(s)` e há um commit `change: update status` no bare. `trama status` lista `change <id>: 1 step(s) not merged`.
- [ ] **Step 6:** Se algo falhar, corrigir a skill ou o CLI (teste antes no CLI), repetir num cenário novo e registrar no ledger.

---

### Task 8: Release `v0.0.6` e a mudança real nos repos de demo

**Files:** `.claude-plugin/plugin.json` (`0.0.6`); no `trama-demo`: `mise.toml` (pin `0.0.6`) e `changes/`.

- [ ] **Step 1:** `plugin.json` em `0.0.6`. Run `mise exec -- go test -race ./... && mise exec -- goreleaser check && claude plugin validate .` → PASS. Commit `chore: release 0.0.6`.
- [ ] **Step 2: ⚠️ Confirmar com o usuário** a publicação: tag `v0.0.6` → release → push da `main` (ordem do README) e pin `0.0.6` no `trama-demo` com push.
- [ ] **Step 3: ⚠️ Confirmar com o usuário** a demo real, que é pública:
  1. tag `start` nos dois repos de demo, no commit atual da `main`, com push;
  2. seguir `skills/change/SKILL.md` no `trama-demo` com o pedido "renomear message para greeting no GET /hello", abrindo os 2 PRs de verdade;
  3. registro commitado e enviado para a `main` do `trama-demo`.
- [ ] **Step 4:** Entregar ao usuário os links dos 2 PRs e do registro, na ordem de merge. Os merges são dele. Depois dos merges: `trama sync`, `trama changes` (passos `merged`, sobra a remoção) e `/trama:map --all`.
