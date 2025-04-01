package reporting

import (
	"time"

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
	Task       *ReportTask       `json:"Task,omitempty"`
	Repository *ReportRepository `json:"Repository,omitempty"`
	Snapshot   *ReportSnapshot   `json:"Snapshot,omitempty"`
}
