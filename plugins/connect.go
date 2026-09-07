package plugins

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func connectPlugin(ctx context.Context, runner Runner) (grpc.ClientConnInterface, error) {
	conn, err := runner.Run(ctx)
	if err != nil {
		return nil, err
	}

	clientConn, err := grpc.NewClient("127.0.0.1:0",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithIdleTimeout(0),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return conn, nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc client creation failed: %w", err)
	}
	return clientConn, nil
}
