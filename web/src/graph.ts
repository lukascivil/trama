import type { Edge, Model } from "./types";

// Node ids: a repository is "r:<name>", and every consumer with unproven
// relations gets its own "?" satellite, "u:<name>", so one unproven relation
// does not drag every repository into a single knot in the middle.
export const repoID = (name: string) => `r:${name}`;
export const unknownID = (consumer: string) => `u:${consumer}`;
export const isUnknown = (id: string) => id.startsWith("u:");

export const trusted = (e: Edge) => e.status === "proven" || e.status === "declared";

export const KINDS = ["http", "event", "package", "declared", "unverified", "pending"] as const;

// edgeGroup is the relation filter an edge belongs to.
export function edgeGroup(e: Edge): string {
  if (e.status === "pending" || e.status === "unverified") return e.status;
  return e.kind || "declared";
}

export const roleOf = (role?: string) => role || "other";

// A Bundle is one arrow: a repository that calls seven routes of another is
// one dependency, not seven. idxs point into model.edges.
export type Bundle = { id: string; source: string; target: string; group: string; idxs: number[] };

export type Bundles = { bundles: Bundle[]; unknowns: string[] };

export function buildBundles(model: Model): Bundles {
  const names = new Set(model.repos.map((r) => r.name));
  const byKey = new Map<string, Bundle>();
  const unknowns = new Set<string>();
  model.edges.forEach((e, i) => {
    if (!names.has(e.consumer)) return;
    const known = !!e.provider && names.has(e.provider);
    const target = known ? repoID(e.provider as string) : unknownID(e.consumer);
    if (!known) unknowns.add(e.consumer);
    const group = edgeGroup(e);
    const key = `${e.consumer}|${target}|${group}`;
    const bundle = byKey.get(key);
    if (bundle) {
      bundle.idxs.push(i);
      return;
    }
    byKey.set(key, { id: "", source: repoID(e.consumer), target, group, idxs: [i] });
  });
  const bundles = [...byKey.values()];
  bundles.forEach((b, n) => (b.id = `e:${n}`));
  return { bundles, unknowns: [...unknowns] };
}

// neighborhood is the node itself and everything within `levels` arrows of
// it, in either direction, hidden or not.
export function neighborhood(bundles: Bundle[], start: string, levels: number): Set<string> {
  const kept = new Set([start]);
  let frontier = [start];
  for (let i = 0; i < levels; i++) {
    const next: string[] = [];
    for (const b of bundles) {
      for (const [a, z] of [[b.source, b.target], [b.target, b.source]] as const) {
        if (frontier.includes(a) && !kept.has(z)) {
          kept.add(z);
          next.push(z);
        }
      }
    }
    frontier = next;
  }
  return kept;
}

export type Filters = {
  hiddenRepos: ReadonlySet<string>;
  hiddenKinds: ReadonlySet<string>;
  hiddenRoles: ReadonlySet<string>;
  hideIsolated: boolean;
  onlyChanges: boolean;
  focus: ReadonlySet<string> | null;
};

export const noFilters: Filters = {
  hiddenRepos: new Set(),
  hiddenKinds: new Set(),
  hiddenRoles: new Set(),
  hideIsolated: false,
  onlyChanges: false,
  focus: null,
};

export type Visible = { nodes: string[]; edges: Bundle[] };

// visibleGraph decides what the filters leave on the canvas. An arrow shows
// when both ends do and its relation kind is on; a "?" shows only with an
// arrow; a repository with no visible arrow can be hidden on its own.
export function visibleGraph(model: Model, { bundles, unknowns }: Bundles, f: Filters): Visible {
  const keep = (id: string) => !f.focus || f.focus.has(id);
  const shown = new Set(
    model.repos
      .filter(
        (r) =>
          !f.hiddenRepos.has(r.name) &&
          !f.hiddenRoles.has(roleOf(r.role)) &&
          keep(repoID(r.name)) &&
          (!f.onlyChanges || (r.openChanges ?? []).length > 0),
      )
      .map((r) => repoID(r.name)),
  );
  const edges = bundles.filter((b) => {
    const ends = shown.has(b.source) && (isUnknown(b.target) ? keep(b.target) : shown.has(b.target));
    return ends && !f.hiddenKinds.has(b.group);
  });
  const linked = new Set(edges.flatMap((b) => [b.source, b.target]));
  const nodes = [
    ...model.repos.map((r) => repoID(r.name)).filter((id) => shown.has(id) && (!f.hideIsolated || linked.has(id))),
    ...unknowns.map(unknownID).filter((id) => linked.has(id)),
  ];
  return { nodes, edges };
}
