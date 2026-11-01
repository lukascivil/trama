import type { Selection } from "./marks";
import type { Model } from "./types";

export type SearchResult = { sel: Selection; reveal?: string };

// search answers what was typed in the top box: an id (a route, a topic, a
// package) shows its provider and everyone who uses it; a repository name
// selects it, with the fields that also match; a field or column shows where
// it lives.
export function search(model: Model, q: string): SearchResult {
  const query = q.trim();
  if (!query) return { sel: { kind: "none" } };
  const lower = query.toLowerCase();

  const users = model.edges.flatMap((e, i) => (e.id.toLowerCase() === lower ? [i] : []));
  const providers = model.repos.filter((r) => (r.provides ?? []).some((p) => p.id.toLowerCase() === lower)).map((r) => r.name);
  if (users.length || providers.length) {
    const named = users.map((i) => model.edges[i].provider).filter((n): n is string => !!n);
    return {
      sel: {
        kind: "uses",
        id: users.length ? model.edges[users[0]].id : query,
        providers: [...new Set([...providers, ...named])],
        users,
      },
    };
  }

  const hits = (model.data ?? []).filter((d) => d.match.toLowerCase().includes(lower));
  const match = model.repos.find((r) => r.name.toLowerCase().includes(lower));
  if (match) {
    // The repo wins the panel, but the data it does not mention is still an
    // answer to what was typed: show it instead of dropping it.
    return { sel: { kind: "repo", name: match.name, data: hits.length ? hits : undefined }, reveal: match.name };
  }
  if (hits.length) return { sel: { kind: "data", term: query, hits } };
  return { sel: { kind: "nomatch", term: query } };
}
