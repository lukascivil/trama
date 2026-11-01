# trama — Descoberta de dado — Plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A ficha de catálogo passa a dizer quais campos cada endpoint recebe e devolve e quais tabelas o repositório é dono, e `trama data <termo>` responde quem tem um dado — primeiro por endpoint que já existe, depois por tabela.

**Architecture:** Dois campos novos em `catalog.Provide` (`request`, `response`, mapas planos de caminho para tipo), um campo novo em `catalog.Identity` (`tables`, com colunas) e validadores primitivos em `internal/catalog/field.go`, no lugar onde `ValidateTool` já mora. `internal/validate` ganha as regras; `internal/linker`, `internal/graph` e `internal/view` não mudam, porque tabela não é relação. Um comando novo, `trama data`, varre as fichas em três passadas (response, request, tabela) e imprime na ordem em que as respostas são úteis.

**Tech Stack:** Go 1.25, cobra, `go.yaml.in/yaml/v3`. Nenhuma dependência nova.

**Spec:** `docs/superpowers/specs/2026-11-01-trama-descoberta-de-dado-design.md`. Base: `main` em `67b07b0`. Trabalho na branch `data-discovery`.

## Global Constraints

- Nenhuma dependência Go nova: só `cobra` e `yaml`.
- Tabela não é relação: nada muda em `internal/linker`, `internal/graph`, `internal/view` nem em `trama uses`. A lista `vias` em `internal/validate/links.go:14` e a lista `kinds` em `internal/validate/card.go:22` ficam como estão.
- Evidência continua por entrada, nunca por campo.
- Caminho de campo é pontilhado e plano (`customer.email`), nunca YAML aninhado.
- Tipos aceitos, e só eles: `string`, `number`, `boolean`, `object`, `[]string`, `[]number`, `[]boolean`, `[]object`.
- `request` só em `kind: http`; `response` em `http` e `event`; nenhum dos dois em `package`.
- Código, mensagens de erro, docs do repositório e nomes de teste em inglês. Este plano e o spec em português.
- `catalog.Load` (não estrito) continua ignorando campo desconhecido; quem recusa `shape:` é `trama validate`.
- Commits terminam com `Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>`. Sem push, tag ou release sem OK explícito do usuário.

## Review Focus

1. **Tipo fora do vocabulário** (`str`, `string?`, `String`, vazio, `[]`): erro nomeando os tipos aceitos, nunca ficha aceita com tipo que a busca não entende. Teste na Task 1.
2. **`[]` no último segmento** (`tags[]`): erro dizendo para usar o tipo `[]string`. É a regra mais fácil de errar ao mapear. Teste na Task 1.
3. **Ficha antiga com `shape:`**: `trama validate` erra nomeando o rename, e os outros comandos (que usam `catalog.Load`) continuam rodando. Teste na Task 1.
4. **Tabela sem `columns`, ou com coluna vazia**: erro. Ficha que diz "sou dono de users" sem coluna não responde nada e não pode passar na validação. Teste na Task 2.
5. **Termo de busca com caractere de regex** (`.*`, `[`, `()`): `trama data` compara substring literal, nunca compila regex, e não acha nada em vez de casar com tudo. Teste na Task 3.

---

### Task 1: Campos de request e response nos provides

**Files:**
- Create: `internal/catalog/field.go`, `internal/catalog/field_test.go`
- Modify: `internal/catalog/catalog.go:88-94` (struct `Provide`)
- Modify: `internal/validate/card.go:80-90` (laços de `Provides` e `Consumes`)
- Modify: `skills/map/card-reference.md` (tabela de campos de `provides`, seção `provides`, exemplo)
- Test: `internal/catalog/field_test.go`, `internal/validate/card_test.go`

**Interfaces:**
- Produces:
  - `var catalog.FieldTypes = []string{"string", "number", "boolean", "object", "[]string", "[]number", "[]boolean", "[]object"}`
  - `func catalog.ValidateField(path, typ string) error`: checa o caminho sempre; checa o tipo só quando `typ != ""`, para o `fields_used` do consumidor poder reusar a mesma regra de caminho. Erro descreve o que está errado e é usado direto na mensagem do `validate`.
  - `catalog.Provide` ganha `Request map[string]string \`yaml:"request,omitempty"\`` e `Response map[string]string \`yaml:"response,omitempty"\``. `Shape map[string]any` continua no struct **só** para o `validate` poder recusá-lo com uma mensagem útil.

- [ ] **Step 1: Criar a branch e commitar o spec e o plano**

