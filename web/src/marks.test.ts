import { describe, expect, it } from "vitest";
import { sampleModel } from "../dev/sample";
import { buildBundles, repoID, unknownID } from "./graph";
import { computeMarks, isFaded } from "./marks";

const model = sampleModel();
const { bundles } = buildBundles(model);
const marks = (sel: Parameters<typeof computeMarks>[2]) => computeMarks(model, bundles, sel);

describe("computeMarks", () => {
  it("marks nothing without a selection", () => {
    const m = marks({ kind: "none" });
    expect(m.active).toBe(false);
    expect(isFaded(m, repoID("trama-demo-api"))).toBe(false);
  });

  it("marks the selected repo, its direct consumers and the arrows between", () => {
    const m = marks({ kind: "repo", name: "trama-demo-api" });
    expect(m.nodes.get(repoID("trama-demo-api"))).toBe("selected");
    expect(m.nodes.get(repoID("trama-demo-web"))).toBe("direct");
    const arrow = bundles.find((b) => b.source === repoID("trama-demo-web") && b.target === repoID("trama-demo-api"));
    expect(m.edges.get(arrow?.id ?? "")).toBe("direct");
    expect(isFaded(m, repoID("grafana-rules"))).toBe(true);
    expect(isFaded(m, repoID("trama-demo-web"))).toBe(false);
  });

  it("marks the repos that reach the selected one in more steps as indirect, along the path", () => {
    const m = marks({ kind: "repo", name: "idp-authorizer" });
    // stone-admin-platform -> idp-authentication -> idp-authorizer
    expect(m.nodes.get(repoID("stone-admin-platform"))).toBe("indirect");
    const hop = bundles.find((b) => b.source === repoID("stone-admin-platform") && b.target === repoID("idp-authentication"));
    expect(m.edges.get(hop?.id ?? "")).toBe("indirect");
  });

  it("does not turn a direct or indirect repo into a possible one", () => {
    const m = marks({ kind: "repo", name: "idp-resource-server" });
    // legal-entity-affiliation has an unverified relation to it, but also reaches it through proven ones
    expect(model.repos.find((r) => r.name === "idp-resource-server")?.affected.possible.map((h) => h.repo)).toContain("legal-entity-affiliation");
    expect(m.nodes.get(repoID("legal-entity-affiliation"))).toBe("indirect");
  });

  it("marks a consumer of an unproven relation as possible and keeps its ? node", () => {
    const m = marks({ kind: "repo", name: "questions-answers-dispatcher" });
    expect(m.nodes.get(repoID("idp-authentication"))).toBe("possible");
    expect(isFaded(m, unknownID("idp-authentication"))).toBe(false);
    const dashed = bundles.find((b) => b.source === repoID("idp-authentication") && b.group === "pending");
    expect(m.edges.get(dashed?.id ?? "")).toBe("possible");
  });

  it("keeps both ends of a selected arrow", () => {
    const b = bundles[0];
    const m = marks({ kind: "bundle", id: b.id });
    expect(m.edges.get(b.id)).toBe("edge");
    expect(isFaded(m, b.source)).toBe(false);
    expect(isFaded(m, b.target)).toBe(false);
    expect(isFaded(m, repoID("grafana-rules"))).toBe(true);
  });

  it("marks the provider and every user of an id", () => {
    const users = model.edges.flatMap((e, i) => (e.id === "POST /authorize" ? [i] : []));
    const m = marks({ kind: "uses", id: "POST /authorize", providers: ["idp-authorizer"], users });
    expect(m.nodes.get(repoID("idp-authorizer"))).toBe("selected");
    expect(m.nodes.get(repoID("idp-authentication"))).toBe("direct");
    expect(m.nodes.get(repoID("questions-answers-dispatcher"))).toBe("direct");
  });

  it("marks where a field lives", () => {
    const hits = (model.data ?? []).filter((d) => d.path === "email");
    const m = marks({ kind: "data", term: "email", hits });
    expect([...m.nodes.keys()].sort()).toEqual([repoID("idp-authentication"), repoID("idp-resource-server")]);
  });
});
