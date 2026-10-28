# trama — Fase 2 (Mapa) — Plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A trama passa a mapear os repos. Entram as fichas completas (`catalog/<repo>.yaml`) com evidência `arquivo:linha`, o `trama validate`, o `trama stamp`, o `trama graph` e as skills `/trama:add-repo` e `/trama:map`. Junto vão os ajustes pequenos que ficaram da fase 1, e a fase fecha com a release `v0.0.4` e o `trama-demo` mapeado.

**Architecture:** O CLI ganha a parte exata: ler e validar fichas, carimbar o commit mapeado e gerar o grafo. As skills ficam com o julgamento: ler o código, escrever a ficha e provar cada vínculo. Validar e gerar o grafo leem só arquivos locais (fichas, `workspace.yaml` e os clones em `repos/`), sem rede.

**Tech Stack:** Go (cobra, `go.yaml.in/yaml/v3`), git, mise, Mermaid (no `docs/graph.md`), skills do Claude Code.

**Spec:** `docs/superpowers/specs/2026-10-27-trama-design.md`. Esta é a fase 2 da §11 e cobre §6.2, §6.4 (parte gerada), `validate`/`graph` da §7, `/trama:add-repo` e `/trama:map` da §8 e os critérios 1 e 2 da §10. Base: `v0.0.3` (commit `0f4d02e`).

**Fora desta fase:** `setup`/`dev`/`up`/`test` (fase 3); `impact`, `change` e `changes` (fase 4); hook de sessão (fase 5). O `status` passa a listar as pendências `from: unknown`; as mudanças abertas em `changes/` ficam para a fase 4.

## Decisões de construção (o spec deixou para quem constrói)

1. **`trama stamp <repo>`**: comando novo que grava `mapped_at_commit` (SHA completo de `origin/<branch>`) e `mapped_at` (data de hoje). Recusa se o clone não estiver limpo, na branch principal e igual a `origin/<branch>`. Assim a IA nunca digita SHA, e a regra "a ficha descreve a branch principal" (§6.2, §8) vira código.
2. **Vínculo declarado (`via: declared`)**: lista `links` no topo do `workspace.yaml`, com itens `{consumer, provider, id}`.
3. **Leitura estrita no `validate`**: um campo desconhecido na ficha (ex.: `evidnce`) é erro, com o número da linha. O `status` continua lendo de forma tolerante.
4. **Evidência conferida no disco**: `validate` checa se o arquivo existe em `repos/<repo>` e se a linha cabe no arquivo. Se o repo não estiver clonado, isso vira aviso. A evidência `workspace.yaml` só vale para `identity.hosts`, e o host precisa estar declarado lá.
5. **`graph` recusa catálogo inválido**: roda as mesmas checagens do `validate` e não escreve nada se houver erro. Fora dos marcadores, o `AGENTS.md` nunca é tocado; sem os marcadores, é erro.
6. **Avisos não reprovam**: ficha faltando e `from: unknown` são avisos; `validate` só sai com erro quando há erro.
7. **Texto das fichas em inglês**, como o resto do conteúdo dos repos.
8. **Ajuste #11 da fase 1 não entra** (o `Save` perde linhas em branco): é limitação do `yaml.v3`, e consertar exige reescrever o encoder. Vira nota no README.

## Global Constraints

- Tudo das Global Constraints da fase 1 continua valendo (módulo, dependências, git sem prompts, nunca descartar trabalho local, inglês, MIT, macOS e Linux).
- Fichas: `version: 1`; arquivo `catalog/<name>.yaml`, com `name` igual ao nome do arquivo e presente no `workspace.yaml`.
- `kind` ∈ `http`, `event`, `package`. O `id` de `http` é `MÉTODO /caminho`, com MÉTODO ∈ `GET POST PUT PATCH DELETE HEAD OPTIONS` e parâmetros como `{nome}` (nunca `:nome`), sem query string.
- `via` ∈ `env`, `package`, `topic`, `declared`; `from` é um repo do workspace ou `unknown`.
- Evidência: `caminho:linha` ou `caminho:início-fim`, com caminho relativo à raiz do repo, sem `..` e sem `/` inicial; ou o literal `workspace.yaml`.
- `mapped_at_commit`: 7 a 40 caracteres hexadecimais minúsculos. `mapped_at`: `AAAA-MM-DD`.
- Marcadores do `AGENTS.md`: `<!-- trama:begin -->` e `<!-- trama:end -->`.
- ⚠️ Push, tag, release ou repo novo no GitHub: pare e confirme com o usuário imediatamente antes.

## Review Focus

1. **Evidência que envelheceu** (arquivo encurtado ou apagado depois do mapeamento) → `validate` acusa arquivo inexistente ou linha fora do arquivo. (Task 4 `TestValidateEvidenceLineOutOfRange`, `TestValidateEvidenceMissingFile`.)
2. **O mesmo `id` servido por dois repos** (`GET /health`) → nada é ligado sem `via`; `from: unknown` com candidatos é aceito como pendência e aparece no `status` e no grafo. (Task 5 `TestValidateUnknownWithCandidatesIsWarning`, `TestValidateFromWithoutViaIsError`; Task 7 `TestGraphShowsPending`; Task 8 `TestStatusListsPending`.)
3. **Ficha editada à mão com erro de digitação** (`evidnce:`) → erro com arquivo e linha, não um campo ignorado. (Task 3 `TestLoadStrictRejectsUnknownField`.)
4. **Ficha órfã ou com nome trocado** (`catalog/old-api.yaml` sem repo, `name:` diferente do arquivo) → erro. (Task 4 `TestValidateOrphanCard`, `TestValidateNameMismatch`.)
5. **`AGENTS.md` editado por pessoas** → `graph` só reescreve entre os marcadores; sem marcadores, falha sem escrever. (Task 7 `TestReplaceMarkedKeepsOutside`, `TestGraphRefusesWithoutMarkers`.)

