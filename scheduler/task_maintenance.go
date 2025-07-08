package scheduler

import (
	"time"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/plakar/subcommands/maintenance"
)

type MaintenanceTask struct {
	TaskBase
	Retention time.Duration
	Cmd       maintenance.Maintenance
}

func (task *MaintenanceTask) Run(ctx *TaskContext) {
	err := task.LoadRepository(ctx)
	if err != nil {
		ctx.GetLogger().Error("Error loading repository: %s", err)
		return
	}

	retval, err := task.Cmd.Execute(ctx.AppContext, ctx.Repository)
	if err != nil || retval != 0 {
		ctx.GetLogger().Error("Error executing maintenance: %s", err)
		ctx.Reporter.TaskFailed(1, "Error executing maintenance: retval=%d, err=%s", retval, err)
		return
	}

	ctx.GetLogger().Info("maintenance of repository %s succeeded", task.Repository)
	ctx.Reporter.TaskDone()

	if task.Retention != 0 {
		runRmTask(ctx, task.Repository, task.Retention)
	}
}

func (task MaintenanceTask) Event(ctx *TaskContext, event events.Event) {
}

func (task *MaintenanceTask) String() string {
	return "maintenance on " + task.Repository
}
