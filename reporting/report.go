package reporting

import (
	"time"

	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot/header"
	"github.com/PlakarKorp/kloset/storage"
)

type TaskStatus string
type TaskErrorCode uint32

const (
	StatusOK      TaskStatus = "OK"
	StatusWarning TaskStatus = "WARNING"
	StatusFailed  TaskStatus = "FAILURE"
)

type ReportSnapshot struct {
	header.Header
}

type ReportRepository struct {
	Name    string                `json:"name"`
	Storage storage.Configuration `json:"storage"`
}

type ReportTask struct {
	Type         string        `json:"type"`
	Name         string        `json:"name"`
	StartTime    time.Time     `json:"start_time"`
	Duration     time.Duration `json:"duration"`
	Status       TaskStatus    `json:"status"`
	ErrorCode    TaskErrorCode `json:"error_code"`
	ErrorMessage string        `json:"error_message"`
}

type Report struct {
	Timestamp  time.Time         `json:"timestamp"`
	Task       *ReportTask       `json:"report_task,omitempty"`
	Repository *ReportRepository `json:"report_repository,omitempty"`
	Snapshot   *ReportSnapshot   `json:"report_snapshot,omitempty"`

	repo     *repository.Repository `json:"-"`
	logger   *logging.Logger        `json:"-"`
	reporter chan *Report           `json:"-"`
	ignore   bool                   `json:"-"`
}
