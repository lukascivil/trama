# trama — Grafo visual — Plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `trama graph --serve` abre no navegador uma página local com todos os repos e relações do workspace, a release de cada repo e, ao clicar num repo, quem pode ser afetado se o dev publicar algo ali.

**Architecture:** Um pacote novo, `internal/view`, monta em Go um modelo JSON do workspace: relações com o estado da prova, afetados diretos, indiretos e possíveis, release, estado da ficha e mudanças abertas. Ele usa `status`, `catalog`, `validate`, `changes` e um pacote novo, `internal/release`. Um servidor HTTP no mesmo pacote entrega esse JSON e uma página estática embutida (HTML, CSS, JS e Cytoscape.js vendorizado). O comando `trama graph` ganha `--serve`, `--port` e `--no-open`. A página só desenha: nenhuma regra de relação mora no JavaScript.

**Tech Stack:** Go 1.25 (`net/http`, `embed`, cobra, `go.yaml.in/yaml/v3`), git, Cytoscape.js 3.30.2 vendorizado, HTML/CSS/JS sem build.

**Spec:** `docs/superpowers/specs/2026-10-31-trama-grafo-visual-design.md`. Base: `main` em `5350750`. Trabalho na branch `graph-ui`.

## Global Constraints

- Dependências Go: só `cobra` e `yaml`, como hoje.
- A página é HTML, CSS e JS puros, embutidos com `go:embed`. Cytoscape.js vendorizado em versão fixa, com a licença. Nenhuma extensão de layout (usar o `cose` embutido).
- Nada sai da máquina: sem CDN, fontes externas ou telemetria. CSP `default-src 'self'` em toda resposta.
- O servidor escuta só em `127.0.0.1`, aceita só `GET` e só `Host` igual a `127.0.0.1:<porta>` ou `localhost:<porta>`.
- O comando não faz `fetch`; usa o que o `trama sync` trouxe.
- Todo texto vindo de fichas, registros e git entra na página com `textContent`. Link só com `href` que começa com `https://`.
- Inglês por padrão; português quando `navigator.language` começa com `pt`. `?lang=en` ou `?lang=pt` força o idioma.
- `trama graph` sem `--serve` não muda.
- macOS e Linux. O `.goreleaser.yaml` não muda.
- Commits terminam com `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Nada de push, tag ou release nesta fase sem OK explícito do usuário.

## Review Focus

1. **Ficha que aponta `from` para um repo fora do workspace** (ou um fornecedor sem ficha): a página não pode quebrar nem desenhar seta para um nó que não existe. A relação vira não verificada, com o motivo. Teste na Task 4.
2. **Texto hostil nas fichas** (`<script>`, aspas, Unicode em nomes e ids): o JSON preserva o texto literal e a página o mostra como texto. Testes nas Tasks 4 e 7.
3. **Porta ocupada**: com `--port` explícito, erro claro; sem `--port`, outra porta livre e a URL impressa certa. Testes na Task 6.
4. **Ciclos e auto-consumo** (A↔B, A consome de A): o cálculo de afetados termina e nunca lista o próprio repo. Teste na Task 3.
5. **Repo sem release legível**: não clonado, sem `origin/<branch>`, sem tags, tags fora do semver (`start`, `v1.2`, `release-2024`), tag fora da branch principal. Tudo vira "sem release" ou "não clonado", nunca erro na página. Testes na Task 1.

---

### Task 1: Release de um repo (tags semver)

**Files:**
- Create: `internal/release/semver.go`, `internal/release/release.go`, `internal/release/semver_test.go`, `internal/release/release_test.go`
- Modify: `internal/gitx/git.go` (dois métodos novos)
- Test: `internal/gitx/tags_test.go`

**Interfaces:**
- Produces:
  - `gitx.Tag struct { Name, Date string }`: `Date` em ISO 8601 estrito (`%(creatordate:iso-strict)`: data do tagger na tag anotada, do commit na tag leve).
  - `func (r Repo) TagsMerged(ref string) ([]Tag, error)`: `git for-each-ref --merged=<ref> --format=%(refname)%09%(creatordate:iso-strict) refs/tags`, tirando o prefixo `refs/tags/`.
  - `func (r Repo) CountCommits(from, to string) (int, error)`: `git rev-list --count <from>..<to>`.
  - `release.Release struct { Version string \`json:"version"\`; Date string \`json:"date"\`; CommitsAfter int \`json:"commitsAfter"\` }`.
  - `func release.Latest(repo gitx.Repo, ref string) (*Release, error)`: `nil, nil` quando não há tag semver em `ref`; erro só quando o git falha.

- [ ] **Step 1: Criar a branch e commitar o spec e o plano**

```bash
git switch -c graph-ui
git add docs/superpowers/specs/2026-10-31-trama-grafo-visual-design.md docs/superpowers/plans/2026-10-31-trama-grafo-visual.md
git commit -m "docs: graph page design and plan"
```

- [ ] **Step 2: Escrever os testes que falham**

`internal/release/semver_test.go`:
- `TestParseSemver`: aceita `v1.2.3`, `1.2.3`, `1.0.0-rc.1`, `1.0.0+build.5`; recusa `start`, `v1.2`, `release-2024`, `01.2.3`, `1.2.3-`, `v1.2.3.4`.
- `TestCompareSemver`: a ordem da spec do semver, cada um menor que o próximo: `1.0.0-alpha`, `1.0.0-alpha.1`, `1.0.0-alpha.beta`, `1.0.0-beta`, `1.0.0-beta.2`, `1.0.0-beta.11`, `1.0.0-rc.1`, `1.0.0`; e `0.9.0 < 0.10.0`; `+build` não muda a ordem.

`internal/release/release_test.go`, num clone de `testutil.NewRemote` (`origin/main`):
- `TestLatestPicksHighestSemver`: tags `v0.9.0`, `v0.10.0`, `v0.10.0-rc.1`, `start` e `release-2024` → `Version == "v0.10.0"`.
- `TestLatestNoSemverTag`: só a tag `start` → `nil, nil`.
- `TestLatestIgnoresTagOffMain`: `v9.0.0` num commit de outra branch, fora da `origin/main` → é ignorada.
- `TestLatestCountsCommitsAfter`: `v0.1.0` e mais dois commits enviados → `CommitsAfter == 2`.
- `TestLatestDates`: com `GIT_COMMITTER_DATE` fixo, uma tag leve usa a data do commit e uma tag anotada criada com outra data usa a data do tagger (comparar o prefixo `2026-01-01` / `2026-02-01`).
- `TestLatestMissingRef`: `ref` inexistente → erro.

`internal/gitx/tags_test.go`:
- `TestFetchBringsTagOnKnownCommit`: clonar, criar no remoto uma tag num commit que o clone já tem, enviar a tag, `Fetch()`; `TagsMerged("origin/main")` lista a tag.

- [ ] **Step 3: Rodar e ver falhar**

Run: `go test ./internal/release/ ./internal/gitx/ -run 'Semver|Latest|FetchBringsTag'`
Expected: FAIL (o pacote `release` e os métodos não existem).

- [ ] **Step 4: Implementar**

Os métodos do gitx usam `run` como os outros. O semver é próprio (sem dependência): `parse(tag string) (version, bool)` e `compare(a, b version) int`, com identificadores de pré-release comparados pela regra do semver 2.0. Com duas tags de mesma precedência (`v1.0.0` e `1.0.0`), vence a de nome menor em ordem lexical, para ser determinístico.

Se o `TestFetchBringsTagOnKnownCommit` falhar, o `Fetch` faz um segundo fetch sem `--prune`: `git fetch -q origin "refs/tags/*:refs/tags/*"`. Nunca `--tags` junto com `--prune`, porque apagaria as tags locais.

- [ ] **Step 5: Rodar e ver passar**

Run: `go test -race ./internal/release/ ./internal/gitx/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/release internal/gitx
git commit -m "feat: latest semver release of a repository from its git tags"
```

---

### Task 2: `status.Collect` em paralelo, com o SHA da origin

**Files:**
- Modify: `internal/status/status.go`
- Test: `internal/status/status_test.go`

**Interfaces:**
- Produces: `RepoStatus.OriginSHA string` (o `origin/<branch>` resolvido; `""` quando não existe). `Collect` mantém a assinatura e a ordem do workspace.

- [ ] **Step 1: Escrever os testes que falham**

- `TestCollectKeepsWorkspaceOrder`: 12 repos, metade clonados (com `testutil.NewRemote`) e metade não; os nomes em `Collect` saem na ordem do `workspace.yaml`.
- `TestCollectOriginSHA`: no repo clonado, `OriginSHA` é igual a `git rev-parse origin/main`; no não clonado, `""`.

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/status/ -run 'KeepsWorkspaceOrder|OriginSHA'`
Expected: FAIL (`OriginSHA` não existe).

- [ ] **Step 3: Implementar**

`Collect` roda `collectRepo` com no máximo 8 goroutines e guarda cada resultado no índice do repo. `collectRepo` preenche `OriginSHA` com o valor que já calcula.

- [ ] **Step 4: Rodar e ver passar**

Run: `go test -race ./internal/status/ ./internal/cli/`
Expected: PASS (as saídas de `trama status` não mudam)

- [ ] **Step 5: Commit**

```bash
git add internal/status
git commit -m "perf: inspect repositories in parallel in status"
```

---

### Task 3: Arestas e afetados

**Files:**
- Create: `internal/view/edge.go`, `internal/view/affected.go`, `internal/view/affected_test.go`

**Interfaces:**
- Produces:

```go
type EdgeStatus string

const (
	Proven     EdgeStatus = "proven"     // from a repo, validation found no error in this consume
	Declared   EdgeStatus = "declared"   // a workspace.yaml link that no card cites
	Unverified EdgeStatus = "unverified" // from a repo, but validation found errors
	Pending    EdgeStatus = "pending"    // from: unknown
)

type Link struct {
	Text string `json:"text"`          // "src/api/hello.ts:15" or "workspace.yaml"
	URL  string `json:"url,omitempty"` // https link to the line, when the host is GitHub
}

type Edge struct {
	Consumer   string     `json:"consumer"`
	Provider   string     `json:"provider,omitempty"` // "" when Pending
	Kind       string     `json:"kind,omitempty"`     // "" for a Declared link
	ID         string     `json:"id"`
	Via        string     `json:"via,omitempty"`
	Status     EdgeStatus `json:"status"`
	Candidates []string   `json:"candidates,omitempty"`
	Evidence   []Link     `json:"evidence,omitempty"`
	Problems   []string   `json:"problems,omitempty"` // why it is Unverified
}

type Hit struct {
	Repo     string   `json:"repo"`
	Kind     string   `json:"kind,omitempty"`
	ID       string   `json:"id,omitempty"`
	Path     []string `json:"path,omitempty"`   // Indirect: the repos between Repo and the selected one, nearest to Repo first
	Reason   string   `json:"reason,omitempty"` // Possible: "unknown" or "unverified"
	Evidence []Link   `json:"evidence,omitempty"`
}

type Affected struct {
	Direct   []Hit `json:"direct"`
	Indirect []Hit `json:"indirect"`
	Possible []Hit `json:"possible"`
}

func AffectedBy(repo string, edges []Edge) Affected
```

Regras (spec §6.1):
- **Direct:** uma `Hit` por aresta `Proven` ou `Declared` com `Provider == repo` e `Consumer != repo`, na ordem das arestas.
- **Indirect:** busca em largura de trás para frente pelas arestas `Proven` e `Declared`, a partir de `repo`. Cada repo a dois passos ou mais vira uma `Hit` com o caminho mais curto (o primeiro encontrado, seguindo a ordem das arestas). Nunca inclui `repo` nem quem já é direto. Repo visitado não volta.
- **Possible:** arestas `Pending` com `repo` em `Candidates` (`Reason: "unknown"`) e arestas `Unverified` com `Provider == repo` (`Reason: "unverified"`), com `Consumer != repo`. Arestas `Unverified` e `Pending` não entram na busca dos indiretos.

- [ ] **Step 1: Escrever os testes que falham**

Com o exemplo do esboço (`web-app → api-gateway`, `mobile-app → api-gateway`, `api-gateway → orders-api`, `payments-api → orders-api`, `notifications-worker → orders-api` por evento):
- `TestAffectedDirect`: para `orders-api`, os diretos são `api-gateway`, `payments-api` e `notifications-worker`, com os `id`s.
- `TestAffectedIndirectThroughGateway`: os indiretos são `web-app` com `Path == ["api-gateway"]` e `mobile-app` com o mesmo caminho.
- `TestAffectedCycleAndSelf`: `a → b`, `b → a`, `a → a`; para `a`, o direto é só `b`, não há indiretos e a função termina.
- `TestAffectedUnverifiedIsOnlyPossible`: `c → orders-api` não verificada e `d → c` provada; `c` aparece em possíveis com `Reason "unverified"` e `d` não aparece em lugar nenhum.
- `TestAffectedPendingCandidates`: `x` com `from: unknown` e candidatos `[orders-api, legacy]` aparece em possíveis de `orders-api` com `Reason "unknown"`.
- `TestAffectedDeclaredIsDirect`: uma aresta `Declared` conta como direta.

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/view/ -run Affected`
Expected: FAIL (o pacote não existe)

- [ ] **Step 3: Implementar `AffectedBy`**

- [ ] **Step 4: Rodar e ver passar**

Run: `go test -race ./internal/view/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/view
git commit -m "feat: direct, indirect and possible affected repositories"
```

---

### Task 4: O modelo do workspace

**Files:**
- Create: `internal/view/model.go`, `internal/view/model_test.go`, `internal/view/fixture_test.go`
- Modify: `internal/changes/url.go` (extrair o parser do GitHub), `internal/changes/url_test.go`

**Interfaces:**
- Consumes: `release.Latest` (Task 1), `status.Collect` e `RepoStatus.OriginSHA` (Task 2), `Edge`, `Link`, `Affected` e `AffectedBy` (Task 3), `validate.Check`, `catalog.Load`, `catalog.ParseEvidence`, `changes.Load`, `changes.RecordURL`.
- Produces:
  - `func changes.GitHubSlug(origin string) string`: `"owner/repo"` para URLs `https://github.com/`, `git@github.com:` e `ssh://git@github.com/`; `""` para outros hosts. `RecordURL` passa a usá-la.

```go
type Model struct {
	Workspace string   `json:"workspace"`
	Repos     []Repo   `json:"repos"`
	Edges     []Edge   `json:"edges"`
	Warnings  []string `json:"warnings"`
}

type Repo struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Stack       []string         `json:"stack,omitempty"`
	URL         string           `json:"url,omitempty"`  // https://github.com/<slug>, when on GitHub
	Cloned      bool             `json:"cloned"`
	Card        string           `json:"card"`           // current | stale | missing | unknown
	Release     *release.Release `json:"release"`        // nil: no semver tag, not cloned, or no origin ref
	Run         []Command        `json:"run,omitempty"`  // {kind, cmd, port}, in setup, dev, test order
	Provides    []Provide        `json:"provides,omitempty"`
	OpenChanges []Change         `json:"openChanges,omitempty"`
	Affected    Affected         `json:"affected"`
}

type Command struct { Kind, Cmd string; Port int }        // json: kind, cmd, port,omitempty
type Provide struct { Kind, ID string; Evidence []Link }  // json: kind, id, evidence
type Change struct { ID, Title, URL string }              // json: id, title, url,omitempty

type Builder struct{ /* root, validation cache */ }

func NewBuilder(root string) *Builder
func (b *Builder) Build() (*Model, error) // error only when workspace.yaml cannot be loaded
```

O que o painel chama de "consome" a página tira das arestas com `Consumer == repo`: o modelo não repete.

`Build`, a cada chamada:
1. `workspace.Load(root)`; erro aqui é o único erro de `Build`.
2. `status.Collect(ws)`. Cada `RepoStatus.Err` vira um aviso `"<repo>: <err>"`.
3. `catalog.Load` de cada repo (tolerante). Ficha que não abre vira aviso, e o repo aparece sem relações.
4. Validação com cache. A chave é o sha256 de `workspace.yaml`, dos bytes de cada `catalog/*.yaml` e dos `OriginSHA` de todos os repos; com a mesma chave, reusa o `validate.Report` anterior. A função fica numa variável do pacote (`var check = validate.Check`) para o teste contar as chamadas. Todos os achados entram em `Warnings` como `Finding.String()`.
   - Um erro com `File == "catalog/<repo>.yaml"` e `Where` começando com `consumes[<i>]` marca o consumo `i` como não verificado, com a mensagem em `Problems`.
   - Um erro com `Where == ""` nesse arquivo marca todos os consumos da ficha.
5. As arestas, na ordem do workspace e dos consumos: `from: unknown` dá `Pending`; consumo marcado no passo 4, ou cujo `from` não é um repo do workspace, dá `Unverified`; o resto dá `Proven`.
   - Evidência do consumidor: `Link{Text: <como na ficha>, URL: https://github.com/<slug>/blob/<mapped_at_commit>/<path>#L<a>` ou `#L<a>-L<b>` quando é intervalo, só se o `url` do repo no `workspace.yaml` for GitHub e o `mapped_at_commit` não for vazio}`. A evidência `workspace.yaml` fica só com o texto.
   - As evidências de `provides` seguem a mesma regra, com o `url` e o `mapped_at_commit` do próprio repo.
   - Depois, cada `link` do `workspace.yaml` sem aresta do mesmo consumidor, fornecedor e `id` vira `Declared`, com `Evidence: [{Text: "workspace.yaml"}]`.
6. A release dos repos com `OriginSHA != ""`: `release.Latest(gitx.Repo{Dir: ws.RepoDir(name)}, "origin/"+branch)`, com no máximo 8 em paralelo. Erro vira aviso, e a release fica `nil`.
7. As mudanças abertas: `changes.Load(root)`. Um registro `open` entra em cada repo que aparece num dos `OpenSteps()` dele. A URL vem de `changes.RecordURL(<origin do workspace>, <OriginHead do workspace>, "changes/<id>.md")`. Registro que não abre vira aviso.
8. `Affected` de cada repo: `AffectedBy(name, edges)`.
9. `Warnings` nunca é `nil` (sai `[]` no JSON).

- [ ] **Step 1: Escrever os testes que falham**

`fixture_test.go` segue o padrão de `internal/validate/helpers_test.go`: `workspace.yaml`, fichas e arquivos citados em `repos/<repo>/` sem git, para os testes de relação; e repos clonados de `testutil.NewRemote` para release e status.

`model_test.go`:
- `TestBuildDemo`: com o demo de dois repos, a aresta `trama-demo-web → trama-demo-api` `GET /hello` é `Proven`, com `Evidence[0] == {Text: "src/api/hello.ts:10", URL: "https://github.com/lukascivil/trama-demo-web/blob/697bd1b/src/api/hello.ts#L10"}`. O `Affected.Direct` da api tem o web.
- `TestBuildUnverifiedOnValidationError`: evidência `src/api/hello.ts:999` → aresta `Unverified`, um `Problems` que contém `line 999`, o web nos possíveis da api com `Reason "unverified"` e o achado em `Warnings`.
- `TestBuildFromRepoOutsideWorkspace` (Review Focus 1): `from: ghost` → aresta `Unverified`; nenhuma aresta aponta para um repo que não está em `Repos`.
- `TestBuildPending`: `from: unknown` com candidatos → aresta `Pending` com `Candidates`.
- `TestBuildDeclaredLinkWithoutCard`: um `link` no `workspace.yaml` que nenhuma ficha cita → aresta `Declared` e uma `Hit` direta.
- `TestBuildBrokenCardIsWarning`: ficha com YAML inválido → o repo aparece, sem arestas dele, e com um aviso.
- `TestBuildKeepsHostileText` (Review Focus 2): descrição `<script>alert(1)</script>` e `id` com aspas e acento → depois de `json.Marshal` e `json.Unmarshal`, os textos voltam iguais.
- `TestBuildOpenChanges`: registro `open` com um passo `open` em `trama-demo-web` → `OpenChanges == [{ID, Title, ...}]` no web, e nada na api quando o passo dela está `merged`.
- `TestBuildReleaseAndCard` (git): repo clonado com `v0.1.0` e dois commits depois → `Release{Version: "v0.1.0", CommitsAfter: 2}`; sem tag → `Release == nil`; repo não clonado → `Cloned == false` e `Release == nil`.
- `TestBuildCachesValidation`: duas chamadas a `Build` → `check` roda uma vez; mudar um byte de uma ficha → roda de novo.
- `TestBuildHundredRepos`: só com `TRAMA_PERF=1` (senão `t.Skip`). Monta 100 repos clonados com fichas e relações entre vizinhos, chama `Build` duas vezes, registra os dois tempos com `t.Logf` e exige que a segunda leve menos de 2 s.

`internal/changes/url_test.go`:
- `TestGitHubSlug`: as três formas de URL do GitHub → `"owner/repo"` (sem `.git` e sem a barra final); GitLab e caminho local → `""`.

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/view/ ./internal/changes/ -run 'Build|GitHubSlug'`
Expected: FAIL

