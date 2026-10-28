# trama — documento de design

Data: 2026-10-27 · Status: rascunho para aprovação

## 1. Intenção e porquê

Empresas têm vários repositórios que dependem uns dos outros (ex.: um front que consome a API de um back), mas cada repo tem vida própria: ferramentas, forma de rodar local, documentação e times diferentes. Migrar tudo para um monorepo é caro e muitas vezes inviável.

Agentes de IA (Claude Code) trabalham dentro de um repo por vez e não sabem que `front-app` depende de `backend-app`. Por isso não conseguem responder "se eu mudar isso, o que quebra?" nem planejar uma mudança que atravessa repos.

**trama** é um meta-harness local: uma camada acima do Claude Code que junta os repos de uma empresa num **workspace** (um repo próprio, sem código de aplicação), mapeia como eles se relacionam **com evidência no código**, sabe rodá-los e coordena mudanças entre eles. Nada sai da máquina além do que já vai para o GitHub.

O nome vem de "trama" do tecido: cada repo é um fio, a ferramenta é a trama que liga.

Referência de mercado: o Nx Polygraph resolve um problema parecido, mas é SaaS e exige mandar o código para terceiros, o que não é aceitável. A trama é a alternativa local, versionada no git e sem fornecedor.

## 2. Objetivos, não-objetivos e anti-objetivos

### Objetivos

1. Declarar quais repos fazem parte de um workspace e clonar/atualizar todos com um comando.
2. Gerar e manter uma **ficha** por repo (stack, como rodar, o que expõe, o que consome) em que toda relação tem evidência (`arquivo:linha`).
3. Gerar um grafo de dependências legível por humanos e pela IA.
4. Responder "se eu mudar X, o que é afetado?" com repos, arquivos e ordem segura.
5. Executar uma mudança que atravessa repos: branches com o mesmo nome, implementação em cada repo, PRs ligados entre si, na ordem segura.
6. Guardar a memória de cada mudança entre repos (porquê, plano, PRs) no git do workspace, para que outro dev e a IA dele saibam o que foi feito.
7. Rodar um repo ou todos juntos sem a pessoa conhecer o processo de cada um.
8. Avisar quando uma ficha está desatualizada em relação ao código.

### Não-objetivos (fora do MVP)

- Banco de dados de qualquer tipo. A fonte da verdade são arquivos no git.
- Docker como requisito. Docker compose pode entrar depois, só quando algum projeto precisar de infra (Postgres, Redis, fila).
- Portal, interface web ou dashboard.
- GitLab, Bitbucket ou outros hosts. MVP usa só GitHub via `gh`.
- Outros harnesses além do Claude Code (o `AGENTS.md` deixa o workspace legível por outros, mas só o Claude Code é testado).
- Entender dependências de infraestrutura (Terraform, Helm, Dockerfile `FROM`).
- Merge automático de PRs.
- Migrar repos para monorepo ou exigir mudanças dentro dos repos dos times.

### Anti-objetivos (seria um fracasso mesmo funcionando)

- Mudar código de um repo de código sem PR, ou empurrar direto na branch principal dele. (O repo do workspace é a exceção, só para os registros em `changes/`.)
- Registrar uma relação sem evidência. Uma ficha que "chuta" é pior que nenhuma ficha.
- Ficha desatualizada sem aviso. Contexto errado leva a IA a errar com confiança.
- Exigir que cada time adote algo dentro do próprio repo para a trama funcionar.
- Depender de qualquer serviço externo além do GitHub.
- Descartar alterações locais de alguém ao sincronizar.

## 3. Restrições

- Tudo local. Dependências da máquina: `git`, `gh` (autenticado), `mise`, Claude Code. O resto (o binário `trama` e os runtimes de cada repo) é instalado pelo mise.
- macOS é a plataforma principal; Linux deve funcionar. Windows fora do MVP.
- Repos públicos no GitHub da conta `lukascivil`, licença MIT.
- A ferramenta não pode conter nada específico de nenhuma empresa. O workspace de uma empresa real é sempre privado; o de demonstração é público.
- A trama nunca deduz relação pelo nome do repo. O prefixo `trama-` é só organização.

