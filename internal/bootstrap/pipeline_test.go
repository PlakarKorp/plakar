package bootstrap

import (
	"errors"
	"strings"
	"testing"
)

type stubStage struct {
	name string
	run  func(*ConfigContext) error
}

func (s stubStage) Name() string { return s.name }

func (s stubStage) Execute(ctx *ConfigContext) error {
	if s.run != nil {
		return s.run(ctx)
	}
	return nil
}

func TestPipelineStopsOnEarlyExit(t *testing.T) {
	ctx := NewConfigContext([]string{"plakar"})
	defer ctx.Close()

	invoked := false

	pipeline := NewPipeline(
		stubStage{name: "stopper", run: func(c *ConfigContext) error {
			c.ShouldExit = true
			return nil
		}},
		stubStage{name: "next", run: func(c *ConfigContext) error {
			invoked = true
			return nil
		}},
	)

	if err := pipeline.Run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if invoked {
		t.Fatal("second stage executed despite early exit")
	}
}

func TestPipelineReturnsStageError(t *testing.T) {
	ctx := NewConfigContext([]string{"plakar"})
	defer ctx.Close()

	errBoom := errors.New("boom")
	pipeline := NewPipeline(
		stubStage{name: "failing", run: func(c *ConfigContext) error { return errBoom }},
	)

	err := pipeline.Run(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failing") {
		t.Fatalf("stage name missing from error: %v", err)
	}
}

func TestConfigContextCleanupOrder(t *testing.T) {
	ctx := NewConfigContext([]string{"plakar"})

	runs := make([]string, 0, 3)
	ctx.RegisterCleanupNoErr(func() {
		runs = append(runs, "first")
	})
	ctx.RegisterCleanup(func() error {
		runs = append(runs, "second")
		return nil
	})
	ctx.RegisterCleanupNoErr(func() {
		runs = append(runs, "third")
	})

	if err := ctx.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}

	expected := []string{"second", "third", "first"}
	if len(runs) != len(expected) {
		t.Fatalf("cleanup count mismatch: %v", runs)
	}

	for i, want := range expected {
		if runs[i] != want {
			t.Fatalf("cleanup order mismatch at %d: got %s want %s", i, runs[i], want)
		}
	}
}
