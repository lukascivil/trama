# trama: grafo visual

Desenho de uma fase nova, depois das fases 1 a 4 do [design do trama](2026-10-27-trama-design.md). Ela tira do MVP o não-objetivo "Portal, interface web ou dashboard", só para esta página local.

> **Atualização (2026-11-01):** a página deixou o Cytoscape e o JavaScript sem build. Hoje é React Flow, gerado por Vite, e o anti-objetivo sobre Node passou a valer só para quem usa o trama. Veja [grafo em React Flow](2026-11-01-trama-grafo-react-flow-design.md). O resto deste documento (dados, segurança, aceite) continua valendo.

## 1. Intenção e porquê

Nas palavras de quem pediu: o dev "quer saber todas as relações em todos os projetos que ele adicionou", "enxergar como se tudo fosse um único monorepo e trabalhar por ali tendo esse suporte visual", com a "data de release" de cada projeto, e "entender que subir algo num projeto" pode afetar outros. A referência é o `nx graph`: o dev roda um comando e abre no navegador.

As fichas em `catalog/` já guardam todas as relações provadas. O `docs/graph.md` (Mermaid) mostra o grafo no GitHub, mas é estático: não dá para clicar, focar nem filtrar, e fica ilegível com dezenas de repos. Esta fase põe uma página interativa por cima dos mesmos dados, rodando na máquina do dev, porque o código da empresa não sai dela (o motivo de o trama existir).

## 2. Objetivos, não-objetivos e anti-objetivos

### Objetivos

1. Um comando abre no navegador uma página local com todos os repos do workspace e todas as relações das fichas.
2. Ao clicar num repo, a página mostra quem pode ser afetado se o dev publicar algo ali: os afetados diretos (relação provada, com o `id` e o `arquivo:linha` da prova) e os indiretos (dependem de quem depende), sempre distinguidos.
3. Cada repo mostra a última release: versão, data e quantos commits da branch principal ainda não saíram numa release.
4. A página mostra o que o trama já sabe do estado: relações pendentes (`?`), ficha desatualizada ou faltando, repo não clonado e mudanças abertas em `changes/` que passam pelo repo.
5. Funciona com cerca de 100 repos: busca, filtros, foco num repo e nos vizinhos, e esconder repos sem relação.
6. O painel de um repo mostra a ficha: descrição, stack, como rodar, o que fornece, o que consome e as evidências, com link para o código.

### Não-objetivos

- Mudar qualquer coisa pela página: editar fichas, rodar repos, criar mudanças. Ela só lê.
- Servidor compartilhado ou hospedado para o time, com login.
- Exportar a página como HTML estático (pode vir depois).
- Chamar a API do GitHub ou de outro host (releases, deploys, PRs).
- Saber o que está em produção (data de deploy).
- Tipos novos de relação (gRPC, GraphQL, infraestrutura) e seguir uma rota exata através de um gateway: é outra fase.
- Atualizar sozinha quando as fichas mudam: recarregar a página basta.

### Anti-objetivos (seria um fracasso mesmo funcionando)

- Qualquer coisa sair da máquina: CDN, fonte externa, telemetria, pedido da página para fora do próprio servidor.
- Mostrar como provada uma relação sem prova, ou um afetado indireto como se fosse direto.
- Aceitar conexões de outras máquinas.
- Interpretar texto das fichas ou dos registros como HTML ou script. As fichas descrevem código de terceiros.
- Exigir Node, npm ou uma etapa de build para desenvolver ou publicar o trama.

## 3. Restrições

- Tudo no binário `trama` em Go. A página é HTML, CSS e JavaScript puros, embutidos com `go:embed`.
- Cytoscape.js vendorizado no repositório, em versão fixa e com a licença dele. No máximo uma extensão de layout, também vendorizada.
- As dependências Go continuam só `cobra` e `yaml`.
- Offline: o comando não faz `fetch`. Usa os clones e as tags que o `trama sync` trouxe.
- macOS e Linux, como o resto do trama. O release (GoReleaser) não muda.

## 4. Como o dev usa

```
trama graph --serve [--port <n>] [--no-open]
```

- Sem `--serve`, o `trama graph` faz o que faz hoje (escreve `docs/graph.md` e a seção do `AGENTS.md`).
- Com `--serve`, sobe um servidor em `127.0.0.1`, imprime a URL e abre o navegador (`open` no macOS, `xdg-open` no Linux), a não ser com `--no-open`. Roda até Ctrl+C, que sai com código 130 sem mensagem, como `trama dev` e `trama up`.
- Cada carregamento da página relê os arquivos do workspace e o git dos clones. Depois de um `/trama:map` ou `trama sync`, recarregar mostra os dados novos.
- O `--serve` só serve: não reescreve o `docs/graph.md` nem o `AGENTS.md`, e sobe mesmo quando o `trama validate` acusa erros (eles aparecem na página, seção 6).
- Funciona de qualquer pasta dentro do workspace e com `-C`, como os outros comandos.

## 5. A página

O dev aprovou este esboço: topo com o nome do workspace, as contagens (repos, relações, pendentes) e uma busca; filtros à esquerda; o grafo no centro; o painel do repo selecionado à direita.

