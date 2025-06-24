package scheduler

import (
	"sync"
	"time"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/reporting"
	"github.com/PlakarKorp/plakar/services"
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
}

func (s *Scheduler) NewTaskReporter(ctx *appcontext.AppContext, repo *repository.Repository, taskType, taskName, repoName string) *reporting.Reporter {
	doReport := true
	authToken, err := s.ctx.GetAuthToken(repo.Configuration().RepositoryID)
	if err != nil || authToken == "" {
		doReport = false
	} else {
		sc := services.NewServiceConnector(s.ctx, authToken)
		enabled, err := sc.GetServiceStatus("alerting")
		if err != nil || !enabled {
			doReport = false
		}
	}
	reporter := reporting.NewReporter(ctx, doReport, repo, s.ctx.GetLogger())
	reporter.TaskStart(taskType, taskName)
	reporter.WithRepositoryName(repoName)
	reporter.WithRepository(repo)
	return reporter
}
