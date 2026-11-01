import { describe, expect, it } from "vitest";
import { sampleModel } from "../dev/sample";
import { buildBundles, neighborhood, noFilters, repoID, unknownID, visibleGraph } from "./graph";

const model = sampleModel();
const built = buildBundles(model);

describe("buildBundles", () => {
  it("makes one arrow per pair of repositories and kind", () => {
    const keys = built.bundles.map((b) => `${b.source}|${b.target}|${b.group}`);
    expect(new Set(keys).size).toBe(keys.length);
    expect(built.bundles.reduce((n, b) => n + b.idxs.length, 0)).toBe(model.edges.length);
  });

  it("gives every consumer with unproven relations its own ? node", () => {
    expect(built.unknowns.sort()).toEqual(["idp-authentication", "one-stone-migration-service"]);
    const pending = built.bundles.filter((b) => b.group === "pending");
    expect(pending.map((b) => b.target).sort()).toEqual([unknownID("idp-authentication"), unknownID("one-stone-migration-service")]);
  });

  it("keeps an unverified relation apart from the proven ones between the same repos", () => {
    const between = built.bundles.filter((b) => b.source === repoID("legal-entity-affiliation") && b.target === repoID("idp-resource-server"));
    expect(between.map((b) => b.group)).toEqual(["unverified"]);
  });
});

describe("visibleGraph", () => {
  it("shows everything with no filter", () => {
    const v = visibleGraph(model, built, noFilters);
    expect(v.nodes).toHaveLength(model.repos.length + built.unknowns.length);
    expect(v.edges).toHaveLength(built.bundles.length);
  });

  it("drops the arrows of a hidden kind, and the ? nodes that lose their only arrow", () => {
    const v = visibleGraph(model, built, { ...noFilters, hiddenKinds: new Set(["pending"]) });
    expect(v.edges.some((b) => b.group === "pending")).toBe(false);
    expect(v.nodes.some((id) => id.startsWith("u:"))).toBe(false);
  });

  it("drops the arrows of a hidden repository", () => {
    const v = visibleGraph(model, built, { ...noFilters, hiddenRepos: new Set(["idp-resource-server"]) });
    expect(v.nodes).not.toContain(repoID("idp-resource-server"));
    expect(v.edges.every((b) => b.source !== repoID("idp-resource-server") && b.target !== repoID("idp-resource-server"))).toBe(true);
  });

  it("hides repositories without relations only when asked", () => {
    expect(visibleGraph(model, built, noFilters).nodes).toContain(repoID("grafana-rules"));
    expect(visibleGraph(model, built, { ...noFilters, hideIsolated: true }).nodes).not.toContain(repoID("grafana-rules"));
  });

  it("keeps only repositories with open changes", () => {
    const v = visibleGraph(model, built, { ...noFilters, onlyChanges: true });
    expect(v.nodes.filter((id) => id.startsWith("r:")).sort()).toEqual([repoID("legal-entity-affiliation"), repoID("reg-blocking-orchestrator")]);
  });

  it("hides a whole type of repository", () => {
    const v = visibleGraph(model, built, { ...noFilters, hiddenRoles: new Set(["web"]) });
    const webs = model.repos.filter((r) => r.role === "web").map((r) => repoID(r.name));
    expect(v.nodes.some((id) => webs.includes(id))).toBe(false);
  });

  it("follows the focus", () => {
    const focus = neighborhood(built.bundles, repoID("trama-demo-web"), 1);
    const v = visibleGraph(model, built, { ...noFilters, focus });
    expect(v.nodes.sort()).toEqual([repoID("trama-demo-api"), repoID("trama-demo-web")]);
    expect(v.edges).toHaveLength(1);
  });
});

describe("neighborhood", () => {
  it("grows one level of arrows at a time, in both directions", () => {
    const one = neighborhood(built.bundles, repoID("risk-profiler"), 1);
    expect(one.has(repoID("legal-entity-affiliation"))).toBe(true); // consumes it
    expect(one.has(repoID("idp-authorizer"))).toBe(true); // it consumes this one
    expect(one.has(repoID("idp-resource-server"))).toBe(false);
    expect(neighborhood(built.bundles, repoID("risk-profiler"), 2).has(repoID("idp-resource-server"))).toBe(true);
  });
});
