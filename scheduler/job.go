package scheduler

import (
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
)

type Job struct {
	Name      string
	Task      Task
	Schedules []Schedule

	isRunning     bool
	lastRun       time.Time
	lastActualRun time.Time

	mu sync.Mutex
}

type ScheduledJob struct {
	event     *Event[*ScheduledJob]
	scheduled time.Time
	job       *Job
}

func (s *ScheduledJob) Execute(ctx *appcontext.AppContext) {
	s.job.mu.Lock()

	// Do not execute a job if the previous invocation is sill running.
	if s.job.isRunning {
		s.job.mu.Unlock()
		ctx.GetLogger().Warn("job %q: still running", s.job.Name)
		return
	}

	delay := time.Since(s.scheduled)
	if delay > 5*time.Second {
		// This might happen if the machine was suspended.
		ctx.GetLogger().Warn("job %q: overdue by %s", s.job.Name, delay)
	}

	s.job.mu.Unlock()
	s.job.lastRun = s.scheduled
	s.job.lastActualRun = time.Now()

	go func() {
		ctx.GetLogger().Info("job %q: running", s.job.Name)

		taskCtx := TaskContext{
			JobName:    s.job.Name,
			AppContext: appcontext.NewAppContextFrom(ctx),
		}

		err := taskCtx.Prepare(s.job.Task)
		if err == nil {
			s.job.Task.Run(&taskCtx)
		}
		taskCtx.Finalize()

		ctx.GetLogger().Info("job %q: done", s.job.Name)
		// lock is not needed here
		s.job.isRunning = false
	}()
}
