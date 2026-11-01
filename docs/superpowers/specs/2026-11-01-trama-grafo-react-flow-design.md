# trama: grafo em React Flow

Emenda o [grafo visual](2026-10-31-trama-grafo-visual-design.md). O que a página faz continua igual (selecionado, direto, indireto, possível, foco, busca, filtros); muda como ela é feita e como se lê.

## 1. Intenção e porquê

Nas palavras de quem pediu: com o Cytoscape os nós "ficam todos amontoados" na primeira carga, o drawer da direita "vem tudo amontoado", e faltava busca em providers, consumers e tabelas. A meta é uma tela que entregue "melhor experiência para o usuário".

O diagnóstico, antes de trocar de biblioteca: o amontoado vinha do layout `cose` (força), que não conhece direção, e não do Cytoscape. O drawer era uma lista só. O que o Cytoscape realmente não faz é um nó com conteúdo (nome, versão, tipo), e é por isso que a página passa para React Flow.

## 2. Objetivos, não-objetivos e anti-objetivos

### Objetivos

1. Primeira carga arrumada: consumidores antes de fornecedores, sem sobrepor nós, repos sem relação em grade à parte.
2. Cada nó mostra o nome, a versão e a data da release e um chip do tipo do repo (api, web, app, worker, lib, infra).
3. Clicar num nó mantém a visão de relações de antes: selecionado, direto, indireto e possível, com a legenda, e o resto esmaecido.
4. O drawer da direita tem cabeçalho fixo (nome, tipo, release, descrição, stack, Focar) e abas com contagem (Afetados, Fornece, Consome, Tabelas, Rodar, Mudanças), cada uma com seu campo de filtro.
5. O drawer da esquerda escolhe tipos de relação, tipos de repo e repositórios: marcar e desmarcar, Todos, Nenhum, um filtro de nome e "só este" por repo.
6. Clicar num repo dentro do drawer o seleciona e o traz ao centro do grafo.

### Não-objetivos

- Mudar o que o servidor Go calcula: afetados, relações, provas e release continuam em Go.
- Um campo novo na ficha para o tipo do repo (ver 5).

### Anti-objetivos

- A página pedir qualquer coisa fora do servidor local, ou o dev que só usa o trama precisar de algo além do binário.
- Texto das fichas virar HTML. O React escapa o texto, e nada na página usa `dangerouslySetInnerHTML`.

## 3. Restrições

- **Muda:** o anti-objetivo "exigir Node, npm ou uma etapa de build" passa a valer só para quem **usa** o trama. Quem mantém o código da página pode precisar de Node.
- O bundle (um `app.js` e um `app.css`, via Vite) fica commitado em `internal/view/assets/` e entra no binário por `go:embed`. `go install`, `go build` e o GoReleaser seguem sem Node.
- O CI reconstrói o bundle e falha se o commitado não for o que o código-fonte gera.
- O CSP continua `default-src 'self'`; o teste de assets confere que o bundle só chama `/api/workspace`.

## 4. Como funciona

- Código-fonte em `web/` (React, `@xyflow/react`, `@dagrejs/dagre`, TypeScript). A lógica sem tela (setas, filtros, marcas, busca, layout) fica em módulos puros com testes (`vitest`).
- O layout é o dagre. Ele roda nas duas direções, da esquerda para a direita e de cima para baixo, e a página usa a que desenha o grafo maior no tamanho da área do grafo. Só muda quando o que está na tela muda, nunca ao clicar.
- Os rótulos das setas aparecem só quando marcadas ou com o mouse em cima, se há mais de 12 setas e o zoom está baixo.

## 5. Decisões

### Tomadas por você

- React Flow no lugar do Cytoscape, com Vite gerando um bundle único.
- Nó com nome, versão, data e chip de tipo; manter a seleção e as cores de afetados.
- Seleção de repos e de tipos de relação no drawer da esquerda.
- Quem mantém o trama pode ter Node; quem usa instala o mínimo.

### Tomadas por mim (revise)

- O tipo do repo vem de uma heurística em Go (`internal/view/role.go`), lida da `stack` e do que a ficha fornece e consome, e não de um campo novo: app (stack mobile), web (react, next, vue…), infra (terraform, helm…), api (fornece http), lib (só pacote) e worker (eventos). Sem sinal, o nó fica sem chip. Pode errar; se incomodar, o passo seguinte é um campo `kind` opcional na ficha.
- A direção do layout é automática (4).
- Há um filtro por tipo de repo no drawer da esquerda, além do pedido.
- A atribuição "React Flow" do canto foi mantida.

## 6. Critérios de aceite

1. `trama graph --serve` no `trama-demo` abre a página com os dois repos em cartões, cada um com o chip do tipo, e a seta `GET /hello`; clicar na API marca a web como direta, com a evidência e o link.
2. Num workspace com 14 repos o grafo abre em camadas, sem nós sobrepostos, e os repos sem relação ficam numa grade depois do grafo.
3. O painel de rede mostra só `index`, `app.css`, `app.js` e `/api/workspace`, e o console não mostra violação de CSP.
4. `go test ./...` e, em `web/`, `npm run typecheck`, `npm test` e `npm run build` passam; o build não altera os arquivos commitados.
5. Uma ficha com `<script>alert(1)</script>` na descrição mostra o texto e não executa nada.
