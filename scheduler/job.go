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

	lastRun       time.Time
	lastActualRun time.Time

	isRunning bool
	runLock   sync.Mutex
}

func (job *Job) Start() bool {
	job.runLock.Lock()
	if job.isRunning {
		job.runLock.Unlock()
		return false
	}
	job.isRunning = true
	job.runLock.Unlock()
	return true
}

func (job *Job) Done() {
	job.isRunning = false
}

func (job *Job) Execute(ctx *appcontext.AppContext, scheduledAt time.Time) {
	if !job.Start() {
		ctx.GetLogger().Warn("job %q: still running", job.Name)
		return
	}

	delay := time.Since(scheduledAt)
	if delay > 5*time.Second {
		// This might happen if the machine/process was suspended.
		ctx.GetLogger().Warn("job %q: overdue by %s", job.Name, delay)
	}

	job.lastRun = scheduledAt
	job.lastActualRun = time.Now()

	go func() {
		defer job.Done()

		ctx.GetLogger().Info("job %q: running", job.Name)

		taskCtx := TaskContext{
			JobName:    job.Name,
			AppContext: appcontext.NewAppContextFrom(ctx),
		}

		err := taskCtx.Prepare(job.Task)
		if err == nil {
			job.Task.Run(&taskCtx)
		}
		taskCtx.Finalize()

		ctx.GetLogger().Info("job %q: done", job.Name)
	}()
}
