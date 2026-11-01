# trama — Varredura de schema — Design

## Problema

Num workspace real, `/trama:map --all` mapeou `tables` em 3 de 10 repositórios. A busca em Go não tem culpa: o Go só valida e pesquisa o que já está na ficha. Quem mapeia é o agente, seguindo `skills/map/SKILL.md`, e a instrução tem uma assimetria:

- A seção "Read the repository" manda **caçar** o router e o OpenAPI, com nome de arquivo e tudo.
- Migration e schema não aparecem ali. A orientação sobre tabela está só na seção de **escrever** a ficha: diz o que citar quando o arquivo já está na mão, nunca manda ir atrás dele.

O agente só mapeia tabela se tropeçar no schema procurando outra coisa. E quando não mapeia, ninguém fica sabendo: a ficha sem `tables` é idêntica para um repositório sem banco e para um que o mapeador pulou.

## Decisões

**Achar arquivo é mecânico; ler arquivo é julgamento.** O agente não é determinístico e não dá para torná-lo. O que dá é encolher o que ele decide. Uma varredura em Go lista, sempre igual, os arquivos que declaram tabela. O agente continua lendo e citando a linha — isso é dele, porque é ele que entende o arquivo —, mas deixa de poder não achar.

**Uma varredura, dois consumidores.** `trama schemas <repo>` entrega a lista ao agente. `trama validate` usa a mesma lista para avisar quando o clone tem schema e a ficha não tem `tables`. A omissão silenciosa vira uma pergunta, que aparece também nos avisos da página do grafo.

**Aviso, não erro.** Um `.sql` com `CREATE TABLE` pode ser seed, fixture ou o schema de outro serviço. Quem decide é quem leu. Por isso o mapeador tem uma saída declarada: `tables: []` quer dizer "olhei e este repositório não é dono de tabela nenhuma", e cala o aviso. Chave ausente continua querendo dizer "não olhei". O `stamp` e o `link` editam a árvore de nós do YAML, então o `[]` escrito à mão sobrevive aos dois.

**A regra de onde ler sobe para o `workspace`.** "Ler de `origin/<branch>` quando o clone tem, do disco senão" hoje mora dentro do `validate`. O comando novo precisa da mesma regra, então ela vira `ws.Files(repo)`, e o `validate` passa a usá-la em vez de ter a sua. Uma regra, um lugar.

## O que a varredura acha

Um arquivo entra em exatamente um tipo, nesta precedência:

| Tipo | Regra |
|---|---|
| `schema` | nome `schema.prisma`, `schema.rb`, `structure.sql` ou `schema.sql`, ou extensão `.prisma` — o estado atual do banco num arquivo só |
| `migration` | algum diretório do caminho se chama `migrations`, `migration`, `migrate`, `alembic`, `flyway` ou `liquibase` |
| `sql` | extensão `.sql` com `CREATE TABLE` no conteúdo, sem distinguir maiúsculas |

Ficam de fora: diretórios `node_modules`, `vendor`, `dist`, `build`, `target`, `testdata`, `fixtures`, `test`, `tests`, `spec`, `__tests__`; arquivos que começam com ponto; `__init__.py`; `.md` e `.txt`.

A saída vem por tipo (schema, migration, sql) e, dentro de cada tipo, em ordem de caminho. O schema vem primeiro de propósito: quando existe, é a verdade atual, e o agente não precisa reconstruir a tabela a partir de quinhentas migrations.

No modo git a lista é `git ls-tree -r` em `origin/<branch>`, só arquivos regulares — a mesma regra do `ReadFile`, que já recusa symlink e submódulo.

## Forma

```
$ trama schemas api
api
  schema     prisma/schema.prisma
  migration  prisma/migrations/20240101_init/migration.sql
  sql        db/init.sql
```

```
warning: catalog/api.yaml: identity.tables: repos/api has 3 schema files (prisma/schema.prisma, …) but the card lists no tables: map them, or write "tables: []" if this repository owns none
```

## O que não entra

- Achar entidade de ORM pelo nome do arquivo. Não tem padrão confiável; a instrução continua mandando o agente procurar.
- Extrair tabela e coluna em Go. Isso seria parsear SQL, Prisma, Ruby e Python, e é exatamente o julgamento que fica com o agente.
- Aviso no `trama status`. O `validate` já chega na página do grafo e no fim do `/trama:map`.
