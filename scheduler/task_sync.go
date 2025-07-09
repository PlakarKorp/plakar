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

func (task *SyncTask) Base() *TaskBase {
	return &task.TaskBase
}

func (task *SyncTask) Run(ctx *TaskContext) {
	task.Cmd.PeerRepositoryLocation = task.Kloset // XXX

	retval, err := task.Cmd.Execute(ctx.AppContext, ctx.Repository)
	if err != nil || retval != 0 {
		ctx.GetLogger().Error("sync: %s", err)
		ctx.ReportFailure("Error executing sync: retval=%d, err=%s", retval, err)
		return
	}

	ctx.GetLogger().Info("sync: synchronization succeeded")
}

func (task *SyncTask) Event(ctx *TaskContext, event events.Event) {
}

func (task *SyncTask) String() string {
	return fmt.Sprintf("sync %s %s %s", task.Repository, task.Cmd.Direction, task.Kloset)
}