```bash
git switch -c data-discovery
git add docs/superpowers/specs/2026-11-01-trama-descoberta-de-dado-design.md docs/superpowers/plans/2026-11-01-trama-descoberta-de-dado.md
git commit -m "docs: data discovery design and plan"
```

- [ ] **Step 2: Escrever os testes que falham**

`internal/catalog/field_test.go`:
- `TestValidateFieldPath`: aceita `id`, `customer.email`, `items[].sku`, `a.b.c`, `user_id`, `userId`; recusa `""`, `.email`, `email.`, `a..b`, `1id`, `items[].`, `tags[]` (`[]` no último segmento), `a b`, `customer.*`.
- `TestValidateFieldType`: aceita cada item de `catalog.FieldTypes`; recusa `str`, `String`, `string?`, `[]`, `[]foo`, `""` quando o tipo é exigido.
- `TestValidateFieldSkipsEmptyType`: `ValidateField("customer.email", "")` devolve `nil`.

`internal/validate/card_test.go`, sobre a ficha de `helpers_test.go`:
- `TestCardResponseFieldTypeUnknown`: `response: { id: str }` → erro citando `provides[0] GET /hello` e os tipos aceitos.
- `TestCardResponseFieldPathInvalid`: `response: { "tags[]": string }` → erro.
- `TestCardRequestOnlyForHTTP`: `kind: event` com `request:` → erro `request is only for kind: http`.
- `TestCardResponseNotForPackage`: `kind: package` com `response:` → erro.
- `TestCardShapeRenamed`: `shape: { message: string }` → erro `shape was renamed to response`.
- `TestCardFieldsUsedPathInvalid`: `fields_used: ["a..b"]` → erro.
- `TestCardBodiesOK`: `request: { user_id: string }` e `response: { id: string, "customer.email": string }` → sem erro.

- [ ] **Step 3: Rodar os testes para ver falhar**

Run: `go test ./internal/catalog/ ./internal/validate/`
Expected: FAIL — `undefined: catalog.ValidateField`, `undefined: catalog.FieldTypes` e os campos novos desconhecidos no YAML.

- [ ] **Step 4: Implementar `ValidateField` e `FieldTypes` em `internal/catalog/field.go`**

Caminho: segmentos separados por `.`, cada um `[A-Za-z_][A-Za-z0-9_-]*` com `[]` opcional no fim, e `[]` proibido no último segmento (lista de escalares é o tipo). Segue o estilo de `ValidateTool`: regexp no nível do pacote, erro de uma frase.

- [ ] **Step 5: Adicionar `Request`, `Response` e manter `Shape` em `catalog.Provide`**

- [ ] **Step 6: Implementar `checkBodies(rep report, where string, p catalog.Provide)` em `internal/validate/card.go`**

Chamada do laço de `Provides`, logo depois de `checkKindID`, com o mesmo `where`. Itera as chaves com `slices.Sorted(maps.Keys(...))`, como o laço de `Tools` já faz, para a saída ser determinística. No laço de `Consumes`, validar cada `fields_used` com `catalog.ValidateField(f, "")`.

- [ ] **Step 7: Trocar `shape:` por `response:` nas fixtures existentes**

`internal/validate/helpers_test.go:42`, `internal/view/fixture_test.go:40` e `internal/catalog/card_test.go:60` escrevem `shape: { message: string }` numa ficha, e a regra nova recusa isso, então a suíte quebra sem este passo. Em `internal/catalog/card_test.go:104`, a afirmação sobre `Shape["message"]` passa a ser sobre `Response["message"]`. O `shape` em `docs/superpowers/specs/2026-10-27-trama-design.md:160` fica como está: é registro de um design passado, não ficha viva.

- [ ] **Step 8: Rodar os testes para ver passar**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 9: Documentar os campos em `skills/map/card-reference.md`**

Em `provides`: `request` e `response` na lista de campos, a regra de caminho pontilhado, os tipos aceitos, onde cada um vale, e `[]` só em segmento intermediário. Tirar `shape` da tabela e do exemplo, trocando por `response` no exemplo de `trama-demo-api`. Em `consumes`, dizer que `fields_used` usa a mesma forma de caminho.

- [ ] **Step 10: Commit**

```bash
git add internal/catalog internal/validate internal/view skills/map/card-reference.md
git commit -m "feat: request and response fields on provides"
```

---

### Task 2: Tabelas de que o repositório é dono

