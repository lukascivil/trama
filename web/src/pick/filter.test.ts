import { describe, expect, it } from "vitest";
import { filterRepos, setMatching } from "./filter";
import type { Repo } from "./types";

const repo = (name: string, extra: Partial<Repo> = {}): Repo => ({
  id: `acme/${name}`,
  name,
  description: "",
  language: "",
  pushedAt: "",
  archived: false,
  blocked: "",
  ...extra,
});

const repos = [
  repo("payments-api", { description: "Charges and refunds" }),
  repo("Checkout-Web", { description: "The checkout page" }),
  repo("old-billing", { archived: true, description: "retired" }),
  repo("ledger", { blocked: "repo ledger already in workspace" }),
];

describe("filterRepos", () => {
  it("finds a repo by a piece of its name, ignoring case", () => {
    expect(filterRepos(repos, "CHECK", false).map((r) => r.name)).toEqual(["Checkout-Web"]);
  });

  it("finds a repo by its description", () => {
    expect(filterRepos(repos, "refunds", false).map((r) => r.name)).toEqual(["payments-api"]);
  });

  it("returns every repo that is not archived when the query is empty", () => {
    expect(filterRepos(repos, "  ", false).map((r) => r.name)).toEqual(["payments-api", "Checkout-Web", "ledger"]);
  });

  it("hides archived repos unless asked", () => {
    expect(filterRepos(repos, "billing", false)).toEqual([]);
    expect(filterRepos(repos, "billing", true).map((r) => r.name)).toEqual(["old-billing"]);
  });
});

describe("setMatching", () => {
  const matching = filterRepos(repos, "", true);

  it("marks the matching repos that can be added, never a blocked one", () => {
    const next = setMatching(new Set(), matching, true);
    expect([...next].sort()).toEqual(["acme/Checkout-Web", "acme/old-billing", "acme/payments-api"]);
  });

  it("leaves repos outside the matching list alone", () => {
    const next = setMatching(new Set(["acme/elsewhere"]), [repos[0]], true);
    expect([...next].sort()).toEqual(["acme/elsewhere", "acme/payments-api"]);
  });

  it("unmarks only the matching repos", () => {
    const start = new Set(["acme/payments-api", "acme/Checkout-Web"]);
    expect([...setMatching(start, [repos[0]], false)]).toEqual(["acme/Checkout-Web"]);
  });

  it("returns a new set and keeps the old one as it was", () => {
    const start = new Set(["acme/payments-api"]);
    const next = setMatching(start, [repos[1]], true);
    expect(next).not.toBe(start);
    expect([...start]).toEqual(["acme/payments-api"]);
  });
});