---

### Task 1: Ajustes da fase 1 no workspace, no `add` e no `init`

**Files:**
- Modify: `internal/workspace/workspace.go` (`Find`), `internal/cli/add.go`, `internal/scaffold/scaffold.go` (`yamlScalar`, criação da pasta)
- Test: `internal/workspace/find_test.go`, `internal/cli/add_local_test.go`, `internal/scaffold/init_dir_test.go`

**Interfaces:**
- Produces: `func LocalURL(url, cwd string) string` (package `workspace`): devolve o caminho absoluto quando `url` é um caminho local (não tem `://` e não é do tipo `usuario@host:caminho` ou `host:caminho`); caso contrário devolve `url` sem mudança.
- `Find` passa a pular `workspace.yaml` sem a chave `version` no topo e continua subindo. Um arquivo com `version` diferente de 1 ainda para a busca, e o `Load` dá o erro.
- `scaffold.Init` cria `Dir` (com `MkdirAll`) quando não existe. `yamlScalar` coloca aspas em todo nome que o YAML leria como outra coisa que não a mesma string (`null`, `true`, `123`, `~`).

- [ ] **Step 1: Testes que falham:**

```go
// find_test.go
func TestFindSkipsForeignWorkspaceYAML(t *testing.T) // root/workspace.yaml (trama) + root/repos/x/workspace.yaml ("load_from: []") → Find(root/repos/x/src) == root
func TestFindStopsAtOtherVersion(t *testing.T)       // root/workspace.yaml with "version: 2" → Find returns root (Load reports the version)

// add_local_test.go
func TestAddStoresLocalPathAsAbsolute(t *testing.T) // chdir to parent of remote; add "./remote.git" (relative) → workspace URL == absolute remote.URL
func TestLocalURL(t *testing.T) {
	cases := map[string]string{
		"https://github.com/x/y.git": "https://github.com/x/y.git",
		"git@github.com:x/y.git":     "git@github.com:x/y.git",
		"github.com:x/y.git":         "github.com:x/y.git",
		"../y.git":                   "/work/y.git",
		"/abs/y.git":                 "/abs/y.git",
	}
	// workspace.LocalURL(k, "/work/sub") == v
}

// init_dir_test.go
func TestInitCreatesMissingDir(t *testing.T)     // Dir = t.TempDir()+"/new/ws" → created; workspace.yaml loads
func TestInitQuotesSpecialNames(t *testing.T)    // for name in {"null", "true", "123", "~"}: Init with Name → workspace.Load succeeds and ws.Name == name
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/workspace/ ./internal/cli/ ./internal/scaffold/` → Expected: FAIL nos testes novos.
- [ ] **Step 3:** Implementar. No `add`, aplicar `workspace.LocalURL(url, os.Getwd())` antes de `NameFromURL` e `CheckNew`. Em `yamlScalar`, decodificar `s` como YAML; se o resultado não for a string `s`, usar aspas.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `git add` só dos arquivos da task (conferir `git status` antes) e `git commit -m "fix: workspace lookup, local paths and init edge cases"`

---

### Task 2: Shim, workflow de release e README

**Files:**
- Modify: `bin/trama`, `.github/workflows/release.yml`, `README.md`, `README.pt-BR.md`
- Test: `internal/plugin/plugin_test.go`

**Interfaces:**
- `bin/trama`: se a versão lida do `plugin.json` vier vazia, escreve `trama: could not read the version from .claude-plugin/plugin.json` no stderr e sai com 1, antes de chamar o `mise`.
- `release.yml`: um passo `go test ./...` entre o setup-go e o GoReleaser.
- README, em inglês e em português:
  - seção "Releasing" com a ordem: atualizar `plugin.json`, fazer push da tag, esperar a release, só então fazer push da `main` e atualizar o pin dos workspaces;
  - nota de que `trama add` regrava o `workspace.yaml` sem as linhas em branco.

- [ ] **Step 1: Teste que falha:**

```go
func TestShimRejectsMissingVersion(t *testing.T) // copy bin/trama to tmp/bin/trama and write tmp/.claude-plugin/plugin.json = {"name":"trama"}; fake mise on PATH → exit 1, stderr contains "could not read the version", fake mise never called (marker file absent)
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/plugin/` → Expected: FAIL.
- [ ] **Step 3:** Implementar o shim, o passo do workflow e o texto dos dois READMEs.
- [ ] **Step 4:** Run `mise exec -- go test ./internal/plugin/ && claude plugin validate .` → Expected: PASS e validação sem erros.
- [ ] **Step 5: Commit** — `git commit -m "build: shim version check, tests before release, release order docs"`

