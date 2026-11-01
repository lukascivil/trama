# trama: seletor de repositórios

Desenho de uma melhoria no `/trama:add-repo`, depois do [grafo visual](2026-10-31-trama-grafo-visual-design.md). Convenção deste documento: o que o Lucas decidiu está escrito sem marca; o que foi escolha de quem desenhou leva **(decisão minha)** e pode ser trocado.

## 1. Intenção e porquê

Hoje montar um workspace numa organização grande é manual: o dev procura cada repo no GitHub, copia o link e pede ao Claude `/trama:add-repo <url>`, um repo de cada vez. Nas palavras de quem pediu: "foi trabalhoso, teve que procurar um a um pelo github, pegar o link e adicionar um a um pelo claude".

O dev já sabe quais repos usa ("cada dev sabe dos repos que trabalha na empresa"), e a organização pode ter mais de 100 repos de vários times. O custo está em achar e copiar, não em decidir. A melhoria é listar os repos da organização, deixar o dev filtrar e marcar vários, e adicionar todos de uma vez.

## 2. Objetivos, não-objetivos e anti-objetivos

### Objetivos

1. Dado o nome de uma organização do GitHub, o dev vê todos os repos dela que o token dele alcança, incluindo os privados.
2. Numa tela com checkboxes e um filtro de texto, ele marca os repos que quer, em quantas buscas precisar, e confirma uma vez.
3. Funciona com mais de 100 repos: o filtro responde na hora e "marcar todos os que casam com o filtro" existe.
4. Os repos escolhidos entram no `workspace.yaml` com a mesma validação do `trama add`, e o Claude segue com `trama sync` e `/trama:map` neles, sem o dev repetir pedido.
5. O caminho atual continua: `/trama:add-repo <url>` e `trama add <url>` funcionam como hoje.

### Não-objetivos

- GitLab, Bitbucket e outros hosts. **(decisão minha)** O desenho assume GitHub; o Lucas respondeu "ok" à pergunta se tudo está no GitHub, e isso foi lido como sim.
- Remover repos do workspace, clonar ou mapear pela página. A página só escolhe.
- Guardar uma lista de organizações no `workspace.yaml`. **(decisão minha)** A organização vem do argumento ou da lista de organizações do token.
- Filtrar por time dono, topics ou convenções da empresa. O filtro é por texto.

### Anti-objetivos (seria um fracasso mesmo funcionando)

- O token do GitHub chegar à página, ao Claude, a um arquivo ou ao terminal. Ele vive só no processo do `trama`.
- A página falar com qualquer host que não seja o servidor local. A listagem do GitHub é pedida pelo binário, nunca pelo navegador.
- Outro site, ou outra máquina, conseguir escolher repos ou ler a lista por esse servidor.
- Interpretar nome, descrição ou qualquer texto vindo do GitHub como HTML ou script.
- Adicionar um repo que o `trama add` recusaria, ou deixar o `workspace.yaml` meio escrito se algo falhar.
- Exigir algo além do binário para quem usa: Node, CDN ou download na primeira execução.

## 3. Restrições

- Tudo no binário `trama`. A página é embutida com `go:embed`, como a do grafo, e gerada pelo mesmo ferramental que o grafo usa hoje. Node e build são aceitáveis para quem mantém o trama, não para quem usa.
- Dependências Go novas: nenhuma além da biblioteca padrão para falar com a API do GitHub. **(decisão minha)**
- O servidor local segue as regras do servidor do grafo (`internal/view/server.go`): escuta só em `127.0.0.1`, recusa `Host` fora do esperado e recusa `Sec-Fetch-Site` de outro site, com a mesma CSP.
- macOS e Linux. O release (GoReleaser) não muda.
- Os testes não usam rede: o cliente do GitHub aceita a URL base, e os testes apontam para um `httptest`.

## 4. Como o dev usa

### Pelo Claude

`/trama:add-repo` decide pelo argumento:

- **URL ou caminho** (contém `/`, `:` ou termina em `.git`): igual a hoje.
- **Um nome simples** (`minha-org`): se existe uma pasta com esse nome na pasta atual, é o caminho local de hoje (o `trama add` já trata um nome simples como caminho relativo), e nada muda. Se não existe, é uma organização e abre o seletor. **(decisão minha)** Quem quer forçar um lado usa `trama add --pick <org>` ou passa `./nome`.
- **Sem argumento:** o seletor, com a lista de organizações do token.

No caminho do seletor, a skill:

1. roda o comando do seletor **em segundo plano**, porque ele espera uma pessoa e pode levar minutos, e diz ao dev que a página abriu no navegador;
2. quando o comando termina, lê os nomes que ele imprimiu e roda `trama sync <nomes>`;
3. mapeia todos os repos adicionados, um por vez, como a skill `/trama:map` já faz, sem perguntar nem limitar a quantidade. Mapear faz parte de adicionar, mesmo que a seleção seja grande. Se ficar lento demais, otimiza-se depois; isso não é desta fase.

