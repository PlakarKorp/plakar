package scheduler

import (
	"fmt"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/plakar/subcommands/sync"
)

type SyncTask struct {
	TaskBase
	Kloset string
	Cmd    sync.Sync
}

func (task *SyncTask) Run(ctx *TaskContext) {
	err := task.LoadRepository(ctx)
	if err != nil {
		ctx.GetLogger().Error("Error loading repository: %s", err)
		return
	}

	task.Cmd.PeerRepositoryLocation = task.Kloset // XXX

	retval, err := task.Cmd.Execute(ctx.AppContext, ctx.Repository)
	if err != nil || retval != 0 {
		ctx.GetLogger().Error("sync: %s", err)
		ctx.Reporter.TaskFailed(1, "Error executing sync: retval=%d, err=%s", retval, err)
		return
	}

	ctx.GetLogger().Info("sync: synchronization succeeded")
	ctx.Reporter.TaskDone()
}

func (task *SyncTask) Event(ctx *TaskContext, event events.Event) {
}

func (task *SyncTask) String() string {
	return fmt.Sprintf("sync %s %s %s", task.Repository, task.Cmd.Direction, task.Kloset)
}
