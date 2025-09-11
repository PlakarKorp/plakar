package scheduler

import (
	"fmt"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/plakar/subcommands/backup"
)

type BackupTask struct {
	TaskBase
	IgnoreFile string
	Ignore     []string
	Cmd        backup.Backup
}

func (task *BackupTask) Base() *TaskBase {
	return &task.TaskBase
}

func (task *BackupTask) Run(ctx *TaskContext) {
	task.Cmd.Job = ctx.JobName

	var excludes []string
	if task.IgnoreFile != "" {
		lines, err := backup.LoadIgnoreFile(task.IgnoreFile)
		if err != nil {
			ctx.GetLogger().Error("Failed to load ignore file: %s", err)
			ctx.ReportFailure("Failed to load ignore file: %s", err)
			return
		}
		for _, line := range lines {
			excludes = append(excludes, line)
		}
	}
	for _, line := range task.Ignore {
		excludes = append(excludes, line)
	}
	task.Cmd.Excludes = excludes

	retval, err := task.Cmd.Execute(ctx.AppContext, ctx.Repository)
	if err != nil || retval != 0 {
		ctx.GetLogger().Error("Error creating backup: %s", err)
		ctx.ReportFailure("Error creating backup: retval=%d, err=%s", retval, err)
	}
}

func (task BackupTask) Event(ctx *TaskContext, event events.Event) {
	switch event := event.(type) {
	case events.StartImporter:
		ctx.snapshotId = event.SnapshotID
	}
}

func (task *BackupTask) String() string {
	return fmt.Sprintf("backup %s on %s", task.Cmd.Path, task.Repository)
}
