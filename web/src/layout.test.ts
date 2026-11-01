import { describe, expect, it } from "vitest";
import { sampleModel } from "../dev/sample";
import { buildBundles, noFilters, visibleGraph } from "./graph";
import { bestDirection, layout, sizeOf, type Direction } from "./layout";

const model = sampleModel();
const v = visibleGraph(model, buildBundles(model), noFilters);

describe.each<Direction>(["LR", "TB"])("layout %s", (dir) => {
  const { at } = layout(v.nodes, v.edges, dir);
  const axis = dir === "LR" ? "x" : "y";

  it("places every node", () => {
    expect([...at.keys()].sort()).toEqual([...v.nodes].sort());
  });

  it("puts a consumer before its provider when nothing loops back", () => {
    const web = at.get("r:trama-demo-web");
    const api = at.get("r:trama-demo-api");
    expect(web && api && web[axis] < api[axis]).toBe(true);
  });

  it("never overlaps two nodes", () => {
    const boxes = v.nodes.map((id) => ({ id, ...(at.get(id) as { x: number; y: number }), ...sizeOf(id) }));
    for (const a of boxes) {
      for (const b of boxes) {
        if (a.id >= b.id) continue;
        const apart = a.x + a.w <= b.x || b.x + b.w <= a.x || a.y + a.h <= b.y || b.y + b.h <= a.y;
        expect(apart, `${a.id} overlaps ${b.id}`).toBe(true);
      }
    }
  });

  it("puts nodes without arrows after the graph, not in its first rank", () => {
    const lone = at.get("r:grafana-rules");
    const last = Math.max(...v.nodes.filter((id) => id !== "r:grafana-rules").map((id) => (at.get(id) as { y: number }).y));
    expect(lone && lone.y > last).toBe(true);
  });
});

describe("layout", () => {
  it("lays out a graph with no arrows at all", () => {
    const { at } = layout(["r:a", "r:b", "r:c"], [], "LR");
    expect(at.size).toBe(3);
  });

  it("reports the size it takes", () => {
    const l = layout(v.nodes, v.edges, "LR");
    expect(l.width).toBeGreaterThan(0);
    expect(l.height).toBeGreaterThan(0);
  });
});

describe("bestDirection", () => {
  const lr = layout(v.nodes, v.edges, "LR");
  const tb = layout(v.nodes, v.edges, "TB");

  it("is left to right until the canvas has a size", () => {
    expect(bestDirection(lr, tb, { w: 0, h: 0 })).toBe("LR");
  });

  it("takes the direction that draws the graph larger", () => {
    const wide = bestDirection(lr, tb, { w: 3000, h: 500 });
    const tall = bestDirection(lr, tb, { w: 500, h: 3000 });
    expect(wide).toBe("LR");
    expect(tall).toBe("TB");
  });
});

describe("layout at scale", () => {
  it("lays out 100 repositories without overlap, quickly", async () => {
    const { bigModel } = await import("../dev/sample");
    const big = bigModel(100);
    const g = visibleGraph(big, buildBundles(big), noFilters);
    const started = performance.now();
    for (const dir of ["LR", "TB"] as const) {
      const { at } = layout(g.nodes, g.edges, dir);
      expect(at.size).toBe(g.nodes.length);
      const boxes = g.nodes.map((id) => ({ ...(at.get(id) as { x: number; y: number }), ...sizeOf(id) }));
      boxes.forEach((a, i) =>
        boxes.slice(i + 1).forEach((b) => {
          const apart = a.x + a.w <= b.x || b.x + b.w <= a.x || a.y + a.h <= b.y || b.y + b.h <= a.y;
          if (!apart) throw new Error(`overlap in ${dir}`);
        }),
      );
    }
    expect(performance.now() - started).toBeLessThan(2000);
  });
});
