package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/github"
	"github.com/lukascivil/trama/internal/pick"
	"github.com/lukascivil/trama/internal/runner"
	"github.com/lukascivil/trama/internal/workspace"
)

// githubAPI, ghRun and pickIdle are where tests stand in for GitHub, the gh
// binary and the wait.
var (
	githubAPI           = "https://api.github.com"
	ghRun     github.Gh = github.ExecGh
	pickIdle            = 30 * time.Minute
)

// runPick serves the page that lists the repositories of org (or lets the
// person choose an account when org is empty) until they add repositories or
// cancel. The names it added go to stdout, one per line; the URL, the refusals
// and everything else go to stderr, so a caller can read stdout as a list.
// A signal or a cancelled context returns runner.ErrInterrupted.
func runPick(cmd *cobra.Command, ws *workspace.Workspace, org string, port int, explicit, open bool) error {
	token := github.Token(ghRun, os.Getenv)
	client := github.NewClient(token)
	client.BaseURL = githubAPI
	p := pick.New(pick.Options{
		Workspace:     ws,
		Org:           org,
		Orgs:          client.Orgs,
		List:          client.Repos,
		Protocol:      github.Protocol(ghRun),
		Authenticated: token != "",
	})

	ln, err := listenLocal(port, explicit)
	if err != nil {
		return err
	}
	port = ln.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()
	fmt.Fprintf(errOut, "serving %s (Ctrl+C to stop)\n", url)
	if open {
		if err := openURL(url); err != nil {
			fmt.Fprintf(errOut, "open %s in your browser\n", url)
		}
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()
	srv := &http.Server{Handler: p.Handler(port), ReadHeaderTimeout: 5 * time.Second}
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()
	waited := make(chan error, 1)
	go func() { waited <- p.Wait(ctx, pickIdle) }()

	var result error
	select {
	case err := <-served:
		return err
	case result = <-waited:
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)

	switch {
	case result == nil:
		for _, r := range p.Refused() {
			fmt.Fprintf(errOut, "refused %s: %s\n", r.ID, r.Reason)
		}
		for _, name := range p.Added() {
			fmt.Fprintln(out, name)
		}
		return nil
	case errors.Is(result, context.Canceled):
		return runner.ErrInterrupted
	case errors.Is(result, pick.ErrIdle):
		return fmt.Errorf("stopped after %s without a request: %w", pickIdle, result)
	}
	return result
}
