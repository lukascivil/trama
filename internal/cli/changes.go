package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/changes"
	"github.com/lukascivil/trama/internal/gitx"
	"github.com/lukascivil/trama/internal/workspace"
)

// stateFunc reads a pull request's state; tests replace it.
var stateFunc changes.PRState = changes.GHState

// recordBranch returns the workspace repository and its main branch,
// failing unless the workspace is on that branch.
func recordBranch(ws *workspace.Workspace) (gitx.Repo, string, error) {
	repo := gitx.Repo{Dir: ws.Root}
	main := repo.OriginHead()
	branch, err := repo.CurrentBranch()
	if err != nil {
		return repo, "", err
	}
	if branch != main {
		if branch == "" {
			branch = "a detached HEAD"
		}
		return repo, "", fmt.Errorf("workspace is on %s; change records go to %s", branch, main)
	}
	return repo, main, nil
}

// commitRecords commits the change records in paths and, with push, sends
// to origin every unpushed commit, as long as they only touch changes/.
func commitRecords(repo gitx.Repo, main string, out io.Writer, msg string, paths []string, push bool) error {
	if len(paths) > 0 {
		if _, err := repo.CommitPaths(msg, paths...); err != nil {
			return err
		}
	}
	if _, err := repo.OriginURL(); err != nil {
		if len(paths) > 0 {
			fmt.Fprintln(out, "committed locally; the workspace has no origin remote")
		}
		return nil
	}
	if !push {
		return nil
	}
	upstream := "origin/" + main
	if _, err := repo.RevParse(upstream); err == nil {
		files, err := repo.DiffNames(upstream, "HEAD")
		if err != nil {
			return err
		}
		var records, outside []string
		for _, f := range files {
			if strings.HasPrefix(f, "changes/") {
				records = append(records, f)
			} else {
				outside = append(outside, f)
			}
		}
		if len(outside) > 0 {
			return fmt.Errorf("the workspace's %s has unpushed commits outside changes/ (%s): push or undo them first; the record is committed locally", main, strings.Join(outside, ", "))
		}
		if len(records) == 0 {
			return nil
		}
	}
	if err := repo.PushHead(main); err != nil {
		return pushError(main, err)
	}
	return nil
}

// pushError explains why pushing the workspace's main branch failed.
func pushError(main string, err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "fetch first") || strings.Contains(msg, "non-fast-forward"):
		return fmt.Errorf("origin's %s has commits the workspace does not have: run \"git pull --rebase\" in the workspace, then run this again (the record is committed locally)", main)
	case strings.Contains(msg, "declined") || strings.Contains(msg, "protected"):
		return fmt.Errorf("push to %s was rejected (protected branch?): %v", main, err)
	default:
		return fmt.Errorf("push to %s failed: %v (the record is committed locally)", main, err)
	}
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
			repo, main, err := recordBranch(ws)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			records, errs := changes.Load(ws.Root)
			for _, err := range errs {
				fmt.Fprintf(out, "warning: %v\n", err)
			}
			var paths []string
			unread := 0
			for _, r := range records {
				if r.Status == "open" {
					updated, errs := changes.Refresh(r, stateFunc)
					unread += len(errs)
					for _, err := range errs {
						fmt.Fprintf(out, "warning: %v\n", err)
					}
					if updated {
						if err := r.Save(); err != nil {
							return err
						}
						paths = append(paths, filepath.ToSlash(filepath.Join("changes", r.ID+".md")))
					}
				}
				if r.Status == "done" {
					fmt.Fprintf(out, "%s: done\n", r.ID)
				} else {
					fmt.Fprintf(out, "%s: %d open step(s)\n", r.ID, len(r.OpenSteps()))
				}
			}
			modified, err := repo.ModifiedPaths("changes/")
			if err != nil {
				return err
			}
			for _, m := range modified {
				if strings.HasSuffix(m, ".md") && !slices.Contains(paths, m) {
					paths = append(paths, m)
				}
			}
			if err := commitRecords(repo, main, out, "change: update status", paths, !noPush); err != nil {
				return err
			}
			if unread > 0 {
				return fmt.Errorf("could not read the state of %d pull request(s)", unread)
			}
			return nil
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
			rel := filepath.ToSlash(filepath.Join("changes", id+".md"))
			repo, main, err := recordBranch(ws)
			if err != nil {
				return err
			}
			records, errs := changes.Load(ws.Root)
			var rec *changes.Record
			for _, r := range records {
				if r.ID == id {
					rec = r
				}
			}
			if rec == nil {
				for _, e := range errs {
					if strings.HasPrefix(e.Error(), id+".md:") {
						return e
					}
				}
				return fmt.Errorf("no change record %s", rel)
			}
			// Compare with the committed version to say what this save changes.
			summary := "update record"
			data, err := repo.ReadFile("HEAD", rel)
			if err == nil {
				if old, err := changes.Parse(rel, data); err == nil {
					summary = changes.Summary(old, rec)
				}
			} else if _, noHead := repo.RevParse("HEAD"); errors.Is(err, gitx.ErrNotExist) || noHead != nil {
				summary = changes.Summary(nil, rec)
			}
			out := cmd.OutOrStdout()
			if err := commitRecords(repo, main, out, fmt.Sprintf("change: %s: %s", rec.ID, summary), []string{rel}, true); err != nil {
				return err
			}
			fmt.Fprintf(out, "saved %s\n", id)
			if origin, err := repo.OriginURL(); err == nil {
				if url := changes.RecordURL(origin, main, rel); url != "" {
					fmt.Fprintf(out, "record: %s\n", url)
				}
			}
			return nil
		},
	}
}
