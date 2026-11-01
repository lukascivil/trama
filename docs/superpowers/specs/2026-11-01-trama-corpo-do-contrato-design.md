# trama — Corpo do contrato — Design

## Problema

O catálogo responde "este dado já vem de algum endpoint?" pelos campos `request` e `response` de cada `provides`. Quem escreve esses campos é o agente do `/trama:map`, e a instrução é condicional: "adicione quando conseguir ler o body".

Na prática:

- **As fichas voltam sem bodies**, mesmo nos repositórios que commitam um OpenAPI 3 com o schema de cada body.
- **Quase nenhuma ficha aponta `contract` para esse spec**, embora o arquivo esteja no repositório. E ninguém confere o `contract`: um ponteiro quebrado passa em silêncio.
- **Nada avisa a omissão.** Uma ficha sem bodies é igual à de um repositório que não tem body nenhum.
- **O `trama data <termo>` só responde pelas tabelas**, porque o lado dos endpoints fica vazio. A camada que a descoberta de dado considerou a mais útil ("um endpoint que já devolve o dado dá para chamar hoje") nunca é preenchida.
- **Rotas ficam de fora da ficha.** O skill aceita ler as rotas "do router ou do spec", e nada compara uma coisa com a outra. Rota esquecida é relação sem prova na ficha de outro repositório.

Transcrever à mão não escala: uma ficha com centenas de rotas teria milhares de campos para copiar. É a mesma assimetria que motivou a varredura de schema: o agente pula o que dá trabalho, e ninguém fica sabendo.

## Decisões

**OpenAPI 3 é dado estruturado, não julgamento.** O design de descoberta de dado deixou de fora "ler OpenAPI, GraphQL ou proto para preencher os campos". Esta fase reverte isso só para OpenAPI 3. A varredura de schema deixou a leitura de tabela com o agente porque SQL, Prisma, Ruby e Python são linguagens, e entender cada uma exige julgamento. Um OpenAPI 3 é um documento de forma definida: `paths.<path>.<method>`, `requestBody`, `responses` e `$ref` dentro do arquivo. Achar os campos de um body nele é tão mecânico quanto casar uma rota, e o mesmo spec dá o mesmo resultado para todo mundo. Quem faz é o Go, como no `trama link`.

**O spec é dono dos bodies das rotas que declara.** Do mesmo jeito que o `link` é dono de `from` e `via`: toda execução reescreve `request` e `response` das rotas que um spec declara, e remove o body que o spec não declara mais. Body escrito à mão fica só para as rotas que nenhum spec commitado declara (spec servido em runtime, rota sem spec). A evidência da rota não muda: a linha do router prova a rota, o `contract` diz de onde vieram os campos.

**O router é a fonte das rotas; o spec, dos bodies e da conferência.** Um spec gerado e commitado pode estar atrás do router. A ficha lista o que o router serve. O `trama bodies` avisa, sem escrever, as rotas que o spec declara e a ficha não lista: quem mapeia confere no router e decide.

**Uma leitura, dois consumidores.** `trama bodies` escreve. `trama validate` usa o mesmo plano para avisar quando a ficha está atrás do spec. Um `contract` que não é arquivo do repositório passa a ser erro, como evidência inexistente.

## Qual spec vale para uma rota

1. **O `contract` da ficha vence.** Se ele aponta para um OpenAPI 3, a rota é procurada nesse spec.
2. **Sem `contract`, vale o único spec OpenAPI 3 do repositório que declara a rota.** A regra do `link`: um prova, dois não dizem qual. Achado assim, o comando também escreve o `contract`.
3. **Dois ou mais specs declaram a rota:** nada é escrito, e a saída pede para definir o `contract`.
4. **Um `contract` que não é YAML nem JSON** (proto, GraphQL) não é erro: o comando só não o lê.

Rotas se casam como no `link` (`catalog.NormalizeID`). Só entram os métodos que uma ficha aceita: `get`, `put`, `post`, `delete`, `options`, `head`, `patch`.

## Como achar os specs

Pelo conteúdo, não pelo nome: pelo nome escapariam `docs/api.yaml` ou `spec/v1.yaml`.

- **Candidatos:** `.yaml`, `.yml` e `.json` fora de `node_modules`, `vendor`, `dist`, `build`, `target`, `testdata`, `fixtures`, `test`, `tests`, `__tests__`, `venv`, `site-packages` e diretórios ocultos. Ao contrário da varredura de schema, `spec` não fica de fora: é onde muito spec mora. Lockfiles (`package-lock.json`, `npm-shrinkwrap.json`, `pnpm-lock.yaml`) ficam de fora pelo tamanho.
- **Filtro barato:** só é parseado o arquivo que contém `openapi:` (com ou sem aspas).
- **Conta como spec:** o topo tem `openapi: 3.x` e um mapping `paths`. Arquivo que não parseia é ignorado.
- **Uma leitura em lote**, pela regra do `ws.Files(repo)`. O repositório só é varrido quando alguma rota não tem `contract`.