- [ ] **Step 3: Implementar `GitHubSlug` e `Builder.Build`**

- [ ] **Step 4: Rodar e ver passar**

Run: `go test -race ./internal/view/ ./internal/changes/ && TRAMA_PERF=1 go test ./internal/view/ -run Hundred -v`
Expected: PASS, e o tempo da segunda chamada no log, abaixo de 2 s. Se passar disso, medir onde está o tempo antes de subir o paralelismo.

- [ ] **Step 5: Commit**

```bash
git add internal/view internal/changes
git commit -m "feat: workspace model for the graph page"
```

---

### Task 5: O servidor HTTP

**Files:**
- Create: `internal/view/server.go`, `internal/view/server_test.go`, `internal/view/assets/index.html` (por enquanto mínimo; a Task 7 escreve a página)

**Interfaces:**
- Consumes: `Model`, `Builder.Build` (Task 4).
- Produces: `func Handler(build func() (*Model, error), port int) http.Handler`.

Rotas (`http.ServeMux` com padrões de método):
- `GET /` → `assets/index.html`;
- `GET /assets/` → arquivos do `embed.FS` (`fs.Sub(assets, "assets")`);
- `GET /api/workspace` → `json.NewEncoder` do modelo, com `Content-Type: application/json` e `Cache-Control: no-store`; erro de `build` → 500 com `{"error": "<mensagem>"}`.
- O resto: 404. Outro método: 405 (o `ServeMux` já responde assim).

