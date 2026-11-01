# trama — Descoberta de dado — Design

## Problema

O catálogo responde por **id** (rota, tópico, pacote), não por **dado**. Um dev que precisa do e-mail do usuário não tem como perguntar ao workspace quem tem esse dado: ou alguém já declarou a relação, ou o catálogo é cego. As duas perguntas reais do dia a dia são:

1. *"Esse dado já vem de algum endpoint que existe?"* — hoje precisaria dos campos do response. Existe `provides[].shape`, mas é `map[string]any` sem validação nenhuma e na prática fica vazio.
2. *"E se não vem, quem é dono do dado?"* — não existe nada. É a tabela de quem é dono.

Request body não existe em lugar nenhum, e é o que mais quebra quando muda.

## Decisões

**Tabela é índice de descoberta, não relação.** Serviço não lê banco de outro serviço neste contexto, então tabela não vira `kind`, não vira `via`, não mexe no `linker`, no `trama uses` nem no grafo. Entra em `identity`, que é exatamente "como os outros me reconhecem".

**Tabela sem coluna não serve.** `users` sozinho responde "usuário está no B", que o dev já sabia. O nível de campo é o que responde a pergunta, então `columns` é obrigatório.

**Campo é caminho pontilhado, nunca YAML aninhado.** `"customer.email"` se compara com uma comparação de string e é o mesmo formato do `fields_used`, que já é lista plana. Aninhado obriga a andar na árvore para responder qualquer pergunta.

**Vocabulário de tipos fechado e pequeno.** `string`, `number`, `boolean`, `object` e as versões em lista. Não modela required, não modela enum, não modela formato: isso é trabalho do spec, e aqui só faria a ficha mentir.

**`contract` continua ponteiro.** Quem quer o detalhe completo abre o OpenAPI; quem quer buscar usa a ficha. A ficha é o índice, não a verdade.

**`shape` vira `response`.** Duas formas de dizer a mesma coisa é pior que renomear num pré-release. `trama validate` recusa `shape:` nomeando o rename, para a ficha antiga não perder o campo em silêncio.

**A ordem da resposta é o valor.** Busca primeiro no response dos provides (achou: chama o endpoint que já existe), depois no request, depois nas tabelas (o dado existe, o dono é o B, e a conversa com o time do B passa a ser "me expõe isso" em vez de "vocês têm isso?").

## Forma

```yaml
identity:
  tables:
    - value: users
      columns: [id, email, name, created_at]
      evidence: ["migrations/001_users.sql:3-8"]
provides:
  - kind: http
    id: "POST /orders"
    request:  { user_id: string, "items[].sku": string }
    response: { id: string, "customer.email": string }
    evidence: ["internal/api/orders.go:42"]
```

- `request` só em `kind: http`. `response` em `http` e em `event` (aí é o payload publicado). Nenhum dos dois em `package`.
- `[]` marca segmento que é lista de objetos pela qual se desce (`items[].sku`). Lista de escalares é o tipo (`[]string`), nunca `[]` no último segmento.
- Evidência continua por entrada, não por campo: uma linha por `provides`, uma por tabela.
- Nome de tabela aceita `schema.tabela`.

## Comando

`trama data <termo>` busca substring, sem distinguir maiúsculas, em caminho de campo e nome de coluna:

```
$ trama data email
email
  response  api  GET /orders/{id}  customer.email: string  internal/api/orders.go:42
  request   api  POST /orders  email: string  internal/api/orders.go:60
  table     api  users.email  migrations/001_users.sql:3-8
not mapped: web
```

## O que não entra

- Ligar coluna a campo de endpoint automaticamente. É fluxo de dados, não casamento de id; não tem evidência `file:line` honesta e quebraria a regra que sustenta o resto do catálogo.
- Tabela no grafo, no `trama uses`, no `trama link`.
- Impacto no nível de campo (`/trama:impact` cruzando `fields_used` com `response`). O dado para isso passa a existir depois desta fase; a análise fica para outra.
- Ler OpenAPI, GraphQL ou proto para preencher os campos. Quem mapeia lê e cita a linha, como no resto do catálogo.
