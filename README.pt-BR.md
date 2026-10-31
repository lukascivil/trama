# trama

[English](README.md)

Empresas têm vários repositórios que dependem uns dos outros — um front que chama a API de um back, serviços que trocam eventos —, mas cada repositório tem vida própria: ferramentas, forma de rodar local, documentação e time. Migrar tudo para um monorepo é caro e muitas vezes inviável. Agentes de IA trabalham dentro de um repositório por vez, então não sabem que o `front-app` depende do `backend-app`, não respondem "o que quebra se eu mudar isso?" e não planejam uma mudança que atravessa os dois.

**trama** é um meta-harness local: uma camada acima do Claude Code que junta os repositórios de uma empresa num **workspace** (um repositório próprio, sem código de aplicação), mapeia como eles se relacionam com evidência `arquivo:linha`, sabe rodá-los e coordena mudanças entre eles. Nada sai da sua máquina além do que já vai para o GitHub. O nome vem da trama do tecido: cada repositório é um fio, e a trama é o que liga todos.

## Status

Pré-lançamento. As fases 1 a 4 de 5 estão prontas: workspaces, `add`, `sync`, `status`, fichas do catálogo com evidência, validação, grafo de dependências, rodar os repositórios (`setup`, `dev`, `test`, `up`), análise de impacto e mudanças entre repositórios. Uma checagem no início da sessão e o fluxo de quem chega no time vêm a seguir.

## Requisitos

- [git](https://git-scm.com) e o [GitHub CLI](https://cli.github.com) (`gh`), autenticado
- [mise](https://mise.jdx.dev)
- [Claude Code](https://code.claude.com)

## Instalar o plugin

No Claude Code:

```
/plugin marketplace add lukascivil/trama
/plugin install trama@trama
```

O plugin coloca um comando `trama` no `PATH` do Claude que roda, via mise, o binário da trama com a mesma versão do plugin.

## Criar um workspace

Abra o Claude Code numa pasta vazia e rode `/trama:init`. Ele cria:

```
workspace.yaml        os repositórios deste workspace
mise.toml             fixa a versão do binário trama
AGENTS.md, CLAUDE.md  instruções para os agentes
.claude/settings.json habilita o plugin trama para quem clonar o workspace
catalog/ docs/ changes/
repos/                os clones (ignorados pelo git)
```

## Mapear os repositórios

No Claude Code, a partir do workspace:

- `/trama:add-repo <url>` adiciona um repositório, clona e mapeia.
- `/trama:map <repo>` (ou `--all`) lê um repositório e escreve a ficha dele, `catalog/<repo>.yaml`: stack, como rodar, o que expõe e consome, e a evidência `arquivo:linha` de cada fato. Um consumidor só é ligado a um fornecedor com prova (um valor de configuração que bate com o host ou a porta do fornecedor, um pacote, um tópico ou um vínculo declarado no `workspace.yaml`); sem prova, a relação fica `from: unknown` com os candidatos.

Depois, `trama graph` escreve o `docs/graph.md` e a tabela de repositórios no `AGENTS.md`. As skills nunca fazem commit: revise e commite `catalog/`, `docs/graph.md` e `AGENTS.md`.

## Mudanças entre repositórios

- `/trama:impact <mudança>` diz, sem alterar nada, quais repositórios e linhas uma mudança afeta, se ela quebra algum deles e a ordem segura para publicar (expandir → migrar → contrair).
- `/trama:change <mudança>` escreve um registro em `changes/`, pede sua aprovação, cria uma branch `trama/<slug>` em cada repositório a partir da branch principal atualizada, faz a mudança, roda os testes de cada repositório e abre PRs ligados entre si, na ordem de merge. Nunca envia para a branch principal e nunca faz merge.
- Depois dos merges, `trama changes` atualiza o registro a partir dos PRs, e `/trama:map --all` atualiza as fichas.

## Comandos

Rode de qualquer lugar dentro do workspace (no seu terminal, via `mise exec -- trama ...` ou com o mise ativado):

| Comando | O que faz |
|---|---|
| `trama add <url>` | Adiciona um repositório ao `workspace.yaml`. Detecta a branch padrão. Não clona. |
| `trama sync [repo...]` | Clona os repositórios que faltam. Busca os outros e só avança os que estão limpos e na branch principal. Nunca descarta alterações locais. |
| `trama status [--quiet]` | Mostra branch, alterações locais, distância do remoto, estado da ficha e relações sem prova de cada repositório. Sem rede. `--quiet` mostra só os problemas. |
| `trama validate` | Confere todas as fichas: campos obrigatórios, evidência que existe no clone e prova de cada relação. |
| `trama stamp <repo>` | Grava na ficha o commit que ela descreve (`origin/<branch>`). Recusa se o clone não estiver limpo e atualizado. |
| `trama graph` | Escreve o `docs/graph.md` (Mermaid) e a tabela entre os marcadores do `AGENTS.md`. Recusa catálogo inválido. |
| `trama setup [repo...]` | Roda o `run.setup` de cada ficha (ex.: `npm install`) e lembra disso, para que `dev`, `test` e `up` só repitam depois de commits novos, de um comando novo ou de versões novas das ferramentas. |
| `trama dev <repo>` | Roda o `run.dev` em primeiro plano, depois do setup quando preciso, com as versões de runtime da ficha via `mise exec`. |
| `trama test [repo...]` | Roda o `run.test` de cada repositório, depois do setup quando preciso, e mostra `passed` ou `failed` para cada um. |
| `trama up [repo...]` | Roda o `run.dev` de todos os repositórios ao mesmo tempo, cada linha com o nome do repositório. Recusa subir se uma porta de dev já estiver em uso. Ctrl+C para todos os processos, inclusive os filhos. |
| `trama uses <id>` | Lista os repositórios que fornecem ou consomem um id (ex.: `"GET /hello"`), com as evidências. |
| `trama branch <repo> <nome>` | Cria uma branch a partir da branch principal atualizada do repositório; recusa repositório com alterações locais. `--main` volta para a principal. |
| `trama pr <repo> --title T --body-file F` | Envia a branch atual e abre um PR com o `gh`. Recusa a branch principal. |
| `trama changes [--no-push]` | Atualiza cada registro de mudança aberto a partir dos PRs, depois commita e envia os registros. `trama changes save <id>` commita e envia um registro. |
| `trama init [dir]` | Cria os arquivos do workspace (é o que o `/trama:init` roda). |
| `trama version` | Mostra a versão. |

## Desenvolvimento

```bash
go test ./...
```

Para testar o plugin a partir deste checkout com um build local:

```bash
go build -o /tmp/trama-dev ./cmd/trama
TRAMA_BIN=/tmp/trama-dev claude --plugin-dir .
```

## Publicar uma versão

O plugin é servido a partir da branch padrão e roda o binário com a própria versão, então os binários precisam existir antes de a `main` mudar:

1. Coloque a nova versão em `.claude-plugin/plugin.json` e faça o commit.
2. Faça push da tag `v<versão>`. O workflow de release roda os testes, recusa uma tag diferente da versão do plugin e publica os binários com o GoReleaser.
3. Quando a release existir, faça push da `main`.
4. Atualize o pin `github:lukascivil/trama` no `mise.toml` de cada workspace.

## Limitações conhecidas

- `trama add` regrava o `workspace.yaml` sem as linhas em branco. Comentários e campos desconhecidos são mantidos.
- `trama stamp` regrava a ficha do mesmo jeito: as linhas em branco somem e os mapas em linha são compactados (`{ cmd: x }` vira `{cmd: x}`).

## Licença

[MIT](LICENSE)
