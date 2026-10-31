package view

import (
	"github.com/lukascivil/trama/internal/validate"
	"github.com/lukascivil/trama/internal/workspace"
)

// CountChecks makes validation count its runs until the test ends.
func CountChecks(cleanup func(func()), n *int) {
	old := check
	check = func(ws *workspace.Workspace) validate.Report {
		*n++
		return old(ws)
	}
	cleanup(func() { check = old })
}
