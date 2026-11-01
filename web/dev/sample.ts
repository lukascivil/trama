import type { Affected, DataEntry, Edge, Hit, Link, Model, Provide, Repo } from "../src/types";

// A made-up workspace for working on the page without a real one, and for
// the tests. It mirrors what internal/view builds in Go, including the
// affected lists, which AffectedBy computes there.

const ev = (repo: string, path: string, line: number): Link => ({
  text: `${path}:${line}`,
  url: `https://github.com/acme/${repo}/blob/0a1b2c3/${path}#L${line}`,
});

type R = [name: string, role: string, stack: string[], days: number | null, extra?: Partial<Repo>];

const REPOS: R[] = [
  ["trama-demo-api", "api", ["node", "express"], 3],
  ["trama-demo-web", "web", ["node", "react", "vite"], 5],
  ["grafana-rules", "infra", ["yaml"], null],
  ["reg-blocking-orchestrator", "worker", ["node", "typescript", "kafka"], 1, { openChanges: [{ id: "2026-10-31-block-rules", title: "Block rules per affiliation" }] }],
  ["resource-affiliation-dispatcher", "worker", ["node", "typescript", "kafka"], 1],
  ["questions-answers-dispatcher", "worker", ["node", "typescript", "kafka"], 1],
  ["idp-authorizer", "api", ["go", "postgres"], 1],
  ["infra-confluent-kafka-topics", "infra", ["terraform"], 20],
  ["idp-authentication", "api", ["go", "postgres"], 2],
  ["stone-admin-platform", "web", ["node", "typescript", "next", "react"], 4],
  ["legal-entity-affiliation", "api", ["java", "spring", "postgres"], 8, { openChanges: [{ id: "2026-11-01-affiliation-v2", title: "Affiliation v2 payload" }] }],
  ["idp-resource-server", "api", ["go", "postgres"], 1],
  ["risk-profiler", "api", ["python", "fastapi"], 12],
  ["one-stone-migration-service", "web", ["node", "typescript", "next", "react", "prisma", "postgres", "kafka"], null],
];

type E = [consumer: string, provider: string | null, kind: string, id: string, status: Edge["status"], extra?: Partial<Edge>];

const EDGES: E[] = [
  ["trama-demo-web", "trama-demo-api", "http", "GET /hello", "proven"],
  ["stone-admin-platform", "idp-authentication", "http", "POST /auth/login", "proven"],
  ["stone-admin-platform", "legal-entity-affiliation", "http", "GET /affiliations/{id}", "proven"],
  ["stone-admin-platform", "reg-blocking-orchestrator", "http", "POST /blocks", "proven"],
  ["one-stone-migration-service", "reg-blocking-orchestrator", "http", "GET /resources/v1/questionnaires", "proven"],
  ["one-stone-migration-service", "idp-resource-server", "http", "GET /resources/v1/accounts", "proven"],
  ["one-stone-migration-service", "risk-profiler", "http", "POST /risk/score", "proven"],
  ["one-stone-migration-service", null, "http", "GET /legacy/stone", "pending", { candidates: ["idp-resource-server", "risk-profiler"] }],
  ["reg-blocking-orchestrator", "questions-answers-dispatcher", "event", "qa.requested", "proven"],
  ["reg-blocking-orchestrator", "resource-affiliation-dispatcher", "event", "affiliation.requested", "proven"],
  ["reg-blocking-orchestrator", "idp-resource-server", "http", "GET /resources/v1/stone-accounts", "proven"],
  ["reg-blocking-orchestrator", "infra-confluent-kafka-topics", "", "qa.requested", "declared"],
  ["resource-affiliation-dispatcher", "idp-resource-server", "http", "GET /resources/v1/stone-accounts/references/from", "proven"],
  ["resource-affiliation-dispatcher", "legal-entity-affiliation", "http", "POST /affiliations", "proven"],
  ["questions-answers-dispatcher", "idp-authorizer", "http", "POST /authorize", "proven"],
  ["idp-authentication", "idp-authorizer", "http", "POST /authorize", "proven"],
  ["idp-authentication", "idp-resource-server", "http", "GET /resources/v1/users", "proven"],
  ["idp-authentication", null, "http", "POST /notify", "pending", { candidates: ["questions-answers-dispatcher"] }],
  ["idp-authorizer", "idp-resource-server", "http", "GET /resources/v1/policies", "proven"],
  ["legal-entity-affiliation", "risk-profiler", "http", "POST /risk/score", "proven"],
  ["legal-entity-affiliation", "idp-resource-server", "http", "GET /resources/v1/missing", "unverified", { problems: ["evidence src/missing.ts:999 does not exist"] }],
  ["risk-profiler", "idp-authorizer", "package", "@acme/authz-client", "proven"],
];

export function affectedBy(repo: string, edges: Edge[]): Affected {
  const a: Affected = { direct: [], indirect: [], possible: [] };
  const sure = (e: Edge) => e.status === "proven" || e.status === "declared";
  for (const e of edges) {
    if (e.consumer === repo) continue;
    const hit: Hit = { repo: e.consumer, kind: e.kind, id: e.id, evidence: e.evidence };
    if (sure(e) && e.provider === repo) a.direct.push(hit);
    else if (e.status === "unverified" && e.provider === repo) a.possible.push({ ...hit, reason: "unverified" });
    else if (e.status === "pending" && (e.candidates ?? []).includes(repo)) a.possible.push({ ...hit, reason: "unknown" });
  }
  const parent = new Map<string, string>([[repo, ""]]);
  const depth = new Map<string, number>([[repo, 0]]);
  const queue = [repo];
  const found: string[] = [];
  while (queue.length) {
    const u = queue.shift() as string;
    for (const e of edges) {
      if (!sure(e) || e.provider !== u || parent.has(e.consumer)) continue;
      parent.set(e.consumer, u);
      depth.set(e.consumer, (depth.get(u) ?? 0) + 1);
      queue.push(e.consumer);
      found.push(e.consumer);
    }
  }
  for (const name of found) {
    if ((depth.get(name) ?? 0) < 2) continue;
    const path: string[] = [];
    for (let p = parent.get(name) as string; p !== repo; p = parent.get(p) as string) path.push(p);
    a.indirect.push({ repo: name, path });
  }
  return a;
}

