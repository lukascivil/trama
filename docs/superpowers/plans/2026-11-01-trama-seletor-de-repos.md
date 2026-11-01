# Seletor de repositórios: plano de implementação

> **Para quem executa:** SUB-SKILL OBRIGATÓRIA: use superpowers:subagent-driven-development (recomendado) ou superpowers:executing-plans para implementar este plano tarefa por tarefa. Os passos usam checkbox (`- [ ]`).

**Objetivo:** `trama add --pick [<org>]` lista os repos de uma organização do GitHub numa página local com filtro e checkboxes, adiciona os escolhidos ao `workspace.yaml` de uma vez, e a skill `/trama:add-repo` segue com `sync` e `map`.

**Arquitetura:** um pacote `internal/github` fala com a API (o token nunca sai dele). Um pacote `internal/pick` guarda o estado da escolha e serve a página e uma API JSON em `127.0.0.1`, reaproveitando as proteções do servidor do grafo. O `internal/cli` liga tudo ao `trama add --pick`. A página é React, gerada pelo mesmo Vite do grafo e commitada em `internal/pick/assets`.

**Tecnologias:** Go (só biblioteca padrão, `cobra`, `yaml`), React 19 + Vite + Vitest como a página do grafo.

**Spec:** [2026-11-01-trama-seletor-de-repos-design.md](../specs/2026-11-01-trama-seletor-de-repos-design.md)

## Restrições globais

- Dependências Go novas: nenhuma além da biblioteca padrão.
- Servidor só em `127.0.0.1`; recusa `Host` fora do esperado e `Sec-Fetch-Site` de outro site; mesma CSP do grafo (`default-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'`).
- O token do GitHub nunca aparece na página, nas respostas, nos erros nem na saída.
- A página só chama `/api/repos`, `/api/add` e `/api/cancel`, e renderiza texto do GitHub como texto.
- O bundle da página fica commitado em `internal/pick/assets`; quem usa o trama não precisa de Node.
- Testes sem rede: o cliente do GitHub aponta para um `httptest.Server`.
- `trama add <url>` e `/trama:add-repo <url>` não mudam de comportamento.
- Saída do comando: nomes adicionados, um por linha, em stdout; URL, avisos e recusas em stderr.
- Janela ociosa de 30 minutos; depois disso o comando sai sem alterar o `workspace.yaml`.

## Foco da revisão

Entradas que a spec implica e nenhuma tarefa exercita por si só; cada uma tem seu teste na tarefa dona:

1. Repo `.github` (comum em organizações): o nome é inválido para o trama. Aparece bloqueado com o motivo, e os outros seguem. (Tarefa 3)
2. `description` e `language` nulos na API: viram texto vazio, não quebram a página. (Tarefas 1 e 5)
3. Organização inexistente, sem repos ou 403 por SSO / limite de requisições: a página mostra a mensagem do GitHub, nunca o token. (Tarefas 1 e 3)
4. Clique duplo em "Adicionar": o segundo pedido recusa tudo ("já no workspace") e nada é gravado duas vezes. (Tarefa 3)
5. Mesmo nome de repo em duas organizações: o segundo é bloqueado com o motivo, não renomeado. (Tarefa 3)

---

### Tarefa 1: cliente do GitHub

**Arquivos:**
- Criar: `internal/github/github.go`, `internal/github/token.go`
- Testar: `internal/github/github_test.go`, `internal/github/token_test.go`