## 4. Visão geral

Separação central: **ferramenta** (reutilizável, pública) × **workspace** (os dados de uma empresa).

| Repo | Papel |
|---|---|
| `trama` | A ferramenta: CLI em Go + plugin do Claude Code (skills e hook), no mesmo repo |
| `trama-demo` | Workspace de exemplo (lista de repos, fichas, grafo, memória) |
| `trama-demo-api` | Backend hello world em Go |
| `trama-demo-web` | Front hello world em Vite + React + TypeScript |

Divisão de trabalho dentro da ferramenta:

- **CLI `trama` (Go)** faz o que precisa ser exato e repetível: clonar, status, validar, gerar grafo, rodar. Humanos e a IA chamam os mesmos comandos.
- **Skills** fazem o que exige julgamento: ler código e montar a ficha, analisar impacto, planejar e executar mudanças.

## 5. Estrutura do workspace

```
trama-demo/
├── AGENTS.md                 ← fonte das instruções para a IA (parte gerada entre marcadores)
├── CLAUDE.md                 ← só "@AGENTS.md"
├── workspace.yaml            ← lista dos repos
├── mise.toml                 ← fixa a versão do binário trama
├── .gitignore                ← ignora repos/
├── .claude/settings.json     ← habilita o plugin trama no projeto + hook de início de sessão
├── catalog/
│   ├── trama-demo-api.yaml
│   └── trama-demo-web.yaml
├── docs/
│   └── graph.md              ← grafo Mermaid gerado a partir do catálogo
├── changes/
│   └── 2026-11-03-renomear-message.md
└── repos/                    ← ignorado pelo git; cada repo clonado com seu próprio .git
    ├── trama-demo-api/
    └── trama-demo-web/
```

O Claude Code é aberto na raiz do workspace. O `CLAUDE.md`/`AGENTS.md` da raiz carrega no início; os `CLAUDE.md` de cada repo em `repos/` carregam quando a IA mexe nos arquivos de lá, então as regras de cada repo continuam valendo.

## 6. Formatos de arquivo

### 6.1 `workspace.yaml`

```yaml
version: 1
name: trama-demo
repos:
  - name: trama-demo-api
    url: https://github.com/lukascivil/trama-demo-api.git
    branch: main
  - name: trama-demo-web
    url: https://github.com/lukascivil/trama-demo-web.git
    branch: main
```

Cada repo é clonado em `repos/<name>`. `name` é único no workspace.

### 6.2 `catalog/<repo>.yaml` (ficha)

Formato inspirado no `catalog-info.yaml` do Backstage, sem depender dele.

```yaml
version: 1
name: trama-demo-web
description: Front que mostra a saudação vinda da API
stack: [node, react, typescript]
tools:                      # versões que o mise usa para rodar este repo
  node: "22"
mapped_at_commit: 8b77d02   # commit do repo quando a ficha foi gerada
mapped_at: 2026-10-27
run:
  dev:  { cmd: "npm run dev", port: 5173 }
  test: { cmd: "npm test" }
  setup: { cmd: "npm install" }
provides: []
consumes:
  - kind: http
    id: "GET /hello"
    from: trama-demo-api
    via: env                 # como o vínculo com o fornecedor foi provado (ver regras)
    fields_used: [message]
    evidence: ["src/api/hello.ts:5", "src/App.tsx:12"]
config:
  - env: VITE_API_URL
    value: "http://localhost:8080"
    points_to: trama-demo-api
    evidence: [".env.example:1"]
```

E do lado de quem expõe:

```yaml
identity:                    # como outros repos podem reconhecer este
  ports: [{ value: 8080, evidence: ["cmd/api/main.go:10"] }]
  hosts: []                  # ex.: api.acme.com; pode vir declarado no workspace.yaml
  package: null              # nome do pacote publicado, se houver
  topics: []                 # tópicos de eventos publicados
provides:
  - kind: http
    id: "GET /hello"
    contract: null           # caminho para OpenAPI/GraphQL/proto quando existir
    shape: { message: string }
    evidence: ["internal/http/hello.go:14"]
```

