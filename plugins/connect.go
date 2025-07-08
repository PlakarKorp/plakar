//go:build !windows

package plugins

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func connectPlugin(pluginPath string) (grpc.ClientConnInterface, error) {
	fd, err := forkChild(pluginPath)
	if err != nil {
		return nil, err
	}

	connFile := os.NewFile(uintptr(fd), "grpc-conn")
	conn, err := net.FileConn(connFile)
	if err != nil {
		return nil, fmt.Errorf("net.FileConn failed: %w", err)
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

func forkChild(pluginPath string) (int, error) {
	sp, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return -1, fmt.Errorf("failed to create socketpair: %w", err)
	}

	childFile := os.NewFile(uintptr(sp[0]), "child-conn")

	cmd := exec.Command(pluginPath)
	cmd.Stdin = childFile
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return -1, fmt.Errorf("failed to start plugin: %w", err)
	}

	childFile.Close()
	return sp[1], nil
}
