package agent

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