---

### Task 3: `links` no workspace e o esquema completo da ficha

**Files:**
- Modify: `internal/workspace/workspace.go`, `internal/catalog/catalog.go`
- Test: `internal/workspace/links_test.go`, `internal/catalog/card_test.go`, `internal/catalog/evidence_test.go`

**Interfaces:**
- Produces (`package workspace`):

```go
type Link struct {
	Consumer string `yaml:"consumer"`
	Provider string `yaml:"provider"`
	ID       string `yaml:"id"`
}
// Workspace gains: Links []Link `yaml:"links,omitempty"`
// Validate: consumer and provider are repos of the workspace, differ, and id is not empty.
```

- Produces (`package catalog`). `Entry` sai; `status` passa a usar `Card`:

```go
type Card struct {
	Version        int                `yaml:"version"`
	Name           string             `yaml:"name"`
	Description    string             `yaml:"description"`
	Stack          []string           `yaml:"stack"`
	Tools          map[string]string  `yaml:"tools,omitempty"`
	MappedAtCommit string             `yaml:"mapped_at_commit"`
	MappedAt       string             `yaml:"mapped_at"`
	Run            map[string]Command `yaml:"run,omitempty"` // keys: setup, dev, test
	Identity       Identity           `yaml:"identity,omitempty"`
	Provides       []Provide          `yaml:"provides,omitempty"`
	Consumes       []Consume          `yaml:"consumes,omitempty"`
	Config         []ConfigEntry      `yaml:"config,omitempty"`
}
type Command struct {
	Cmd  string `yaml:"cmd"`
	Port int    `yaml:"port,omitempty"`
}
type Fact struct {
	Value    string   `yaml:"value"`
	Evidence []string `yaml:"evidence"`
}
type PortFact struct {
	Value    int      `yaml:"value"`
	Evidence []string `yaml:"evidence"`
}
type Identity struct {
	Ports   []PortFact `yaml:"ports,omitempty"`
	Hosts   []Fact     `yaml:"hosts,omitempty"`
	Package *Fact      `yaml:"package,omitempty"`
	Topics  []Fact     `yaml:"topics,omitempty"`
}
type Provide struct {
	Kind     string         `yaml:"kind"`
	ID       string         `yaml:"id"`
	Contract string         `yaml:"contract,omitempty"`
	Shape    map[string]any `yaml:"shape,omitempty"`
	Evidence []string       `yaml:"evidence"`
}
type Consume struct {
	Kind       string   `yaml:"kind"`
	ID         string   `yaml:"id"`
	From       string   `yaml:"from"`
	Via        string   `yaml:"via,omitempty"`
	Candidates []string `yaml:"candidates,omitempty"`
	FieldsUsed []string `yaml:"fields_used,omitempty"`
	Evidence   []string `yaml:"evidence"`
}
type ConfigEntry struct {
	Env      string   `yaml:"env"`
	Value    string   `yaml:"value,omitempty"`
	PointsTo string   `yaml:"points_to,omitempty"`
	Evidence []string `yaml:"evidence"`
}

const Unknown = "unknown"

func Path(root, name string) string
func Load(root, name string) (*Card, error)       // tolerant; (nil, nil) when missing
func LoadStrict(root, name string) (*Card, error) // KnownFields(true); (nil, nil) when missing; errors carry the YAML line
func Names(root string) ([]string, error)         // names of catalog/*.yaml, sorted

type Evidence struct {
	Path       string // relative to the repo root; "" when Declared
	Start, End int    // End == Start for a single line
	Declared   bool   // the literal "workspace.yaml"
}
func ParseEvidence(s string) (Evidence, error)
```

- [ ] **Step 1: Testes que falham:**

```go
// links_test.go
func TestLoadLinks(t *testing.T)            // links [{consumer: web, provider: api, id: "GET /hello"}] → ws.Links[0] fields
func TestLoadRejectsBadLinks(t *testing.T)  // unknown consumer, unknown provider, consumer == provider, empty id → each an error

// card_test.go — the spec §6.2 demo cards (web and api) as fixtures
func TestLoadFullCard(t *testing.T)               // web: Consumes[0] == {http, "GET /hello", "trama-demo-api", "env", nil, ["message"], ["src/api/hello.ts:5", "src/App.tsx:12"]}; Config[0].Value == "http://localhost:8080"; Run["dev"] == {"npm run dev", 5173}
func TestLoadIdentity(t *testing.T)               // api: Identity.Ports[0] == {8080, ["cmd/api/main.go:10"]}; Identity.Package == nil
func TestLoadStrictRejectsUnknownField(t *testing.T) // "evidnce:" in consumes → error contains "evidnce" and "line "
func TestLoadStrictMissingReturnsNil(t *testing.T)
func TestNames(t *testing.T)                      // catalog/{b.yaml,a.yaml,notes.md} → ["a", "b"]

// evidence_test.go
func TestParseEvidence(t *testing.T) {
	ok := map[string]catalog.Evidence{
		"src/api/hello.ts:5":    {Path: "src/api/hello.ts", Start: 5, End: 5},
		"internal/api/api.go:40-44": {Path: "internal/api/api.go", Start: 40, End: 44},
		"workspace.yaml":        {Declared: true},
	}
	bad := []string{"src/a.go", "src/a.go:0", "src/a.go:9-3", "/etc/passwd:1", "../x.go:1", "a/../../x.go:1", "src/a.go:x", ""}
	// ParseEvidence(k) == v for ok; error for each bad
}
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/workspace/ ./internal/catalog/` → Expected: FAIL.
- [ ] **Step 3:** Implementar. `LoadStrict` usa `yaml.NewDecoder` com `KnownFields(true)` e coloca o caminho do arquivo nos erros. Trocar `catalog.Entry` por `catalog.Card` em `internal/status`.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS (os testes do `status` continuam verdes).
- [ ] **Step 5: Commit** — `git commit -m "feat: full catalog card schema and declared links"`

