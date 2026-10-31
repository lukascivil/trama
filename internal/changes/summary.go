package changes

import (
	"fmt"
	"strings"
)

// Summary describes what changed from old (the committed record, nil when
// there is none) to cur, for a commit message: the plan for a new record,
// else the steps that changed, else "update record".
func Summary(old, cur *Record) string {
	if old == nil {
		return fmt.Sprintf(`plan "%s"`, cur.Title)
	}
	// Steps can share an order (consumers that migrate in parallel), so a
	// step is its order and its repository.
	type key struct {
		order int
		repo  string
	}
	before := map[key]Step{}
	for _, s := range old.Steps {
		before[key{s.Order, s.Repo}] = s
	}
	var parts []string
	for _, s := range cur.Steps {
		o, ok := before[key{s.Order, s.Repo}]
		delete(before, key{s.Order, s.Repo})
		switch {
		case !ok:
			parts = append(parts, fmt.Sprintf("add step %d (%s)", s.Order, s.Repo))
		case o.State != s.State:
			parts = append(parts, fmt.Sprintf("step %d %s (%s)", s.Order, s.State, s.Repo))
		case o != s:
			parts = append(parts, fmt.Sprintf("update step %d (%s)", s.Order, s.Repo))
		}
	}
	for _, s := range old.Steps {
		if _, gone := before[key{s.Order, s.Repo}]; gone {
			parts = append(parts, fmt.Sprintf("remove step %d (%s)", s.Order, s.Repo))
		}
	}
	if len(parts) == 0 {
		return "update record"
	}
	return strings.Join(parts, ", ")
}
