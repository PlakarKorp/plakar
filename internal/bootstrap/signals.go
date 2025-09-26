package bootstrap

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// SignalHandler cancels the application context upon termination signals.
type SignalHandler struct {
	ch   chan os.Signal
	once sync.Once
}

func NewSignalHandler() *SignalHandler {
	return &SignalHandler{ch: make(chan os.Signal, 1)}
}

func (h *SignalHandler) listen(ctx *ConfigContext) {
	for range h.ch {
		fmt.Fprintf(ctx.App.Stderr, "%s: Interrupting, it might take a while...\n", ctx.ProgramName)
		ctx.App.Cancel()
	}
}

func (h *SignalHandler) stop() {
	h.once.Do(func() {
		signal.Stop(h.ch)
		close(h.ch)
	})
}

// SignalStage installs signal handlers for graceful shutdown.
type SignalStage struct{}

// NewSignalStage builds a signal stage instance.
func NewSignalStage() *SignalStage {
	return &SignalStage{}
}

func (s *SignalStage) Name() string { return "signals" }

func (s *SignalStage) Execute(ctx *ConfigContext) error {
	handler := NewSignalHandler()
	ctx.Signals = handler

	ctx.RegisterCleanupNoErr(func() {
		handler.stop()
	})

	go handler.listen(ctx)
	signal.Notify(handler.ch, os.Interrupt, syscall.SIGTERM)

	return nil
}
