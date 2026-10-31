package runner

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
	"unsafe"
)

// ErrInterrupted means trama was asked to stop while a command ran.
var ErrInterrupted = errors.New("interrupted")

// grace is how long a command gets to stop after a signal before SIGKILL.
const grace = 5 * time.Second

// Run runs j in the foreground and returns the child's error, or
// ErrInterrupted when trama got SIGINT, SIGTERM or SIGHUP meanwhile.
//
// On a terminal the child stays in trama's process group: the terminal's
// Ctrl+C reaches it directly and it can read the keyboard, so trama only
// passes on SIGTERM and SIGHUP. Without a terminal the child gets its own
// process group: every signal goes to the whole group, and when the command
// ends the group gets SIGTERM, so background children do not outlive it.
// A command still running grace after a signal gets SIGKILL.
func Run(ctx context.Context, j Job, stdin io.Reader, stdout, stderr io.Writer) error {
	args := j.Args()
	c := exec.CommandContext(ctx, args[0], args[1:]...)
	c.Dir = j.Dir
	c.Stdin = stdin
	c.Stdout = stdout
	c.Stderr = stderr
	c.WaitDelay = time.Second // a grandchild holding the output open must not block Wait
	tty := isTerminal(stdin)
	if !tty {
		c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)
	if err := c.Start(); err != nil {
		return err
	}
	pgid := c.Process.Pid
	waited := make(chan error, 1)
	go func() { waited <- c.Wait() }()
	interrupted := false
	var kill <-chan time.Time
	for {
		select {
		case err := <-waited:
			if !tty {
				_ = syscall.Kill(-pgid, syscall.SIGTERM)
			}
			if interrupted {
				return ErrInterrupted
			}
			return err
		case sig := <-signals:
			interrupted = true
			s := sig.(syscall.Signal)
			switch {
			case !tty:
				_ = syscall.Kill(-pgid, s)
			case s != syscall.SIGINT:
				_ = c.Process.Signal(s)
			}
			if kill == nil {
				kill = time.After(grace)
			}
		case <-kill:
			if tty {
				_ = c.Process.Kill()
			} else {
				_ = syscall.Kill(-pgid, syscall.SIGKILL)
			}
		}
	}
}

// isTerminal reports whether r is a terminal.
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok || f == nil {
		return false
	}
	var t syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), ioctlReadTermios, uintptr(unsafe.Pointer(&t)))
	return errno == 0
}
