//go:build windows

package plugins

import (
	"context"
	"errors"
)

func Load(ctx context.Context, pluginPath string) error {
	return errors.ErrUnsupported
}