**Interfaces:**
- Produz:
  - `type Repo struct { FullName, Name, Description, Language, DefaultBranch, CloneURL, SSHURL string; Archived bool; PushedAt time.Time }` com tags JSON da API (`full_name`, `default_branch`, `clone_url`, `ssh_url`, `pushed_at`); `Description` e `Language` aceitam `null`.
  - `type Client struct { BaseURL, Token string; HTTP *http.Client }` e `func NewClient(token string) *Client` (BaseURL `https://api.github.com`, `HTTP` com timeout de 30 s).
  - `func (c *Client) Repos(ctx context.Context, org string) ([]Repo, error)`: `GET /orgs/{org}/repos?per_page=100&type=all`, segue `Link: rel="next"`; em 404 tenta `GET /users/{org}/repos`.
  - `func (c *Client) Orgs(ctx context.Context) ([]string, error)`: o login do dono do token (`/user`) seguido dos de `/user/orgs`; sem token devolve `nil, nil`.
  - `type APIError struct { Status int; Message string }` com `Error()` no formato `github: <status> <message>`.
  - `type Gh func(args ...string) (string, error)`; `func ExecGh(args ...string) (string, error)` roda `gh` e devolve a saída sem a quebra final.
  - `func Token(gh Gh, getenv func(string) string) string`: `gh auth token`, depois `GH_TOKEN`, depois `GITHUB_TOKEN`; vazio se nenhum.
  - `func Protocol(gh Gh) string`: `"ssh"` se `gh config get git_protocol` devolve `ssh`; senão `"https"`.

- [ ] **Passo 1: testes que falham** (`github_test.go`, contra `httptest`)
  - `TestReposFollowsPagination`: 3 páginas (100, 100, 50) ligadas por `Link`; espera 250 repos em ordem.
  - `TestReposSendsToken`: com `Token: "SECRET"`, o servidor vê `Authorization: Bearer SECRET`; sem token, não há o cabeçalho.
  - `TestReposFallsBackToUser`: `/orgs/x/repos` dá 404 e `/users/x/repos` dá 200; espera os repos de `/users`.
  - `TestReposNullFields`: JSON com `"description": null, "language": null` decodifica para strings vazias.
  - `TestReposRefusesForeignNextLink`: `Link` aponta para outro host; espera erro e que o outro host não receba nenhum pedido.
  - `TestErrorsNeverHoldTheToken`: o servidor responde 403 `{"message":"Resource protected by organization SAML enforcement"}` e ecoa o cabeçalho no corpo; `err.Error()` contém a mensagem e não contém `SECRET`.
  - `TestOrgsListsOwnerThenOrgs` e `TestOrgsWithoutTokenIsEmpty`.
- [ ] **Passo 2: rodar e ver falhar**: `go test ./internal/github/ -v` → FAIL (pacote não existe).
- [ ] **Passo 3: implementar** `Repos`, `Orgs`, `APIError` em `github.go`. O `Link` seguinte só vale se tiver o mesmo host de `BaseURL`. Mensagem de erro vem do campo `message` do corpo JSON, nunca do corpo bruto.
- [ ] **Passo 4: testes de token** (`token_test.go`): `TestTokenOrder` (gh vence env; `GH_TOKEN` vence `GITHUB_TOKEN`; tudo vazio dá `""`), `TestProtocolSSH`, `TestProtocolDefaultsToHTTPS` (gh falha ou devolve `https`). Implementar `Token`, `Protocol`, `ExecGh` em `token.go`.
- [ ] **Passo 5: verificar**: `go test ./internal/github/ -race -v` → PASS.
- [ ] **Passo 6: commit**: `git add internal/github && git commit -m "feat: a GitHub client that lists the repositories of an organization"`

---

### Tarefa 2: guarda de segurança reaproveitável

**Arquivos:**
- Modificar: `internal/view/server.go` (a parte de `Handler` que põe cabeçalhos e checa `Host` e `Sec-Fetch-Site`)
- Testar: `internal/view/server_test.go`

**Interfaces:**
- Produz: `func Guard(next http.Handler, port int) http.Handler` com exatamente as checagens que `Handler` faz hoje, sem restringir o método. `Handler` passa a ser `Guard(mux, port)`.

