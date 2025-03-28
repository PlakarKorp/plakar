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
	"github.com/PlakarKorp/plakar/encryption"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/PlakarKorp/plakar/versioning"
)

func loadRepository(newCtx *appcontext.AppContext, name string) (*repository.Repository, storage.Store, error) {
	storeConfig, err := newCtx.Config.GetRepository(name)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get repository configuration: %w", err)
	}

	store, config, err := storage.Open(storeConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to open storage: %w", err)
	}

	repoConfig, err := storage.NewConfigurationFromWrappedBytes(config)
	if err != nil {
		store.Close()
		return nil, nil, fmt.Errorf("unable to read repository configuration: %w", err)
	}

	if repoConfig.Version != versioning.FromString(storage.VERSION) {
		store.Close()
		return nil, nil, fmt.Errorf("incompatible repository version: %s != %s", repoConfig.Version, storage.VERSION)
	}

	if passphrase, ok := storeConfig["passphrase"]; ok {
		key, err := encryption.DeriveKey(repoConfig.Encryption.KDFParams, []byte(passphrase))
		if err != nil {
			store.Close()
			return nil, nil, fmt.Errorf("error deriving key: %w", err)
		}
		if !encryption.VerifyCanary(repoConfig.Encryption, key) {
			store.Close()
			return nil, nil, fmt.Errorf("invalid passphrase")
		}
		newCtx.SetSecret(key)
	}

	repo, err := repository.New(newCtx, store, config)
	if err != nil {
		store.Close()
		return nil, store, fmt.Errorf("unable to open repository: %w", err)
	}
	return repo, store, nil
}

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

	backupSubcommand := &backup.Backup{}
	backupSubcommand.Silent = true
	backupSubcommand.Job = taskset.Name
	backupSubcommand.Path = task.Path
	backupSubcommand.Quiet = true
	if task.Check.Enabled {
		backupSubcommand.OptCheck = true
	}

	rmSubcommand := &rm.Rm{}
	rmSubcommand.OptJob = task.Name

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		reporter := s.NewReporter()
		firstRun := true
		for {
			if firstRun {
				firstRun = false
			} else {
				time.Sleep(interval)
			}

			reporter.TaskStart("backup", taskset.Name)
			reporter.WithRepositoryName(taskset.Repository)

			newCtx := appcontext.NewAppContextFrom(s.ctx)
			repo, store, err := loadRepository(newCtx, taskset.Repository)
			if err != nil {
				s.ctx.GetLogger().Error("Error loading repository: %s", err)
				reporter.TaskFailed(1, "Error loading repository: %s", err)
				continue
			}
			reporter.WithRepository(repo)

			backupCtx := appcontext.NewAppContextFrom(newCtx)
			retval, err := backupSubcommand.Execute(backupCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error creating backup: %s", err)
				reporter.TaskFailed(1, "Error creating backup: retval=%d, err=%s", retval, err)
				backupCtx.Close()
				goto close
			}
			reporter.WithSnapshotID(repo, backupCtx.SnapshotID)

			backupCtx.Close()

			if task.Retention != "" {
				rmCtx := appcontext.NewAppContextFrom(newCtx)
				rmSubcommand.OptBefore = time.Now().Add(-retention)
				retval, err = rmSubcommand.Execute(rmCtx, repo)
				if err != nil || retval != 0 {
					reporter.TaskWarning("Error removing obsolete backups: retval=%d, err=%s", retval, err)
					s.ctx.GetLogger().Error("Error removing obsolete backups: %s", err)
				} else {
					reporter.TaskDone()
				}
				rmCtx.Close()
			} else {
				reporter.TaskDone()
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

	checkSubcommand := &check.Check{}
	checkSubcommand.OptJob = taskset.Name
	checkSubcommand.OptLatest = task.Latest
	checkSubcommand.Silent = true
	if task.Path != "" {
		checkSubcommand.Snapshots = []string{":" + task.Path}
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		reporter := s.NewReporter()
		firstRun := true
		for {
			if firstRun {
				firstRun = false
			} else {
				time.Sleep(interval)
			}

			reporter.TaskStart("check", taskset.Name)
			reporter.WithRepositoryName(taskset.Repository)

			newCtx := appcontext.NewAppContextFrom(s.ctx)

			repo, store, err := loadRepository(newCtx, taskset.Repository)
			if err != nil {
				s.ctx.GetLogger().Error("Error loading repository: %s", err)
				reporter.TaskFailed(1, "Error loading repository: %s", err)
				continue
			}
			reporter.WithRepository(repo)

			retval, err := checkSubcommand.Execute(newCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing check: %s", err)
				reporter.TaskFailed(1, "Error executing check: retval=%d, err=%s", retval, err)
			} else {
				reporter.TaskDone()
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

	restoreSubcommand := &restore.Restore{}
	restoreSubcommand.OptJob = taskset.Name
	restoreSubcommand.Target = task.Target
	restoreSubcommand.Silent = true
	if task.Path != "" {
		restoreSubcommand.Snapshots = []string{":" + task.Path}
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		reporter := s.NewReporter()
		firstRun := true
		for {
			if firstRun {
				firstRun = false
			} else {
				time.Sleep(interval)
			}

			reporter.TaskStart("restore", taskset.Name)
			reporter.WithRepositoryName(taskset.Repository)

			newCtx := appcontext.NewAppContextFrom(s.ctx)

			repo, store, err := loadRepository(newCtx, taskset.Repository)
			if err != nil {
				s.ctx.GetLogger().Error("Error loading repository: %s", err)
				reporter.TaskFailed(1, "Error loading repository: %s", err)
				continue
			}
			reporter.WithRepository(repo)

			retval, err := restoreSubcommand.Execute(newCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing restore: %s", err)
				reporter.TaskFailed(1, "Error executing restore: retval=%d, err=%s", retval, err)
			} else {
				reporter.TaskDone()
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

	syncSubcommand := &sync.Sync{}
	syncSubcommand.PeerRepositoryLocation = task.Peer
	if task.Direction == SyncDirectionTo {
		syncSubcommand.Direction = "to"
	} else if task.Direction == SyncDirectionFrom {
		syncSubcommand.Direction = "from"
	} else if task.Direction == SyncDirectionWith {
		syncSubcommand.Direction = "with"
	} else {
		return fmt.Errorf("invalid sync direction: %s", task.Direction)
	}
	//	if taskset.Repository.Passphrase != "" {
	//		syncSubcommand.DestinationRepositorySecret = []byte(taskset.Repository.Passphrase)
	//		_ = syncSubcommand.DestinationRepositorySecret

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

			newCtx := appcontext.NewAppContextFrom(s.ctx)

			repo, store, err := loadRepository(newCtx, taskset.Repository)
			if err != nil {
				s.ctx.GetLogger().Error("Error loading repository: %s", err)
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

	maintenanceSubcommand := &maintenance.Maintenance{}
	rmSubcommand := &rm.Rm{}

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

			newCtx := appcontext.NewAppContextFrom(s.ctx)

			repo, store, err := loadRepository(newCtx, task.Repository)
			if err != nil {
				s.ctx.GetLogger().Error("Error loading repository: %s", err)
				continue
			}

			retval, err := maintenanceSubcommand.Execute(newCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing maintenance: %s", err)
			} else {
				s.ctx.GetLogger().Info("maintenance of repository %s succeeded", task.Repository)
			}

			if task.Retention != "" {
				rmCtx := appcontext.NewAppContextFrom(newCtx)
				rmSubcommand.OptBefore = time.Now().Add(-retention)
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