---

### Task 4: `validate`, regras de cada ficha

**Files:**
- Create: `internal/validate/validate.go`, `internal/validate/card.go`
- Test: `internal/validate/card_test.go`, `internal/validate/helpers_test.go`

**Interfaces:**
- Consumes: `catalog.*`, `workspace.Workspace`.
- Produces (`package validate`):

```go
type Severity string

const (
	Error   Severity = "error"
	Warning Severity = "warning"
)

type Finding struct {
	Severity Severity
	File     string // slash path relative to the workspace root, e.g. "catalog/trama-demo-web.yaml"
	Where    string // e.g. `consumes[0] GET /hello`; "" for the whole file
	Message  string
}

func (f Finding) String() string // "<severity>: <file>: <where>: <message>" (Where omitted when empty)

// Workspace checks every card against workspace.yaml, the other cards and the
// clones in repos/. Findings are sorted by file, then order of appearance.
func Workspace(ws *workspace.Workspace) []Finding

func Errors(fs []Finding) int
```

Regras desta task (cada uma é um `Error`, salvo indicação):
- Repo sem ficha → `Warning` `not mapped yet: run "/trama:map <name>"`.
- Ficha sem repo no workspace → `card for a repository that is not in workspace.yaml`.
- Falha do `LoadStrict` → um erro com a mensagem do decoder.
- `version != 1`; `name` diferente do arquivo; `description` vazia; `stack` vazio; `mapped_at_commit` fora do padrão; `mapped_at` fora de `AAAA-MM-DD`; chave de `run` fora de `setup`, `dev` e `test`, ou `cmd` vazio.
- Em `provides`, `consumes`, `config` e `identity`: lista de evidência vazia → `no evidence`; evidência que não faz parse → mensagem do `ParseEvidence`.
- `kind` fora da lista; `id` de `http` fora da forma `MÉTODO /caminho`; `id` vazio para `event` e `package`.
- **Evidência no disco** (só com o repo clonado; sem clone, um `Warning` por ficha `repos/<name> is not cloned: evidence not checked`): o arquivo existe dentro de `repos/<name>`, e `End` ≤ número de linhas do arquivo.
- `workspace.yaml` como evidência: só vale em `identity.hosts`, e o host precisa estar em `repos[name].hosts` do `workspace.yaml`.

- [ ] **Step 1: Testes que falham.** Em `helpers_test.go`, um helper `fixture(t)` monta um workspace num diretório temporário com `workspace.yaml`, `repos/<name>/` (pastas com arquivos, sem git) e fichas. Ele devolve `func(name, card string)` para escrever fichas e `func(repo, path, content string)` para escrever arquivos no repo. Em `card_test.go`, cada teste afirma a mensagem e o `Where`:

```go
func TestValidateCleanDemo(t *testing.T)               // demo cards + matching files → Errors == 0, no findings
func TestValidateMissingCardIsWarning(t *testing.T)    // repo without card → one Warning "not mapped yet"
func TestValidateOrphanCard(t *testing.T)              // catalog/old.yaml, "old" not in workspace → Error
func TestValidateNameMismatch(t *testing.T)            // catalog/api.yaml with name: web → Error
func TestValidateRequiredFields(t *testing.T)          // table: empty description, empty stack, mapped_at_commit "main", mapped_at "28/10/2026", run key "start" → each one Error
func TestValidateNoEvidence(t *testing.T)              // provides[0].evidence: [] → Error, Where == "provides[0] GET /hello"
func TestValidateHTTPID(t *testing.T)                  // ids "GET /orders/{id}" ok; "get /x", "GET orders", "GET /orders/:id", "GET /x?y=1", "FETCH /x" → Error
func TestValidateEvidenceMissingFile(t *testing.T)     // evidence "src/gone.ts:3" → Error mentions "src/gone.ts"
func TestValidateEvidenceLineOutOfRange(t *testing.T)  // 3-line file, evidence "f.go:4" → Error mentions "line 4" and "3 lines"
func TestValidateNotClonedIsWarning(t *testing.T)      // no repos/<name> dir → Warning, no evidence errors
func TestValidateDeclaredHost(t *testing.T)            // identity.hosts [{value: api.acme.com, evidence: [workspace.yaml]}]: ok when workspace.yaml repos[api].hosts has it; Error otherwise; "workspace.yaml" under provides → Error
func TestFindingString(t *testing.T)                   // Finding{Error, "catalog/a.yaml", "provides[0] GET /x", "no evidence"}.String() == "error: catalog/a.yaml: provides[0] GET /x: no evidence"
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/validate/` → Expected: FAIL.
- [ ] **Step 3:** Implementar `Workspace` (carrega `catalog.Names` mais os repos do workspace e roda as regras de ficha) e as regras acima em `card.go`.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat: validate catalog cards"`

