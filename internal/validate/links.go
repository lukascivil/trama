package validate

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/lukascivil/trama/internal/catalog"
	"github.com/lukascivil/trama/internal/workspace"
)

var vias = []string{"env", "package", "topic", "declared"}

// checkLinks applies the rules between cards and returns how many consumes
// have a proven provider.
func checkLinks(ws *workspace.Workspace, l loaded, fs *[]Finding) int {
	proven := 0
	for _, r := range ws.Repos {
		c := l.cards[r.Name]
		if c == nil {
			continue
		}
		rep := report{file: cardFile(r.Name), out: fs}
		for i, cf := range c.Config {
			if cf.PointsTo == "" {
				continue
			}
			if _, ok := ws.Repo(cf.PointsTo); !ok {
				rep.errorf(fmt.Sprintf("config[%d] %s", i, cf.Env), "points_to %q is not a repo of the workspace", cf.PointsTo)
			}
		}
		for i, cs := range c.Consumes {
			before := Errors(*fs)
			where := fmt.Sprintf("consumes[%d] %s", i, cs.ID)
			if checkConsume(ws, l, r.Name, c, cs, rep, where) && Errors(*fs) == before {
				proven++
			}
		}
	}
	return proven
}

// checkConsume reports whether cs names a provider (from is a repo) and
// adds a finding for every rule it breaks.
func checkConsume(ws *workspace.Workspace, l loaded, name string, c *catalog.Card, cs catalog.Consume, rep report, where string) bool {
	switch {
	case cs.From == "":
		rep.errorf(where, "from is empty (use a repository name or unknown)")
		return false
	case cs.From == catalog.Unknown:
		if cs.Via != "" {
			rep.errorf(where, "via must be empty when from is unknown")
		}
		for _, cand := range cs.Candidates {
			if _, ok := ws.Repo(cand); !ok {
				rep.errorf(where, "candidate %q is not a repo of the workspace", cand)
			}
		}
		if len(cs.Candidates) == 0 {
			rep.warnf(where, "no proven provider (no candidates)")
		} else {
			rep.warnf(where, "no proven provider (candidates: %s)", strings.Join(cs.Candidates, ", "))
		}
		return false
	}

	from := cs.From
	if _, ok := ws.Repo(from); !ok {
		rep.errorf(where, "from %q is not a repo of the workspace", from)
		return false
	}
	if from == name {
		rep.errorf(where, "a repository cannot consume from itself")
		return false
	}
	if l.broken[from] {
		return false // the provider's own card error is already reported
	}
	pc := l.cards[from]
	if pc == nil {
		rep.errorf(where, "%s is not mapped", from)
		return false
	}
	if !slices.ContainsFunc(pc.Provides, func(p catalog.Provide) bool { return p.Kind == cs.Kind && p.ID == cs.ID }) {
		rep.errorf(where, "%s does not provide %s", from, cs.ID)
	}

	switch cs.Via {
	case "":
		rep.errorf(where, "from needs proof in via")
	case "env":
		if !envProof(c, from, pc) {
			rep.errorf(where, "via env: no config entry points to %s with a value matching its identity", from)
		}
	case "package":
		if cs.Kind != "package" || pc.Identity.Package == nil || pc.Identity.Package.Value != cs.ID {
			rep.errorf(where, "via package: needs kind package and an id equal to %s's identity.package", from)
		}
	case "topic":
		if cs.Kind != "event" || !slices.ContainsFunc(pc.Identity.Topics, func(f catalog.Fact) bool { return f.Value == cs.ID }) {
			rep.errorf(where, "via topic: needs kind event and %s listing %s in identity.topics", from, cs.ID)
		}
	case "declared":
		link := workspace.Link{Consumer: name, Provider: from, ID: cs.ID}
		if !slices.Contains(ws.Links, link) {
			rep.errorf(where, "via declared: workspace.yaml has no links entry for %s -> %s %s", name, from, cs.ID)
		}
	default:
		rep.errorf(where, "unknown via %q (use %s)", cs.Via, strings.Join(vias, ", "))
	}
	return true
}

// envProof reports whether a config entry of c points to provider with a
// value whose host or port matches the provider's identity.
func envProof(c *catalog.Card, provider string, pc *catalog.Card) bool {
	for _, cf := range c.Config {
		if cf.PointsTo == provider && valueMatches(cf.Value, pc.Identity) {
			return true
		}
	}
	return false
}

func valueMatches(value string, id catalog.Identity) bool {
	v := strings.TrimSpace(value)
	if v == "" {
		return false
	}
	if !strings.Contains(v, "://") {
		v = "http://" + v
	}
	u, err := url.Parse(v)
	if err != nil {
		return false
	}
	host := u.Hostname()
	for _, h := range id.Hosts {
		if strings.EqualFold(h.Value, host) {
			return true
		}
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || !slices.Contains([]string{"localhost", "127.0.0.1", "0.0.0.0", ""}, host) {
		return false
	}
	return slices.ContainsFunc(id.Ports, func(p catalog.PortFact) bool { return p.Value == port })
}
