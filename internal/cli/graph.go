package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/lukascivil/trama/internal/graph"
	"github.com/lukascivil/trama/internal/runner"
	"github.com/lukascivil/trama/internal/validate"
	"github.com/lukascivil/trama/internal/view"
)

// defaultPort is where trama graph --serve listens when --port is not set
// and the port is free.
const defaultPort = 7300

// openURL opens url in the browser; tests replace it.
var openURL = func(url string) error {
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	cmd := exec.Command(name, url)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func newGraphCmd() *cobra.Command {
	var serve, noOpen bool
	var port int
	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Write docs/graph.md and the repository table in AGENTS.md from the catalog, or serve the graph page with --serve",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !serve && (cmd.Flags().Changed("port") || noOpen) {
				return errors.New("--port and --no-open need --serve")
			}
			ws, err := loadWorkspace(cmd)
			if err != nil {
				return err
			}
			if serve {
				return serveGraph(cmd, ws.Root, port, cmd.Flags().Changed("port"), !noOpen)
			}
			if n := validate.Errors(validate.Workspace(ws)); n > 0 {
				return fmt.Errorf(`fix %d validation error(s) first: run "trama validate"`, n)
			}
			if err := graph.Write(ws); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "wrote docs/graph.md")
			fmt.Fprintln(out, "updated AGENTS.md")
			return nil
		},
	}
	cmd.Flags().BoolVar(&serve, "serve", false, "serve an interactive graph of the workspace on 127.0.0.1 until Ctrl+C")
	cmd.Flags().IntVar(&port, "port", defaultPort, "port for --serve; 0 picks a free one")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "with --serve, do not open the browser")
	return cmd
}

// serveGraph serves the graph page of the workspace at root until the
// command's context ends or trama gets a signal, which returns
// runner.ErrInterrupted. Without an explicit port, a busy default port
// falls back to a free one.
func serveGraph(cmd *cobra.Command, root string, port int, explicit, open bool) error {
	b := view.NewBuilder(root)
	if _, err := b.Build(); err != nil {
		return err
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if errors.Is(err, syscall.EADDRINUSE) {
		if explicit {
			return fmt.Errorf("port %d is in use: choose another with --port", port)
		}
		ln, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err != nil {
		return err
	}
	port = ln.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "serving %s (Ctrl+C to stop)\n", url)
	if open {
		if err := openURL(url); err != nil {
			fmt.Fprintf(out, "open %s in your browser\n", url)
		}
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()
	srv := &http.Server{Handler: view.Handler(b.Build, port), ReadHeaderTimeout: 5 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
	return runner.ErrInterrupted
}
