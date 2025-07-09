package scheduler

import (
	"fmt"
	"time"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/plakar/subcommands/rm"
	"github.com/PlakarKorp/plakar/utils"
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
		task.Cmd.LocateOptions = utils.NewDefaultLocateOptions()
	}
	task.Cmd.LocateOptions.Job = ctx.JobName
	task.Cmd.LocateOptions.Before = time.Now().Add(-task.Retention)

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
