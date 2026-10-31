package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/changes"
	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/workspace"
)

// stateFunc reads a pull request's state; tests replace it.
var stateFunc changes.PRState = changes.GHState

// commitRecords commits the given change records to the workspace's main
// branch and, when push is set and there is an origin, pushes them.
func commitRecords(ws *workspace.Workspace, out io.Writer, msg string, paths []string, push bool) error {
	repo := gitx.Repo{Dir: ws.Root}
	main := repo.OriginHead()
	branch, err := repo.CurrentBranch()
	if err != nil {
		return err
	}
	if branch != main {
		return fmt.Errorf("workspace is on %s; change records go to %s", branch, main)
	}
	committed, err := repo.CommitPaths(msg, paths...)
	if err != nil {
		return err
	}
	if !committed {
		fmt.Fprintln(out, "nothing to commit")
	}
	if _, err := repo.OriginURL(); err != nil {
		fmt.Fprintln(out, "committed locally; the workspace has no origin remote")
		return nil
	}
	if !push {
		return nil
	}
	if err := repo.PushHead(main); err != nil {
		return fmt.Errorf("push to %s was rejected (protected branch?): %v", main, err)
	}
	return nil
}

func newChangesCmd() *cobra.Command {
	var noPush bool
	cmd := &cobra.Command{
		Use:   "changes",
		Short: "Update the state of every open change record from its pull requests, then commit and push the records",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			records, errs := changes.Load(ws.Root)
			for _, err := range errs {
				fmt.Fprintf(out, "warning: %v\n", err)
			}
			var changed []string
			for _, r := range records {
				if r.Status == "open" {
					updated, errs := changes.Refresh(r, stateFunc)
					for _, err := range errs {
						fmt.Fprintf(out, "warning: %v\n", err)
					}
					if updated {
						if err := r.Save(); err != nil {
							return err
						}
						changed = append(changed, filepath.Join("changes", r.ID+".md"))
					}
				}
				if r.Status == "done" {
					fmt.Fprintf(out, "%s: done\n", r.ID)
				} else {
					fmt.Fprintf(out, "%s: %d open step(s)\n", r.ID, len(r.OpenSteps()))
				}
			}
			if len(changed) == 0 {
				return nil
			}
			return commitRecords(ws, out, "change: update status", changed, !noPush)
		},
	}
	cmd.Flags().BoolVar(&noPush, "no-push", false, "commit the updated records without pushing them")
	cmd.AddCommand(newChangesSaveCmd())
	return cmd
}

func newChangesSaveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "save <id>",
		Short: "Commit one change record to the workspace's main branch and push it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			id := args[0]
			rel := filepath.Join("changes", id+".md")
			records, errs := changes.Load(ws.Root)
			var rec *changes.Record
			for _, r := range records {
				if r.ID == id {
					rec = r
				}
			}
			if rec == nil {
				for _, e := range errs {
					if strings.Contains(e.Error(), id+".md") {
						return e
					}
				}
				return fmt.Errorf("no change record %s", rel)
			}
			out := cmd.OutOrStdout()
			if err := commitRecords(ws, out, fmt.Sprintf("change: %s (%s)", rec.Title, rec.ID), []string{rel}, true); err != nil {
				return err
			}
			fmt.Fprintf(out, "saved %s\n", id)
			repo := gitx.Repo{Dir: ws.Root}
			if origin, err := repo.OriginURL(); err == nil {
				if url := changes.RecordURL(origin, repo.OriginHead(), filepath.ToSlash(rel)); url != "" {
					fmt.Fprintf(out, "record: %s\n", url)
				}
			}
			return nil
		},
	}
}
