package reporting

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/services"
)

const PLAKAR_API_URL = "https://api.plakar.io/v1/reporting/reports"

// reportPool provides a pool for Report objects to reduce allocations
var reportPool = sync.Pool{
	New: func() interface{} {
		return &Report{}
	},
}

type Emitter interface {
	Emit(ctx context.Context, report *Report) error
}

type Reporter struct {
	ctx             *appcontext.AppContext
	reportCount     atomic.Int32
	reports         chan *Report
	stop            chan any
	done            chan any
	emitter         Emitter
	emitter_timeout time.Time
}

func NewReporter(ctx *appcontext.AppContext) *Reporter {
	r := &Reporter{
		ctx:     ctx,
		reports: make(chan *Report, 100),
		stop:    make(chan any),
		done:    make(chan any),
	}

	go func() {
		var rp *Report
		for {
			select {
			case <-ctx.Done():
				goto done
			case <-r.stop:
				goto done
			case rp = <-r.reports:
				r.Process(rp)
				r.reportCount.Add(-1)
			}
		}
	done:
		// drain remaining reports
		for r.reportCount.Load() != 0 {
			rp = <-r.reports
			r.Process(rp)
			r.reportCount.Add(-1)
		}
		close(r.reports)
		close(r.done)
	}()

	return r
}

func (reporter *Reporter) Process(report *Report) {
	// Return report to pool after processing
	defer func() {
		// Clear sensitive data before returning to pool
		report.logger = nil
		report.reporter = nil
		reportPool.Put(report)
	}()

	if report.ignore {
		return
	}

	attempts := 3
	backoffUnit := time.Minute
	for i := range attempts {
		err := reporter.getEmitter().Emit(reporter.ctx, report)
		if err == nil {
			return
		}
		reporter.ctx.GetLogger().Warn("failed to emit report: %s", err)
		time.Sleep(backoffUnit << i)
	}
	reporter.ctx.GetLogger().Error("failed to emit report after %d attempts", attempts)
}

func (reporter *Reporter) StopAndWait() {
	close(reporter.stop)
	<-reporter.done
}

func (reporter *Reporter) getEmitter() Emitter {
	// Check if emitter should be reloaded
	if reporter.emitter != nil && reporter.emitter_timeout.After(time.Now()) {
		return reporter.emitter
	}

	// By default do nothing
	reporter.emitter = &NullEmitter{}
	reporter.emitter_timeout = time.Now().Add(time.Minute)

	// Check if user is logged
	token, _ := reporter.ctx.GetCookies().GetAuthToken()
	if token == "" {
		return reporter.emitter
	}

	sc := services.NewServiceConnector(reporter.ctx, token)
	enabled, err := sc.GetServiceStatus("alerting")
	if err != nil {
		reporter.ctx.GetLogger().Warn("failed to check alerting service: %v", err)
		return reporter.emitter
	}
	if !enabled {
		return reporter.emitter
	}

	// User is logged and alerting service is enabled
	url := os.Getenv("PLAKAR_API_URL")
	if url == "" {
		url = PLAKAR_API_URL
	}

	reporter.emitter = &HttpEmitter{
		url:   url,
		token: token,
	}
	return reporter.emitter
}

func (reporter *Reporter) NewReport() *Report {
	reporter.reportCount.Add(1)

	// Get report from pool to reduce allocations
	report := reportPool.Get().(*Report)

	// Reset report fields
	report.logger = reporter.ctx.GetLogger()
	report.reporter = reporter.reports
	report.ignore = false
	report.Timestamp = time.Time{}
	report.Task = nil
	report.Repository = nil
	report.Snapshot = nil

	return report
}

func (report *Report) SetIgnore() {
	report.ignore = true
}

func (report *Report) TaskStart(kind string, name string) {
	if report.Task != nil {
		report.logger.Warn("already in a task")
	}
	report.Task = &ReportTask{
		StartTime: time.Now(),
		Type:      kind,
		Name:      name,
	}
}

func (report *Report) WithRepositoryName(name string) {
	if report.Repository != nil {
		report.logger.Warn("already has a repository")
	}
	report.Repository = &ReportRepository{
		Name: name,
	}
}

func (report *Report) WithRepository(repository *repository.Repository) {
	report.repo = repository
	configuration := repository.Configuration()
	report.Repository.Storage = configuration
}

func (report *Report) WithSnapshotID(snapshotId objects.MAC) {
	snap, err := snapshot.Load(report.repo, snapshotId)
	if err != nil {
		report.logger.Warn("failed to load snapshot: %s", err)
		return
	}
	report.WithSnapshot(snap)
	snap.Close()
}

func (report *Report) WithSnapshot(snapshot *snapshot.Snapshot) {
	if report.Snapshot != nil {
		report.logger.Warn("already has a snapshot")
	}
	report.Snapshot = &ReportSnapshot{
		Header: *snapshot.Header,
	}
}

func (report *Report) TaskDone() {
	report.taskEnd(StatusOK, 0, "")
}

func (report *Report) TaskWarning(errorMessage string, args ...interface{}) {
	report.taskEnd(StatusWarning, 0, errorMessage, args...)
}

func (report *Report) TaskFailed(errorCode TaskErrorCode, errorMessage string, args ...interface{}) {
	report.taskEnd(StatusFailed, errorCode, errorMessage, args...)
}

func (report *Report) taskEnd(status TaskStatus, errorCode TaskErrorCode, errorMessage string, args ...interface{}) {
	report.Task.Status = status
	report.Task.ErrorCode = errorCode
	if len(args) == 0 {
		report.Task.ErrorMessage = errorMessage
	} else {
		report.Task.ErrorMessage = fmt.Sprintf(errorMessage, args...)
	}
	report.Task.Duration = time.Since(report.Task.StartTime)
	report.Publish()
}

func (report *Report) Publish() {
	report.Timestamp = time.Now()
	report.reporter <- report
}