Hosts que o código não revela (ex.: URLs de produção) podem ser declarados por uma pessoa no `workspace.yaml`, em `repos[].hosts`. Declaração humana conta como evidência (`evidence: ["workspace.yaml"]`).

Regras:

- `kind` aceita `http`, `event` e `package` no MVP. O demo exercita só `http`.
- `id` de `http` é normalizado como `"MÉTODO /caminho"`, com parâmetros como `{param}` (ex.: `GET /orders/{id}`).
- **Relação** = um `consumes` com `from: X` e o mesmo `id` de um `provides` da ficha de `X`.
- Toda entrada de `provides`, `consumes`, `config` e `identity` tem pelo menos uma evidência. Sem evidência, não entra.
- **Ligar um consumo a um fornecedor exige prova própria**, registrada em `via`:
  - `env`: o valor de uma variável de ambiente do consumidor bate com um `host` ou `port` da `identity` do fornecedor;
  - `package`: o consumidor importa o `package` do fornecedor;
  - `topic`: o consumidor assina um tópico da `identity` do fornecedor;
  - `declared`: uma pessoa declarou o vínculo no `workspace.yaml`.
- Só o `id` igual não basta (ex.: `GET /health` existe em vários serviços). Nesse caso a IA registra `from: unknown` com `candidates: [X, Y]`. Isso aparece como pendência no grafo e no status, nunca é escondido. A pessoa resolve declarando um host ou o vínculo no `workspace.yaml`.
- `mapped_at_commit` é sempre um commit da branch principal do repo e é o que permite detectar ficha desatualizada.

### 6.3 `changes/AAAA-MM-DD-<slug>.md` (memória)

Um arquivo por mudança entre repos, escrito pela skill `change` e commitado no workspace:

- **Porquê:** o pedido original e a motivação.
- **Impacto:** repos e arquivos afetados (saída do `impact`).
- **Plano:** passos por repo, na ordem, com o padrão expandir → migrar → contrair.
- **Execução:** branch usada, links dos PRs, status de cada passo (pendente, PR aberto, mergeado).
- **Pendências:** passos deixados para depois (ex.: remover o campo antigo).
- **Continua:** link para um registro anterior, quando esta mudança fecha uma pendência dele.

Continuar uma pendência é uma mudança nova que aponta para a antiga. Ao começar, a `change` procura em `changes/` pendências que tocam o mesmo `id` e oferece continuar a partir delas. Não há modo de "retomar" no MVP.

### 6.4 `AGENTS.md` da raiz

Escrito pelo `init`, com uma parte gerada pelo comando `trama graph` entre marcadores (`<!-- trama:begin -->` / `<!-- trama:end -->`). Conteúdo:

- O que é este workspace e como ele está organizado.
- Tabela resumo dos repos e das relações (gerada).
- Regras de roteamento para a IA:
  - Antes de editar um repo, ler o `AGENTS.md`/`CLAUDE.md` dele.
  - Se a mudança toca algo listado em `provides`, `consumes` ou `config`, rodar `/trama:impact` antes.
  - Mudanças que atravessam repos usam `/trama:change`.
  - Nunca commitar código de `repos/` no repo do workspace.
  - Se o status disser que uma ficha está desatualizada, não confiar nela: rodar `/trama:map` antes.

## 7. CLI `trama` (Go)

Um binário único em Go, no próprio repo `trama` (`cmd/trama`). Ele faz a parte determinística e é usado tanto por pessoas quanto pelas skills.

- Lê e valida o YAML com structs tipadas: o esquema das fichas vive no código e é testado com `go test`.
- Para git e GitHub, chama os comandos `git` e `gh` da máquina em vez de reimplementá-los. Assim respeita as credenciais e configurações de quem usa.
- É distribuído como binário pelo GitHub Releases (GoReleaser, macOS e Linux). O `mise.toml` do workspace fixa a versão em `[tools]` (ex.: `"github:lukascivil/trama" = "0.1.0"`) e `mise install` instala.
- Binário e plugin saem juntos, com a mesma tag. As skills checam se a versão do binário bate com a do plugin e avisam se não bater.
- Inicia em milissegundos, o que importa para o hook de início de sessão.

