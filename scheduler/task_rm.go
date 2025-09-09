package scheduler

import (
	"fmt"
	"time"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/plakar/subcommands/rm"
)

type RmTask struct {
	TaskBase
	Cmd       rm.Rm
	Retention time.Duration
}

func (task *RmTask) Base() *TaskBase {
	return &task.TaskBase
}

func (task *RmTask) Run(ctx *TaskContext) {
	if task.Cmd.LocateOptions == nil {
		task.Cmd.LocateOptions = locate.NewDefaultLocateOptions()
	}
	task.Cmd.LocateOptions.Filters.Job = ctx.JobName
	task.Cmd.LocateOptions.Filters.Before = time.Now().Add(-task.Retention)

	if retval, err := task.Cmd.Execute(ctx.AppContext, ctx.Repository); err != nil || retval != 0 {
		ctx.GetLogger().Error("Error removing snapshots: %s", err)
		ctx.ReportFailure("Error removing snapshots: retval=%d, err=%s", retval, err)
	}
}

func (task *RmTask) Event(ctx *TaskContext, event events.Event) {
}

func (task *RmTask) String() string {
	return fmt.Sprintf("rm on %s", task.Repository)
}

func runRmTask(ctx *TaskContext, repoName string, duration time.Duration) {
	task := &RmTask{
		TaskBase: TaskBase{
			Repository: repoName,
			Type:       "RM",
		},
		Retention: duration,
	}
	task.Run(ctx)
}
