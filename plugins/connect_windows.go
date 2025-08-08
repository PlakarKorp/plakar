package plugins

import (
	"context"
	"errors"

	"google.golang.org/grpc"
)

func connectPlugin(ctx context.Context, pluginPath string, args []string) (grpc.ClientConnInterface, error) {
	return nil, errors.ErrUnsupported
}
