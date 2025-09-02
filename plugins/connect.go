package plugins

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func connectPlugin(ctx context.Context, pluginPath string, args []string) (grpc.ClientConnInterface, error) {
	conn, err := spawn(ctx, pluginPath, args)
	if err != nil {
		return nil, err
	}

	clientConn, err := grpc.NewClient("127.0.0.1:0",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return conn, nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc client creation failed: %w", err)
	}
	return clientConn, nil
}

func spawn(ctx context.Context, pluginPath string, args []string) (net.Conn, error) {
	cmd := exec.CommandContext(ctx, pluginPath, args...)
	cmd.Stderr = os.Stderr

	wr, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	rd, err := cmd.StdoutPipe()
	if err != nil {
		wr.Close()
		return nil, err
	}

	stdin, ok := rd.(*os.File)
	if !ok {
		wr.Close()
		rd.Close()
		reason := "stdin is not a file"
		return nil, fmt.Errorf("failed to spawn plugin: %s", reason)
	}

	stdout, ok := wr.(*os.File)
	if !ok {
		wr.Close()
		rd.Close()
		reason := "stdout is not a file"
		return nil, fmt.Errorf("failed to spawn plugin: %s", reason)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start plugin: %w", err)
	}

	return NewStdioConn(stdin, stdout, cmd), nil
}