| Comando | O que faz |
|---|---|
| `trama sync` | Clona os repos que faltam. Nos existentes, sempre faz `fetch` e, só se estiver limpo e na branch principal, avança para a última versão (fast-forward). Caso contrário, avisa e pula o avanço. Nunca descarta alteração local. |
| `trama status` | Por repo: branch atual, alterações locais, à frente/atrás do remoto e se `origin/<branch>` (último fetch) ≠ `mapped_at_commit` (ficha desatualizada). Não usa rede. Lista também pendências `from: unknown` e mudanças em `changes/` com passos em aberto. Tem um modo silencioso para o hook. |
| `trama changes` | Usa `gh` para consultar o estado de cada PR listado nos registros de `changes/` e atualiza o status de cada passo (PR aberto, mergeado, fechado). Se algo mudou, commita só os arquivos de `changes/` e envia para a main do workspace. |
| `trama validate` | Confere `workspace.yaml` e fichas: campos obrigatórios, evidência em toda relação, `from` existente, `via` coerente com a `identity` do fornecedor, `id` de consumo existente no fornecedor. Sai com erro se algo falhar. |
| `trama graph` | Gera `docs/graph.md` (Mermaid, uma aresta por relação com o `id`) e a tabela entre marcadores no `AGENTS.md`. Mesma entrada → mesma saída. |
| `trama setup <repo>` | Roda o `run.setup` do repo (ex.: `npm install`), se houver. |
| `trama dev <repo>` | Roda o `run.dev` do repo, na pasta dele, com as versões de `tools` da ficha (via `mise exec`). |
| `trama up` | Roda `trama setup` se necessário e depois o `run.dev` de todos os repos em paralelo, com a saída prefixada pelo nome do repo. |
| `trama test <repo>` | Roda o `run.test` do repo. Usado pela skill `change`. |

## 8. Skills (plugin `trama`)

Distribuídas como plugin do Claude Code, com o marketplace no próprio repo `trama`. Instalação manual:

```
/plugin marketplace add lukascivil/trama
/plugin install trama@trama
```

O `.claude/settings.json` do workspace declara o marketplace e habilita o plugin no escopo do projeto, para que quem clonar o workspace seja convidado a instalá-lo.

Nenhuma skill commita sozinha no workspace, com uma exceção: o registro em `changes/` vai **direto para a main do workspace** (commit e push), sem PR. A `change` faz isso depois que você aprova o plano e a cada atualização de status. Os PRs ficam só nos repos de código. Se a main do workspace estiver protegida e o push for recusado, a skill para e avisa. O resto (fichas, grafo) fica como alteração para você revisar e commitar.

### `/trama:init`

Cria a estrutura da seção 5 na pasta atual (vazia ou com git iniciado). Fixa no `mise.toml` a versão do binário `trama` igual à do plugin e roda `mise install`. Não cria repo no GitHub.

### `/trama:add-repo <url>`

Adiciona o repo ao `workspace.yaml`, roda `trama sync` e em seguida `/trama:map` para esse repo.

### `/trama:map [repo | --all]`

**A ficha sempre descreve a branch principal.** Antes de mapear, a skill roda `trama sync` para o repo. Se o repo não estiver limpo e na branch principal atualizada, ela para e explica o que falta, em vez de mapear outra coisa.

Lê o código do repo e cria ou atualiza a ficha:

1. Detecta stack, versões (`go.mod`, `.nvmrc`, `package.json` `engines`, `.tool-versions` etc.) e comandos de rodar e testar, preferindo o que o README/`AGENTS.md` do repo diz.
2. Encontra o que o repo expõe (rotas, eventos publicados, pacotes) e o que consome (chamadas HTTP, eventos assinados, dependências de pacote internas, variáveis de ambiente que apontam para outros serviços), sempre com `arquivo:linha`.
3. Cruza com as outras fichas para preencher `from`.
4. Atualiza `mapped_at_commit`.
5. Roda `trama validate` e `trama graph`.
6. Mostra um resumo do que mudou na ficha (relações novas, removidas, alteradas).

