package plugins

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	dockerSockDir  = "/run/plakar"
	dockerSockName = "plugin.sock"

	defaultStartTimeout = 30 * time.Second
)

// DockerRunner runs the plugin as a detached container and talks gRPC over a
// unix socket bind-mounted into it. The plugin side is told where to listen
// through the PLAKAR_PLUGIN_LISTEN environment variable.
type DockerRunner struct {
	Binary       string        // container CLI to shell out to, "docker" when empty
	Image        string        // image reference or ID to run
	Args         []string      // arguments passed to the connector inside the container
	StartTimeout time.Duration // max wait for the plugin socket, 30s when zero
}

func (r *DockerRunner) Run(ctx context.Context) (net.Conn, error) {
	bin := r.Binary
	if bin == "" {
		bin = "docker"
	}

	// The socket dir is world-writable so the container-side uid can create
	// the socket in it, and nested in a 0700 dir so no other host user can
	// reach it.
	base, err := os.MkdirTemp("", "plakar-plugin-")
	if err != nil {
		return nil, err
	}
	sockdir := filepath.Join(base, "sock")
	if err := os.Mkdir(sockdir, 0777); err != nil {
		os.RemoveAll(base)
		return nil, err
	}

	// umask may have changed us force world-writable
	if err := os.Chmod(sockdir, 0777); err != nil {
		os.RemoveAll(base)
		return nil, err
	}

	c := &container{bin: bin, name: filepath.Base(base), dir: base}

	args := []string{
		"run", "--detach", "--name", c.name,
		"--network", "host",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--read-only", "--tmpfs", "/tmp",
		"--volume", sockdir + ":" + dockerSockDir + ":Z",
		"--env", "PLAKAR_PLUGIN_LISTEN=unix://" + path.Join(dockerSockDir, dockerSockName),
		r.Image,
	}
	args = append(args, r.Args...)

	if _, err := exec.CommandContext(ctx, bin, args...).Output(); err != nil {
		os.RemoveAll(base)
		var exit *exec.ExitError
		if errors.As(err, &exit) && len(exit.Stderr) > 0 {
			return nil, fmt.Errorf("failed to start plugin container: %w: %s",
				err, strings.TrimSpace(string(exit.Stderr)))
		}
		return nil, fmt.Errorf("failed to start plugin container: %w", err)
	}

	timeout := r.StartTimeout
	if timeout == 0 {
		timeout = defaultStartTimeout
	}
	conn, err := c.dial(ctx, filepath.Join(sockdir, dockerSockName), timeout)
	if err != nil {
		if logs := c.logs(); logs != "" {
			err = fmt.Errorf("%w; container logs:\n%s", err, logs)
		}
		c.teardown()
		return nil, err
	}

	dc := &dockerConn{Conn: conn, container: c}
	// Mirror the native runner, whose plugin process is killed through
	// exec.CommandContext when the connector context is cancelled.
	context.AfterFunc(ctx, func() { dc.Close() })
	return dc, nil
}

// container tracks a running plugin container and its host-side socket dir.
type container struct {
	bin  string
	name string
	dir  string

	stop    sync.Once
	stopErr error
}

// dial waits for the plugin to serve its socket, failing fast when the
// container exits before that.
func (c *container) dial(ctx context.Context, sock string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()

	for i := 0; ; i++ {
		if conn, err := net.Dial("unix", sock); err == nil {
			return conn, nil
		}
		if i%10 == 9 && !c.running(ctx) {
			return nil, fmt.Errorf("plugin container exited before serving its socket")
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("waiting for plugin socket: %w", ctx.Err())
		case <-tick.C:
		}
	}
}

func (c *container) running(ctx context.Context) bool {
	out, err := exec.CommandContext(ctx, c.bin, "inspect", "--format", "{{.State.Running}}", c.name).Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func (c *container) logs() string {
	out, _ := exec.Command(c.bin, "logs", "--tail", "20", c.name).CombinedOutput()
	return strings.TrimSpace(string(out))
}

func (c *container) teardown() error {
	c.stop.Do(func() {
		if out, err := exec.Command(c.bin, "rm", "--force", c.name).CombinedOutput(); err != nil {
			c.stopErr = fmt.Errorf("failed to remove plugin container: %s: %s",
				err, strings.TrimSpace(string(out)))
		}
		if err := os.RemoveAll(c.dir); err != nil && c.stopErr == nil {
			c.stopErr = err
		}
	})
	return c.stopErr
}

type dockerConn struct {
	net.Conn
	container *container
}

func (dc *dockerConn) Close() error {
	err := dc.Conn.Close()
	if terr := dc.container.teardown(); terr != nil && err == nil {
		err = terr
	}
	return err
}