Em toda resposta, antes da rota:
- `Host` diferente de `127.0.0.1:<port>` ou `localhost:<port>` (sem diferenciar maiúsculas) → 403;
- `Content-Security-Policy: default-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'`;
- `X-Content-Type-Options: nosniff` e `Referrer-Policy: no-referrer`.

- [ ] **Step 1: Escrever os testes que falham** (com `httptest.NewRecorder` e `Host` explícito)

- `TestServerServesIndexAndAPI`: `GET /` → 200 com HTML; `GET /api/workspace` → 200 com o `workspace` do modelo de teste.
- `TestServerRejectsForeignHost`: `Host: evil.example:<port>` e `Host: localhost:<outra porta>` → 403.
- `TestServerOnlyGET`: `POST /api/workspace` → 405.
- `TestServerSecurityHeaders`: o CSP e o `nosniff` estão em `/`, em `/assets/...` e em `/api/workspace`.
- `TestServerAPIErrorIsJSON`: `build` que falha → 500 com `{"error": ...}`.
- `TestServerDoesNotReadDisk`: `GET /assets/../workspace.yaml` e `GET /assets/%2e%2e/go.mod` → não servem nada do disco (404 ou o que o `ServeMux` responder para caminho sujo, nunca 200).

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/view/ -run Server`
Expected: FAIL

- [ ] **Step 3: Implementar `Handler` e o `//go:embed assets`**

