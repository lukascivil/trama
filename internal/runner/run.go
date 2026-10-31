package runner

import (
	"context"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// Run runs j in the foreground. The child stays in trama's process group, so
// the terminal's Ctrl+C reaches it directly and it can read the keyboard;
// trama itself ignores the signal and waits for the child to end. The
// returned error is the child's (an *exec.ExitError carries the code).
func Run(ctx context.Context, j Job, stdin io.Reader, stdout, stderr io.Writer) error {
	args := j.Args()
	c := exec.CommandContext(ctx, args[0], args[1:]...)
	c.Dir = j.Dir
	c.Stdin = stdin
	c.Stdout = stdout
	c.Stderr = stderr
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	return c.Run()
}
