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
	Base() *TaskBase
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
	Done       chan struct{}

	reporter     *reporting.Reporter
	taskStatus   reporting.TaskStatus
	taskErrorMsg string
}

func (ctx *TaskContext) GetLogger() *logging.Logger {
	return ctx.AppContext.GetLogger()
}

func (ctx *TaskContext) ReportWarning(format string, args ...any) {
	if ctx.taskStatus == "" {
		ctx.taskStatus = reporting.StatusWarning
		if len(args) == 0 {
			ctx.taskErrorMsg = format
		} else {
			ctx.taskErrorMsg = fmt.Sprintf(format, args...)
		}
	}
}

func (ctx *TaskContext) ReportFailure(format string, args ...any) {
	if ctx.taskStatus != reporting.StatusFailed {
		ctx.taskStatus = reporting.StatusFailed
		if len(args) == 0 {
			ctx.taskErrorMsg = format
		} else {
			ctx.taskErrorMsg = fmt.Sprintf(format, args...)
		}
	}
}

func (ctx *TaskContext) Prepare(task Task) error {
	err := ctx.loadRepository(task.Base())
	if err != nil {
		ctx.GetLogger().Error("Error loading repository: %s", err)
		return err
	}

	ctx.reporter = ctx.newReporter(task.Base())
	return nil
}

func (ctx *TaskContext) Finalize() {
	if ctx.Repository != nil {
		_ = ctx.Repository.Close()
		ctx.Repository = nil
	}
	if ctx.Store != nil {
		_ = ctx.Store.Close()
		ctx.Store = nil
	}

	if ctx.reporter != nil {
		switch ctx.taskStatus {
		case reporting.StatusWarning:
			ctx.reporter.TaskWarning(ctx.taskErrorMsg)
		case reporting.StatusFailed:
			ctx.reporter.TaskFailed(1, ctx.taskErrorMsg)
		default:
			ctx.reporter.TaskDone()
		}
	}
}

func (ctx *TaskContext) loadRepository(task *TaskBase) error {
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
	return nil
}

func (ctx *TaskContext) newReporter(task *TaskBase) *reporting.Reporter {
	doReport := false
	if task.Reporting {
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
	reporter.TaskStart(strings.ToLower(task.Type), ctx.JobName)
	reporter.WithRepositoryName(task.Repository)
	reporter.WithRepository(ctx.Repository)
	return reporter
}
