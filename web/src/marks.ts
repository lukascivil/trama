import { repoID, trusted, unknownID, type Bundle } from "./graph";
import type { DataEntry, Model } from "./types";

// What the panel shows, and so what the canvas marks.
export type Selection =
  | { kind: "none" }
  | { kind: "repo"; name: string; data?: DataEntry[] }
  | { kind: "bundle"; id: string }
  | { kind: "uses"; id: string; providers: string[]; users: number[] }
  | { kind: "data"; term: string; hits: DataEntry[] }
  | { kind: "nomatch"; term: string };

export type NodeMark = "selected" | "direct" | "indirect" | "possible";
export type EdgeMark = "direct" | "indirect" | "possible" | "edge";

// Marks say how every element is drawn for a selection: marked ones in
// their color, the other kept ones normally, and the rest faded. A mark that
// is "stronger" replaces a weaker one on the same element.
export type Marks = {
  active: boolean;
  kept: ReadonlySet<string>;
  nodes: ReadonlyMap<string, NodeMark>;
  edges: ReadonlyMap<string, EdgeMark>;
};

const NODE_RANK: Record<NodeMark, number> = { selected: 0, direct: 1, indirect: 2, possible: 3 };
const EDGE_RANK: Record<EdgeMark, number> = { direct: 1, indirect: 2, possible: 3, edge: 4 };

export const noMarks: Marks = { active: false, kept: new Set(), nodes: new Map(), edges: new Map() };

class Painter {
  kept = new Set<string>();
  nodes = new Map<string, NodeMark>();
  edges = new Map<string, EdgeMark>();

  keep(id: string) {
    this.kept.add(id);
  }
  node(id: string, mark: NodeMark) {
    this.kept.add(id);
    const old = this.nodes.get(id);
    if (!old || NODE_RANK[mark] > NODE_RANK[old]) this.nodes.set(id, mark);
  }
  edge(id: string, mark: EdgeMark) {
    this.kept.add(id);
    const old = this.edges.get(id);
    if (!old || EDGE_RANK[mark] > EDGE_RANK[old]) this.edges.set(id, mark);
  }
  done(): Marks {
    return { active: true, kept: this.kept, nodes: this.nodes, edges: this.edges };
  }
}

// computeMarks is the whole of what clicking or searching highlights.
export function computeMarks(model: Model, bundles: Bundle[], sel: Selection): Marks {
  const p = new Painter();
  // edgesWhere lists the arrows that stand for at least one relation that matches.
  const edgesWhere = (pred: (i: number) => boolean) => bundles.filter((b) => b.idxs.some(pred));

  switch (sel.kind) {
    case "none":
    case "nomatch":
      return noMarks;

    case "repo": {
      const repo = model.repos.find((r) => r.name === sel.name);
      if (!repo) return noMarks;
      const name = repo.name;
      p.node(repoID(name), "selected");
      const a = repo.affected;
      for (const h of a.direct) {
        p.node(repoID(h.repo), "direct");
        for (const b of edgesWhere((i) => {
          const e = model.edges[i];
          return trusted(e) && e.consumer === h.repo && e.provider === name && e.id === h.id;
        })) p.edge(b.id, "direct");
      }
      for (const h of a.indirect) {
        p.node(repoID(h.repo), "indirect");
        const chain = [h.repo, ...(h.path ?? []), name];
        for (let i = 0; i + 1 < chain.length; i++) {
          for (const b of edgesWhere((k) => {
            const e = model.edges[k];
            return trusted(e) && e.consumer === chain[i] && e.provider === chain[i + 1];
          })) p.edge(b.id, "indirect");
        }
      }
      for (const h of a.possible) {
        // A repository that already depends on this one for sure stays direct or indirect.
        const sure = p.nodes.get(repoID(h.repo));
        if (sure !== "direct" && sure !== "indirect") p.node(repoID(h.repo), "possible");
        for (const b of edgesWhere((i) => {
          const e = model.edges[i];
          return e.consumer === h.repo && e.id === h.id && (e.provider === name || (e.candidates ?? []).includes(name));
        })) p.edge(b.id, "possible");
        p.keep(unknownID(h.repo));
      }
      return p.done();
    }

    case "bundle": {
      const b = bundles.find((x) => x.id === sel.id);
      if (!b) return noMarks;
      p.edge(b.id, "edge");
      p.keep(b.source);
      p.keep(b.target);
      return p.done();
    }

    case "uses": {
      sel.providers.forEach((n) => p.node(repoID(n), "selected"));
      for (const i of sel.users) {
        const e = model.edges[i];
        p.node(repoID(e.consumer), trusted(e) ? "direct" : "possible");
        if (e.provider) p.node(repoID(e.provider), "selected");
        for (const b of edgesWhere((k) => k === i)) {
          p.edge(b.id, trusted(e) ? "direct" : "possible");
          p.keep(b.source);
          p.keep(b.target);
        }
      }
      return p.done();
    }

    case "data": {
      new Set(sel.hits.map((d) => d.repo)).forEach((n) => p.node(repoID(n), "selected"));
      return p.done();
    }
  }
}

// Whether an element of the canvas is faded under these marks.
export const isFaded = (m: Marks, id: string) => m.active && !m.kept.has(id);
