# trama — Varredura de schema — Plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** O mapeador deixa de poder não achar o schema de um repositório: uma varredura em Go lista os arquivos que declaram tabela, `trama schemas <repo>` os entrega ao agente, e `trama validate` avisa quando o clone tem schema e a ficha não tem `tables`.

**Architecture:** A regra de onde ler os arquivos de um repositório sai do `validate` e vira `workspace.Files`. Um pacote novo, `internal/schemafind`, classifica os arquivos (schema, migration, sql) a partir de duas funções, listar e ler. O comando novo e o aviso do `validate` usam o mesmo pacote. A instrução do `/trama:map` passa a mandar rodar o comando.

**Tech Stack:** Go 1.25, cobra, `go.yaml.in/yaml/v3`, git.

**Spec:** `docs/superpowers/specs/2026-11-01-trama-varredura-de-schema-design.md`. Base: `main` em `896fa66`. Trabalho na branch `schema-scan`.

## Global Constraints

- Nenhuma dependência Go nova.
- A varredura nunca extrai tabela nem coluna: só diz quais arquivos abrir.
- O aviso é `warning`, nunca `error`: o `trama graph` não pode passar a recusar um catálogo por causa dele.
- `tables: []` (lista vazia explícita) cala o aviso; chave ausente não.
- No modo git, só arquivos regulares, a mesma regra do `gitx.ReadFile`.
- A refatoração do `validate` não muda nenhuma mensagem nem resultado: os testes de evidência que já existem ficam intactos e passando.
- Commits terminam com `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Sem push, tag ou release sem OK explícito do usuário.

## Review Focus

1. **Repositório gigante** (milhares de arquivos, quinhentas migrations): a varredura lista tudo sem truncar e sem ler conteúdo de arquivo que não seja `.sql`. Teste na Task 2.
2. **Symlink e submódulo no clone**: não entram na lista, nem no modo git nem no disco — uma lista que inclui o que o `ReadFile` recusa daria aviso que ninguém consegue resolver. Testes na Task 1.
3. **`tables: []` versus chave ausente** depois de um `trama stamp`: o `[]` tem que sobreviver à reescrita, senão a saída declarada some na primeira vez que alguém carimba a ficha. Teste na Task 4.
4. **Repositório não clonado**: o `validate` não avisa nada sobre tabela (já avisa que não está clonado), e `trama schemas` erra dizendo para rodar `trama sync`. Testes nas Tasks 3 e 4.
5. **Arquivo `.sql` de query, sem `CREATE TABLE`** (o `query.sql` do sqlc): não entra. Teste na Task 2.

---

### Task 1: `gitx.ListFiles` e `workspace.Files`

**Files:**
- Modify: `internal/gitx/git.go` (método novo), `internal/workspace/workspace.go` (tipo e método novos)
- Modify: `internal/validate/card.go` (`newEvidenceChecker` passa a usar `ws.Files`; `readRegular` sai daqui)
- Test: `internal/gitx/git_test.go` (ou o arquivo de teste do `gitx` que já cobre `ReadFile`), `internal/workspace/files_test.go`

**Interfaces:**
- Produces:
  - `func (r gitx.Repo) ListFiles(rev string) ([]string, error)`: `git ls-tree -r -z <rev>`, só entradas `blob` com modo `100644` ou `100755`, caminhos com `/`.
  - `workspace.Files struct { Place string; Cloned bool; List func() ([]string, error); Read func(path string) ([]byte, error) }`.
  - `func (ws *Workspace) Files(r Repo) Files`: `Cloned` falso quando `repos/<name>` não existe (e então `List`/`Read` são `nil`); `origin/<branch>` quando o clone é git e tem a ref, com `Place` igual a `repos/<name> at origin/<branch>`; o disco senão, com `Place` igual a `repos/<name>`. No disco, `List` percorre a árvore pulando `.git` e `node_modules`, só arquivos regulares, caminhos relativos com `/`.

- [ ] **Step 1: Criar a branch e commitar o spec e o plano**
- [ ] **Step 2: Testes que falham** — `TestListFilesOnlyRegular` (um arquivo comum, um executável, um symlink; só os dois primeiros saem); `TestFilesGitMode` (clone de `testutil.NewRemote` com um arquivo só em `origin/main` e outro só no disco: `List` devolve o de `origin/main`, não o do disco); `TestFilesDiskMode` (diretório comum: lista o que está no disco e pula `.git` e `node_modules`); `TestFilesNotCloned` (`Cloned == false`).
- [ ] **Step 3: Rodar e ver falhar**: `mise exec -- go test ./internal/gitx/ ./internal/workspace/` → FAIL, métodos indefinidos.
- [ ] **Step 4: Implementar `ListFiles` e `Files`**; mover `readRegular` para o `workspace`.
- [ ] **Step 5: `newEvidenceChecker` usa `ws.Files(repo)`**, guardando `List` ao lado de `Read`.
- [ ] **Step 6: Rodar tudo**: `mise exec -- go test ./...` → PASS, com os testes do `validate` intactos.
- [ ] **Step 7: Commit** — `refactor: one place decides where a repository's files are read from`

---

### Task 2: `internal/schemafind`

**Files:**
- Create: `internal/schemafind/schemafind.go`, `internal/schemafind/schemafind_test.go`

