package scheduler

import (
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/reporting"
)

type Scheduler struct {
	config *Configuration
	ctx    *appcontext.AppContext
	wg     sync.WaitGroup
}

func stringToDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	return d, nil
}

func NewScheduler(ctx *appcontext.AppContext, config *Configuration) *Scheduler {
	return &Scheduler{
		ctx:    ctx,
		config: config,
		wg:     sync.WaitGroup{},
	}
}

func (s *Scheduler) Run() {
	for _, cleanupCfg := range s.config.Agent.Maintenance {
		err := s.maintenanceTask(cleanupCfg)
		if err != nil {
			s.ctx.GetLogger().Error("Error configuring maintenance task: %s", err)
		}
	}

	for _, tasksetCfg := range s.config.Agent.Tasks {
		if tasksetCfg.Backup != nil {
			err := s.backupTask(tasksetCfg, *tasksetCfg.Backup)
			if err != nil {
				s.ctx.GetLogger().Error("Error configuring backup task: %s", err)
			}
		}

		for _, checkCfg := range tasksetCfg.Check {
			err := s.checkTask(tasksetCfg, checkCfg)
			if err != nil {
				s.ctx.GetLogger().Error("Error configuring check task: %s", err)
			}
		}

		for _, restoreCfg := range tasksetCfg.Restore {
			err := s.restoreTask(tasksetCfg, restoreCfg)
			if err != nil {
				s.ctx.GetLogger().Error("Error configuring restore task: %s", err)
			}
		}

		for _, syncCfg := range tasksetCfg.Sync {
			err := s.syncTask(tasksetCfg, syncCfg)
			if err != nil {
				s.ctx.GetLogger().Error("Error configuring sync task: %s", err)
			}
		}
	}
	<-make(chan struct{})
}

func (s *Scheduler) NewReporter() *reporting.Reporter {
	return reporting.NewReporter(s.config.Agent.Reporting.Emitter, s.ctx.GetLogger())
}