## Como um body vira campo

| No spec | Na ficha |
|---|---|
| `requestBody.content` | `application/json`; sem ele, o primeiro tipo `+json`; sem ele, o primeiro tipo |
| `responses` | o menor status 2xx que tem `content`, depois `2XX`; `default` e erros não contam |
| `$ref: "#/…"` | seguido, inclusive `components/requestBodies` e `components/responses`, e encadeado |
| objeto com `properties` | desce: `customer.email` |
| lista de objetos | desce com `[]`: `items[].sku` |
| lista de escalares | o tipo: `[]string` |
| objeto sem `properties` (mapa livre) | `object` |
| lista de listas | `[]object` |
| `integer` | `number` |
| `type: [string, "null"]`, `nullable: true` | o tipo não nulo |
| `allOf`, `oneOf`, `anyOf` | a união das propriedades: a busca quer todo campo que pode vir |
| `readOnly` / `writeOnly` | fora do request / fora do response |
| nome que não é segmento de caminho (`$schema`, nome com `.`) | fora |
| ciclo de `$ref` | `object` / `[]object` no ponto em que volta |
| campo sem tipo legível | fora |

Os campos saem na ordem do spec, e um caminho repetido pela união aparece uma vez.

Um body que não é lido como campos fica como está na ficha, e a saída diz por quê: `is a list, not an object`, `is a string, not an object`, `is an object with no properties`, `has no schema`, `has no type trama can read`, `uses $ref "<ref>", outside the file`, `uses $ref "<ref>", which is not in the file`, `uses $ref "<ref>", which points back to itself`, `is too large for trama to read (more than 100000 schemas, or 32 levels deep)`.

## Como a ficha é editada

Na árvore de nós YAML, como `link` e `stamp`: comentários, ordem das chaves e o resto do arquivo ficam como estavam. A ordem das chaves é `kind`, `id`, `contract`, `request`, `response`, `evidence`. Uma entrada escrita numa linha (`{kind: http, …}`) vira bloco quando ganha body, e o comentário de fim de linha dela vai para a última linha da entrada. Antes de escrever, o `Apply` confere que a entrada ainda tem o mesmo `id`. Uma segunda execução não muda nada.

Os helpers de edição de nós, que o `linker` tinha só para ele, vão para `internal/yamlx`. No caminho, o `Set` deixa de inserir antes da chave de referência quando ela é a última do mapping.

## Forma

```
$ trama bodies
api: POST /orders <- openapi.yaml:42 (request 3 fields, response 5 fields)
api: GET /orders/{id} <- openapi.yaml:88 (contract, response 1 field)
api: DELETE /orders/{id} <- openapi.yaml:120 (no request)
api: GET /export: response is a list, not an object: left as it is
api: GET /legacy: not in contract openapi.yaml
api: GET /health: declared by a.yaml and b.yaml: set contract to the one that serves it
api: 12 routes are in no OpenAPI spec of the repository
api: openapi.yaml declares 2 routes the card does not list: GET /stats, POST /orders/{id}/cancel
web: repos/web is not cloned: run "trama sync web"

wrote 3 provides; review the diff in catalog/ and run "trama validate"
```

```
warning: catalog/api.yaml: provides: 2 provides differ from the OpenAPI spec that declares them (POST /orders, GET /orders/{id}): run "trama bodies api"
error: catalog/api.yaml: provides[0] GET /hello: contract missing.yaml: missing.yaml does not exist (in repos/api)
error: catalog/api.yaml: provides[0] GET /hello: contract "../x.yaml": path must be relative to the repository root, without ..
```

O aviso é um por ficha e não aparece quando o clone não existe. As notas (rota ambígua, body que é lista, rota que o spec declara e a ficha não lista) não viram aviso: são para quem roda o comando.

## Segurança

O spec vem de um repositório do workspace: é entrada não confiável.

- `contract` com `..` ou absoluto é recusado antes de qualquer leitura (`catalog.ValidateContract`, a regra da evidência).
- Symlink e submódulo: `ws.Files` já recusa.
- Expansão exponencial (alias YAML que contém a si mesmo, `$ref` em leque a cada nível): teto por body de 100 000 schemas visitados e 32 níveis. Passou, o body vira problema dito na saída.
- `$ref` na saída sai com `%q`. Nome de campo só entra na ficha se passar por `catalog.ValidateField`, e o tipo é de vocabulário fechado.
- Escrita atômica (`fsx.ReplaceFile`). Nenhuma rede, nenhuma execução.

## O que não entra

- Swagger 2.0, AsyncAPI, GraphQL e proto. O `contract` pode apontar para eles; o comando só não lê.
- `$ref` para outro arquivo. O body fica como está e a saída diz por quê.
- `servers:` com caminho. A rota é casada como está escrita.
- Spec servido só em runtime: não há arquivo para ler, e o agente continua lendo o handler.
- `fields_used` de quem consome: ler o que o código usa é julgamento.