- **Nós:** um por repo, com o nome e a release (`v2.3.0 · há 11 dias`, `sem release` ou `não clonado`). Marcas para ficha desatualizada ou faltando e para mudança aberta.
- **Setas:** do consumidor para o fornecedor, com o `id` (`GET /orders/{id}`, o tópico, o pacote). Relação pendente é uma seta tracejada para um nó `?`. Com muitas setas, os rótulos podem aparecer só com zoom ou ao passar o mouse.
- **Clique num repo:** ele fica destacado, os afetados diretos com uma cor, os indiretos com outra, e o resto esmaece. O painel da direita mostra:
  - nome, descrição, stack e última release (versão, data, commits depois dela);
  - o que fornece e o que consome (de quem e com qual prova);
  - "Se você publicar algo aqui, podem ser afetados": os diretos com o `id` e o `arquivo:linha` da prova no consumidor; os indiretos com o caminho (`via api-gateway`); os possíveis, de relações sem prova (seção 6.1);
  - as mudanças abertas que passam pelo repo (`id` e título do registro);
  - as evidências, com link para o código.
- **Clique numa seta:** `id`, tipo, a prova (`via`) e a evidência no consumidor.
- **Foco:** mostra só o repo e os vizinhos até N níveis.
- **Busca:** por nome de repo ou por `id`. Um `id` destaca o fornecedor e todos os consumidores, como o `trama uses`.
- **Filtros:** repos, tipos de relação (HTTP, eventos, pacotes, pendentes), esconder repos sem relação, só repos com mudança aberta.
- **Escala:** legível com 100 repos, com layout automático, zoom e arrastar.
- **Idioma:** inglês por padrão e português quando o navegador está em português, com uma tabela de textos pequena na própria página (como o README e o site, que têm as duas versões). Os textos citados neste documento são os da versão em português.

## 6. Dados

O servidor monta em Go um JSON com tudo o que a página desenha (`GET /api/workspace`); a página não calcula relações. Fontes:

- `workspace.yaml`: os repos e os `links` declarados. Um `link` que nenhuma ficha cita vira mesmo assim uma relação, com a prova "declarada no `workspace.yaml`" no lugar do `arquivo:linha`, porque uma pessoa a declarou.
- `catalog/*.yaml`: as fichas, lidas de forma tolerante, como o `trama status`. Relações e pendentes vêm de `graph.Build`, a mesma regra do `docs/graph.md`.
- `validate.Check`: o `graph.Build` confia no `from` da ficha, e o que mantém o `docs/graph.md` honesto é o `trama graph` recusar rodar com erros de validação. A página não recusa: ela roda a validação e marca como **não verificada** toda relação cujo consumo tem erro (prova que não fecha, evidência que não existe, fornecedor que não fornece o `id`). Relação não verificada é desenhada diferente das provadas e conta só como "possível" (6.1). Os erros e avisos aparecem numa faixa no topo da página. A validação lê as evidências pelo git de cada clone; para caber no critério de 2 segundos, o resultado pode ser guardado enquanto a ficha e o `origin/<branch principal>` do clone não mudarem.
- `status.Collect`: clonado ou não, estado da ficha (atual, desatualizada, faltando, desconhecida).
- `changes.Load`: registros com `status: open`. Um registro passa por um repo quando algum passo dele, não mergeado, é desse repo.
- O git de cada clone, para a release (6.2).

Uma ficha que não abre vira um aviso no topo da página, e o repo aparece sem relações. Nunca derruba a página.

### 6.1 Afetados por um repo R

- **Diretos:** cada relação provada (inclusive declarada) com fornecedor R. Mostra o consumidor, o `id` e a evidência do consumidor, ou "declarada no `workspace.yaml`".
- **Indiretos:** repos que chegam a R seguindo relações provadas de trás para frente em dois ou mais passos (consumidores de consumidores), com o caminho mais curto. A página diz que são "podem ser afetados, via X", porque o trama não prova que a rota usada passa por R. Ciclos não repetem repos.
- **Possíveis:** consumidores com `from: unknown` que têm R entre os `candidates`, e consumidores de relações não verificadas com fornecedor R. A página diz por que a relação não tem prova.

### 6.2 Release

- A última release de um repo é a maior tag semver (`v1.2.3` ou `1.2.3`, pré-releases ordenadas pela regra do semver) alcançável a partir de `origin/<branch principal>` no clone.
- A data é a do tagger para tag anotada e a do commit para tag leve.
- "Commits depois" é o número de commits em `origin/<branch principal>` que não estão na tag.
- Sem tag semver: `sem release`. Repo não clonado: `não clonado`.
- Tags em vez de GitHub Releases: funciona offline, com qualquer host e sem limite de API, e todo GitHub Release tem uma tag.
- Confirmar que o `fetch` do `trama sync` traz tags novas, inclusive em commits já baixados. Se não trouxer, o sync passa a buscá-las.

### 6.3 Links