**Files:**
- Modify: `internal/catalog/field.go` (dois validadores novos)
- Modify: `internal/catalog/catalog.go:79-86` (`Identity`, tipo novo `TableFact`)
- Modify: `internal/validate/card.go:64-79` (laços de `Identity`)
- Modify: `skills/map/card-reference.md` (seção `identity`)
- Test: `internal/catalog/field_test.go`, `internal/catalog/card_test.go`, `internal/validate/card_test.go`

**Interfaces:**
- Consumes: o estilo de erro de `catalog.ValidateField` (Task 1).
- Produces:
  - `func catalog.ValidateColumn(name string) error`: `[A-Za-z_][A-Za-z0-9_]*`.
  - `func catalog.ValidateTable(name string) error`: um identificador, ou `schema.tabela` (dois no máximo).
  - `catalog.TableFact struct { Value string \`yaml:"value"\`; Columns []string \`yaml:"columns"\`; Evidence []string \`yaml:"evidence"\` }`
  - `catalog.Identity` ganha `Tables []TableFact \`yaml:"tables,omitempty"\``.

- [ ] **Step 1: Escrever os testes que falham**

`internal/catalog/field_test.go`:
- `TestValidateTable`: aceita `users`, `public.users`, `user_roles`; recusa `""`, `a.b.c`, `1users`, `users;`, `.users`.
- `TestValidateColumn`: aceita `id`, `email`, `created_at`; recusa `""`, `a.b`, `1a`, `a b`.

`internal/catalog/card_test.go`: estender a ficha do teste existente com `identity.tables` e afirmar que `c.Identity.Tables[0].Value == "users"` e `Columns` tem as colunas na ordem escrita.

`internal/validate/card_test.go`:
- `TestCardTableWithoutColumns`: `tables: [{ value: users, evidence: ["main.go:1"] }]` → erro citando `identity.tables[0] users` e pedindo as colunas.
- `TestCardTableInvalidColumn`: `columns: [id, "a b"]` → erro.
- `TestCardTableInvalidName`: `value: "a.b.c"` → erro.
- `TestCardTableNoEvidence`: `evidence: []` → o erro `no evidence` que o `evidenceChecker` já dá.
- `TestCardTableOK`: `value: users`, `columns: [id, email]`, `evidence: ["main.go:1"]` → sem erro.

- [ ] **Step 2: Rodar os testes para ver falhar**

Run: `go test ./internal/catalog/ ./internal/validate/`
Expected: FAIL — `undefined: catalog.ValidateTable` e `Identity.Tables` desconhecido.

- [ ] **Step 3: Implementar `ValidateTable` e `ValidateColumn` em `internal/catalog/field.go`**

- [ ] **Step 4: Adicionar `TableFact` e `Identity.Tables` em `internal/catalog/catalog.go`**

- [ ] **Step 5: Validar as tabelas em `internal/validate/card.go`**

Laço novo depois do de `Identity.Topics`, com `where` no formato `identity.tables[%d] %s` que os outros laços já usam, e `ev.check(where, t.Evidence, false)` no fim.

- [ ] **Step 6: Rodar os testes para ver passar**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 7: Documentar `identity.tables` em `skills/map/card-reference.md`**

Linha na tabela de `identity` com a forma do item e a evidência esperada (a linha da migration ou do schema que cria a tabela), a obrigatoriedade de `columns`, e a frase que diz que tabela não é relação: não entra em `provides`, não liga nada, serve para descobrir o dono do dado.

- [ ] **Step 8: Commit**

```bash
git add internal/catalog internal/validate skills/map/card-reference.md
git commit -m "feat: owned tables in identity"
```

---

### Task 3: `trama data <termo>`

**Files:**
- Create: `internal/cli/data.go`, `internal/cli/data_test.go`
- Modify: `internal/cli/root.go:40` (registrar o comando)
- Modify: `README.md`, `README.pt-BR.md` (tabela de comandos)
- Test: `internal/cli/data_test.go`

**Interfaces:**
- Consumes: `catalog.Provide.Request`, `catalog.Provide.Response` (Task 1), `catalog.Identity.Tables` (Task 2).
- Produces: `func newDataCmd() *cobra.Command`, `Use: "data <term>"`, `Args: cobra.ExactArgs(1)`.

Formato de saída, no estilo de `internal/cli/uses.go` (rótulo de 8 colunas, campos separados por dois espaços):

```
email
  response  api  GET /orders/{id}  customer.email: string  internal/api/orders.go:42
  request   api  POST /orders  email: string  internal/api/orders.go:60
  table     api  users.email  migrations/001_users.sql:3-8
not mapped: web
```