- [ ] **Step 4: Rodar e ver passar**

Run: `go test -race ./internal/view/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/view
git commit -m "feat: local HTTP server for the graph page"
```

---

### Task 6: `trama graph --serve`

**Files:**
- Modify: `internal/cli/graph.go`
- Create: `internal/cli/graph_serve_test.go`

**Interfaces:**
- Consumes: `view.NewBuilder`, `view.Handler` (Tasks 4 e 5), `runner.ErrInterrupted`.
- Produces: as flags `--serve`, `--port <n>` (padrão 7300; 0 é qualquer porta livre) e `--no-open`; e `var openURL = func(url string) error` (`open` no macOS, `xdg-open` no Linux), que os testes trocam.

Comportamento:
- `--port` ou `--no-open` sem `--serve` → erro `--port and --no-open need --serve`.
- Sem `--serve`, o caminho de hoje, sem nenhuma mudança.
- Com `--serve`:
  1. Acha o workspace como os outros comandos e roda um `Build()`. Se falhar, devolve o erro. Esse `Build` também aquece o cache.
  2. Escuta em `127.0.0.1:<porta>`. Com `--port` explícito e a porta ocupada → `port <n> is in use: choose another with --port`. Sem `--port` e 7300 ocupada → outra porta livre (`:0`).
  3. Imprime `serving http://127.0.0.1:<porta>/ (Ctrl+C to stop)`.
  4. Sem `--no-open`, chama `openURL`. Se falhar, imprime `open http://127.0.0.1:<porta>/ in your browser` e segue.
  5. Serve com `http.Server{ReadHeaderTimeout: 5 * time.Second}` até `signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)` acabar. Então faz `Shutdown` com prazo de 2 s e devolve `runner.ErrInterrupted`, que sai com 130 sem mensagem.
  6. Não escreve `docs/graph.md` nem `AGENTS.md`, e sobe mesmo com erros de validação.