---

### Task 5: `validate`, regras entre fichas, e o comando `trama validate`

**Files:**
- Create: `internal/validate/links.go`, `internal/cli/validate.go`
- Test: `internal/validate/links_test.go`, `internal/cli/validate_test.go`

**Interfaces:**
- Consumes: Task 4 (`Finding`, `Workspace`, helpers de teste).
- `Workspace` passa a rodar também estas regras para cada `consumes[i]`:
  - `from` vazio → Error.
  - `from: unknown`: um `Warning` `no proven provider (candidates: a, b)` ou `(no candidates)`; `via` precisa estar vazio; cada candidato precisa ser repo do workspace.
  - `from: X` com X fora do workspace → Error. X igual ao próprio repo → Error. X sem ficha → Error `X is not mapped`.
  - A ficha de X não tem `provides` com o mesmo `kind` e `id` → Error `X does not provide <id>`.
  - `via` vazio → Error `from needs proof in via`.
  - `via` fora da lista → Error.
  - `via: env`: alguma entrada de `config` desta ficha tem `points_to: X` e um `value` que bate com a `identity` de X → senão Error `via env: no config entry points to X with a value matching its identity`. Regra do valor: interpretar como URL; bate se o host for igual (sem diferenciar maiúsculas) a um `identity.hosts[].value`, ou se a porta for igual a um `identity.ports[].value` com host `localhost`, `127.0.0.1` ou `0.0.0.0`. Caminho e esquema não importam (`http://localhost:8080/api` bate com a porta 8080).
  - `via: package`: `kind` é `package` e `id` é igual a `X.identity.package.value`.
  - `via: topic`: `kind` é `event` e `id` está em `X.identity.topics[].value`.
  - `via: declared`: o `workspace.yaml` tem `links` com `{consumer: este, provider: X, id}`.
- `config[i].points_to`, quando preenchido, é um repo do workspace.
- CLI `trama validate`: imprime cada `Finding.String()`. Com erros, devolve `catalog has <N> error(s)`. Sem erros, imprime `ok: <cards> card(s), <relations> relation(s), <warnings> warning(s)`. Aqui *relation* é um `consumes` com `from` diferente de `unknown` que passou em todas as regras; exponha a contagem como `func Relations(ws *workspace.Workspace) int`, para não duplicar a regra no CLI.

- [ ] **Step 1: Testes que falham:**

```go
// links_test.go
func TestValidateDemoRelation(t *testing.T)               // web consumes GET /hello from api via env, config VITE_API_URL=http://localhost:8080 points_to api, api identity port 8080 → no findings
func TestValidateEnvMatchesPathAndHost(t *testing.T)      // value "http://localhost:8080/api" ok; value "https://API.acme.com" with api hosts [api.acme.com] ok; value "http://localhost:9090" → Error
func TestValidateFromWithoutViaIsError(t *testing.T)      // from: api, via missing → Error "from needs proof in via"
func TestValidateUnknownWithCandidatesIsWarning(t *testing.T) // from: unknown, candidates [api, billing] → one Warning, Errors == 0; candidates [ghost] → Error
func TestValidateProviderMustProvide(t *testing.T)        // api provides nothing → Error "api does not provide GET /hello"
func TestValidateProviderNotMapped(t *testing.T)          // api has no card → Error "api is not mapped"
func TestValidateViaPackageAndTopic(t *testing.T)         // package: kind package, id "@acme/ui" == provider identity.package → ok; topic: kind event, id "orders.created" in provider topics → ok; mismatches → Error
func TestValidateViaDeclared(t *testing.T)                // workspace.yaml links entry → ok; without it → Error
func TestValidateSelfReference(t *testing.T)              // from: web inside web's card → Error

// validate_test.go (package cli)
func TestValidateCommandReportsErrors(t *testing.T) // workspace with one bad card → err "catalog has 1 error(s)"; output contains "error: catalog/"
func TestValidateCommandOK(t *testing.T)            // clean demo with every cited file present under repos/ → no error; output starts with "ok: 2 card(s), 1 relation(s), 0 warning(s)"
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/validate/ ./internal/cli/` → Expected: FAIL.
- [ ] **Step 3:** Implementar `links.go` e o comando, registrado no root.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat: trama validate"`

---

### Task 6: `trama stamp`

**Files:**
- Create: `internal/catalog/stamp.go`, `internal/cli/stamp.go`
- Test: `internal/catalog/stamp_test.go`, `internal/cli/stamp_test.go`

**Interfaces:**
- Consumes: `gitx.Repo`, `workspace.Workspace`.
- Produces:

```go
// Stamp records in catalog/<name>.yaml the commit the card describes
// (origin/<branch>) and the date. It edits the YAML in place, keeping every
// other field and comment, and adds the two keys when they are missing.
func Stamp(ws *workspace.Workspace, name string, today time.Time) (sha string, err error)
```

