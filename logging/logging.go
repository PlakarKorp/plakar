package logging

import (
	"io"
	"log"
	"strings"
	"sync"
)

type Logger struct {
	EnabledInfo       bool
	EnabledTracing    string
	mutraceSubsystems sync.Mutex
	traceSubsystems   map[string]bool
	stdoutLogger      *log.Logger
	stderrLogger      *log.Logger
}

func NewLogger(stdout io.Writer, stderr io.Writer) *Logger {
	return &Logger{
		EnabledInfo:     false,
		EnabledTracing:  "",
		stdoutLogger:    log.New(stdout, "", 0),
		stderrLogger:    log.New(stderr, "", 0),
		traceSubsystems: make(map[string]bool),
	}
}

func (l *Logger) SetOutput(w io.Writer) {
	l.stdoutLogger.SetOutput(w)
	l.stderrLogger.SetOutput(w)
}

func (l *Logger) SetSyslogOutput(w io.Writer) {
	l.stdoutLogger = log.New(w, "stdout", 0)
	l.stderrLogger = log.New(w, "stderr", 0)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.stdoutLogger.Printf(format, args...)
}

func (l *Logger) Stdout(format string, args ...interface{}) {
	l.stdoutLogger.Printf(format, args...)
}

func (l *Logger) Stderr(format string, args ...interface{}) {
	l.stderrLogger.Printf(format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	if l.EnabledInfo {
		l.stdoutLogger.Printf("info: "+format, args...)
	}
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.stderrLogger.Printf("warn: "+format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.stderrLogger.Printf("error: "+format, args...)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.stderrLogger.Printf("debug: "+format, args...)
}

func (l *Logger) Trace(subsystem string, format string, args ...interface{}) {
	if l.EnabledTracing != "" {
		l.mutraceSubsystems.Lock()
		_, exists := l.traceSubsystems[subsystem]
		if !exists {
			_, exists = l.traceSubsystems["all"]
		}
		l.mutraceSubsystems.Unlock()
		if exists {
			l.stdoutLogger.Printf("trace: "+subsystem+": "+format, args...)
		}
	}
}

func (l *Logger) EnableInfo() {
	l.EnabledInfo = true
}

func (l *Logger) InfoEnabled() bool {
	return l.EnabledInfo
}

func (l *Logger) EnableTracing(traces string) {
	l.EnabledTracing = traces
	l.traceSubsystems = make(map[string]bool)
	for _, subsystem := range strings.Split(traces, ",") {
		l.traceSubsystems[subsystem] = true
	}
}

func (l *Logger) TracingEnabled() string {
	return l.EnabledTracing
}
