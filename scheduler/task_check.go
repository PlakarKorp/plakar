package scheduler

import (
	"fmt"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/plakar/subcommands/check"
)

type CheckTask struct {
	TaskBase
	Cmd check.Check
}

func (task *CheckTask) Base() *TaskBase {
	return &task.TaskBase
}

func (task *CheckTask) Run(ctx *TaskContext) {
	retval, err := task.Cmd.Execute(ctx.AppContext, ctx.Repository)
	if err != nil || retval != 0 {
		ctx.GetLogger().Error("Error executing check: %s", err)
		ctx.ReportFailure("Error executing check: retval=%d, err=%s", retval, err)
	}
}

func (task CheckTask) Event(ctx *TaskContext, event events.Event) {
}

func (task *CheckTask) String() string {
	return fmt.Sprintf("check %s", task.Repository)
}