export function sampleModel(now = Date.now()): Model {
  const edges: Edge[] = EDGES.map(([consumer, provider, kind, id, status, extra]) => ({
    consumer,
    ...(provider ? { provider } : {}),
    ...(kind ? { kind } : {}),
    id,
    status,
    evidence: status === "declared" ? [{ text: "workspace.yaml" }] : [ev(consumer, `src/clients/${id.replace(/\W+/g, "-").slice(0, 24)}.ts`, 12)],
    ...extra,
  }));

  const provides = new Map<string, Provide[]>();
  for (const e of edges) {
    if (!e.provider || e.status === "declared" || !e.kind) continue;
    const list = provides.get(e.provider) ?? [];
    if (!list.some((p) => p.id === e.id)) list.push({ kind: e.kind, id: e.id, evidence: [ev(e.provider, "src/routes.ts", 20 + list.length * 7)] });
    provides.set(e.provider, list);
  }

  const repos: Repo[] = REPOS.map(([name, role, stack, days, extra]) => ({
    name,
    description: `${name} (made-up repository for the sample workspace).`,
    stack,
    role,
    url: `https://github.com/acme/${name}`,
    cloned: name !== "grafana-rules",
    card: name === "risk-profiler" ? "stale" : "current",
    release: days === null ? null : { version: `v${1 + (name.length % 3)}.${name.length % 9}.0`, date: new Date(now - days * 86400000).toISOString(), commitsAfter: name.length % 4 },
    run: [
      { kind: "setup", cmd: "pnpm install" },
      { kind: "dev", cmd: "pnpm dev", port: 3000 + (name.length % 50) },
      { kind: "test", cmd: "pnpm test" },
    ],
    provides: provides.get(name),
    affected: affectedBy(name, edges),
    ...extra,
  }));

  const data: DataEntry[] = [
    { layer: "response", repo: "idp-resource-server", id: "GET /resources/v1/users", path: "email", type: "string", match: "email", evidence: [ev("idp-resource-server", "src/users.go", 40)] },
    { layer: "response", repo: "idp-resource-server", id: "GET /resources/v1/users", path: "name", type: "string", match: "name", evidence: [ev("idp-resource-server", "src/users.go", 41)] },
    { layer: "request", repo: "idp-authentication", id: "POST /auth/login", path: "email", type: "string", match: "email", evidence: [ev("idp-authentication", "src/login.go", 12)] },
    { layer: "table", repo: "idp-resource-server", table: "users", path: "email", type: "text", match: "users email", evidence: [ev("idp-resource-server", "db/schema.sql", 8)] },
    { layer: "table", repo: "idp-resource-server", table: "users", path: "id", type: "uuid", match: "users id", evidence: [ev("idp-resource-server", "db/schema.sql", 7)] },
    { layer: "table", repo: "idp-authorizer", table: "policies", path: "name", type: "text", match: "policies name", evidence: [ev("idp-authorizer", "db/schema.sql", 3)] },
  ];

  return {
    workspace: "trama-sample",
    repos,
    edges,
    data,
    warnings: ["catalog/risk-profiler.yaml: mapped_at_commit is behind origin/main", "legal-entity-affiliation: consumes[3] evidence src/missing.ts:999 does not exist"],
  };
}

// bigModel is a workspace of n repositories for checking the page at scale.
// Repositories sit in six tiers and each one consumes from one to three
// repositories of the tiers below it, as services do; a few have unproven
// relations, and every tenth has none at all.
export function bigModel(n: number, now = Date.now()): Model {
  const roles = ["api", "api", "web", "worker", "lib", "infra"];
  const name = (i: number) => `service-${String(i).padStart(3, "0")}`;
  const tier = Math.ceil(n / 6);
  const edges: Edge[] = [];
  for (let i = tier; i < n; i++) {
    if (i % 10 === 0) continue;
    const below = Math.floor(i / tier) * tier; // everything before this tier
    const links = 1 + (i % 3);
    for (let k = 1; k <= links; k++) {
      const p = (i * 7 + k * 13) % below;
      if (p % 10 === 0) continue;
      const id = `GET /v1/${name(p)}/${k}`;
      edges.push({ consumer: name(i), provider: name(p), kind: "http", id, status: "proven", evidence: [ev(name(i), "src/client.ts", k)] });
    }
    if (i % 17 === 0) edges.push({ consumer: name(i), kind: "http", id: "GET /legacy", status: "pending", candidates: [name(i - 1)] });
  }
  const repos: Repo[] = Array.from({ length: n }, (_, i) => ({
    name: name(i),
    stack: ["node"],
    role: roles[i % roles.length],
    cloned: true,
    card: "current",
    release: { version: `v1.${i % 20}.0`, date: new Date(now - (i % 40) * 86400000).toISOString(), commitsAfter: i % 5 },
    affected: affectedBy(name(i), edges),
  }));
  return { workspace: `big-${n}`, repos, edges, warnings: [] };
}
