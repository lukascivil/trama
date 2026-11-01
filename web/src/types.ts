// The model that GET /api/workspace returns. Every relation rule lives in
// Go (internal/view): the page only draws what these types describe.

export type Link = { text: string; url?: string };

export type EdgeStatus = "proven" | "declared" | "unverified" | "pending";

export type Edge = {
  consumer: string;
  provider?: string;
  kind?: string;
  id: string;
  via?: string;
  status: EdgeStatus;
  candidates?: string[];
  evidence?: Link[];
  problems?: string[];
};

export type Hit = {
  repo: string;
  kind?: string;
  id?: string;
  path?: string[];
  reason?: string;
  evidence?: Link[];
};

export type Affected = { direct: Hit[]; indirect: Hit[]; possible: Hit[] };

export type Release = { version: string; date: string; commitsAfter: number };

export type Provide = { kind: string; id: string; evidence: Link[] };

export type Command = { kind: string; cmd: string; port?: number };

export type Change = { id: string; title: string; url?: string };

export type Repo = {
  name: string;
  description?: string;
  stack?: string[];
  role?: string;
  url?: string;
  cloned: boolean;
  card: string;
  release: Release | null;
  run?: Command[];
  provides?: Provide[];
  openChanges?: Change[];
  affected: Affected;
};

export type DataEntry = {
  layer: "response" | "request" | "table";
  repo: string;
  id?: string;
  table?: string;
  path: string;
  type?: string;
  match: string;
  evidence: Link[];
};

export type Model = {
  workspace: string;
  repos: Repo[];
  edges: Edge[];
  data?: DataEntry[];
  warnings: string[];
};
