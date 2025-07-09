package scheduler

import (
	"fmt"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/plakar/subcommands/restore"
)

type RestoreTask struct {
	TaskBase
	Cmd restore.Restore
}

func (task *RestoreTask) Base() *TaskBase {
	return &task.TaskBase
}

func (task *RestoreTask) Run(ctx *TaskContext) {
	retval, err := task.Cmd.Execute(ctx.AppContext, ctx.Repository)
	if err != nil || retval != 0 {
		ctx.GetLogger().Error("Error executing restore: %s", err)
		ctx.ReportFailure("Error executing restore: retval=%d, err=%s", retval, err)
	}
}

func (task *RestoreTask) Event(ctx *TaskContext, event events.Event) {
}

func (task *RestoreTask) String() string {
	return fmt.Sprintf("restore %s to %q", task.Repository, task.Cmd.Target)
}
