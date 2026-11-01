const plural = (n: number, one: string, many: string) => (n === 1 ? one : many);

type Strings = {
  relations: string;
  types: string;
  view: string;
  repos: string;
  hideIsolated: string;
  onlyChanges: string;
  search: string;
  hint: string;
  counts: (repos: number, relations: number, notProven: number) => string;
  warnings: (n: number) => string;
  loadError: (m: string) => string;
  loading: string;
  today: string;
  daysAgo: (n: number) => string;
  noRelease: string;
  notCloned: string;
  commitsAfter: (n: number) => string;
  kind: Record<string, string>;
  role: Record<string, string>;
  card: Record<string, string>;
  port: (p: number) => string;
  inResponse: string;
  inRequest: string;
  inTable: string;
  direct: string;
  indirect: string;
  possible: string;
  nobody: string;
  via: (path: string[]) => string;
  reason: Record<string, string>;
  focus: string;
  levels: (n: number) => string;
  showAll: string;
  status: Record<string, string>;
  from: string;
  proof: string;
  candidates: string;
  problems: string;
  evidence: string;
  provider: string;
  consumers: string;
  noMatch: (q: string) => string;
  legend: { selected: string; direct: string; indirect: string; possible: string };
  bundle: (n: number, kind: string) => string;
  together: (n: number, kind: string) => string;
  data: string;
  dataHits: (n: number) => string;
  tabs: { affected: string; provides: string; consumes: string; tables: string; run: string; changes: string };
  filter: string;
  nothingHere: string;
  noneMatch: string;
  all: string;
  none: string;
  only: string;
  onlyTitle: (name: string) => string;
  select: string;
  fitView: string;
  affectedTitle: string;
  openChanges: string;
};

const en: Strings = {
  relations: "Relations",
  types: "Type",
  view: "View",
  repos: "Repositories",
  hideIsolated: "Hide repos without relations",
  onlyChanges: "Only repos with open changes",
  search: "Search a repo, an id or a field (email…)",
  hint: "Click a repository to see who may be affected if you release something there. Click an arrow to see its proof.",
  counts: (r, e, p) => `${r} ${plural(r, "repo", "repos")} · ${e} ${plural(e, "relation", "relations")} · ${p} not proven`,
  warnings: (n) => `${n} ${plural(n, "warning", "warnings")}`,
  loadError: (m) => `Could not load the workspace: ${m}`,
  loading: "Loading the workspace…",
  today: "today",
  daysAgo: (n) => `${n} ${plural(n, "day", "days")} ago`,
  noRelease: "no release",
  notCloned: "not cloned",
  commitsAfter: (n) => `${n} ${plural(n, "commit", "commits")} after it`,
  kind: { http: "HTTP", event: "Events", package: "Packages", declared: "Declared", pending: "Not proven (?)", unverified: "Not verified" },
  role: { api: "api", web: "web", app: "app", worker: "worker", lib: "lib", infra: "infra", other: "other" },
  card: { stale: "card out of date: run /trama:map", missing: "no card yet: run /trama:map", unknown: "card not checked: repository not cloned" },
  port: (p) => `port ${p}`,
  inResponse: "returns",
  inRequest: "accepts",
  inTable: "stores",
  direct: "Direct",
  indirect: "Indirect",
  possible: "Possible",
  nobody: "Nobody, as far as the cards show.",
  via: (path) => `may be affected, via ${path.join(" → ")}`,
  reason: { unknown: "relation without proof (from: unknown)", unverified: "relation not verified: see the warnings" },
  focus: "Focus",
  levels: (n) => `${n} ${plural(n, "level", "levels")}`,
  showAll: "Show all",
  status: { proven: "proven", declared: "declared in workspace.yaml", unverified: "not verified", pending: "not proven yet" },
  from: "from",
  proof: "Proof",
  candidates: "Candidates",
  problems: "Problems",
  evidence: "Evidence",
  provider: "Provider",
  consumers: "Consumers",
  noMatch: (q) => `No repository, id or field matches “${q}”.`,
  legend: { selected: "selected", direct: "direct", indirect: "indirect", possible: "possible" },
  bundle: (n, kind) => `${n} × ${kind}`,
  together: (n, kind) => `${n} ${kind} ${plural(n, "relation", "relations")}`,
  data: "Data",
  dataHits: (n) => `${n} ${plural(n, "place", "places")}`,
  tabs: { affected: "Affected", provides: "Provides", consumes: "Consumes", tables: "Tables", run: "Run", changes: "Changes" },
  filter: "Filter…",
  nothingHere: "Nothing here.",
  noneMatch: "Nothing matches.",
  all: "All",
  none: "None",
  only: "only",
  onlyTitle: (name) => `Show only ${name}`,
  select: "Select this repository",
  fitView: "Fit to screen",
  affectedTitle: "If you release something here, these may be affected",
  openChanges: "Open changes",
};

