package reporting

import (
	"fmt"
	"os"
	"time"

	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot/header"
	"github.com/PlakarKorp/plakar/storage"
)

type TaskStatus string
type TaskErrorCode uint32

const (
	StatusOK      TaskStatus = "OK"
	StatusWarning TaskStatus = "WARNING"
	StatusFailed  TaskStatus = "FAILURE"
)

type ReportSnapshot struct {
	Header *header.Header
}

type ReportRepository struct {
	Name    string
	Storage *storage.Configuration
}

type ReportTask struct {
	Type         string
	Name         string
	StartTime    time.Time
	Duration     time.Duration
	Status       TaskStatus
	ErrorCode    TaskErrorCode
	ErrorMessage string
}

type Report struct {
	Timestamp  time.Time
	Task       *ReportTask
	Repository *ReportRepository
	Snapshot   *ReportSnapshot
}

type Emitter interface {
	Emit(report Report, logger *logging.Logger)
}

type Reporter struct {
	logger            *logging.Logger
	emitter           Emitter
	currentTask       *ReportTask
	currentRepository *ReportRepository
	currentSnapshot   *ReportSnapshot
}

func NewReporter(emitterString string, logger *logging.Logger) *Reporter {
	if logger == nil {
		logger = logging.NewLogger(os.Stdout, os.Stderr)
	}

	var emitter Emitter

	if emitterString == "" {
		emitter = &NullEmitter{}
	} else {
		emitter = &HttpEmitter{
			url:   emitterString,
			retry: 3,
		}
	}

	return &Reporter{
		logger:  logger,
		emitter: emitter,
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
	configuration := repository.Configuration()
	reporter.currentRepository.Storage = &configuration
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
	reporter.currentSnapshot = nil
	reporter.currentRepository = nil
	reporter.emitter.Emit(report, reporter.logger)
}
