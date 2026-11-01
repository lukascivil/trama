# trama — Dado no grafo — Plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A página do `trama graph --serve` responde quem tem um dado: a caixa de busca acha campo de endpoint e coluna de tabela, e o painel de um repositório mostra os campos que ele devolve e as tabelas de que é dono.

**Architecture:** A busca que o `trama data` faz sai de `internal/cli/data.go` para `internal/datafind`, que devolve os resultados já na ordem útil (response, request, tabela) com o texto pesquisável que o Go decidiu. O comando e o `internal/view` passam a usar o mesmo pacote: `Model.Data` é `datafind.All`. O JavaScript só faz `includes` nesse texto e desenha, sem nenhuma regra nova.

**Tech Stack:** Go 1.25, cobra, `go.yaml.in/yaml/v3`, HTML/CSS/JS sem build, Cytoscape.js vendorizado.

**Spec:** `docs/superpowers/specs/2026-11-01-trama-dado-no-grafo-design.md`. Base: `main` em `08e7e7f`. Trabalho na branch `data-in-graph`.

## Global Constraints

- Nenhuma dependência Go nova. Nenhuma biblioteca JavaScript nova.
- Nenhuma regra de busca no JavaScript: o que casa é o campo `match` que o Go escreve, e a ordem é a ordem do array. O JS faz `includes` e desenha.
- Endpoint novo é proibido: `GET /{$}` e `GET /api/workspace` continuam sendo a superfície inteira do servidor.
- Tabela não é nó nem aresta nem entra no cálculo de afetados.
- Todo texto vindo de ficha entra na página com `textContent` (helpers `el` e `section`), nunca com HTML.
- A ordem da busca é id exato → nome de repo → dado. Os dois primeiros degraus não mudam de comportamento.
- Inglês e português na página, como já é; `?lang=` continua valendo.
- Commits terminam com `Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>`. Sem push, tag ou release sem OK explícito do usuário.

## Review Focus

1. **Workspace sem nenhum campo e sem nenhuma tabela**: `Model.Data` vem vazio, a página não quebra e a busca cai em "nada encontrado" como hoje. Testes nas Tasks 2 e 3.
2. **Termo que casa id exato e também coluna** (um repo `users` com uma tabela `users`): os dois primeiros degraus da busca têm que continuar ganhando, senão a busca de hoje regride. Teste na Task 3.
3. **Repositório sem ficha ou não clonado** (`cards[name] == nil`): `datafind.All` não entra em pânico e simplesmente não rende hits para ele. Teste na Task 1.
4. **Texto hostil em caminho de campo e nome de coluna** (`<script>`, aspas, Unicode): chega à página como texto, nunca como HTML. Teste na Task 3.
5. **Termo vazio ou só com espaço**: `Find` devolve nada, o comando continua errando com a mesma mensagem e a página não lista o workspace inteiro. Testes nas Tasks 1 e 3.

---

### Task 1: `internal/datafind`

**Files:**
- Create: `internal/datafind/datafind.go`, `internal/datafind/datafind_test.go`
- Modify: `internal/cli/data.go` (passa a usar o pacote)
- Test: `internal/datafind/datafind_test.go`, `internal/cli/data_test.go` (sem mudança: o comportamento do comando não muda)

**Interfaces:**
- Produces:
  - `datafind.Hit struct { Layer, Repo, ID, Table, Path, Type, Match string; Evidence []string }`, com `Layer` em `response | request | table`, `ID` vazio num hit de tabela, `Table` e `Path` vazios num hit de campo (o `Path` é a coluna no hit de tabela), `Type` vazio num hit de tabela, e `Match` o texto que o termo casa: o caminho do campo, ou `tabela.coluna`.
  - `func datafind.All(repos []string, cards map[string]*catalog.Card) []Hit`: toda entrada pesquisável, agrupada por camada na ordem response, request, tabela, e dentro de cada camada na ordem de `repos`, com os caminhos de campo ordenados. Um nome de `repos` sem ficha em `cards` é ignorado.
  - `func datafind.Find(repos []string, cards map[string]*catalog.Card, term string) []Hit`: `All` filtrado por `Match` contendo `term`, sem distinguir maiúsculas; `nil` quando `term` é vazio ou só espaço.

- [ ] **Step 1: Criar a branch e commitar o spec e o plano**

```bash
git switch -c data-in-graph
git add docs/superpowers/specs/2026-11-01-trama-dado-no-grafo-design.md docs/superpowers/plans/2026-11-01-trama-dado-no-grafo.md
git commit -m "docs: data in the graph page design and plan"
```

- [ ] **Step 2: Escrever os testes que falham**

