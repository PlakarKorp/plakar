package scheduler

import (
	"errors"
	"fmt"
	"time"

	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/plakar/agent"
	"github.com/PlakarKorp/plakar/subcommands/backup"
	"github.com/PlakarKorp/plakar/subcommands/check"
	"github.com/PlakarKorp/plakar/subcommands/maintenance"
	"github.com/PlakarKorp/plakar/subcommands/restore"
	"github.com/PlakarKorp/plakar/subcommands/rm"
	"github.com/PlakarKorp/plakar/subcommands/sync"
	ptask "github.com/PlakarKorp/plakar/task"
)

// Needs to go.
var ErrCantUnlock = errors.New("failed to unlock repository")

func (s *Scheduler) backupTask(taskset Task, task BackupConfig) {
	backupSubcommand := &backup.Backup{}
	backupSubcommand.Tags = task.Tags
	backupSubcommand.Job = taskset.Name
	backupSubcommand.Path = task.Path
	backupSubcommand.Opts = make(map[string]string)
	backupSubcommand.PreHook = task.PreHook
	backupSubcommand.PostHook = task.PostHook
	backupSubcommand.FailHook = task.FailHook
	if task.Check.Enabled {
		backupSubcommand.OptCheck = true
	}

	rmSubcommand := &rm.Rm{}
	rmSubcommand.Apply = true
	rmSubcommand.LocateOptions = locate.NewDefaultLocateOptions(locate.WithJob(task.Name))

	for {
		tick := time.After(task.Interval)
		select {
		case <-s.ctx.Done():
			return
		case <-tick:

			var excludes []string
			if task.IgnoreFile != "" {
				lines, err := backup.LoadIgnoreFile(task.IgnoreFile)
				if err != nil {
					s.ctx.GetLogger().Error("Failed to load ignore file: %s", err)
					continue
				}
				for _, line := range lines {
					excludes = append(excludes, line)
				}
			}
			for _, line := range task.Ignore {
				excludes = append(excludes, line)
			}
			backupSubcommand.Excludes = excludes

			storeConfig, err := s.ctx.Config.GetRepository(taskset.Repository)
			if err != nil {
				s.ctx.GetLogger().Error("Error getting repository config: %s", err)
				continue
			}

			repo, err := s.makeRepository(storeConfig)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				continue
			}

			if _, err := agent.RebuildStateFromCached(s.ctx, repo.Configuration().RepositoryID, storeConfig); err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				continue
			}

			retval, err := ptask.RunCommand(s.ctx, backupSubcommand, repo, "@scheduler")
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error creating backup: %s", err)
				continue
			}

			if task.Retention != 0 {
				if _, err := agent.RebuildStateFromCached(s.ctx, repo.Configuration().RepositoryID, storeConfig); err != nil {
					s.ctx.GetLogger().Error("Error opening repository: %s", err)
					continue
				}

				rmSubcommand.LocateOptions.Filters.Before = time.Now().Add(-task.Retention)
				retval, err := ptask.RunCommand(s.ctx, rmSubcommand, repo, "@scheduler")
				if err != nil || retval != 0 {
					s.ctx.GetLogger().Error("Error removing obsolete backups: %s", err)
					continue
				} else {
					s.ctx.GetLogger().Info("Retention purge succeeded")
				}
			}
		}
	}
}

func (s *Scheduler) checkTask(taskset Task, task CheckConfig) {
	checkSubcommand := &check.Check{}
	checkSubcommand.LocateOptions = locate.NewDefaultLocateOptions(
		locate.WithJob(taskset.Name),
		locate.WithLatest(task.Latest),
	)
	if task.Path != "" {
		checkSubcommand.Snapshots = []string{":" + task.Path}
	}

	for {
		tick := time.After(task.Interval)
		select {
		case <-s.ctx.Done():
			return
		case <-tick:
			storeConfig, err := s.ctx.Config.GetRepository(taskset.Repository)
			if err != nil {
				s.ctx.GetLogger().Error("Error getting repository config: %s", err)
				continue
			}

			repo, err := s.makeRepository(storeConfig)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				continue
			}

			if _, err := agent.RebuildStateFromCached(s.ctx, repo.Configuration().RepositoryID, storeConfig); err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				continue
			}

			retval, err := ptask.RunCommand(s.ctx, checkSubcommand, repo, "@scheduler")
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing check: %s", err)
				continue
			}
		}
	}
}

func (s *Scheduler) restoreTask(taskset Task, task RestoreConfig) {
	restoreSubcommand := &restore.Restore{}
	restoreSubcommand.OptJob = taskset.Name
	restoreSubcommand.Target = task.Target
	if task.Path != "" {
		restoreSubcommand.Snapshots = []string{":" + task.Path}
	}

	for {
		tick := time.After(task.Interval)
		select {
		case <-s.ctx.Done():
			return
		case <-tick:
			storeConfig, err := s.ctx.Config.GetRepository(taskset.Repository)
			if err != nil {
				s.ctx.GetLogger().Error("Error getting repository config: %s", err)
				continue
			}

			repo, err := s.makeRepository(storeConfig)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				continue
			}

			if _, err := agent.RebuildStateFromCached(s.ctx, repo.Configuration().RepositoryID, storeConfig); err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				continue
			}

			retval, err := ptask.RunCommand(s.ctx, restoreSubcommand, repo, "@scheduler")
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing restore: %s", err)
				continue
			}
		}
	}
}