- [ ] **Passo 1: teste que falha**: `TestGuardRefusesForeignHostAndSite` em `server_test.go`: um `Guard` sobre um handler que responde 204 devolve 403 para `Host: evil.test`, 403 para `Sec-Fetch-Site: cross-site`, 204 para `127.0.0.1:<port>` sem `Sec-Fetch-Site`; os cabeçalhos CSP e `X-Content-Type-Options` aparecem nas três respostas.
- [ ] **Passo 2**: `go test ./internal/view/ -run TestGuard -v` → FAIL (`Guard` indefinido).
- [ ] **Passo 3: extrair `Guard`** de `Handler`, sem mudar o comportamento.
- [ ] **Passo 4: verificar**: `go test ./internal/view/ -race` → PASS (os testes existentes do servidor do grafo seguem passando).
- [ ] **Passo 5: commit**: `git commit -am "refactor: share the local-server guard between pages"`

---

### Tarefa 3: estado da escolha e API do seletor

**Arquivos:**
- Criar: `internal/pick/pick.go` (estado e `Apply`), `internal/pick/server.go` (HTTP), `internal/pick/assets/index.html` (provisório, a Tarefa 5 o substitui)
- Testar: `internal/pick/pick_test.go`, `internal/pick/server_test.go`

**Interfaces:**
- Consome: `github.Repo`, `view.Guard`, `workspace.Workspace` (`CheckNew`, `AddRepo`, `Save`), `workspace.NameFromURL`.
- Produz:
  - `type Options struct { Workspace *workspace.Workspace; Org string; Orgs func(context.Context) ([]string, error); List func(context.Context, string) ([]github.Repo, error); Protocol string; Authenticated bool }`
  - `func New(o Options) *Picker`
  - `func (p *Picker) Handler(port int) http.Handler`
  - `func (p *Picker) Wait(ctx context.Context, idle time.Duration) error`: `nil` quando a escolha termina (adicionou pelo menos 1, ou cancelou), `ErrIdle` quando passa `idle` sem requisição, `ctx.Err()` se o contexto acabar.
  - `func (p *Picker) Added() []string`: nomes adicionados, na ordem.
  - `var ErrIdle = errors.New("no request for too long")`
  - Rotas: `GET /api/repos?org=<org>` → `{"token": "...", "org": "...", "orgs": [...], "authenticated": bool, "error": "", "repos": [{"id","name","description","language","pushedAt","archived","blocked"}]}` onde `id` é o `full_name` e `blocked` é o texto do erro de `CheckNew` (vazio se livre). `POST /api/add` com `{"org","ids"}` → `{"added":[{"name","url","branch"}],"refused":[{"id","reason"}]}`. `POST /api/cancel` → 204. Os dois `POST` exigem `Content-Type: application/json` e o cabeçalho `X-Trama-Token` igual ao valor de `token` (aleatório por `Picker`); senão 403.

- [ ] **Passo 1: testes de `Apply`/estado que falham** (`pick_test.go`, com um workspace temporário de `workspace.Load` e repos fabricados):
  - `TestAddWritesOnceAndKeepsOrder`: ids de 3 repos livres → 3 `added`; `workspace.yaml` relido tem os 3 com `Branch` igual ao `DefaultBranch` e URL `clone_url` (protocolo `https`) ou `ssh_url` (protocolo `ssh`).
  - `TestAddRefusesWithReason`: um id com nome já existente e outro de nome `.github` → ficam em `refused` com o motivo do `CheckNew`; os demais entram.
  - `TestAddSameNameFromTwoOrgs`: `a/api` e `b/api` marcados juntos → o primeiro entra, o segundo é recusado.
  - `TestAddDoubleSubmit`: a mesma chamada duas vezes → a segunda recusa tudo, o arquivo tem cada repo uma vez.
  - `TestAddIgnoresIdsNotInTheList`: id que o servidor não listou é recusado ("not in the list").
  - `TestBlockedMarksWorkspaceRepos`: repo já no workspace vem com `blocked` preenchido.