- Evidência: quando o origin do repo é do GitHub, link para `https://github.com/<dono>/<repo>/blob/<mapped_at_commit>/<caminho>#L<a>-L<b>`. O commit em que a ficha foi mapeada garante que a linha bate. Com outro host, o `arquivo:linha` aparece como texto.
- Registro de mudança: o mesmo link que `trama changes save` imprime (`changes.RecordURL`), ou o caminho quando não houver.

## 7. Segurança

- Escuta só em `127.0.0.1`. Recusa pedidos cujo `Host` não seja `localhost` ou `127.0.0.1` com a porta certa (proteção contra DNS rebinding).
- Só `GET`. Serve apenas os arquivos embutidos e o JSON; nada do disco por caminho vindo do pedido.
- `Content-Security-Policy` que só permite o próprio servidor (`default-src 'self'`), para garantir que a página não pede nada lá fora.
- Todo texto vindo das fichas, dos registros e do git entra na página como texto (`textContent`), nunca como HTML.

## 8. Testes

- Go, com testes que falham antes do código:
  - o JSON: relações, pendentes, `link` declarado sem ficha que o cite, relação não verificada por erro de validação, afetados diretos, indiretos com ciclo e possíveis; ficha que não abre; repo não clonado;
  - a release: ordem semver com pré-release, tag anotada e tag leve, sem tag, commits depois da tag;
  - o servidor: só `GET`, recusa de `Host` estranho, o cabeçalho CSP, os arquivos embutidos presentes;
  - o comando: `--serve --no-open` sobe e para com Ctrl+C; porta ocupada dá erro claro.
- A página não tem testes automáticos de JS (sem npm). Ela é conferida no navegador nos critérios de aceite.

## 9. Critérios de aceite

1. No `trama-demo`, `trama graph --serve` abre a página com os dois repos e a seta `trama-demo-web → trama-demo-api` com `GET /hello`.
2. Ao clicar em `trama-demo-api`, `trama-demo-web` aparece como afetado direto, com a evidência `src/api/hello.ts` e um link para a linha certa no GitHub.
3. Um repo sem tag semver mostra `sem release`. Com uma tag `v0.1.0` local e dois commits depois dela, mostra `v0.1.0`, a data e "2 commits depois".
4. Um workspace de teste com 100 repos fictícios abre em menos de 2 segundos depois de o servidor estar no ar, e a busca, o foco e os filtros funcionam.
5. Com uma relação `from: unknown`, a seta tracejada para `?` aparece, e o repo candidato lista esse consumidor em "possíveis".
6. Uma ficha com `<script>alert(1)</script>` na descrição mostra o texto e não executa nada.
7. Com uma evidência quebrada na ficha do consumidor (linha 999), a página sobe, mostra o erro na faixa do topo, e a seta aparece como não verificada; o fornecedor lista o consumidor em "possíveis", não em "diretos".
8. Com o navegador em inglês, os textos aparecem em inglês; em português, em português.
9. O painel de rede do navegador não mostra nenhum pedido para fora do servidor local.
10. `trama graph` sem `--serve` continua igual: os testes atuais passam.

## 10. Decisões

### Tomadas por você

- Uma página local no navegador, no estilo do `nx graph`, para ver todas as relações do workspace como se fosse um monorepo.
- Mostrar a release de cada repo e quem pode ser afetado ao publicar algo num repo (você aprovou o esboço com isso).
- Cytoscape.js na página; o Mermaid continua no `docs/graph.md`.

### Tomadas por mim (revise)

- O comando é `trama graph --serve`, e não um comando novo.
- A release vem das tags semver do git, não de GitHub Releases nem de data de deploy; e a página mostra os commits depois da release.
- Os afetados indiretos são calculados pelas relações existentes e aparecem como "podem ser afetados", sem prova por rota. Os possíveis vêm das relações sem prova.
- A página só lê, e os dados são relidos a cada carregamento, sem atualização automática.
- Segurança: só loopback, checagem do `Host`, CSP restrito e texto sempre escapado; sem chave de sessão, porque só processos da própria máquina chegam ao servidor, e esses já leem os arquivos.
- O JSON é calculado em Go (testável); a página só desenha.
- O `--serve` sobe mesmo com erros de validação: roda a validação, marca as relações com erro como não verificadas (contam só como possíveis) e mostra os erros no topo. Não reescreve `docs/graph.md` nem `AGENTS.md`.
- Um `link` declarado no `workspace.yaml` sem ficha que o cite aparece como relação declarada, e conta como afetado direto.
- A página fala inglês por padrão e português quando o navegador está em português.
- Os links de evidência apontam para o commit em que a ficha foi mapeada.

## 11. Deixado para quem construir

- O layout do Cytoscape (um dos embutidos ou uma extensão vendorizada), o estilo visual e o comportamento dos rótulos com muitas setas.
- Os nomes exatos dos campos do JSON e a organização dos arquivos (um pacote novo para o servidor e os arquivos embutidos, por exemplo).
- A porta padrão e o que fazer quando está ocupada (erro claro ou outra porta livre, desde que a URL impressa seja a certa).
- Quantos níveis o foco mostra por padrão.
- Como gerar o workspace de 100 repos fictícios para o critério 4.
