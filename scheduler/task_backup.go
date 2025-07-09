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

func (task *BackupTask) Base() *TaskBase {
	return &task.TaskBase
}

func (task *BackupTask) Run(ctx *TaskContext) {
	task.Cmd.Job = ctx.JobName
	retval, err, snapId, reportWarning := task.Cmd.DoBackup(ctx.AppContext, ctx.Repository)
	if err != nil || retval != 0 {
		ctx.GetLogger().Error("Error creating backup: %s", err)
		ctx.ReportFailure("Error creating backup: retval=%d, err=%s", retval, err)
		return
	}
	// XXX use events
	ctx.reporter.WithSnapshotID(snapId)
	if reportWarning != nil {
		ctx.ReportWarning("Warning during backup: %s", reportWarning)
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