- [ ] **Step 1: Escrever os testes que falham**

`internal/cli/data_test.go`, com `catalogWorkspace` e `run` de `validate_test.go`, citando só os arquivos que o helper escreve (`main.go` tem 2 linhas, `app.ts` tem 1):
- `TestDataFindsResponseRequestAndTable`: ficha `api` com `response: { "customer.email": string }` em `GET /hello`, um `provides` `POST /orders` com `request: { email: string }` e `identity.tables` com a coluna `email`; `run(t, "data", "email", "--dir", ws)` imprime as três linhas, nessa ordem (response, request, table).
- `TestDataOrdersEndpointsBeforeTables`: afirmar que o índice da linha `response` é menor que o da linha `table` na saída.
- `TestDataIsCaseInsensitive`: `data EMAIL` acha a coluna `email`.
- `TestDataMatchesSubstringNotRegex`: `data ".*"` devolve o erro de nada encontrado, mesmo com campos na ficha.
- `TestDataListsUnmapped`: workspace onde `web` não tem ficha → saída termina com `not mapped: web`.
- `TestDataUnknownTerm`: `data cpf` → erro `no catalog card has a field or column matching "cpf"`.

- [ ] **Step 2: Rodar os testes para ver falhar**

Run: `go test ./internal/cli/ -run TestData -v`
Expected: FAIL — `unknown command "data"`.

- [ ] **Step 3: Implementar `newDataCmd` em `internal/cli/data.go`**

Carrega o workspace com `loadWorkspace(cmd)` e as fichas com `catalog.Load`, como `uses.go` faz, acumulando `unmapped` do mesmo jeito. Três passadas sobre `ws.Repos` na ordem response, request, tabela, para a saída ficar agrupada por camada e não por repositório. Comparação: `strings.Contains(strings.ToLower(x), strings.ToLower(term))` sobre o caminho do campo e sobre `tabela.coluna`. Chaves de mapa iteradas ordenadas. Nada encontrado: erro, como `uses` faz.

- [ ] **Step 4: Registrar o comando em `internal/cli/root.go:40`**

- [ ] **Step 5: Rodar os testes para ver passar**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 6: Documentar o comando nos dois READMEs**

Linha na tabela de comandos, depois de `trama uses`: o que busca, em que ordem responde e por quê — endpoint que já existe primeiro, tabela do dono depois.

- [ ] **Step 7: Commit**

```bash
git add internal/cli README.md README.pt-BR.md
git commit -m "feat: trama data searches fields and columns"
```

---

### Task 4: Instruções de mapeamento

**Files:**
- Modify: `skills/map/SKILL.md:32-36` (seções `provides`, `identity`, `consumes`)
- Test: `go test ./...` e uma leitura de consistência contra `card-reference.md`

**Interfaces:**
- Consumes: os campos documentados nas Tasks 1 e 2 em `skills/map/card-reference.md`.

- [ ] **Step 1: Escrever as instruções de `identity.tables` em `skills/map/SKILL.md`**

Onde ler o dono de uma tabela, na ordem em que a evidência é forte: migrations (`CREATE TABLE`), schema declarativo (`schema.prisma`, `schema.rb`), classes de entidade do ORM. Citar a linha que cria a tabela, não a que a consulta. Listar as colunas que existem, não as que o código usa. Repositório sem banco não ganha a chave.

- [ ] **Step 2: Escrever as instruções de `request` e `response` em `skills/map/SKILL.md`**

Onde ler o body de cada rota: o tipo de entrada e de saída do handler, o DTO, o schema de validação, ou o OpenAPI do repositório quando existe — citando a linha lida, como no resto do catálogo. Campos que o endpoint devolve, não os que o banco tem. Deixar o mapa de fora quando não deu para ler, em vez de adivinhar; `contract` continua sendo o ponteiro para o spec completo.

- [ ] **Step 3: Checar que nenhum campo novo ficou sem instrução**

Run: `grep -n "tables" skills/map/SKILL.md; grep -n "request" skills/map/SKILL.md; grep -n "response" skills/map/SKILL.md`
Expected: cada um dos três aparece, e a linha encontrada diz onde ler aquele campo no repositório. Campo documentado em `card-reference.md` nas Tasks 1 e 2 sem frase correspondente aqui é uma falha deste passo.

- [ ] **Step 4: Rodar a suíte inteira**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skills/map/SKILL.md
git commit -m "docs: how to map tables and bodies"
```
