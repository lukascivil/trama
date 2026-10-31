package changes

import "testing"

func TestSummary(t *testing.T) {
	plan := func() *Record {
		return &Record{
			ID: "2026-11-03-rename-message", Title: "Rename message", Status: "open",
			Steps: []Step{
				{Order: 1, Repo: "api", Action: "Add greeting", State: "pending"},
				{Order: 2, Repo: "web", Action: "Read greeting", State: "pending"},
			},
		}
	}
	// Two consumers migrate in parallel, so they share an order.
	parallel := func() *Record {
		r := plan()
		r.Steps = append(r.Steps, Step{Order: 2, Repo: "mobile", Action: "Read greeting", State: "pending"})
		return r
	}
	open := func(r *Record, i int) *Record {
		r.Steps[i].Branch = "trama/rename-message"
		r.Steps[i].PR = "https://github.com/o/r/pull/1"
		r.Steps[i].State = "open"
		return r
	}
	tests := []struct {
		name     string
		old, cur *Record
		want     string
	}{
		{"new record", nil, plan(), `plan "Rename message"`},
		{"one step opened", plan(), open(plan(), 0), "step 1 open (api)"},
		{"two steps opened", plan(), open(open(plan(), 0), 1), "step 1 open (api), step 2 open (web)"},
		{"step merged", open(plan(), 0), func() *Record { r := open(plan(), 0); r.Steps[0].State = "merged"; return r }(), "step 1 merged (api)"},
		{"step added", plan(), func() *Record {
			r := plan()
			r.Steps = append(r.Steps, Step{Order: 3, Repo: "api", Action: "Remove message", State: "pending"})
			return r
		}(), "add step 3 (api)"},
		{"step removed", plan(), func() *Record { r := plan(); r.Steps = r.Steps[:1]; return r }(), "remove step 2 (web)"},
		{"step edited", plan(), func() *Record { r := plan(); r.Steps[1].Action = "Read greeting only"; return r }(), "update step 2 (web)"},
		{"text only", plan(), plan(), "update record"},
		{"quotes in title", nil, func() *Record { r := plan(); r.Title = `Rename "message"`; return r }(), `plan "Rename "message""`},
		{"shared order unchanged", parallel(), parallel(), "update record"},
		{"shared order opened", parallel(), open(parallel(), 1), "step 2 open (web)"},
		{"shared order removed", parallel(), func() *Record { r := parallel(); r.Steps = r.Steps[:2]; return r }(), "remove step 2 (mobile)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Summary(tt.old, tt.cur); got != tt.want {
				t.Fatalf("Summary = %q, want %q", got, tt.want)
			}
		})
	}
}