`internal/datafind/datafind_test.go`, com fichas montadas em memória (`&catalog.Card{...}`, sem disco):
- `TestAllOrdersByLayerThenRepo`: dois repos, cada um com `response`, `request` e `tables`; a ordem dos `Layer` do resultado é todos `response` (repo A, repo B), todos `request`, todos `table`.
- `TestAllSortsFieldPaths`: um `response` com `b`, `a`, `c` sai como `a`, `b`, `c`.
- `TestAllTableHitPerColumn`: uma tabela com três colunas rende três hits, cada um com `Match == "users.<coluna>"`, `Table == "users"`, `Path` a coluna e `Type` vazio.
- `TestAllSkipsRepoWithoutCard`: um nome em `repos` sem entrada em `cards` não rende hit e não entra em pânico.
- `TestAllFieldHitCarriesTypeAndEvidence`: um hit de `response` tem `ID` da rota, `Type` do campo e a evidência do `provides`.
- `TestFindIsCaseInsensitiveSubstring`: `Find(..., "EMAIL")` acha `customer.email` e a coluna `email`.
- `TestFindEmptyTermFindsNothing`: `""` e `"   "` devolvem `nil`.
- `TestFindMatchesOnlyTheMatchField`: um termo igual ao nome de um repo, mas que não aparece em nenhum `Match`, devolve nada.

- [ ] **Step 3: Rodar os testes para ver falhar**

Run: `mise exec -- go test ./internal/datafind/`
Expected: FAIL — o pacote não existe.

- [ ] **Step 4: Implementar `Hit`, `All` e `Find` em `internal/datafind/datafind.go`**

`All` em três passadas sobre `repos`, uma por camada, com `slices.Sorted(maps.Keys(...))` nos mapas de campo, como `internal/cli/data.go` já faz. `Find` filtra com `strings.Contains(strings.ToLower(h.Match), term)`.

- [ ] **Step 5: Reescrever `internal/cli/data.go` sobre o pacote**

O comando continua carregando as fichas e acumulando `unmapped`, chama `datafind.Find` e formata os hits com os mesmos três formatos de linha de hoje (`%-8s` no rótulo). A recusa de termo vazio e a mensagem de nada encontrado não mudam.

- [ ] **Step 6: Rodar os testes para ver passar**

Run: `mise exec -- go test ./internal/datafind/ ./internal/cli/`
Expected: PASS, com os testes de `data_test.go` intactos — é o que prova que a extração não mudou o comando.

- [ ] **Step 7: Commit**

```bash
git add internal/datafind internal/cli
git commit -m "refactor: extract the data search into internal/datafind"
```

---

### Task 2: `Model.Data`

**Files:**
- Modify: `internal/view/model.go` (`Model`, e o `Build` que o preenche)
- Test: `internal/view/model_test.go`

**Interfaces:**
- Consumes: `datafind.All` e `datafind.Hit` (Task 1).
- Produces:
  - `view.DataEntry struct { Layer, Repo, ID, Table, Path, Type, Match string; Evidence []Link }`, com tags JSON minúsculas (`layer`, `repo`, `id,omitempty`, `table,omitempty`, `path`, `type,omitempty`, `match`, `evidence`), porque é o que o JavaScript lê.
  - `Model` ganha `Data []DataEntry \`json:"data,omitempty"\``, preenchido a partir de `datafind.All` sobre os repos na ordem do `workspace.yaml` e as fichas que o `Build` já carregou. A evidência de cada hit passa pelo `links(r.URL, c.MappedAtCommit, ...)` que o pacote já usa, para a linha citada virar link clicável como nas outras seções. `datafind` não conhece URL nem commit, e é por isso que o tipo do JSON é do `view`, não do `datafind`.

- [ ] **Step 1: Escrever os testes que falham**

`internal/view/model_test.go`, sobre a fixture de duas fichas que o pacote já tem:
- `TestModelCarriesData`: a ficha da api ganha um `response` e uma `tables`; o `Model.Data` tem os hits, o primeiro com `Layer == "response"` e o último com `Layer == "table"`.
- `TestModelDataEmptyWithoutFieldsOrTables`: com as fichas como estão hoje, `Model.Data` vem vazio.
- `TestModelDataJSONKey`: o JSON do modelo serializa a chave `data` (e a omite quando vazia), e cada entrada traz as chaves minúsculas `layer`, `repo`, `match` e `evidence`.
- `TestModelDataEvidenceIsLinked`: a evidência de um hit vem como `[]Link` com `url` apontando para o commit no GitHub, igual à de `provides`.

- [ ] **Step 2: Rodar os testes para ver falhar**

Run: `mise exec -- go test ./internal/view/`
Expected: FAIL — `Model` não tem campo `Data`.

