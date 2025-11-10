package reporting

import (
	"sync"

	"github.com/PlakarKorp/kloset/logging"
	errorspkg "github.com/PlakarKorp/plakar/internal/errors"
)

const (
	// ErrEncodeReport is emitted when marshalling a report fails.
	ErrEncodeReport errorspkg.Code = "report.encode"
	// ErrBuildRequest indicates we failed to prepare the HTTP request.
	ErrBuildRequest errorspkg.Code = "report.request_build"
	// ErrDoRequest indicates the HTTP call itself failed.
	ErrDoRequest errorspkg.Code = "report.request_do"
	// ErrBadStatus indicates a non-success HTTP status was returned.
	ErrBadStatus errorspkg.Code = "report.bad_status"
	// ErrEmitReport represents a failure while the reporter pipeline emits a report.
	ErrEmitReport errorspkg.Code = "report.emit"
)

var (
	manager         = errorspkg.NewManager()
	loggerObservers sync.Map
)

// Errors exposes the reporting error manager.
func Errors() *errorspkg.Manager { return manager }

// registerLoggerObserver ensures we only attach a logger once per reporter.
func registerLoggerObserver(logger *logging.Logger) {
	if logger == nil {
		return
	}
	if _, loaded := loggerObservers.LoadOrStore(logger, struct{}{}); loaded {
		return
	}

	manager.Register(func(e *errorspkg.Error) {
		logger.Warn("reporting error: %s", e.Format())
	})
}