- [ ] **Passo 2: testes de HTTP que falham** (`server_test.go`, com `httptest` sobre `Handler`):
  - `TestReposNeedsOrgOrOrgs`: sem `Org` e sem `?org=`, devolve `orgs` e `repos: []`.
  - `TestPostWithoutTokenIsForbidden`, `TestPostWrongContentTypeIsForbidden`, `TestForeignHostIsForbidden`: nenhum escreve no `workspace.yaml`.
  - `TestListErrorReachesThePage`: `List` devolve erro com a mensagem do GitHub → `error` na resposta; a resposta não contém o token.
  - `TestDescriptionIsJSONText`: descrição `<script>x</script>` volta como string JSON escapada (sem interpretação).
  - `TestWaitEndsOnAddAndOnCancel`, `TestWaitStaysOpenWhenEverythingRefused`, `TestWaitIdle` (com `idle` de 50 ms → `ErrIdle`).
- [ ] **Passo 3: rodar e ver falhar**: `go test ./internal/pick/ -v` → FAIL.
- [ ] **Passo 4: implementar** com um `sync.Mutex` em volta de listar-cache/`Apply`. `Apply` valida cada id contra a lista que o próprio `Picker` guardou por organização, deriva o nome com `NameFromURL`, usa `CheckNew` e `AddRepo`, e chama `Save` uma única vez no fim. Se `Save` falhar, devolve 500 com o erro e mantém `Wait` aberto. Cada requisição atualiza o instante usado por `Wait`. Serve `GET /` e `GET /assets/` a partir de `//go:embed assets`.
- [ ] **Passo 5: verificar**: `go test ./internal/pick/ -race -v` → PASS.
- [ ] **Passo 6: commit**: `git add internal/pick && git commit -m "feat: the picker state and its local API"`

---

### Tarefa 4: `trama add --pick`

**Arquivos:**
- Criar: `internal/cli/pick.go`
- Modificar: `internal/cli/add.go` (flags e argumentos), `internal/cli/graph.go` (extrair o `net.Listen` com queda para porta livre de `serveGraph` em `listenLocal`)
- Testar: `internal/cli/pick_test.go`, e rodar `internal/cli/graph_serve_test.go` sem mudanças

**Interfaces:**
- Consome: `pick.New`, `Picker.Handler`, `Picker.Wait`, `Picker.Added`, `github.NewClient`, `github.Token`, `github.Protocol`, `openURL`, `noBrowser` (teste).
- Produz:
  - `func listenLocal(port int, explicit bool) (net.Listener, error)`: o comportamento atual de `serveGraph` (porta ocupada sem `--port` cai para uma livre; com `--port` é erro).
  - `func runPick(cmd *cobra.Command, ws *workspace.Workspace, org string, port int, explicit, open bool) error`
  - Variáveis de pacote para teste: `var githubAPI = "https://api.github.com"`, `var ghRun github.Gh = github.ExecGh`, `var pickIdle = 30 * time.Minute`.
  - Flags novas em `add`: `--pick`, `--no-open`, `--port` (padrão 7300, mesma regra do grafo). Com `--pick`, o argumento é opcional (a organização) e `--name`/`--branch` dão erro. Sem `--pick`, `--no-open` e `--port` dão erro, como no `graph`, e continua exigindo exatamente uma URL.

