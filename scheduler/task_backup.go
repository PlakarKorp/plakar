package scheduler

import (
	"fmt"
	"time"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/plakar/subcommands/backup"
)

type BackupTask struct {
	TaskBase
	Retention time.Duration
	Cmd       backup.Backup
}

func (task *BackupTask) Run(ctx *TaskContext) {
	err := task.LoadRepository(ctx)
	if err != nil {
		ctx.GetLogger().Error("Error loading repository: %s", err)
		return
	}

	task.Cmd.Job = ctx.JobName
	retval, err, snapId, reportWarning := task.Cmd.DoBackup(ctx.AppContext, ctx.Repository)
	if err != nil || retval != 0 {
		ctx.GetLogger().Error("Error creating backup: %s", err)
		ctx.Reporter.TaskFailed(1, "Error creating backup: retval=%d, err=%s", retval, err)
		return
	}

	ctx.Reporter.WithSnapshotID(snapId)
	if reportWarning != nil {
		ctx.Reporter.TaskWarning("Warning during backup: %s", reportWarning)
	} else {
		ctx.Reporter.TaskDone()
	}

	if task.Retention != 0 {
		runRmTask(ctx, task.Repository, task.Retention)
	}
}

func (task BackupTask) Event(ctx *TaskContext, event events.Event) {
}

func (task *BackupTask) String() string {
	return fmt.Sprintf("backup %s on %s", task.Cmd.Path, task.Repository)
}
