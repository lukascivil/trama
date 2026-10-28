package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/workspace"
)

// loadWorkspace finds and loads the workspace that contains --dir, or the
// current directory when --dir is not set.
func loadWorkspace(cmd *cobra.Command) (*workspace.Workspace, error) {
	start := dirFlag(cmd)
	if start == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		start = wd
	}
	root, err := workspace.Find(start)
	if err != nil {
		return nil, err
	}
	return workspace.Load(root)
}