- [ ] **Step 3: Preencher `Model.Data` em `internal/view/model.go`**

No `Build`, depois do laço que monta os repos, com os nomes na ordem de `ws.Repos` e o mapa de fichas que o laço já tem em mão.

- [ ] **Step 4: Rodar os testes para ver passar**

Run: `mise exec -- go test ./internal/view/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/view
git commit -m "feat: the graph model carries the searchable data index"
```

---

### Task 3: A página

**Files:**
- Modify: `internal/view/assets/app.js` (`STRINGS` nas duas línguas, `search`, o painel do repo)
- Modify: `internal/view/assets/app.css` (se as seções novas precisarem de regra; só se precisarem)
- Test: verificação no navegador contra um workspace de fixture, mais `mise exec -- go test ./...`

**Interfaces:**
- Consumes: `model.data` (Task 2), cada item com `layer`, `repo`, `id`, `table`, `path`, `type`, `match` e `evidence` (lista de `{text, url}`, que o helper `evidenceList` já desenha).

- [ ] **Step 1: Escrever as chaves de i18n e o degrau de busca em `app.js`**

Chaves novas em `en` e `pt`: `data` ("Data" / "Dado"), `dataHits` (função com a contagem), `inResponse`/`inRequest`/`inTable` como rótulo de linha, `ownedTables` ("Tables it owns" / "Tabelas de que é dono"), `fields` ("Fields" / "Campos"). O placeholder da busca passa a citar também um campo (`Search a repo, an id or a field (email…)`).

No `search`, um degrau novo **depois** do casamento de id e **depois** do casamento por nome de repo, antes do `noMatch`: filtra `model.data` por `h.match.toLowerCase().includes(lower)`, destaca no grafo os repos dos hits e renderiza o painel com os hits na ordem em que vêm, agrupados por `layer`, cada linha com o id ou a tabela, o caminho, o tipo e a evidência — toda string via `el`, nunca HTML.

- [ ] **Step 2: Mostrar os campos e as tabelas no painel do repositório**

No bloco que já desenha `provides`, acrescentar os campos daquele `provides` a partir de `model.data` (hits de `response` e `request` com `repo` e `id` iguais). Depois do bloco de `provides`, uma seção `ownedTables` com os hits de `layer === "table"` daquele repo, agrupados por tabela, listando as colunas.

- [ ] **Step 3: Verificar no navegador**

```bash
mise exec -- go build -o /tmp/trama-graph ./cmd/trama
```

Montar um workspace de fixture com duas fichas que tenham `response`, `request` e `tables`, servir com `/tmp/trama-graph graph --serve --no-open --port 7399` e abrir `http://127.0.0.1:7399` no navegador embutido.
Expected, conferido na página:
- buscar `email` lista os hits na ordem response → request → tabela, com repo, id/tabela e evidência;
- buscar o nome de um repo ainda seleciona o repositório, e buscar um id exato ainda destaca a relação;
- clicar num repositório mostra os campos sob `provides` e a seção de tabelas;
- um campo chamado `<script>alert(1)</script>` aparece como texto na página;
- buscar um termo que não existe mostra a mensagem de nada encontrado.

- [ ] **Step 4: Rodar a suíte**

Run: `mise exec -- go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/view/assets
git commit -m "feat: the graph page answers who holds a piece of data"
```

---

### Task 4: Docs

**Files:**
- Modify: `README.md`, `README.pt-BR.md` (o parágrafo do `graph --serve`)
- Modify: `site/src/i18n.ts` (o pilar "Map" / "Mapear", nas duas línguas)
- Test: `mise exec -- go test ./...` e uma leitura da página servida

- [ ] **Step 1: Atualizar o parágrafo do `graph --serve` nos dois READMEs**

A frase de hoje lista release, relações com prova, afetados, busca, filtros e foco. Acrescentar que a busca também acha campo de endpoint e coluna de tabela, e que o painel de um repositório mostra os campos que ele devolve e as tabelas de que é dono.

- [ ] **Step 2: Acrescentar uma frase ao pilar de mapear em `site/src/i18n.ts`**

Nas duas línguas: a ficha também guarda o que cada endpoint recebe e devolve e as tabelas de que o repositório é dono, então dá para perguntar quem tem um dado. Sem pilar novo: os quatro são o fluxo de trabalho, e consultar dado não é um passo dele.

- [ ] **Step 3: Rodar a suíte e conferir o site**

Run: `mise exec -- go test ./... && cd site && mise exec -- npm run build`
Expected: PASS e build do site sem erro.

- [ ] **Step 4: Commit**

```bash
git add README.md README.pt-BR.md site/src/i18n.ts
git commit -m "docs: the page and the site mention data discovery"
```
