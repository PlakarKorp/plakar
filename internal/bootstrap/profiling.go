package bootstrap

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

// ProfilingManager handles CPU/MEM profiling and execution timing output.
type ProfilingManager struct {
	cpuProfilePath string
	memProfilePath string
	printDuration  bool

	start int64
	now   func() int64

	cpuFile *os.File
	closed  bool
}

// NewProfilingManager configures the profiling manager for the given options.
func NewProfilingManager(opts Options, now func() int64) *ProfilingManager {
	return &ProfilingManager{
		cpuProfilePath: opts.CPUProfile,
		memProfilePath: opts.MemProfile,
		printDuration:  opts.PrintExecutionTime,
		now:            now,
	}
}

func (m *ProfilingManager) Name() string { return "profiling" }

// Start begins CPU profiling and records the start timestamp.
func (m *ProfilingManager) Start(ctx *ConfigContext) error {
	m.start = m.now()

	if m.cpuProfilePath == "" {
		return nil
	}

	f, err := os.Create(m.cpuProfilePath)
	if err != nil {
		fmt.Fprintf(ctx.App.Stderr, "%s: could not create CPU profile: %s\n", ctx.ProgramName, err)
		return err
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		fmt.Fprintf(ctx.App.Stderr, "%s: could not start CPU profile: %s\n", ctx.ProgramName, err)
		_ = f.Close()
		return err
	}

	m.cpuFile = f
	return nil
}

// Finalize stops profiling, emits timing information and writes memory profiles.
func (m *ProfilingManager) Finalize(ctx *ConfigContext) error {
	if m.printDuration {
		delta := timeFromUnixNano(m.now() - m.start)
		fmt.Fprintln(ctx.App.Stdout, "time:", delta)
	}

	if m.memProfilePath != "" {
		f, err := os.Create(m.memProfilePath)
		if err != nil {
			return fmt.Errorf("could not create memory profile: %w", err)
		}
		defer f.Close()

		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			fmt.Fprintf(ctx.App.Stderr, "%s: could not write MEM profile: %s\n", ctx.ProgramName, err)
			return err
		}
	}

	return m.cleanup()
}

func (m *ProfilingManager) cleanup() error {
	if m.closed {
		return nil
	}
	m.closed = true

	if m.cpuProfilePath != "" {
		pprof.StopCPUProfile()
	}

	if m.cpuFile != nil {
		return m.cpuFile.Close()
	}

	return nil
}

// ProfilingStage wires the profiling manager into the pipeline.
type ProfilingStage struct{}

// NewProfilingStage creates a profiling stage instance.
func NewProfilingStage() *ProfilingStage {
	return &ProfilingStage{}
}

func (s *ProfilingStage) Name() string { return "profiling" }

func (s *ProfilingStage) Execute(ctx *ConfigContext) error {
	manager := NewProfilingManager(ctx.Options, func() int64 { return ctx.now().UnixNano() })
	ctx.Profiling = manager
	ctx.RegisterCleanup(manager.cleanup)
	return manager.Start(ctx)
}

func timeFromUnixNano(ns int64) time.Duration {
	return time.Duration(ns)
}
