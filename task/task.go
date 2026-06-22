package task

import (
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/reporting"
	"github.com/PlakarKorp/plakar/subcommands/backup"
	"github.com/PlakarKorp/plakar/subcommands/check"
	"github.com/PlakarKorp/plakar/subcommands/maintenance"
	"github.com/PlakarKorp/plakar/subcommands/restore"
	"github.com/PlakarKorp/plakar/subcommands/rm"
	"github.com/PlakarKorp/plakar/subcommands/sync"
)

func Report(ctx *appcontext.AppContext, repo *repository.Repository, taskKind, taskName string) error {
	location := ""
	var err error

	if repo != nil {
		location = repo.Origin()
	}

	reporter := reporting.NewReporter(ctx)
	report := reporter.NewReport()

	report.TaskStart(taskKind, taskName)
	if repo != nil {
		report.WithRepositoryName(location)
		report.WithRepository(repo)
	}

	var snapshotID objects.MAC
	var warning error
	if _, ok := cmd.(*backup.Backup); ok {
		cmd := cmd.(*backup.Backup)
		status, err, snapshotID, warning = cmd.DoBackup(ctx, repo)
		if !cmd.DryRun && err == nil {
			report.WithSnapshotID(snapshotID)
		}
	} else {
		status, err = cmd(ctx, repo)
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

	return err
}