Recusa, nesta ordem e com estas mensagens:
- `repo <name> is not in workspace.yaml`
- `no card at catalog/<name>.yaml: write it first`
- `repos/<name> is not cloned: run "trama sync <name>"`
- `repos/<name> is on <x>, not <branch>`
- `repos/<name> has local changes`
- `repos/<name> is not at origin/<branch>: run "trama sync <name>"`

O CLI `trama stamp <repo>` imprime `stamped <name> at <sha7> (<data>)`.

- [ ] **Step 1: Testes que falham** (remotos do `testutil`, workspace com o repo clonado pelo `syncer`):

```go
func TestStampWritesCommitAndDate(t *testing.T)   // card with a comment and mapped_at_commit: "" → after Stamp: LoadStrict gives MappedAtCommit == full origin/main SHA (40 chars), MappedAt == "2026-10-28"; comment kept
func TestStampAddsMissingKeys(t *testing.T)       // card without mapped_* keys → keys added
func TestStampRefuses(t *testing.T)               // table: no card; not cloned; feature branch; dirty tree; behind origin (remote.Commit + Fetch) → each the message above, card unchanged
func TestStampCommand(t *testing.T)               // output starts with "stamped api at "
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/catalog/ ./internal/cli/` → Expected: FAIL.
- [ ] **Step 3:** Implementar. Editar o `yaml.Node` do documento, como o `workspace.AddRepo` faz, e salvar via arquivo temporário e rename.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat: trama stamp"`

---

### Task 7: `trama graph`

**Files:**
- Create: `internal/graph/graph.go`, `internal/cli/graph.go`
- Test: `internal/graph/graph_test.go`, `internal/cli/graph_test.go`

**Interfaces:**
- Consumes: `catalog.Card`, `validate.Workspace`, `validate.Errors`.
- Produces (`package graph`):

```go
type Relation struct{ Consumer, Provider, Kind, ID, Via string }
type Pending struct {
	Consumer, Kind, ID string
	Candidates          []string
}

func Build(ws *workspace.Workspace, cards map[string]*catalog.Card) ([]Relation, []Pending) // workspace order, then card order
func Doc(ws *workspace.Workspace, rels []Relation, pending []Pending) string            // docs/graph.md
func AgentsSection(ws *workspace.Workspace, cards map[string]*catalog.Card) string      // text between the markers
func ReplaceMarked(doc, section string) (string, error)                                 // error when a marker is missing or out of order
func Write(ws *workspace.Workspace) error                                               // loads cards, writes docs/graph.md and AGENTS.md
```

Formato de `docs/graph.md`:
- título `# Dependency graph` e a linha `<!-- Generated by trama graph from catalog/. Do not edit by hand. -->`;
- bloco ```` ```mermaid ```` com `graph LR`, um nó por repo do workspace (`n<i>["<name>"]`, na ordem do workspace) e uma aresta por relação (`n<c> -->|"<id>"| n<p>`). Cada pendência vira uma aresta tracejada até um nó `unknown(("?"))`;
- tabela `## Relations` (`Consumer | Provider | Kind | ID | Via`);
- tabela `## Pending` (`Consumer | Kind | ID | Candidates`), só quando houver pendência.

Seção do `AGENTS.md`: tabela `| Repository | Stack | Provides | Consumes |`, uma linha por repo do workspace. Repo sem ficha mostra `_not mapped yet_`. `Consumes` lista `<id> ← <provider>` ou `<id> ← ?`. A seção termina com `See [docs/graph.md](docs/graph.md).`

CLI `trama graph`: roda `validate.Workspace`. Com erros, devolve `fix <N> validation error(s) first: run "trama validate"` sem escrever nada. Sem erros, escreve e imprime `wrote docs/graph.md` e `updated AGENTS.md`.

- [ ] **Step 1: Testes que falham:**

```go
func TestBuildDemo(t *testing.T)                 // demo cards → rels == [{trama-demo-web, trama-demo-api, http, "GET /hello", env}], no pending
func TestDocDemo(t *testing.T)                   // Doc contains `n0["trama-demo-api"]`, `n1["trama-demo-web"]`, `n1 -->|"GET /hello"| n0`, "| trama-demo-web | trama-demo-api | http | GET /hello | env |"; no "## Pending"
func TestGraphShowsPending(t *testing.T)         // from: unknown with candidates → `-.->|"GET /health"| unknown` and a "## Pending" row listing the candidates
func TestReplaceMarkedKeepsOutside(t *testing.T) // text before, markers, text after → only the inside changes; byte-equal outside
func TestAgentsSection(t *testing.T)             // demo: row "| trama-demo-web | node, react, typescript | — | GET /hello ← trama-demo-api |"; unmapped repo row contains "_not mapped yet_"
func TestGraphRefusesWithoutMarkers(t *testing.T)   // AGENTS.md without markers → Write errors, docs/graph.md not created, AGENTS.md unchanged
func TestGraphCommandRefusesInvalidCatalog(t *testing.T) // bad card → err "fix 1 validation error(s) first"; no docs/graph.md
func TestGraphCommandWrites(t *testing.T)        // clean demo (initialized with trama init) → both files written; running twice gives identical bytes
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/graph/ ./internal/cli/` → Expected: FAIL.
- [ ] **Step 3:** Implementar e registrar o comando.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat: trama graph"`

---

### Task 8: `status` lista as pendências

**Files:**
- Modify: `internal/status/status.go`
- Test: `internal/status/pending_test.go`

