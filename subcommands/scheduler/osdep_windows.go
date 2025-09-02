package scheduler

import (
	"errors"

	"github.com/PlakarKorp/plakar/appcontext"
)

func setupSyslog(ctx *appcontext.AppContext) error {
	return errors.ErrUnsupported
}

func daemonize(argv []string) error {
	return errors.ErrUnsupported
}

func stop() error {
	return errors.ErrUnsupported
}