- [ ] **Passo 1: testes que falham** (`pick_test.go`; GitHub falso via `httptest` com `githubAPI` apontando para ele; `ghRun` devolve `SECRET` para `auth token`; `pickIdle` curto onde precisar; reaproveita `serve` e `noBrowser`):
  - `TestPickAddsChosenRepos`: `add --pick acme --no-open --port 0 --dir <ws>` sobe; o teste faz `GET /api/repos?org=acme`, depois `POST /api/add` com 2 ids; o comando termina com sucesso, stdout é exatamente `api\nweb\n` e o `workspace.yaml` tem os dois.
  - `TestPickCancelLeavesWorkspaceAlone`: `POST /api/cancel` → comando termina com 0, stdout vazio, arquivo idêntico ao de antes.
  - `TestPickInterruptExits130`: cancelar o contexto → `errors.Is(err, runner.ErrInterrupted)`; arquivo idêntico.
  - `TestPickIdleStops`: com `pickIdle = 100ms` e nenhuma requisição, termina sem alterar o arquivo, com erro que diz que ficou ocioso.
  - `TestPickPrintsServingLineToStderrOnly`: a linha `serving http://127.0.0.1:<porta>/` não aparece em stdout.
  - `TestPickNeverPrintsTheToken`: nem stdout nem stderr contêm `SECRET`.
  - `TestPickFlagRules`: `--pick` com `--name` dá erro; `--no-open` sem `--pick` dá erro; sem `--pick` e sem URL dá o erro de argumentos de hoje.
  - Os testes existentes de `add` (`add_test.go`, `add_local_test.go`, `add_validation_test.go`) passam sem alteração.
- [ ] **Passo 2: rodar e ver falhar**: `go test ./internal/cli/ -run Pick -v` → FAIL.
- [ ] **Passo 3: implementar**: extrair `listenLocal`; em `add.go` ramificar para `runPick` quando `--pick`; `runPick` monta `Options` (`Authenticated` = token não vazio; `Protocol` = `github.Protocol(ghRun)`; `Orgs` e `List` pelo `Client`), sobe o servidor como `serveGraph` faz (mesmo tratamento de sinais e `Shutdown` de 2 s), abre o navegador com `openURL` salvo `--no-open`, espera `Picker.Wait(ctx, pickIdle)` e imprime os nomes em stdout. Sinal ou contexto cancelado devolve `runner.ErrInterrupted`.
- [ ] **Passo 4: verificar**: `go test ./internal/cli/ -race` → PASS.
- [ ] **Passo 5: commit**: `git add internal/cli && git commit -m "feat: trama add --pick lists an organization and adds the chosen repositories"`

---

### Tarefa 5: a página

**Arquivos:**
- Criar: `web/vite.pick.config.ts`, `web/pick.html` (só para o `vite dev`), `web/src/pick/main.tsx`, `web/src/pick/App.tsx`, `web/src/pick/filter.ts`, `web/src/pick/types.ts`, `web/src/pick/strings.ts`, `web/src/pick/pick.css`
- Substituir: `internal/pick/assets/index.html` (final); gerados e commitados: `internal/pick/assets/pick.js`, `internal/pick/assets/pick.css`
- Modificar: `web/package.json` (o script `build` passa a rodar também `vite build -c vite.pick.config.ts`), `.github/workflows/ci.yml` (o `git diff --exit-code` cobre `../internal/pick/assets` além de `../internal/view/assets`)
- Testar: `web/src/pick/filter.test.ts`, `internal/pick/assets_test.go`

**Interfaces:**
- Consome: as rotas da Tarefa 3.
- Produz em `filter.ts`:
  - `export function filterRepos(repos: Repo[], query: string, showArchived: boolean): Repo[]`: sem diferenciar maiúsculas, em `name` e `description`; arquivados só com `showArchived`.
  - `export function setMatching(selected: Set<string>, matching: Repo[], on: boolean): Set<string>`: marca ou desmarca só os de `matching` com `blocked === ""`, devolvendo um conjunto novo.

