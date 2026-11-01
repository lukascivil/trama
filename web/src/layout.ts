import dagre from "@dagrejs/dagre";

export const NODE_W = 220;
export const NODE_H = 70;
export const UNKNOWN_SIZE = 40;

export type Size = { w: number; h: number };
export type Point = { x: number; y: number };
export type Direction = "LR" | "TB";

export const sizeOf = (id: string): Size => (id.startsWith("u:") ? { w: UNKNOWN_SIZE, h: UNKNOWN_SIZE } : { w: NODE_W, h: NODE_H });

const GAP = 24;

export type Layout = { at: Map<string, Point>; width: number; height: number };

// layout places the nodes in ranks: whoever consumes comes before whoever
// provides (left to right, or top to bottom), so a release flows against
// the arrows toward the people it may affect. Nodes without any arrow go in
// a grid after the graph instead of one tall column at its edge. Positions
// are the top-left corner of each node.
export function layout(nodes: string[], arrows: { source: string; target: string }[], dir: Direction): Layout {
  const at = new Map<string, Point>();
  const linked = new Set(arrows.flatMap((a) => [a.source, a.target]));
  const connected = nodes.filter((id) => linked.has(id));
  const alone = nodes.filter((id) => !linked.has(id));

  let width = 0;
  let height = 0;
  if (connected.length) {
    const g = new dagre.graphlib.Graph();
    g.setGraph({
      rankdir: dir,
      nodesep: dir === "LR" ? 24 : 28,
      ranksep: dir === "LR" ? 96 : 72,
      edgesep: 16,
      marginx: 0,
      marginy: 0,
      ranker: "network-simplex",
    });
    g.setDefaultEdgeLabel(() => ({}));
    connected.forEach((id) => {
      const { w, h } = sizeOf(id);
      g.setNode(id, { width: w, height: h });
    });
    const have = new Set(connected);
    arrows.forEach((a) => {
      if (have.has(a.source) && have.has(a.target)) g.setEdge(a.source, a.target);
    });
    dagre.layout(g);
    connected.forEach((id) => {
      const n = g.node(id);
      const { w, h } = sizeOf(id);
      at.set(id, { x: n.x - w / 2, y: n.y - h / 2 });
      width = Math.max(width, n.x + w / 2);
      height = Math.max(height, n.y + h / 2);
    });
  }

  if (alone.length) {
    // Rows as wide as the graph, or a square-ish block when there is none.
    const columns = Math.max(1, connected.length ? Math.floor((width + GAP) / (NODE_W + GAP)) : Math.ceil(Math.sqrt(alone.length)));
    const top = connected.length ? height + 70 : 0;
    alone.forEach((id, i) => {
      at.set(id, { x: (i % columns) * (NODE_W + GAP), y: top + Math.floor(i / columns) * (NODE_H + GAP) });
    });
    const rows = Math.ceil(alone.length / columns);
    width = Math.max(width, Math.min(alone.length, columns) * (NODE_W + GAP) - GAP);
    height = top + rows * (NODE_H + GAP) - GAP;
  }
  return { at, width, height };
}

// fitScale is how large the layout can be drawn in a canvas of this size.
export function fitScale(l: Layout, canvas: Size): number {
  if (l.width === 0 || l.height === 0) return 1;
  return Math.min(canvas.w / (l.width + 80), canvas.h / (l.height + 80));
}

// bestDirection picks the direction that lets the graph be drawn largest in
// this canvas: a wide canvas takes a few long rows, a tall one a few columns.
// Before the canvas has a size, it is left to right.
export function bestDirection(lr: Layout, tb: Layout, canvas: Size): Direction {
  if (canvas.w <= 0 || canvas.h <= 0) return "LR";
  return fitScale(tb, canvas) > fitScale(lr, canvas) * 1.05 ? "TB" : "LR";
}