- [ ] **Step 1: Escrever os testes que falham**

Num helper, o comando roda com `ExecuteContext(ctx)` numa goroutine, a saída vai para um buffer protegido por mutex, e o teste espera a linha `serving http://127.0.0.1:<porta>/`.
- `TestGraphServeServesAndStops`: `graph --serve --no-open --port 0` num workspace de teste; `GET /api/workspace` → 200 com o nome do workspace; cancelar o `ctx` → o comando devolve `runner.ErrInterrupted`; `docs/graph.md` não foi criado.
- `TestGraphServeWithValidationErrors`: uma ficha com evidência quebrada → sobe e o JSON tem `warnings`.
- `TestGraphServeBusyPort` (Review Focus 3): uma porta ocupada pelo teste e `--port <ela>` → erro com `is in use`.
- `TestGraphServeDefaultPortFallback` (Review Focus 3): o teste ocupa a 7300 (se não conseguir, `t.Skip`); sem `--port`, a URL impressa usa outra porta, e ela responde.
- `TestGraphServeOpensBrowser`: com `openURL` trocado, sem `--no-open` ele é chamado com a URL impressa; com `--no-open`, não.
- `TestGraphPortNeedsServe`: `graph --port 8000` → o erro acima.

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/cli/ -run 'GraphServe|GraphPort'`
Expected: FAIL

- [ ] **Step 3: Implementar**

- [ ] **Step 4: Rodar e ver passar**

Run: `go test -race ./internal/cli/`
Expected: PASS, inclusive os testes atuais de `trama graph`

- [ ] **Step 5: Commit**

```bash
git add internal/cli
git commit -m "feat: trama graph --serve opens the workspace graph in the browser"
```

---

### Task 7: A página

**Files:**
- Modify: `internal/view/assets/index.html`
- Create:
  - `internal/view/assets/app.css`, `internal/view/assets/app.js`;
  - `internal/view/assets/vendor/cytoscape.min.js`, `internal/view/assets/vendor/LICENSE.cytoscape`, `internal/view/assets/vendor/README.md` (versão, origem e sha256);
  - `internal/view/assets_test.go`.

**Interfaces:**
- Consumes: o JSON de `GET /api/workspace` (Task 4: `Model`, `Repo`, `Edge`, `Hit`, `Link`).

- [ ] **Step 1: Vendorizar o Cytoscape.js 3.30.2**

Baixar `https://unpkg.com/cytoscape@3.30.2/dist/cytoscape.min.js` e o `LICENSE` do mesmo pacote. Conferir que o arquivo começa com o banner do Cytoscape e anotar a versão, as URLs e o `shasum -a 256` no `vendor/README.md`.

