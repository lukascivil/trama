package view

import "slices"

// AffectedBy lists who may be affected when repo releases something:
// direct consumers through proven or declared relations, repos that reach
// repo through them in two or more steps, and consumers whose relation to
// repo is not proven.
func AffectedBy(repo string, edges []Edge) Affected {
	a := Affected{Direct: []Hit{}, Indirect: []Hit{}, Possible: []Hit{}}
	trusted := func(e Edge) bool { return e.Status == Proven || e.Status == Declared }
	for _, e := range edges {
		if e.Consumer == repo {
			continue
		}
		switch {
		case trusted(e) && e.Provider == repo:
			a.Direct = append(a.Direct, Hit{Repo: e.Consumer, Kind: e.Kind, ID: e.ID, Evidence: e.Evidence})
		case e.Status == Unverified && e.Provider == repo:
			a.Possible = append(a.Possible, Hit{Repo: e.Consumer, Kind: e.Kind, ID: e.ID, Reason: "unverified", Evidence: e.Evidence})
		case e.Status == Pending && slices.Contains(e.Candidates, repo):
			a.Possible = append(a.Possible, Hit{Repo: e.Consumer, Kind: e.Kind, ID: e.ID, Reason: "unknown", Evidence: e.Evidence})
		}
	}

	// Breadth-first search from repo against the direction of the trusted
	// edges: the first parent found gives the shortest path.
	parent := map[string]string{repo: ""}
	depth := map[string]int{repo: 0}
	queue := []string{repo}
	var found []string
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		for _, e := range edges {
			if !trusted(e) || e.Provider != u {
				continue
			}
			if _, seen := parent[e.Consumer]; seen {
				continue
			}
			parent[e.Consumer], depth[e.Consumer] = u, depth[u]+1
			queue = append(queue, e.Consumer)
			found = append(found, e.Consumer)
		}
	}
	for _, name := range found {
		if depth[name] < 2 {
			continue
		}
		var path []string
		for p := parent[name]; p != repo; p = parent[p] {
			path = append(path, p)
		}
		a.Indirect = append(a.Indirect, Hit{Repo: name, Path: path})
	}
	return a
}