func (s *Scheduler) syncTask(taskset Task, task SyncConfig) {
	syncSubcommand := &sync.Sync{}
	syncSubcommand.PeerRepositoryLocation = task.Peer
	if task.Direction == SyncDirectionTo {
		syncSubcommand.Direction = "to"
	} else if task.Direction == SyncDirectionFrom {
		syncSubcommand.Direction = "from"
	} else if task.Direction == SyncDirectionWith {
		syncSubcommand.Direction = "with"
	} else {
		s.ctx.Cancel(fmt.Errorf("invalid sync direction: %s", task.Direction))
		return
	}
	//	if taskset.Repository.Passphrase != "" {
	//		syncSubcommand.DestinationRepositorySecret = []byte(taskset.Repository.Passphrase)
	//		_ = syncSubcommand.DestinationRepositorySecret

	//	syncSubcommand.OptJob = taskset.Name
	//	syncSubcommand.Target = task.Target
	//	syncSubcommand.Silent = true

	for {
		tick := time.After(task.Interval)
		select {
		case <-s.ctx.Done():
			return
		case <-tick:
			storeConfig, err := s.ctx.Config.GetRepository(taskset.Repository)
			if err != nil {
				s.ctx.GetLogger().Error("Error getting repository config: %s", err)
				continue
			}

			repo, err := s.makeRepository(storeConfig)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				continue
			}

			if _, err := agent.RebuildStateFromCached(s.ctx, repo.Configuration().RepositoryID, storeConfig); err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				continue
			}

			retval, err := ptask.RunCommand(s.ctx, syncSubcommand, repo, "@scheduler")
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing sync: %s", err)
				continue
			} else {
				s.ctx.GetLogger().Info("sync: synchronization succeeded")
			}
		}
	}
}

func (s *Scheduler) maintenanceTask(task MaintenanceConfig) {
	maintenanceSubcommand := &maintenance.Maintenance{}
	rmSubcommand := &rm.Rm{}
	rmSubcommand.Apply = true
	rmSubcommand.LocateOptions = locate.NewDefaultLocateOptions(locate.WithJob("maintenance"))

	for {
		tick := time.After(task.Interval)
		select {
		case <-s.ctx.Done():
			return
		case <-tick:
			storeConfig, err := s.ctx.Config.GetRepository(task.Repository)
			if err != nil {
				s.ctx.GetLogger().Error("Error getting repository config: %s", err)
				continue
			}

			repo, err := s.makeRepository(storeConfig)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				continue
			}

			if _, err := agent.RebuildStateFromCached(s.ctx, repo.Configuration().RepositoryID, storeConfig); err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				continue
			}

			retval, err := ptask.RunCommand(s.ctx, maintenanceSubcommand, repo, "@scheduler")
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing maintenance: %s", err)
				continue
			} else {
				s.ctx.GetLogger().Info("maintenance of repository %s succeeded", task.Repository)
			}

			if task.Retention != 0 {
				if _, err := agent.RebuildStateFromCached(s.ctx, repo.Configuration().RepositoryID, storeConfig); err != nil {
					s.ctx.GetLogger().Error("Error opening repository: %s", err)
					continue
				}

				rmSubcommand.LocateOptions.Filters.Before = time.Now().Add(-task.Retention)
				retval, err := ptask.RunCommand(s.ctx, rmSubcommand, repo, "@scheduler")
				if err != nil || retval != 0 {
					s.ctx.GetLogger().Error("Error removing obsolete backups: %s", err)
					continue
				} else {
					s.ctx.GetLogger().Info("Retention purge succeeded")
				}
			}
		}
	}
}

func (s *Scheduler) makeRepository(storeConfig map[string]string) (*repository.Repository, error) {
	var serializedConfig []byte
	store, serializedConfig, err := storage.Open(s.ctx.GetInner(), storeConfig)
	if err != nil {
		return nil, err
	}

	repoConfig, err := storage.NewConfigurationFromWrappedBytes(serializedConfig)
	if err != nil {
		return nil, err
	}

	if repoConfig.Version != versioning.FromString(storage.VERSION) {
		return nil, err
	}

	if err := s.setupEncryption(repoConfig); err != nil {
		return nil, err
	}

	// Actual rebuild is done by cached.
	repo, err := repository.NewNoRebuild(s.ctx.GetInner(), s.ctx.GetSecret(), store, serializedConfig)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func (s *Scheduler) setupEncryption(config *storage.Configuration) error {
	if config.Encryption == nil {
		return nil
	}

	if s.ctx.KeyFromFile != "" {
		secret := []byte(s.ctx.KeyFromFile)
		key, err := encryption.DeriveKey(config.Encryption.KDFParams,
			secret)
		if err != nil {
			return err
		}

		if !encryption.VerifyCanary(config.Encryption, key) {
			return ErrCantUnlock
		}
		s.ctx.SetSecret(key)
		return nil
	}

	return ErrCantUnlock
}
