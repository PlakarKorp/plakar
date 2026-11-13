package plugins

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"time"
)

// A Runner starts a plugin and returns the connection carrying its gRPC
// session. Closing the connection tears the plugin down.
type Runner interface {
	Run(ctx context.Context) (net.Conn, error)
}

// NativeRunner executes the plugin binary directly and talks gRPC over its
// stdin/stdout. Whatever the plugin writes to stderr goes to Stderr, or to
// the process stderr when nil.
type NativeRunner struct {
	Path   string
	Args   []string
	Stderr io.Writer
}

func (r *NativeRunner) Run(_ context.Context) (net.Conn, error) {
	cmd := exec.Command(r.Path, r.Args...)
	cmd.SysProcAttr = detachedProcAttr()
	// When Stderr is not a file, Wait also waits for the stderr copy, which
	// never ends if the plugin left a child holding the pipe.
	cmd.WaitDelay = time.Second
	cmd.Stderr = r.Stderr
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}

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
