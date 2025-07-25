package scheduler

import (
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/reporting"
)

type Scheduler struct {
	config   *Configuration
	ctx      *appcontext.AppContext
	wg       sync.WaitGroup
	reporter *reporting.Reporter
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
	s.reporter = reporting.NewReporter(s.ctx)

	for _, cleanupCfg := range s.config.Agent.Maintenance {
		go s.maintenanceTask(cleanupCfg)
	}

	for _, tasksetCfg := range s.config.Agent.Tasks {
		if tasksetCfg.Backup != nil {
			go s.backupTask(tasksetCfg, *tasksetCfg.Backup)
		}

		for _, checkCfg := range tasksetCfg.Check {
			go s.checkTask(tasksetCfg, checkCfg)
		}

		for _, restoreCfg := range tasksetCfg.Restore {
			go s.restoreTask(tasksetCfg, restoreCfg)
		}

		for _, syncCfg := range tasksetCfg.Sync {
			go s.syncTask(tasksetCfg, syncCfg)
		}
	}

	<-s.ctx.Done()
	s.reporter.StopAndWait()
}