**Interfaces:**
- Produces:
  - `schemafind.File struct { Kind, Path string }`, `Kind` em `schema | migration | sql`.
  - `func schemafind.Scan(list func() ([]string, error), read func(string) ([]byte, error)) ([]File, error)`: as regras e a ordem do spec. `read` só é chamado para `.sql` que não caiu em `schema` nem em `migration`.

- [ ] **Step 1: Testes que falham** — `TestScanKinds` (um de cada tipo, na ordem schema, migration, sql); `TestScanPrecedence` (`prisma/migrations/x/migration.sql` com `CREATE TABLE` é `migration`, não `sql`); `TestScanSkipsQueriesWithoutCreateTable` (`db/query.sql` sem `CREATE TABLE` não entra); `TestScanCreateTableIsCaseInsensitive`; `TestScanSkipsExcludedDirs` (um schema em cada diretório excluído, nenhum entra); `TestScanSkipsNoiseInMigrationDirs` (`.keep`, `README.md`, `__init__.py`); `TestScanReadsOnlySQL` (o `read` falha o teste se chamado com algo que não seja `.sql` fora de migration); `TestScanManyMigrations` (quinhentas migrations saem todas, ordenadas); `TestScanListError` (erro do `list` volta como erro).
- [ ] **Step 2: Rodar e ver falhar** → o pacote não existe.
- [ ] **Step 3: Implementar `File` e `Scan`.**
- [ ] **Step 4: Rodar e ver passar**: `mise exec -- go test ./internal/schemafind/`.
- [ ] **Step 5: Commit** — `feat: find the files that declare tables in a repository`

---

### Task 3: `trama schemas <repo>`

**Files:**
- Create: `internal/cli/schemas.go`, `internal/cli/schemas_test.go`
- Modify: `internal/cli/root.go` (registrar), `README.md`, `README.pt-BR.md` (linha na tabela de comandos)

**Interfaces:**
- Consumes: `ws.Files` (Task 1), `schemafind.Scan` (Task 2).
- Produces: `newSchemasCmd()`, `Use: "schemas <repo>"`, `Args: cobra.ExactArgs(1)`. Saída: o nome do repo, depois uma linha por arquivo com o tipo alinhado em 9 colunas (`  %-9s  %s`); sem arquivos, `  no schema files found`. Repo fora do `workspace.yaml`: erro. Não clonado: erro com `run "trama sync <repo>"`.

- [ ] **Step 1: Testes que falham** — `TestSchemasListsByKind`, `TestSchemasNoFiles`, `TestSchemasUnknownRepo`, `TestSchemasNotCloned`.
- [ ] **Step 2: Rodar e ver falhar** → `unknown command "schemas"`.
- [ ] **Step 3: Implementar e registrar.**
- [ ] **Step 4: Rodar e ver passar.**
- [ ] **Step 5: Documentar nos dois READMEs.**
- [ ] **Step 6: Commit** — `feat: trama schemas lists the files that declare tables`

---

### Task 4: O aviso do `validate`

**Files:**
- Modify: `internal/validate/card.go`
- Test: `internal/validate/card_test.go`, `internal/catalog/stamp_test.go`

**Interfaces:**
- Consumes: `schemafind.Scan` (Task 2), o `List` guardado no `evidenceChecker` (Task 1).

- [ ] **Step 1: Testes que falham** — `TestCardWarnsSchemaWithoutTables` (clone com `db/schema.rb`, ficha sem `tables`: um `warning` em `identity.tables` que cita o arquivo e `tables: []`); `TestCardEmptyTablesSilencesWarning` (mesma coisa com `tables: []`: nenhum finding); `TestCardMappedTablesNoWarning` (com uma tabela mapeada: nenhum aviso); `TestCardNoSchemaNoWarning` (clone sem schema, ficha sem `tables`: nenhum aviso); `TestCardNotClonedNoTablesWarning`; `TestStampKeepsEmptyTables` (em `internal/catalog`: depois do `Stamp`, o arquivo ainda tem `tables: []`).
- [ ] **Step 2: Rodar e ver falhar.**
- [ ] **Step 3: Implementar o aviso** em `checkCard`, depois dos laços de `identity`, só quando `Tables == nil` e o repositório está clonado. A mensagem cita a contagem e até três caminhos.
- [ ] **Step 4: Rodar tudo** → PASS.
- [ ] **Step 5: Commit** — `feat: validate warns when a repository has a schema but the card no tables`

---

### Task 5: A instrução do mapeador

**Files:**
- Modify: `skills/map/SKILL.md`, `skills/map/card-reference.md`

- [ ] **Step 1: Na seção "Read the repository"**, ao lado do router e do OpenAPI: rodar `trama schemas <name>` e abrir o que ele listar, o `schema` primeiro.
- [ ] **Step 2: Em `identity.tables`**: preferir o arquivo `schema` quando existe; sem ele, reconstruir a tabela a partir das migrations em ordem; procurar entidades de ORM à mão, que a varredura não acha; quando a lista vier e nenhum arquivo for tabela deste repositório, escrever `tables: []`. Chave ausente só quando `trama schemas` não achar nada.
- [ ] **Step 3: Para `--all`**: um repositório por vez — ler, escrever a ficha, só então abrir o próximo.
- [ ] **Step 4: No `card-reference.md`**: o que `tables: []` quer dizer e o aviso que ele cala.
- [ ] **Step 5: Rodar tudo** → PASS (o `reference_test` valida os exemplos e proíbe `shape`).
- [ ] **Step 6: Commit** — `docs: the map skill runs trama schemas before writing tables`
