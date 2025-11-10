package services

import (
	"sync"

	"github.com/PlakarKorp/kloset/logging"
	errorspkg "github.com/PlakarKorp/plakar/internal/errors"
)

const (
	ErrBuildRequest     errorspkg.Code = "services.request_build"
	ErrDoRequest        errorspkg.Code = "services.request_do"
	ErrUnexpectedStatus errorspkg.Code = "services.bad_status"
	ErrReadBody         errorspkg.Code = "services.read_body"
	ErrDecodeResponse   errorspkg.Code = "services.decode_response"
	ErrEncodeRequest    errorspkg.Code = "services.encode_request"
	ErrValidateConfig   errorspkg.Code = "services.validate_config"
	ErrServiceNotFound  errorspkg.Code = "services.not_found"
)

var (
	manager     = errorspkg.NewManager()
	loggerGuard sync.Map
)

func errorsManager() *errorspkg.Manager {
	return manager
}

func registerServiceLogger(logger *logging.Logger) {
	if logger == nil {
		return
	}
	if _, loaded := loggerGuard.LoadOrStore(logger, struct{}{}); loaded {
		return
	}

	manager.Register(func(e *errorspkg.Error) {
		logger.Warn("service connector error: %s", e.Format())
	})
}