**Interfaces:**
- `RepoStatus` ganha `Pending []string`: o texto de cada `consumes` com `from: unknown`, no formato `<id> (candidates: a, b)` ou `<id> (no candidates)`.
- `Problems()` acrescenta, por pendência, `pending: consumes <texto> from an unknown provider`. A ficha é lida com o `catalog.Load` tolerante.

- [ ] **Step 1: Teste que falha:**

```go
func TestStatusListsPending(t *testing.T) // card at origin/main SHA with consumes [{http, "GET /health", from: unknown, candidates: [a, b]}] → Problems() == [`pending: consumes GET /health (candidates: a, b) from an unknown provider`]
```

- [ ] **Step 2:** Run `mise exec -- go test ./internal/status/` → Expected: FAIL.
- [ ] **Step 3:** Implementar.
- [ ] **Step 4:** Run `mise exec -- go test -race ./...` → Expected: PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat: status lists pending relations"`

---

### Task 9: Skills `/trama:add-repo` e `/trama:map`

**Files:**
- Create: `skills/add-repo/SKILL.md`, `skills/map/SKILL.md`, `skills/map/card-reference.md`
- Modify: `README.md`, `README.pt-BR.md` (comandos novos e as duas skills)

**Interfaces:**
- Consumes: `trama add`, `trama sync`, `trama status`, `trama stamp`, `trama validate`, `trama graph`.
- Produces: `/trama:add-repo <url> [--name N] [--branch B]` e `/trama:map <repo> | --all`.

`skills/add-repo/SKILL.md` (texto exato)
```markdown
---
description: Add a repository to the trama workspace, clone it and map it. Use when the user asks to add or include a repository in the workspace.
---

# Add a repository

Arguments: the repository URL, optionally `--name <name>` and `--branch <branch>` ($ARGUMENTS). Work from the workspace root.

1. Run `trama add` with the same arguments. If it fails, show the error and stop.
2. Run `trama sync <name>`, using the name `trama add` printed.
3. Map it by following the `/trama:map <name>` skill.
```

`skills/map/SKILL.md` (texto exato)
```markdown
---
description: Map one repository, or all of them, of a trama workspace into catalog cards with file:line evidence for every fact. Use when the user asks to map, re-map or refresh the catalog, or when trama status says a card is missing or stale.
---

# Map repositories into the catalog

Arguments: a repository name, or `--all` ($ARGUMENTS). Work from the workspace root, the directory that holds `workspace.yaml`.

A card describes the repository's main branch, and every fact in it is proven by a line you read yourself. A card that guesses is worse than no card.

## 1. Check each repository

1. Run `trama sync <name>`.
2. Run `trama status`. The repository must be on its main branch, clean and `+0 -0`. If it is not, skip it and tell the user what is in the way. Never stash, reset or switch branches yourself.

## 2. Read the repository

In `repos/<name>`:
- Read its `AGENTS.md`, `CLAUDE.md` and `README` first. They say how the team runs and tests it.
- Read the manifests and version files that exist: `go.mod`, `package.json`, `.nvmrc`, `.tool-versions`, `.python-version`, `pyproject.toml`, `pom.xml`, `build.gradle`, `Dockerfile`, `compose.yaml`, `.env.example`.
- If the card already exists, run `git -C repos/<name> diff --stat <mapped_at_commit> HEAD` to see what changed. Still re-check every evidence line already in the card.

## 3. Write the card

Write `catalog/<name>.yaml` following `${CLAUDE_PLUGIN_ROOT}/skills/map/card-reference.md`. For every entry under `identity`, `provides`, `consumes` and `config`:
- Open the file and read the line before you cite it as `path:line`, with the path relative to the repository root.
- If you cannot point at a line, leave the entry out.

Look for:
- **provides**: HTTP routes it serves, events it publishes, packages it publishes.
- **identity**: ports it listens on; hosts listed for it under `hosts` in `workspace.yaml` (evidence `workspace.yaml`); the package name it publishes; topics it publishes.
- **consumes**: HTTP calls to other services, events it subscribes to, internal packages it imports. Add `fields_used` when the code reads specific response fields.
- **config**: environment variables that hold another service's address, with their example value.
- **run** and **tools**: how to set it up, run it (with its port) and test it, and the runtime versions from its version files. Prefer what its own docs say.

## 4. Link consumers to providers

Read the other cards in `catalog/`. A consume gets `from: <repo>` only with its own proof, written in `via`:
- `env`: a `config` entry of this card points to that repository, and its value matches a host or port in that repository's `identity`.
- `package`: this repository imports the package named in that repository's `identity.package`.
- `topic`: this repository subscribes to a topic listed in that repository's `identity.topics`.
- `declared`: `workspace.yaml` has a `links` entry with this consumer, that provider and this `id`.

A matching `id` alone is not proof. Without proof, write `from: unknown` and list in `candidates` the repositories whose cards provide the same `id`.

When mapping several repositories, write all cards first, then revisit every `from: unknown`: a provider mapped later may now prove the link.

## 5. Stamp, validate, graph

1. Run `trama stamp <name>` for each card you wrote. It records the commit the card describes.
2. Run `trama validate` and fix every error from the code. Never silence an error with evidence you did not read: remove the entry, or use `from: unknown`.
3. When there are no errors, run `trama graph`.

## 6. Report

Tell the user:
- for each repository, the relations added, removed or changed (`git diff -- catalog/` shows them);
- every `from: unknown` left, with its candidates and what would prove it (a config value, a declared host, a `links` entry);
- that nothing was committed: they review and commit `catalog/`, `docs/graph.md` and `AGENTS.md`.
```