Se o comando termina sem nada adicionado (dev cancelou ou fechou), a skill diz isso e para.

### O comando

`trama add --pick [<org>]` **(decisão minha: o nome e a forma ficam com quem construir, o comportamento abaixo é o que vale)**.

- Sobe o servidor em `127.0.0.1` (porta como `trama graph --serve`: padrão fixo com queda para uma livre), imprime a URL e abre o navegador, a não ser com `--no-open`.
- Termina quando o dev confirma ou cancela na página, ou com Ctrl+C (código 130, como `trama dev`). Também termina sozinho depois de 30 minutos sem nenhuma requisição à página. **(decisão minha)**
- Na saída, imprime um nome por linha, os repos que entraram no `workspace.yaml`, e uma linha por repo recusado com o motivo.
- Não clona: o `trama sync` clona, como depois do `trama add`.

### A página

- Campo de filtro por texto sobre nome e descrição, sem diferenciar maiúsculas. Foco nele ao abrir.
- Uma linha por repo: checkbox, nome, descrição (como texto puro), linguagem principal e data do último push. **(decisão minha)** para os campos além do nome.
- Arquivados ficam escondidos, com um botão para mostrá-los. **(decisão minha)**
- Repos que já estão no workspace aparecem marcados e desabilitados. **(decisão minha)**
- "Marcar os N que casam" e "desmarcar os que casam", para o caso de "todos os `pay-*`".
- A seleção persiste ao mudar o filtro. Um rodapé fixo mostra quantos estão marcados e tem "Adicionar" e "Cancelar".
- Sem organização no argumento, um seletor no topo lista a conta do dev e as organizações do token. **(decisão minha)**
- Depois de confirmar, a página mostra o resultado (quem entrou, quem foi recusado e por quê) e diz que pode ser fechada.

## 5. Dados e regras

- **Listagem:** `GET /orgs/{org}/repos` paginado (100 por página) até o fim. Se a organização não existir, tenta `GET /users/{org}/repos`. O servidor lista tudo antes de abrir a página e a página recebe a lista por `GET /api/repos`.
- **Token:** `gh auth token` se o `gh` existir, senão `GH_TOKEN` e `GITHUB_TOKEN`. Sem token, lista só os repos públicos e a página avisa que os privados não aparecem. Se o GitHub recusar (403 por SSO, 404, limite de requisições), o erro dele vai para a página e para o terminal, sem o token.
- **URL de clone:** `clone_url` (https) ou `ssh_url`, conforme `gh config get git_protocol` quando o `gh` existe; senão https. **(decisão minha)**
- **Branch:** o `default_branch` que a API devolve. Não se roda `git ls-remote` por repo.
- **Nome:** o mesmo que `trama add` deriva da URL. Nomes iguais de organizações diferentes colidem, e a colisão é recusada, não renomeada.
- **Confirmação:** a página manda os identificadores dos repos marcados por `POST`. O servidor resolve cada um contra a lista que ele próprio carregou (nunca confia em URL vinda da página), roda a mesma checagem do `trama add` (`CheckNew`, `AddRepo`) e grava o `workspace.yaml` **uma única vez**, com os que passaram. Os recusados voltam com o motivo.
- **Escrita protegida:** o `POST` exige um valor aleatório gerado a cada execução, entregue com a página e devolvido em cada requisição. Além disso, os mesmos filtros de `Host` e `Sec-Fetch-Site` do grafo.

## 6. Critérios de aceite

1. Com um servidor GitHub falso com 250 repos em 3 páginas, a página lista os 250, o filtro reduz a lista e "marcar os que casam" seleciona só os visíveis.
2. Confirmar 3 repos grava os 3 no `workspace.yaml` de uma vez, imprime 3 nomes e sai com 0. Se um dos 3 colide com um nome existente, os outros 2 entram e a colisão aparece com o motivo.
3. Cancelar, fechar a página por 30 minutos ou Ctrl+C não altera o `workspace.yaml`.
4. O token nunca aparece na saída, na página, nas respostas do servidor nem nos erros.
5. Um `POST` sem o valor aleatório, com `Host` errado ou com `Sec-Fetch-Site: cross-site` é recusado e não escreve nada.
6. Descrição com `<script>` aparece como texto.
7. `trama add <url>` e `/trama:add-repo <url>` se comportam como antes.

## 7. O que fica com quem construir

- O nome e a forma exata do comando (`add --pick`, `pick`, etc.) e das rotas do servidor.
- Como reaproveitar o servidor e as regras de segurança do `internal/view` sem copiar código.
- Como a página entra no ferramental de build que o grafo já usa (mesmo bundle ou outro) e o layout visual.
- Detalhes de paginação, limite de requisições e tempo de espera das chamadas ao GitHub.
- O texto exato da skill `add-repo`, do README e do `README.pt-BR.md`.
