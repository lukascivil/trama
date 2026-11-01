const params = new URLSearchParams(location.search);
const portuguese = (params.get("lang") || navigator.language || "en").toLowerCase().startsWith("pt");

type Strings = {
  title: string;
  org: string;
  orgPlaceholder: string;
  list: string;
  filter: string;
  showArchived: string;
  hideArchived: string;
  markMatching: (n: number) => string;
  unmarkMatching: string;
  selected: (n: number) => string;
  add: (n: number) => string;
  adding: string;
  cancel: string;
  loading: string;
  anonymous: string;
  empty: string;
  noMatch: string;
  loadError: (m: string) => string;
  addError: (m: string) => string;
  added: (n: number) => string;
  refused: string;
  closeTab: string;
  cancelled: string;
  back: string;
  pickOrg: string;
};

const en: Strings = {
  title: "Add repositories",
  org: "Organization",
  orgPlaceholder: "organization name",
  list: "List",
  filter: "Filter by name or description",
  showArchived: "Show archived",
  hideArchived: "Hide archived",
  markMatching: (n) => `Select the ${n} shown`,
  unmarkMatching: "Unselect the ones shown",
  selected: (n) => `${n} selected`,
  add: (n) => `Add ${n}`,
  adding: "Adding…",
  cancel: "Cancel",
  loading: "Loading repositories…",
  anonymous: "No GitHub token was found, so private repositories are not listed. Log in with gh, or set GH_TOKEN.",
  empty: "This organization has no repositories you can see.",
  noMatch: "No repository matches the filter.",
  loadError: (m) => `Could not list the repositories: ${m}`,
  addError: (m) => `Could not add: ${m}`,
  added: (n) => (n === 1 ? "1 repository added" : `${n} repositories added`),
  refused: "Not added",
  closeTab: "You can close this tab. Claude continues in the terminal.",
  cancelled: "Cancelled. Nothing was added. You can close this tab.",
  back: "Back to the list",
  pickOrg: "Choose an organization to list its repositories.",
};

const pt: Strings = {
  title: "Adicionar repositórios",
  org: "Organização",
  orgPlaceholder: "nome da organização",
  list: "Listar",
  filter: "Filtrar por nome ou descrição",
  showArchived: "Mostrar arquivados",
  hideArchived: "Esconder arquivados",
  markMatching: (n) => `Marcar os ${n} que casam`,
  unmarkMatching: "Desmarcar os que casam",
  selected: (n) => (n === 1 ? "1 marcado" : `${n} marcados`),
  add: (n) => `Adicionar ${n}`,
  adding: "Adicionando…",
  cancel: "Cancelar",
  loading: "Carregando os repositórios…",
  anonymous: "Nenhum token do GitHub foi encontrado, então os repositórios privados não aparecem. Entre com o gh ou defina GH_TOKEN.",
  empty: "Esta organização não tem repositórios que você possa ver.",
  noMatch: "Nenhum repositório casa com o filtro.",
  loadError: (m) => `Não deu para listar os repositórios: ${m}`,
  addError: (m) => `Não deu para adicionar: ${m}`,
  added: (n) => (n === 1 ? "1 repositório adicionado" : `${n} repositórios adicionados`),
  refused: "Não adicionados",
  closeTab: "Pode fechar esta aba. O Claude continua no terminal.",
  cancelled: "Cancelado. Nada foi adicionado. Pode fechar esta aba.",
  back: "Voltar para a lista",
  pickOrg: "Escolha uma organização para listar os repositórios dela.",
};

export const lang = portuguese ? "pt" : "en";
export const T: Strings = portuguese ? pt : en;
