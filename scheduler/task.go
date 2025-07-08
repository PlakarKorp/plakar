package scheduler

import (
	"fmt"
	"strings"

	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/reporting"
	"github.com/PlakarKorp/plakar/services"
)

type Task interface {
	Run(ctx *TaskContext)
	Event(ctx *TaskContext, event events.Event)
	String() string
}

type TaskBase struct {
	Type       string
	Repository string
	Reporting  bool
}

type TaskContext struct {
	JobName    string
	AppContext *appcontext.AppContext
	Store      storage.Store
	Repository *repository.Repository
	Reporter   *reporting.Reporter
	Done       chan struct{}
}

func (ctx *TaskContext) GetLogger() *logging.Logger {
	return ctx.AppContext.GetLogger()
}

func (ctx *TaskContext) Clear() {
	if ctx.Repository != nil {
		_ = ctx.Repository.Close()
		ctx.Repository = nil
	}
	if ctx.Store != nil {
		_ = ctx.Store.Close()
		ctx.Store = nil
	}
}

func (task *TaskBase) LoadRepository(ctx *TaskContext) error {
	storeConfig, err := ctx.AppContext.Config.GetRepository(task.Repository)
	if err != nil {
		return fmt.Errorf("unable to get repository configuration: %w", err)
	}

	store, config, err := storage.Open(ctx.AppContext.GetInner(), storeConfig)
	if err != nil {
		return fmt.Errorf("unable to open storage: %w", err)
	}

	repoConfig, err := storage.NewConfigurationFromWrappedBytes(config)
	if err != nil {
		store.Close()
		return fmt.Errorf("unable to read repository configuration: %w", err)
	}

	if repoConfig.Version != versioning.FromString(storage.VERSION) {
		store.Close()
		return fmt.Errorf("incompatible repository version: %s != %s", repoConfig.Version, storage.VERSION)
	}

	var key []byte
	if passphrase, ok := storeConfig["passphrase"]; ok {
		key, err = encryption.DeriveKey(repoConfig.Encryption.KDFParams, []byte(passphrase))
		if err != nil {
			store.Close()
			return fmt.Errorf("error deriving key: %w", err)
		}
		if !encryption.VerifyCanary(repoConfig.Encryption, key) {
			store.Close()
			return fmt.Errorf("invalid passphrase")
		}
	}

	repo, err := repository.New(ctx.AppContext.GetInner(), key, store, config)
	if err != nil {
		store.Close()
		return fmt.Errorf("unable to open repository: %w", err)
	}

	ctx.Repository = repo
	ctx.Store = store
	ctx.Reporter = task.NewReporter(ctx)

	return nil
}

func (t *TaskBase) NewReporter(ctx *TaskContext) *reporting.Reporter {
	doReport := false
	if t.Reporting {
		doReport = true
		authToken, err := ctx.AppContext.GetAuthToken(ctx.Repository.Configuration().RepositoryID)
		if err != nil || authToken == "" {
			doReport = false
		} else {
			sc := services.NewServiceConnector(ctx.AppContext, authToken)
			enabled, err := sc.GetServiceStatus("alerting")
			if err != nil || !enabled {
				doReport = false
			}
		}
	}

	reporter := reporting.NewReporter(ctx.AppContext, doReport, ctx.Repository, ctx.GetLogger())
	reporter.TaskStart(strings.ToLower(t.Type), ctx.JobName)
	reporter.WithRepositoryName(t.Repository)
	reporter.WithRepository(ctx.Repository)
	return reporter
}
