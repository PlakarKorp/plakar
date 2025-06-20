//go:build windows

package agent

import "errors"

func daemonize(argv []string) error {
	return errors.ErrUnsupported
}