Se já existe ficha, o mapeamento foca no que mudou desde `mapped_at_commit`, mas revalida todas as evidências existentes (uma linha pode ter mudado de lugar).

### `/trama:impact <descrição da mudança>`

Somente leitura. Parte do catálogo e do grafo e depois confirma no código dos repos afetados. Responde:

- quais repos e arquivos são afetados, com `arquivo:linha`;
- se a mudança quebra consumidores ou é compatível;
- a ordem segura para publicar.

Se alguma ficha envolvida estiver desatualizada, avisa antes e sugere `/trama:map`.

### `/trama:change <descrição da mudança>`

1. Roda o `impact`.
2. Escreve o plano em `changes/` usando expandir → migrar → contrair (ex.: o back passa a mandar o campo novo mantendo o antigo; o front passa a usar o novo; a remoção do antigo fica como passo pendente).
3. Pede sua aprovação do plano, commita o registro e envia para a main do workspace.
4. Em cada repo afetado, cria a branch `trama/<slug>` a partir da branch principal atualizada. Se o repo tiver alterações locais, para e pede para você resolver; nunca faz stash nem descarta nada.
5. Implementa repo por repo, seguindo as regras do próprio repo, e roda `trama test` em cada um.
6. Abre os PRs com `gh`, cada um com links para os PRs irmãos e para o registro em `changes/`, e indicando a ordem de merge.
7. Atualiza o registro com os links e o status, commita e envia para a main do workspace.
8. Volta cada repo para a branch principal (a branch `trama/<slug>` continua existindo). Assim `sync` e `map` seguem funcionando.

Nunca empurra para a branch principal de um repo de código, nunca faz merge e nunca pula o teste do repo. Se o teste falhar, para e relata.

Depois dos merges, `trama changes` atualiza o status dos passos e `/trama:map` atualiza as fichas.

### Hook de início de sessão

O plugin registra um hook `SessionStart` que roda `trama status` no modo silencioso quando a sessão abre dentro de um workspace da trama (existe `workspace.yaml` com `version`). Se houver repo faltando, ficha desatualizada ou pendência, mostra um aviso curto. Precisa ser rápido (sem rede: usa só o estado local) e nunca pode impedir a sessão de abrir.

## 9. Repos de demonstração

### `trama-demo-api`

- Go, só biblioteca padrão (`net/http`).
- `GET /hello` → `{"message": "Hello, World"}` na porta 8080.
- CORS liberado para a origem do front em desenvolvimento.
- Um teste do handler.
- `CLAUDE.md` curto com como rodar e testar (para provar que as regras de cada repo são respeitadas).

### `trama-demo-web`

- Vite + React + TypeScript, `npm`.
- Lê `VITE_API_URL` (padrão `http://localhost:8080`), chama `GET /hello` e mostra `message` na tela.
- `.env.example`, `.nvmrc`.
- `CLAUDE.md` curto com como rodar e testar.

Os dois repos não sabem da trama. Não têm nada específico dela.

### Demo reproduzível

- Os dois repos de demo recebem a tag `start` no estado inicial (antes de qualquer mudança).
- O `trama-demo` publicado fica no estado final, com o registro da mudança, como vitrine.
- O README do `trama-demo` explica como refazer tudo: fazer fork dos dois repos de demo a partir da tag `start`, trocar as URLs no `workspace.yaml` e seguir os critérios de aceite.
- Eu crio os repos públicos na conta `lukascivil` (confirmando com você na hora); os merges dos PRs de demonstração são seus.

## 10. Critérios de aceite do MVP