`skills/map/card-reference.md`: documento de referência, em inglês, com:
- cada campo da ficha (§6.2 do spec mais o esquema da Task 3): tipo e se é obrigatório;
- as regras de `kind`, `id`, evidência e `via` das Global Constraints e da Task 5;
- as duas fichas de exemplo do demo, completas e válidas.

- [ ] **Step 1: Cenário de aceite antes das skills (RED).** Build de desenvolvimento em `$SP/devbin/trama` (com `$SP` = scratchpad). Workspace novo em `$SP/accept-ws` com `trama init`, `trama add` dos dois repos de demo do GitHub e `trama sync`. Despachar um subagente novo (general-purpose) com a tarefa: "map every repository of the trama workspace at `$SP/accept-ws` into catalog cards; use `$SP/devbin/trama` for every trama command", **sem** apontar para as skills. Rodar `trama validate` no resultado. Expected: falhas ou relação ausente (é o comportamento sem a skill). Guardar a saída no ledger.
- [ ] **Step 2:** Escrever os três arquivos de skill, os READMEs e um teste novo `TestCardReferenceExamplesValidate` em `internal/validate`. Ele extrai os dois blocos YAML de exemplo do `card-reference.md`, monta um workspace com os arquivos de evidência que eles citam e afirma `Errors == 0`. Assim o exemplo da documentação não pode ficar inválido.
- [ ] **Step 3: Cenário de aceite com as skills (GREEN).** Outro workspace novo `$SP/accept-ws2`, preparado do mesmo jeito. Despachar outro subagente novo com: "follow `<repo>/skills/map/SKILL.md` (with `${CLAUDE_PLUGIN_ROOT}` = `<repo>`) to map `--all` in `$SP/accept-ws2`; use `$SP/devbin/trama` for every trama command". Conferir:
  - `trama validate` → `ok: 2 card(s), 1 relation(s), 0 warning(s)`;
  - ficha do api: `provides` `GET /hello`, com evidência na linha de `grep -n 'GET /hello", Hello' internal/api/api.go`; `identity.ports` 8080, com evidência na linha de `grep -n 'const addr' cmd/api/main.go`;
  - ficha do web: `consumes` `GET /hello` `from: trama-demo-api` `via: env`, com `fields_used: [message]` e evidência dentro de `src/api/hello.ts`; `config` `VITE_API_URL` com `value: http://localhost:8080`, `points_to: trama-demo-api` e evidência `.env.example:1`;
  - `docs/graph.md` tem a aresta `GET /hello`, e o `AGENTS.md` tem a tabela entre os marcadores.
  Se algo falhar, corrigir o texto da skill e repetir num workspace novo.
- [ ] **Step 4:** Critério 2 da §10: apagar uma evidência de uma ficha de `$SP/accept-ws2`. Run `trama validate` → Expected: erro `no evidence`, com arquivo e `Where`. Restaurar.
- [ ] **Step 5:** Run `mise exec -- go test -race ./... && claude plugin validate .` → Expected: PASS.
- [ ] **Step 6: Commit** — `git commit -m "feat: /trama:add-repo and /trama:map skills"`

---

### Task 10: Release `v0.0.4` e o `trama-demo` mapeado

**Files:**
- Modify: `.claude-plugin/plugin.json` (`0.0.4`); no `~/Documents/GitHub/trama-demo`: `mise.toml` (pin `0.0.4`), `catalog/`, `docs/graph.md`, `AGENTS.md`

- [ ] **Step 1:** Atualizar o `plugin.json` para `0.0.4`. Run `mise exec -- go test -race ./... && mise exec -- goreleaser check` → Expected: PASS. Commit: `chore: release 0.0.4`.
- [ ] **Step 2: ⚠️ Confirmar com o usuário** a publicação: tag `v0.0.4`, `main` e o `trama-demo` mapeado. Com o sim, seguir a ordem do README:
  - `git tag v0.0.4 && git push origin v0.0.4`; esperar o workflow `release` e conferir com `mise exec github:lukascivil/trama@0.0.4 -- trama version` → `0.0.4`;
  - `git push origin main`; esperar o CI verde.
- [ ] **Step 3:** No `trama-demo`:
  - trocar o pin para `0.0.4` e rodar `mise install`;
  - seguir `skills/map/SKILL.md` com `--all`, usando `mise exec -- trama`;
  - conferir `mise exec -- trama validate` → `ok: 2 card(s), 1 relation(s), 0 warning(s)` e `mise exec -- trama status --quiet` → nenhuma linha.
- [ ] **Step 4:** Commit no `trama-demo` (`git status` antes) e `git push`, dentro da confirmação do Step 2.
- [ ] **Step 5:** Pedir ao usuário para rodar `/trama:init` e `/trama:add-repo` numa sessão interativa, com o plugin instalado do marketplace. É a parte que o agente não consegue rodar aqui.