- [ ] **Step 2: Escrever os testes que falham** (`assets_test.go`)

- `TestAssetsEmbedded`: o `embed.FS` tem `index.html`, `app.js`, `app.css` e `vendor/cytoscape.min.js`.
- `TestAssetsStaySelfContained`: `index.html`, `app.js` e `app.css` não contêm `http://` nem `https://`, exceto o teste `startsWith("https://")` que filtra os links.
- `TestIndexHasNoInlineCode`: o `index.html` não tem `<style`, `style="`, `<script>` sem `src` nem `on<evento>=`. O CSP bloquearia esses trechos.

- [ ] **Step 3: Rodar e ver falhar**

Run: `go test ./internal/view/ -run Assets`
Expected: FAIL

- [ ] **Step 4: Escrever a página**

O contrato do `app.js` vem da spec §5, que ele implementa inteiro:
- **Dados e idioma:**
  - Carrega `/api/workspace`. Em caso de erro, mostra a mensagem na faixa do topo.
  - Os textos vêm de `STRINGS = {en: {...}, pt: {...}}`. O idioma sai do `?lang=` ou do `navigator.language`.
  - A data é relativa ("há 11 dias" / "11 days ago", "hoje" / "today").
- **Topo:** nome do workspace; contagens de repos, relações (`proven` + `declared`) e pendentes (`pending` + `unverified`); busca; e a faixa de avisos (`warnings`), recolhível.
- **Esquerda:**
  - um checkbox por repo;
  - os tipos: HTTP, eventos, pacotes, pendentes e não verificadas;
  - "esconder repos sem relação";
  - "só repos com mudança aberta".