- [ ] **Passo 1: testes que falham** (`filter.test.ts`): `filterRepos` acha por pedaço do nome, por descrição, ignora maiúsculas, esconde arquivados por padrão e os mostra com `true`; `setMatching` não marca bloqueados, não mexe em ids fora de `matching`, e desmarcar remove só os que casam.
- [ ] **Passo 2**: `cd web && npm test` → FAIL.
- [ ] **Passo 3: implementar** `filter.ts`; depois `App.tsx`: busca `GET /api/repos` (e `?org=` ao trocar a organização), lista com filtro em foco, checkbox por linha, bloqueados desabilitados mostrando o motivo, rótulos "Marcar os N que casam" / "Desmarcar os que casam", botão "Mostrar arquivados", campo para digitar a organização quando não há `org` nem `orgs` (sem token), rodapé fixo com contagem, "Adicionar" e "Cancelar", aviso quando `authenticated` é falso, `error` e organização sem repos como mensagem na tela, e o resultado final (quem entrou, quem foi recusado e por quê, "pode fechar esta aba"). Seguir a detecção de idioma de `web/src/i18n.ts`, com `en` e `pt-BR`. Todo texto do GitHub entra como texto de React, sem `dangerouslySetInnerHTML`. O `POST` leva `Content-Type: application/json` e `X-Trama-Token` com o `token` recebido. Listas longas (mais de 1000 linhas) não podem travar o filtro: renderizar só as linhas visíveis.
- [ ] **Passo 4: build**: `cd web && npm ci && npm run typecheck && npm test && npm run build`; confirmar que `internal/pick/assets/pick.js` e `pick.css` foram gerados e que `internal/view/assets` não mudou (`git diff --stat -- internal/view/assets`).
- [ ] **Passo 5: testes dos arquivos embutidos que falham** (`internal/pick/assets_test.go`, no estilo de `internal/view/assets_test.go`): `TestAssetsEmbedded` (`index.html`, `pick.js`, `pick.css`); `TestAssetsStaySelfContained` (nenhum endereço fora do servidor, salvo os de React); `TestScriptRequestsOnlyThePickerAPI` (só `/api/repos`, `/api/add`, `/api/cancel`; sem `XMLHttpRequest`, `sendBeacon`, `WebSocket`, `EventSource`); `TestScriptNeverInjectsHTML` (`pick.js` não contém `dangerouslySetInnerHTML` nem `innerHTML`); `TestIndexHasNoInlineCode`.
- [ ] **Passo 6: verificar**: `go test ./internal/pick/ -race` → PASS.
- [ ] **Passo 7: commit**: `git add web internal/pick .github && git commit -m "feat: the picker page"`

---

### Tarefa 6: skill e documentação

**Arquivos:**
- Modificar: `skills/add-repo/SKILL.md`, `README.md` (linha 52 e a tabela de comandos, linha 72), `README.pt-BR.md` (mesmas linhas)

**Interfaces:**
- Consome: o comando e o contrato de saída da Tarefa 4.

- [ ] **Passo 1: reescrever a skill** com este fluxo:
  1. Argumento com `/`, `:` ou terminando em `.git`, ou nome simples com pasta de mesmo nome na pasta atual (`test -d`): o caminho de hoje, sem mudança.
  2. Sem argumento, ou nome simples sem pasta: rodar `trama add --pick [<org>]` em segundo plano, dizer ao dev que a página abriu no navegador e esperar o fim do comando.
  3. Se o comando terminou sem imprimir nomes, dizer que nada foi adicionado e parar. Se imprimiu, rodar `trama sync <nomes>` e mapear cada repo, um por vez, seguindo `/trama:map`, sem perguntar e sem limite.
  4. Se o comando falhou, mostrar o erro e parar.
- [ ] **Passo 2: README**: em `README.md` e `README.pt-BR.md`, `/trama:add-repo` aceita uma URL ou abre o seletor; a tabela ganha `trama add --pick [<org>]` (lista a organização no GitHub numa página local, o token vem do `gh` ou de `GH_TOKEN`/`GITHUB_TOKEN`, não clona).
- [ ] **Passo 3: verificar**: `go vet ./... && test -z "$(gofmt -l .)" && go test -race ./...` → sem saída de erro e PASS; ler a skill uma vez e conferir que os quatro casos acima estão nela.
- [ ] **Passo 4: commit**: `git add skills README.md README.pt-BR.md && git commit -m "docs: the add-repo skill opens the picker when it gets no URL"`
