package task

import (
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/reporting"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/subcommands/backup"
	"github.com/PlakarKorp/plakar/subcommands/check"
	"github.com/PlakarKorp/plakar/subcommands/maintenance"
	"github.com/PlakarKorp/plakar/subcommands/restore"
	"github.com/PlakarKorp/plakar/subcommands/rm"
	"github.com/PlakarKorp/plakar/subcommands/sync"
)

func RunCommand(ctx *appcontext.AppContext, cmd subcommands.Subcommand, repo *repository.Repository, taskName string) (int, error) {
	reporter := reporting.NewReporter(ctx)
	report := reporter.NewReport()

	var taskKind string
	switch cmd.(type) {
	case *backup.Backup:
		taskKind = "backup"
	case *check.Check:
		taskKind = "check"
	case *restore.Restore:
		taskKind = "restore"
	case *sync.Sync:
		taskKind = "sync"
	case *rm.Rm:
		taskKind = "rm"
	case *maintenance.Maintenance:
		taskKind = "maintenance"
	default:
		report.SetIgnore()
	}

	report.TaskStart(taskKind, taskName)
	if repo != nil {
		report.WithRepositoryName(repo.Location())
		report.WithRepository(repo)
	}

	var err error
	var status int
	var snapshotID objects.MAC
	var warning error
	if _, ok := cmd.(*backup.Backup); ok {
		cmd := cmd.(*backup.Backup)
		status, err, snapshotID, warning = cmd.DoBackup(ctx, repo)
		if !cmd.DryRun && err == nil {
			report.WithSnapshotID(snapshotID)
		}
	} else {
		status, err = cmd.Execute(ctx, repo)
	}

	if status == 0 {
		if warning != nil {
			report.TaskWarning("warning: %s", warning)
		} else {
			report.TaskDone()
		}
	} else if err != nil {
		report.TaskFailed(0, "error: %s", err)
	}

	reporter.StopAndWait()

	return status, err
}