- **Grafo:** Cytoscape com layout `cose` (`animate: false`, `nodeDimensionsIncludeLabels: true`).
  - Nó: nome e release (`v2.3.0 · há 11 dias`, "sem release" ou "não clonado"). Classes para ficha `stale` ou `missing` e para mudança aberta.
  - Aresta do consumidor para o fornecedor, com rótulo = `id`. Cada estado tem seu estilo: `proven` sólida, `declared` sólida com outro tom, `unverified` pontilhada, `pending` tracejada para um único nó `?`.
  - Com mais de 40 arestas, os rótulos só aparecem com zoom acima de 1.2 ou na aresta selecionada ou sob o mouse.
- **Clique num repo:**
  - classes `selected`, `direct`, `indirect` e `possible`; o resto fica com `faded`;
  - o painel mostra descrição, stack, release com "N commits depois", como rodar, o que fornece, o que consome (arestas com `consumer` = ele), os afetados nas três seções (com o `id`, a evidência, o caminho ou o motivo), as mudanças abertas e o botão Focar.
- **Clique numa aresta:** o painel mostra `id`, tipo, estado, `via`, evidências, candidatos e `problems`.
- **Focar:** mostra só o repo e os vizinhos até N níveis (1 a 3, padrão 1), sem levar em conta a direção. "Mostrar tudo" desfaz.
- **Busca:**
  - um trecho do nome de um repo (sem diferenciar maiúsculas) seleciona o primeiro;
  - um `id` exato destaca o fornecedor e todos os consumidores.
