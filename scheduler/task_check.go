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

func (task *CheckTask) Run(ctx *TaskContext) {
	err := task.LoadRepository(ctx)
	if err != nil {
		ctx.GetLogger().Error("Error loading repository: %s", err)
		return
	}

	retval, err := task.Cmd.Execute(ctx.AppContext, ctx.Repository)
	if err != nil || retval != 0 {
		ctx.GetLogger().Error("Error executing check: %s", err)
		ctx.Reporter.TaskFailed(1, "Error executing check: retval=%d, err=%s", retval, err)
		return
	}

	ctx.Reporter.TaskDone()
}

func (task CheckTask) Event(ctx *TaskContext, event events.Event) {
}

func (task *CheckTask) String() string {
	return fmt.Sprintf("check %s", task.Repository)
}