1. Partindo de uma pasta vazia, `/trama:init` e dois `/trama:add-repo` geram `trama-demo` com as duas fichas e um grafo com a aresta `trama-demo-web → trama-demo-api` via `GET /hello`, com evidências que apontam para as linhas certas.
2. `trama validate` passa; ao apagar uma evidência de uma ficha, falha com mensagem clara.
3. `trama up` sobe os dois e o navegador mostra "Hello, World".
4. `/trama:impact "renomear message para greeting no GET /hello"` aponta `trama-demo-web` com `arquivo:linha`, diz que quebra e sugere a ordem back → front → remoção.
5. `/trama:change` com o mesmo pedido gera o registro em `changes/`, abre um PR no api (adiciona `greeting`, mantém `message`) e um no web (passa a usar `greeting`), ligados entre si, com a remoção de `message` como pendência.
6. Depois dos merges, `trama sync` traz as mudanças; ao abrir uma sessão, o hook avisa que as fichas estão desatualizadas; `trama changes` marca os passos como mergeados; `/trama:map --all` atualiza as fichas e o aviso some.
7. Outra pessoa clona `trama-demo`, abre o Claude Code, é convidada a instalar o plugin, roda `trama sync` e consegue ler o registro da mudança e continuar a pendência.

## 11. Fases

1. **Esqueleto:** repo `trama` com o CLI em Go (`sync` e `status`), release com GoReleaser, instalação via mise, plugin com marketplace e `/trama:init`; repos de demo criados.
2. **Mapa:** `/trama:add-repo`, `/trama:map`, `trama validate`, `trama graph` (critérios 1 e 2).
3. **Rodar:** `trama dev`, `trama up`, `trama test` (critério 3).
4. **Mudança:** `/trama:impact` e `/trama:change` (critérios 4 e 5).
5. **Desatualização e time:** hook de início de sessão e fluxo do segundo dev (critérios 6 e 7). Tag `v0.1.0`.

## 12. Decisões

### Tomadas por você

- Construir uma ferramenta própria, local, em vez de usar SaaS (Polygraph).
- Nome `trama`, com o mesmo prefixo nos repos do projeto.
- Repos públicos.
- Demonstração com um hello world: o back devolve a mensagem e o front mostra.
- Sem banco de dados; mapeamento persistido em arquivos.
- mise como base para instalar ferramentas e runtimes.
- Registro de mudança vai direto para a main do workspace (commit e push), sem PR; os PRs ficam só nos repos de código.
- Parte determinística como CLI em Go (em vez de bash), chamando o `git` e o `gh` da máquina.

### Tomadas por mim (revise)

- Ferramenta distribuída como plugin do Claude Code com marketplace no próprio repo.
- Binário distribuído pelo GitHub Releases e instalado pelo mise, sempre com a mesma versão do plugin.
- Front em Vite + React + TypeScript e back em Go, para exercitar duas stacks.
- Repos de demo com nomes `trama-demo-api` e `trama-demo-web`.
- `AGENTS.md` como fonte e `CLAUDE.md` só importando ele.
- Skills nunca commitam no workspace, exceto o registro de mudança após sua aprovação.
- Branch das mudanças entre repos: `trama/<slug>`; ao terminar, a `change` volta cada repo para a branch principal.
- A ficha sempre descreve a branch principal; desatualização é medida contra `origin/<branch>` do último fetch.
- Vínculo entre consumidor e fornecedor exige prova própria (`via`); `id` igual sozinho vira `from: unknown` com candidatos.
- Continuar uma pendência é uma mudança nova que aponta para a anterior, sem modo de retomada.
- Demo reproduzível por fork a partir da tag `start`; `trama-demo` publicado no estado final.
- Conteúdo dos repos (código, skills, README) em inglês por serem públicos; este documento em português. Um `README.pt-BR.md` na `trama`.
- Licença MIT.

## 13. Deixado para quem construir

- Organização interna do código Go e escolha de bibliotecas (YAML, linha de comando), mensagens e saída colorida.
- Como expressar o esquema das fichas (só structs Go ou também um JSON Schema publicado).
- Estilo do grafo Mermaid.
- Formato da saída paralela do `trama up`.
- Testes do CLI e dos repos de demo além do mínimo descrito.
- Mecanismo exato para manter juntas as versões do plugin e do binário.
- Texto exato dos `SKILL.md`, desde que siga os comportamentos e regras da seção 8.