const pt: Strings = {
  relations: "Relações",
  types: "Tipo",
  view: "Exibir",
  repos: "Repositórios",
  hideIsolated: "Esconder repos sem relação",
  onlyChanges: "Só repos com mudança aberta",
  search: "Buscar repo, id ou campo (email…)",
  hint: "Clique num repositório para ver quem pode ser afetado se você publicar algo ali. Clique numa seta para ver a prova.",
  counts: (r, e, p) => `${r} ${plural(r, "repo", "repos")} · ${e} ${plural(e, "relação", "relações")} · ${p} sem prova`,
  warnings: (n) => `${n} ${plural(n, "aviso", "avisos")}`,
  loadError: (m) => `Não foi possível carregar o workspace: ${m}`,
  loading: "Carregando o workspace…",
  today: "hoje",
  daysAgo: (n) => `há ${n} ${plural(n, "dia", "dias")}`,
  noRelease: "sem release",
  notCloned: "não clonado",
  commitsAfter: (n) => `${n} ${plural(n, "commit depois", "commits depois")}`,
  kind: { http: "HTTP", event: "Eventos", package: "Pacotes", declared: "Declaradas", pending: "Sem prova (?)", unverified: "Não verificadas" },
  role: { api: "api", web: "web", app: "app", worker: "worker", lib: "lib", infra: "infra", other: "outros" },
  card: { stale: "ficha desatualizada: rode /trama:map", missing: "sem ficha ainda: rode /trama:map", unknown: "ficha não conferida: repositório não clonado" },
  port: (p) => `porta ${p}`,
  inResponse: "devolve",
  inRequest: "recebe",
  inTable: "guarda",
  direct: "Diretos",
  indirect: "Indiretos",
  possible: "Possíveis",
  nobody: "Ninguém, pelo que as fichas mostram.",
  via: (path) => `pode ser afetado, via ${path.join(" → ")}`,
  reason: { unknown: "relação sem prova (from: unknown)", unverified: "relação não verificada: veja os avisos" },
  focus: "Focar",
  levels: (n) => `${n} ${plural(n, "nível", "níveis")}`,
  showAll: "Mostrar tudo",
  status: { proven: "provada", declared: "declarada no workspace.yaml", unverified: "não verificada", pending: "ainda sem prova" },
  from: "de",
  proof: "Prova",
  candidates: "Candidatos",
  problems: "Problemas",
  evidence: "Evidência",
  provider: "Fornecedor",
  consumers: "Consumidores",
  noMatch: (q) => `Nenhum repositório, id ou campo corresponde a “${q}”.`,
  legend: { selected: "selecionado", direct: "direto", indirect: "indireto", possible: "possível" },
  bundle: (n, kind) => `${n} × ${kind}`,
  together: (n, kind) => `${n} ${plural(n, "relação", "relações")} ${kind}`,
  data: "Dado",
  dataHits: (n) => `${n} ${plural(n, "lugar", "lugares")}`,
  tabs: { affected: "Afetados", provides: "Fornece", consumes: "Consome", tables: "Tabelas", run: "Rodar", changes: "Mudanças" },
  filter: "Filtrar…",
  nothingHere: "Nada por aqui.",
  noneMatch: "Nada corresponde.",
  all: "Todos",
  none: "Nenhum",
  only: "só",
  onlyTitle: (name) => `Mostrar só ${name}`,
  select: "Selecionar este repositório",
  fitView: "Ajustar à tela",
  affectedTitle: "Se você publicar algo aqui, podem ser afetados",
  openChanges: "Mudanças abertas",
};

const params = new URLSearchParams(location.search);
export const lang: "en" | "pt" = (params.get("lang") || navigator.language || "en").toLowerCase().startsWith("pt") ? "pt" : "en";
export const T: Strings = lang === "pt" ? pt : en;
document.documentElement.lang = lang === "pt" ? "pt-BR" : "en";

export { plural };
