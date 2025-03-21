package scheduler

import (
	"fmt"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/backup"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/check"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/maintenance"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/restore"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/rm"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/sync"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/storage"
)

func (s *Scheduler) backupTask(taskset Task, task BackupConfig) error {
	interval, err := stringToDuration(task.Interval)
	if err != nil {
		return err
	}

	var retention time.Duration
	if task.Retention != "" {
		retention, err = stringToDuration(task.Retention)
		if err != nil {
			return err
		}
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		firstRun := true
		for {
			if firstRun {
				firstRun = false
			} else {
				time.Sleep(interval)
			}

			storeConfig, err := s.ctx.Config.GetRepository(taskset.Repository)
			if err != nil {
				s.ctx.GetLogger().Error("Error getting repository configuration: %s", err)
				continue
			}

			store, config, err := storage.Open(storeConfig)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening storage: %s", err)
				continue
			}

			newCtx := appcontext.NewAppContextFrom(s.ctx)

			repo, err := repository.New(newCtx, store, config)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				store.Close()
				continue
			}

			backupCtx := appcontext.NewAppContextFrom(newCtx)
			backupSubcommand := &backup.Backup{
				Silent: true,
				Job:    taskset.Name,
				Path:   task.Path,
				Quiet:  true,
			}
			if task.Check.Enabled {
				backupSubcommand.OptCheck = true
			}
			retval, err := backupSubcommand.Execute(backupCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error creating backup: %s", err)
				backupCtx.Close()
				goto close
			}
			backupCtx.Close()

			if task.Retention != "" {
				rmCtx := appcontext.NewAppContextFrom(newCtx)
				rmSubcommand := &rm.Rm{
					OptJob:    task.Name,
					OptBefore: time.Now().Add(-retention),
				}
				retval, err = rmSubcommand.Execute(rmCtx, repo)
				if err != nil || retval != 0 {
					s.ctx.GetLogger().Error("Error removing obsolete backups: %s", err)
				}
				rmCtx.Close()
			}

		close:
			newCtx.Close()
			repo.Close()
			store.Close()
		}
	}()

	return nil
}

func (s *Scheduler) checkTask(taskset Task, task CheckConfig) error {
	interval, err := stringToDuration(task.Interval)
	if err != nil {
		return err
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		firstRun := true
		for {
			if firstRun {
				firstRun = false
			} else {
				time.Sleep(interval)
			}

			storeConfig, err := s.ctx.Config.GetRepository(taskset.Repository)
			if err != nil {
				s.ctx.GetLogger().Error("Error getting repository configuration: %s", err)
				continue
			}

			store, config, err := storage.Open(storeConfig)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening storage: %s", err)
				continue
			}

			newCtx := appcontext.NewAppContextFrom(s.ctx)

			repo, err := repository.New(newCtx, store, config)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				store.Close()
				continue
			}

			checkSubcommand := &check.Check{
				OptJob:    taskset.Name,
				OptLatest: task.Latest,
				Silent:    true,
			}
			if task.Path != "" {
				checkSubcommand.Snapshots = []string{":" + task.Path}
			}
			retval, err := checkSubcommand.Execute(newCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing check: %s", err)
			}

			newCtx.Close()
			repo.Close()
			store.Close()
		}
	}()

	return nil
}

func (s *Scheduler) restoreTask(taskset Task, task RestoreConfig) error {
	interval, err := stringToDuration(task.Interval)
	if err != nil {
		return err
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		firstRun := true
		for {
			if firstRun {
				firstRun = false
			} else {
				time.Sleep(interval)
			}

			storeConfig, err := s.ctx.Config.GetRepository(taskset.Repository)
			if err != nil {
				s.ctx.GetLogger().Error("Error getting repository configuration: %s", err)
				continue
			}

			store, config, err := storage.Open(storeConfig)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening storage: %s", err)
				continue
			}

			newCtx := appcontext.NewAppContextFrom(s.ctx)

			repo, err := repository.New(newCtx, store, config)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				store.Close()
				continue
			}

			restoreSubcommand := &restore.Restore{
				OptJob: taskset.Name,
				Target: task.Target,
				Silent: true,
			}
			if task.Path != "" {
				restoreSubcommand.Snapshots = []string{":" + task.Path}
			}
			retval, err := restoreSubcommand.Execute(newCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing restore: %s", err)
			}

			newCtx.Close()
			repo.Close()
			store.Close()
		}
	}()

	return nil
}

