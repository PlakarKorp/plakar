package tui

import (
	"bytes"
	"fmt"
	"io"
	"sync"
)

type switchWriter struct {
	tui      *tui
	stream   string // "stdout" or "stderr"
	fallback io.Writer

	mu  sync.Mutex
	buf []byte
}

func (w *switchWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		line := string(w.buf[:i])
		w.buf = w.buf[i+1:]
		w.writeLine(line)
	}
	return len(p), nil
}

// Flush writes any buffered partial line (no trailing newline).
func (w *switchWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.buf) == 0 {
		return
	}
	w.writeLine(string(w.buf))
	w.buf = w.buf[:0]
}

// writeLine must be called with w.mu held.
func (w *switchWriter) writeLine(line string) {
	app := w.tui.getApp()
	if app != nil && app.prog != nil {
		app.state.logs = append(app.state.logs, fmt.Sprintf("[%s] %s", w.stream, line))
		return
	}
	_, _ = io.WriteString(w.fallback, line+"\n")
}
