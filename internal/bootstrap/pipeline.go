package bootstrap

import "fmt"

// Stage represents a bootstrap pipeline step.
type Stage interface {
	Name() string
	Execute(*ConfigContext) error
}

// Pipeline executes configured bootstrap stages sequentially.
type Pipeline struct {
	stages []Stage
}

// NewPipeline builds a pipeline from the provided stages.
func NewPipeline(stages ...Stage) *Pipeline {
	return &Pipeline{stages: stages}
}

// Run executes the pipeline stages in order. If a stage returns an error,
// execution stops immediately. When the context asks for early exit, the
// pipeline stops without raising an error.
func (p *Pipeline) Run(ctx *ConfigContext) error {
	for _, stage := range p.stages {
		if ctx.ShouldExit {
			return nil
		}

		if err := stage.Execute(ctx); err != nil {
			return fmt.Errorf("bootstrap stage %q failed: %w", stage.Name(), err)
		}
	}

	return nil
}
