package tui

import (
	"bytes"
	"fmt"
	"io"
)

type logLineMsg struct {
	Stream string // "stdout" or "stderr"
	Line   string
}

type asyncTeaLogWriter struct {
	tui    *tui
	ch     chan logLineMsg
	stream string
	buf    []byte
}

func (w *asyncTeaLogWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		line := string(w.buf[:i])
		w.buf = w.buf[i+1:]

		select {
		case w.ch <- logLineMsg{Stream: w.stream, Line: line}:
		default:
			// drop
		}
	}
	return len(p), nil
}

type switchWriter struct {
	tui      *tui
	stream   string // stdout / stderr
	fallback io.Writer

	buf []byte
}

func (w *switchWriter) Write(p []byte) (int, error) {
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

func (w *switchWriter) writeLine(line string) {
	app := w.tui.app
	if app != nil && app.prog != nil {
		app.state.logs = append(app.state.logs, fmt.Sprintf("[%s] %s", w.stream, line))
		return
	}
	_, _ = io.WriteString(w.fallback, line+"\n")
}
