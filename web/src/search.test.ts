import { describe, expect, it } from "vitest";
import { sampleModel } from "../dev/sample";
import { search } from "./search";

const model = sampleModel();

describe("search", () => {
  it("clears on an empty query", () => {
    expect(search(model, "  ").sel).toEqual({ kind: "none" });
  });

  it("finds an id with its provider and users, ignoring case", () => {
    const { sel } = search(model, "get /hello");
    expect(sel.kind).toBe("uses");
    if (sel.kind !== "uses") return;
    expect(sel.providers).toEqual(["trama-demo-api"]);
    expect(sel.users.map((i) => model.edges[i].consumer)).toEqual(["trama-demo-web"]);
  });

  it("selects a repository by part of its name and reveals it", () => {
    const r = search(model, "orchestrator");
    expect(r.sel).toEqual({ kind: "repo", name: "reg-blocking-orchestrator", data: undefined });
    expect(r.reveal).toBe("reg-blocking-orchestrator");
  });

  it("finds a field or a column when no repository matches", () => {
    const { sel } = search(model, "email");
    expect(sel.kind).toBe("data");
    if (sel.kind !== "data") return;
    expect(sel.hits.map((d) => d.layer).sort()).toEqual(["request", "response", "table"]);
  });

  it("says when nothing matches", () => {
    expect(search(model, "zzz").sel).toEqual({ kind: "nomatch", term: "zzz" });
  });
});
