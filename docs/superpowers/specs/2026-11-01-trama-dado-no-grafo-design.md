# trama — Dado no grafo — Design

## Problema

`trama data` responde quem tem um dado, mas a página do grafo não vê nada do que a ficha passou a guardar:

- `internal/view/model.go` expõe `Provide` como `{kind, id, evidence}`: `request` e `response` não atravessam para o JSON, e `Repo` não tem `tables`.
- `internal/view/assets/app.js` busca por igualdade exata de id (`p.id.toLowerCase() === lower`), então `email` não acha nada e nunca vai achar.

A ordem natural fica invertida: a primeira coisa que o dev faz é abrir o grafo e olhar, e é ali que a pergunta não tem resposta. O comando só serve a quem já sabe que ele existe.

## Decisões

**A regra continua em Go.** O spec do grafo diz que a página só desenha. A busca por dado é uma regra (o que é pesquisável, em que ordem responder), então ela não pode nascer de novo em JavaScript — e não há como testar JavaScript neste repositório.

**Uma implementação, dois consumidores.** A busca que o `trama data` faz hoje sai de `internal/cli/data.go` para `internal/datafind`. O comando passa a usar o pacote, e o modelo da página também. Duplicar o casamento de campo seria criar duas verdades que divergem no primeiro ajuste.

**O modelo carrega os resultados, não um endpoint novo.** A página já baixa o modelo inteiro uma vez e busca localmente, sem rede. Mantemos isso: `Model.Data` leva toda entrada pesquisável, já na ordem (response, request, tabela), cada uma com o texto que o Go decidiu ser pesquisável. O JavaScript faz `includes` nesse texto e nada mais — é o Go que define o que casa.

**A busca por dado é o último degrau, não o primeiro.** A ordem da caixa de busca passa a ser: id exato, nome de repo, dado. Pôr dado antes do nome do repo faria `api` listar colunas em vez de selecionar o repositório `api`, que é o que ela faz hoje.

**Tabela continua fora do grafo.** Não é nó nem aresta nem entra no cálculo de afetados. Aparece no painel de um repositório, que é onde `run` e `provides` já aparecem: é atributo do repositório, não relação.

## Forma

```go
// internal/datafind
type Hit struct {
    Layer    string // response | request | table
    Repo     string
    ID       string // a rota ou o tópico; vazio num hit de tabela
    Table    string // a tabela; vazio num hit de campo
    Path     string // o caminho do campo, ou o nome da coluna
    Type     string // o tipo do campo; vazio num hit de tabela
    Match    string // o texto que o termo casa
    Evidence []string
}

func All(repos []string, cards map[string]*catalog.Card) []Hit
func Find(repos []string, cards map[string]*catalog.Card, term string) []Hit
```

`All` devolve tudo na ordem útil; `Find` é `All` filtrado por `Match`, sem distinguir maiúsculas, e devolve nada para termo vazio.

`datafind` não conhece URL de repositório nem commit, então o `Hit` não é o que vai no JSON: o `internal/view` traduz cada `Hit` para um tipo seu, `DataEntry`, com a evidência como `[]Link` — do mesmo jeito que `provides`, `consumes` e `config` já aparecem na página, com linha clicável. `Model.Data` é `[]DataEntry`.

## O que não entra

- Endpoint novo (`/api/data`): o modelo já é o canal, e um fetch por tecla seria uma regressão no jeito como a página funciona.
- Tabela como nó ou aresta do grafo.
- Impacto no nível de campo.
- Tradução do JavaScript: a página já tem suas duas línguas em `app.js`; entram só as chaves novas.

## Limite conhecido

`Model.Data` leva toda coluna e todo campo do workspace. Num workspace grande são alguns milhares de entradas curtas no mesmo payload que já leva o modelo inteiro. Se isso incomodar, o próximo passo é um endpoint de busca, não cortar o índice.
