# trama

[English](README.md)

Empresas têm vários repositórios que dependem uns dos outros — um front que chama a API de um back, serviços que trocam eventos —, mas cada repositório tem vida própria: ferramentas, forma de rodar local, documentação e time. Migrar tudo para um monorepo é caro e muitas vezes inviável. Agentes de IA trabalham dentro de um repositório por vez, então não sabem que o `front-app` depende do `backend-app`, não respondem "o que quebra se eu mudar isso?" e não planejam uma mudança que atravessa os dois.

**trama** é um meta-harness local: uma camada acima do Claude Code que junta os repositórios de uma empresa num **workspace** (um repositório próprio, sem código de aplicação), mapeia como eles se relacionam com evidência `arquivo:linha`, sabe rodá-los e coordena mudanças entre eles. Nada sai da sua máquina além do que já vai para o GitHub. O nome vem da trama do tecido: cada repositório é um fio, e a trama é o que liga todos.

## Status

Pré-lançamento. A fase 1 de 5 está pronta: workspaces, `add`, `sync` e `status`. Fichas do catálogo, grafo de dependências, análise de impacto e mudanças entre repositórios vêm a seguir.

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

## Comandos

Rode de qualquer lugar dentro do workspace (no seu terminal, via `mise exec -- trama ...` ou com o mise ativado):

| Comando | O que faz |
|---|---|
| `trama add <url>` | Adiciona um repositório ao `workspace.yaml`. Detecta a branch padrão. Não clona. |
| `trama sync [repo...]` | Clona os repositórios que faltam. Busca os outros e só avança os que estão limpos e na branch principal. Nunca descarta alterações locais. |
| `trama status [--quiet]` | Mostra branch, alterações locais, distância do remoto e estado da ficha de cada repositório. Sem rede. `--quiet` mostra só os problemas. |
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

## Licença

[MIT](LICENSE)