- **Segurança:**
  - Todo texto entra com `textContent`.
  - Os links são `<a>` com `href` só quando a URL começa com `https://`, mais `target="_blank"` e `rel="noopener noreferrer"`.
  - Sem `innerHTML` com dado vindo do JSON.

- [ ] **Step 5: Rodar e ver passar**

Run: `go test -race ./internal/view/`
Expected: PASS

- [ ] **Step 6: Conferir no navegador**

Gerar um build de desenvolvimento e rodar `trama graph --serve --no-open` numa cópia do `trama-demo` no scratchpad. Abrir a URL no navegador embutido e conferir:
- o console não mostra erro de JavaScript nem violação de CSP;
- o grafo, o clique num repo, o clique numa aresta, o foco, a busca e os filtros funcionam.

- [ ] **Step 7: Commit**

```bash
git add internal/view
git commit -m "feat: interactive graph page with Cytoscape.js"
```

---

### Task 8: Documentação e critérios de aceite

**Files:**
- Modify: `README.md`, `README.pt-BR.md`

- [ ] **Step 1: Documentar**

Nos dois READMEs:
- uma linha na tabela de comandos para `trama graph --serve [--port <n>] [--no-open]`;
- um parágrafo em "Mapear os repositórios" / "Map the repositories": o que a página mostra e que tudo roda na máquina, sem nada sair dela.

- [ ] **Step 2: Conferir os critérios da spec §9**

Os critérios rodam com um build de desenvolvimento e cópias no scratchpad. Nada é enviado, e as tags são locais. A conferência é no navegador embutido.
1. Na cópia do `trama-demo`: a página abre com os dois repos e a seta `GET /hello`.
2. Clicar em `trama-demo-api` mostra o web como afetado direto, com o link para `src/api/hello.ts` no GitHub na linha certa.
3. Sem tag semver aparece "sem release". Com `v0.1.0` local e dois commits locais depois dela, num remoto de teste, aparecem a versão, a data e "2 commits depois".
4. `TRAMA_PERF=1 go test ./internal/view/ -run Hundred -v` fica abaixo de 2 s. A página com esse workspace (gerado pelo mesmo helper num diretório do scratchpad) abre, e a busca, o foco e os filtros respondem.
5. Uma relação `from: unknown` aparece tracejada para o `?`, e o candidato a lista em "possíveis".
6. Uma descrição com `<script>alert(1)</script>` aparece como texto, sem alerta.
7. Uma evidência quebrada (linha 999) aparece na faixa do topo; a seta fica pontilhada; o fornecedor lista o consumidor em "possíveis".
8. `?lang=en` mostra inglês e `?lang=pt` mostra português.
9. A aba de rede (`read_network_requests`) não mostra nenhum pedido fora de `127.0.0.1`.
10. `go test -race ./...`, `go vet ./...` e `gofmt -l .` passam sem falhas.

- [ ] **Step 3: Commit**

```bash
git add README.md README.pt-BR.md
git commit -m "docs: trama graph --serve"
```

- [ ] **Step 4: Revisão final**

Um revisor novo, no modelo mais capaz, lê o branch inteiro (`git diff main...graph-ui`) contra a spec e este plano. Cada achado é corrigido com um teste que falha antes da correção. Release, push e merge só com OK do usuário.
