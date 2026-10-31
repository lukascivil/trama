package runner

import (
	"bytes"
	"fmt"
	"io"
	"sync"
)

type prefixWriter struct {
	out   io.Writer
	label string
	mu    *sync.Mutex
	buf   []byte
}

// NewPrefixWriter writes each complete line as "<label> | <line>"; mu
// serializes writers that share out. Close flushes a last partial line.
func NewPrefixWriter(out io.Writer, label string, mu *sync.Mutex) io.WriteCloser {
	return &prefixWriter{out: out, label: label, mu: mu}
}

func (w *prefixWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			return len(p), nil
		}
		if err := w.line(w.buf[:i]); err != nil {
			return len(p), err
		}
		w.buf = w.buf[i+1:]
	}
}

func (w *prefixWriter) Close() error {
	if len(w.buf) == 0 {
		return nil
	}
	err := w.line(w.buf)
	w.buf = nil
	return err
}

func (w *prefixWriter) line(b []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err := fmt.Fprintf(w.out, "%s | %s\n", w.label, b)
	return err
}
