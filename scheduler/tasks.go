package scheduler

import (
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/plakar/agent"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/subcommands/backup"
	"github.com/PlakarKorp/plakar/subcommands/check"
	"github.com/PlakarKorp/plakar/subcommands/maintenance"
	"github.com/PlakarKorp/plakar/subcommands/restore"
	"github.com/PlakarKorp/plakar/subcommands/rm"
	"github.com/PlakarKorp/plakar/subcommands/sync"
)

func (s *Scheduler) backupTask(taskset Task, task BackupConfig) {
	backupSubcommand := &backup.Backup{}
	backupSubcommand.Flags = subcommands.AgentSupport
	backupSubcommand.Silent = true
	backupSubcommand.Job = taskset.Name
	backupSubcommand.Path = task.Path
	backupSubcommand.Quiet = true
	backupSubcommand.Opts = make(map[string]string)
	backupSubcommand.PreHook = task.PreHook
	backupSubcommand.PostHook = task.PostHook
	if task.Check.Enabled {
		backupSubcommand.OptCheck = true
	}

	rmSubcommand := &rm.Rm{}
	rmSubcommand.Apply = true
	rmSubcommand.Flags = subcommands.AgentSupport
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

			if retval, err := agent.ExecuteRPC(s.ctx, []string{"backup"}, backupSubcommand, storeConfig); err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error creating backup: %s", err)
				continue
			}

			if task.Retention != 0 {
				rmSubcommand.LocateOptions.Filters.Before = time.Now().Add(-task.Retention)
				if retval, err := agent.ExecuteRPC(s.ctx, []string{"rm"}, rmSubcommand, storeConfig); err != nil || retval != 0 {
					s.ctx.GetLogger().Error("Error removing obsolete backups: %s", err)
					continue
				}
			}
		}
	}
}

func (s *Scheduler) checkTask(taskset Task, task CheckConfig) {
	checkSubcommand := &check.Check{}
	checkSubcommand.Flags = subcommands.AgentSupport
	checkSubcommand.LocateOptions = locate.NewDefaultLocateOptions(
		locate.WithJob(taskset.Name),
		locate.WithLatest(task.Latest),
	)
	checkSubcommand.Silent = true
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

			retval, err := agent.ExecuteRPC(s.ctx, []string{"check"}, checkSubcommand, storeConfig)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing check: %s", err)
			}
		}
	}
}

func (s *Scheduler) restoreTask(taskset Task, task RestoreConfig) {
	restoreSubcommand := &restore.Restore{}
	restoreSubcommand.Flags = subcommands.AgentSupport
	restoreSubcommand.OptJob = taskset.Name
	restoreSubcommand.Target = task.Target
	restoreSubcommand.Silent = true
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

			retval, err := agent.ExecuteRPC(s.ctx, []string{"restore"}, restoreSubcommand, storeConfig)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing restore: %s", err)
			}
		}
	}
}

func (s *Scheduler) syncTask(taskset Task, task SyncConfig) {
	syncSubcommand := &sync.Sync{}
	syncSubcommand.Flags = subcommands.AgentSupport
	syncSubcommand.PeerRepositoryLocation = task.Peer
	if task.Direction == SyncDirectionTo {
		syncSubcommand.Direction = "to"
	} else if task.Direction == SyncDirectionFrom {
		syncSubcommand.Direction = "from"
	} else if task.Direction == SyncDirectionWith {
		syncSubcommand.Direction = "with"
	} else {
		//return fmt.Errorf("invalid sync direction: %s", task.Direction)
		s.ctx.Cancel()
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

			retval, err := agent.ExecuteRPC(s.ctx, []string{"sync"}, syncSubcommand, storeConfig)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("sync: %s", err)
			} else {
				s.ctx.GetLogger().Info("sync: synchronization succeeded")
			}
		}
	}
}

func (s *Scheduler) maintenanceTask(task MaintenanceConfig) {
	maintenanceSubcommand := &maintenance.Maintenance{}
	maintenanceSubcommand.Flags = subcommands.AgentSupport
	rmSubcommand := &rm.Rm{}
	rmSubcommand.Apply = true
	rmSubcommand.Flags = subcommands.AgentSupport
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

			retval, err := agent.ExecuteRPC(s.ctx, []string{"maintenance"}, maintenanceSubcommand, storeConfig)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing maintenance: %s", err)
				continue
			} else {
				s.ctx.GetLogger().Info("maintenance of repository %s succeeded", task.Repository)
			}

			if task.Retention != 0 {
				rmSubcommand.LocateOptions.Filters.Before = time.Now().Add(-task.Retention)
				retval, err := agent.ExecuteRPC(s.ctx, []string{"rm"}, rmSubcommand, storeConfig)
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
