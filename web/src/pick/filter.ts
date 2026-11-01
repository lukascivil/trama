import type { Repo } from "./types";

// filterRepos keeps the repos whose name or description holds query, ignoring
// case. Archived repos stay out unless showArchived is set.
export function filterRepos(repos: Repo[], query: string, showArchived: boolean): Repo[] {
  const q = query.trim().toLowerCase();
  return repos.filter((r) => {
    if (r.archived && !showArchived) return false;
    return !q || r.name.toLowerCase().includes(q) || r.description.toLowerCase().includes(q);
  });
}

// setMatching marks or unmarks the repos of matching in a copy of selected.
// A blocked repo is never marked, and ids outside matching are left as they were.
export function setMatching(selected: Set<string>, matching: Repo[], on: boolean): Set<string> {
  const next = new Set(selected);
  for (const r of matching) {
    if (!on) next.delete(r.id);
    else if (!r.blocked) next.add(r.id);
  }
  return next;
}
