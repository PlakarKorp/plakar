package reporting

import (
	"fmt"
	"os"
	"time"

	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
)

const PLAKAR_API_URL = "https://api.plakar.io/v1/reporting/reports"

type Emitter interface {
	Emit(report Report, logger *logging.Logger)
}

type Reporter struct {
	repository        *repository.Repository
	logger            *logging.Logger
	emitter           Emitter
	currentTask       *ReportTask
	currentRepository *ReportRepository
	currentSnapshot   *ReportSnapshot
}

func NewReporter(ctx *appcontext.AppContext, reporting bool, repository *repository.Repository, logger *logging.Logger) *Reporter {
	if logger == nil {
		logger = logging.NewLogger(os.Stdout, os.Stderr)
	}

	var emitter Emitter

	if !reporting {
		emitter = &NullEmitter{}
	} else {

		url := os.Getenv("PLAKAR_API_URL")
		if url == "" {
			url = PLAKAR_API_URL
		}

		var token string

		token, err := ctx.GetCookies().GetAuthToken()
		if err != nil {
			logger.Warn("cannot get auth token")
		}

		emitter = &HttpEmitter{
			url:   url,
			token: token,
			retry: 3,
		}
	}

	return &Reporter{
		repository: repository,
		logger:     logger,
		emitter:    emitter,
	}
}

func (reporter *Reporter) TaskStart(kind string, name string) {
	if reporter.currentTask != nil {
		reporter.logger.Warn("already in a task")
	}

	reporter.currentTask = &ReportTask{
		StartTime: time.Now(),
		Type:      kind,
		Name:      name,
	}
}

func (reporter *Reporter) WithRepositoryName(name string) {
	if reporter.currentRepository != nil {
		reporter.logger.Warn("already has a repository")
	}
	reporter.currentRepository = &ReportRepository{
		Name: name,
	}
}

func (reporter *Reporter) WithRepository(repository *repository.Repository) {
	reporter.repository = repository
	configuration := repository.Configuration()
	reporter.currentRepository.Storage = configuration
}

func (reporter *Reporter) WithSnapshotID(snapshotId objects.MAC) {
	snap, err := snapshot.Load(reporter.repository, snapshotId)
	if err != nil {
		reporter.logger.Warn("failed to load snapshot: %s", err)
		return
	}
	reporter.WithSnapshot(snap)
	snap.Close()
}

func (reporter *Reporter) WithSnapshot(snapshot *snapshot.Snapshot) {
	if reporter.currentSnapshot != nil {
		reporter.logger.Warn("already has a snapshot")
	}
	reporter.currentSnapshot = &ReportSnapshot{
		Header: *snapshot.Header,
	}
}

func (reporter *Reporter) TaskDone() {
	reporter.taskEnd(StatusOK, 0, "")
}

func (reporter *Reporter) TaskWarning(errorMessage string, args ...interface{}) {
	reporter.taskEnd(StatusWarning, 0, errorMessage, args...)
}

func (reporter *Reporter) TaskFailed(errorCode TaskErrorCode, errorMessage string, args ...interface{}) {
	reporter.taskEnd(StatusFailed, errorCode, errorMessage, args...)
}

func (reporter *Reporter) taskEnd(status TaskStatus, errorCode TaskErrorCode, errorMessage string, args ...interface{}) {
	reporter.currentTask.Status = status
	reporter.currentTask.ErrorCode = errorCode
	if len(args) == 0 {
		reporter.currentTask.ErrorMessage = errorMessage
	} else {
		reporter.currentTask.ErrorMessage = fmt.Sprintf(errorMessage, args...)
	}
	reporter.currentTask.Duration = time.Since(reporter.currentTask.StartTime)

	report := Report{
		Timestamp:  time.Now(),
		Task:       reporter.currentTask,
		Repository: reporter.currentRepository,
		Snapshot:   reporter.currentSnapshot,
	}

	reporter.currentTask = nil
	reporter.currentRepository = nil
	reporter.currentSnapshot = nil
	go reporter.emitter.Emit(report, reporter.logger)
}
