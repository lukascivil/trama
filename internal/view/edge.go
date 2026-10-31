// Package view builds what the graph page shows and serves it on the
// loopback interface.
package view

// EdgeStatus says how far a relation is proven.
type EdgeStatus string

const (
	Proven     EdgeStatus = "proven"     // from a repo, validation found no error in this consume
	Declared   EdgeStatus = "declared"   // a workspace.yaml link that no card cites
	Unverified EdgeStatus = "unverified" // from a repo, but validation found errors
	Pending    EdgeStatus = "pending"    // from: unknown
)

// Link is a piece of evidence, with a link to the line when the host is GitHub.
type Link struct {
	Text string `json:"text"`          // "src/api/hello.ts:15" or "workspace.yaml"
	URL  string `json:"url,omitempty"` // https link to the line, when the host is GitHub
}

// Edge is one consume of the catalog, or a declared link, from the
// consumer to the provider.
type Edge struct {
	Consumer   string     `json:"consumer"`
	Provider   string     `json:"provider,omitempty"` // "" when Pending
	Kind       string     `json:"kind,omitempty"`     // "" for a Declared link
	ID         string     `json:"id"`
	Via        string     `json:"via,omitempty"`
	Status     EdgeStatus `json:"status"`
	Candidates []string   `json:"candidates,omitempty"`
	Evidence   []Link     `json:"evidence,omitempty"`
	Problems   []string   `json:"problems,omitempty"` // why it is Unverified
}

// Hit is one repository that a release of the selected one may affect.
type Hit struct {
	Repo     string   `json:"repo"`
	Kind     string   `json:"kind,omitempty"`
	ID       string   `json:"id,omitempty"`
	Path     []string `json:"path,omitempty"`   // Indirect: the repos between Repo and the selected one, nearest to Repo first
	Reason   string   `json:"reason,omitempty"` // Possible: "unknown" or "unverified"
	Evidence []Link   `json:"evidence,omitempty"`
}

// Affected groups the hits by how sure the relation is.
type Affected struct {
	Direct   []Hit `json:"direct"`
	Indirect []Hit `json:"indirect"`
	Possible []Hit `json:"possible"`
}
