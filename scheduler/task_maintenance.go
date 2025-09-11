package scheduler

import (
	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/plakar/subcommands/maintenance"
)

type MaintenanceTask struct {
	TaskBase
	Cmd       maintenance.Maintenance
}

func (task *MaintenanceTask) Base() *TaskBase {
	return &task.TaskBase
}

func (task *MaintenanceTask) Run(ctx *TaskContext) {
	retval, err := task.Cmd.Execute(ctx.AppContext, ctx.Repository)
	if err != nil || retval != 0 {
		ctx.GetLogger().Error("Error executing maintenance: %s", err)
		ctx.ReportFailure("Error executing maintenance: retval=%d, err=%s", retval, err)
		return
	}
	ctx.GetLogger().Info("maintenance of repository %s succeeded", task.Repository)
}

func (task MaintenanceTask) Event(ctx *TaskContext, event events.Event) {
}

func (task *MaintenanceTask) String() string {
	return "maintenance on " + task.Repository
}