func (s *Scheduler) syncTask(taskset Task, task SyncConfig) error {
	interval, err := stringToDuration(task.Interval)
	if err != nil {
		return err
	}

	syncSubcommand := &sync.Sync{
		PeerRepositoryLocation: task.Peer,
	}
	if task.Direction == SyncDirectionTo {
		syncSubcommand.Direction = "to"
	} else if task.Direction == SyncDirectionFrom {
		syncSubcommand.Direction = "from"
	} else if task.Direction == SyncDirectionWith {
		syncSubcommand.Direction = "with"
	} else {
		return fmt.Errorf("invalid sync direction: %s", task.Direction)
	}

	//	syncSubcommand.OptJob = taskset.Name
	//	syncSubcommand.Target = task.Target
	//	syncSubcommand.Silent = true

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		firstRun := true
		for {
			if firstRun {
				firstRun = false
			} else {
				time.Sleep(interval)
			}

			storeConfig, err := s.ctx.Config.GetRepository(taskset.Repository)
			if err != nil {
				s.ctx.GetLogger().Error("Error getting repository configuration: %s", err)
				continue
			}

			store, config, err := storage.Open(storeConfig)
			if err != nil {
				s.ctx.GetLogger().Error("sync: error opening storage: %s", err)
				continue
			}

			newCtx := appcontext.NewAppContextFrom(s.ctx)

			repo, err := repository.New(newCtx, store, config)
			if err != nil {
				s.ctx.GetLogger().Error("sync: error opening repository: %s", err)
				store.Close()
				continue
			}

			retval, err := syncSubcommand.Execute(newCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("sync: %s", err)
			} else {
				s.ctx.GetLogger().Info("sync: synchronization succeeded")
			}

			newCtx.Close()
			repo.Close()
			store.Close()
		}
	}()

	return nil
}

func (s *Scheduler) maintenanceTask(task MaintenanceConfig) error {
	interval, err := stringToDuration(task.Interval)
	if err != nil {
		return err
	}

	var retention time.Duration
	if task.Retention != "" {
		retention, err = stringToDuration(task.Retention)
		if err != nil {
			return err
		}
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		firstRun := true
		for {
			if firstRun {
				firstRun = false
			} else {
				time.Sleep(interval)
			}

			storeConfig, err := s.ctx.Config.GetRepository(task.Repository)
			if err != nil {
				s.ctx.GetLogger().Error("Error getting repository configuration: %s", err)
				continue
			}

			store, config, err := storage.Open(storeConfig)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening storage: %s", err)
				continue
			}

			newCtx := appcontext.NewAppContextFrom(s.ctx)

			repo, err := repository.New(newCtx, store, config)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				store.Close()
				continue
			}

			maintenanceSubcommand := &maintenance.Maintenance{}
			retval, err := maintenanceSubcommand.Execute(newCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing maintenance: %s", err)
			} else {
				s.ctx.GetLogger().Info("maintenance of repository %s succeeded", task.Repository)
			}

			if task.Retention != "" {
				rmCtx := appcontext.NewAppContextFrom(newCtx)
				rmSubcommand := &rm.Rm{
					OptBefore: time.Now().Add(-retention),
				}
				retval, err = rmSubcommand.Execute(rmCtx, repo)
				if err != nil || retval != 0 {
					s.ctx.GetLogger().Error("Error removing obsolete backups: %s", err)
				} else {
					s.ctx.GetLogger().Info("Retention purge succeeded")
				}
				rmCtx.Close()
			}

			newCtx.Close()
			repo.Close()
			store.Close()
		}
	}()

	return nil
}
