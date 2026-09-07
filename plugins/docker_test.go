package plugins

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakeDocker writes a docker-CLI stand-in that re-execs the test binary into
// TestDockerHelper with the given behavior, and points MkdirTemp at a fresh
// TMPDIR so tests can assert the socket dir is cleaned up.
func fakeDocker(t *testing.T, behavior string) (bin string, tmpdir string, statedir string) {
	t.Helper()

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}

	tmpdir = t.TempDir()
	statedir = t.TempDir()
	t.Setenv("TMPDIR", tmpdir)

	bin = filepath.Join(statedir, "docker")
	script := fmt.Sprintf("#!/bin/sh\nFAKE_DOCKER_MODE=%s FAKE_DOCKER_STATE=%s exec %s -test.run='^TestDockerHelper$' -- \"$@\"\n",
		behavior, statedir, exe)
	if err := os.WriteFile(bin, []byte(script), 0755); err != nil {
		t.Fatalf("write fake docker: %v", err)
	}
	return bin, tmpdir, statedir
}

func TestDockerRunnerOK(t *testing.T) {
	bin, tmpdir, statedir := fakeDocker(t, "ok")

	runner := &DockerRunner{Binary: bin, Image: "sha256:feedface", StartTimeout: 5 * time.Second}
	conn, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// The helper's listener echoes; round-trip a byte to prove the socket
	// carries data.
	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buf) != "ping" {
		t.Fatalf("got %q, want %q", buf, "ping")
	}

	if err := conn.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Teardown must remove the socket dir and reap the container (the helper
	// deletes its pidfile when "rm" kills the listener).
	if ents, err := os.ReadDir(tmpdir); err != nil || len(ents) != 0 {
		t.Fatalf("socket dir not cleaned up: %v %v", ents, err)
	}
	if _, err := os.Stat(filepath.Join(statedir, "pids")); !os.IsNotExist(err) {
		if ents, _ := os.ReadDir(filepath.Join(statedir, "pids")); len(ents) != 0 {
			t.Fatalf("container not reaped: %v", ents)
		}
	}
}

func TestDockerRunnerContextCancelTearsDown(t *testing.T) {
	bin, tmpdir, _ := fakeDocker(t, "ok")

	ctx, cancel := context.WithCancel(context.Background())
	runner := &DockerRunner{Binary: bin, Image: "sha256:feedface", StartTimeout: 5 * time.Second}
	if _, err := runner.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}

	// Cancelling the connector context must reap the container, like the
	// native runner's exec.CommandContext kill.
	cancel()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if ents, err := os.ReadDir(tmpdir); err == nil && len(ents) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("socket dir still present after context cancellation")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestDockerRunnerContainerExits(t *testing.T) {
	bin, tmpdir, _ := fakeDocker(t, "exit")

	runner := &DockerRunner{Binary: bin, Image: "sha256:feedface", StartTimeout: 5 * time.Second}
	_, err := runner.Run(context.Background())
	if err == nil {
		t.Fatal("expected Run to fail when the container exits before serving")
	}
	if !strings.Contains(err.Error(), "exited before serving") {
		t.Fatalf("error should mention the container exit, got: %v", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error should carry the container logs, got: %v", err)
	}
	if ents, err := os.ReadDir(tmpdir); err != nil || len(ents) != 0 {
		t.Fatalf("socket dir not cleaned up: %v %v", ents, err)
	}
}

func TestDockerRunnerRunFails(t *testing.T) {
	bin, tmpdir, _ := fakeDocker(t, "runfail")

	runner := &DockerRunner{Binary: bin, Image: "sha256:feedface"}
	_, err := runner.Run(context.Background())
	if err == nil {
		t.Fatal("expected Run to fail when docker run fails")
	}
	if !strings.Contains(err.Error(), "cannot connect to the docker daemon") {
		t.Fatalf("error should carry docker's stderr, got: %v", err)
	}
	if ents, err := os.ReadDir(tmpdir); err != nil || len(ents) != 0 {
		t.Fatalf("socket dir not cleaned up: %v %v", ents, err)
	}
}

func TestDockerRunnerMissingBinary(t *testing.T) {
	runner := &DockerRunner{Binary: "/nonexistent/docker", Image: "sha256:feedface"}
	if _, err := runner.Run(context.Background()); err == nil {
		t.Fatal("expected Run to fail when the container CLI is missing")
	}
}

// TestDockerHelper is not a test: it is the body of the fake docker CLI,
// entered when the wrapper script re-execs the test binary. It emulates the
// docker subcommands DockerRunner uses.
func TestDockerHelper(t *testing.T) {
	mode := os.Getenv("FAKE_DOCKER_MODE")
	if mode == "" {
		t.Skip("helper process for the fake docker CLI")
	}
	args := flag.Args()
	if len(args) == 0 {
		os.Exit(1)
	}

	statedir := os.Getenv("FAKE_DOCKER_STATE")
	pidfile := func(name string) string {
		os.MkdirAll(filepath.Join(statedir, "pids"), 0777)
		return filepath.Join(statedir, "pids", name+".pid")
	}

	switch args[0] {
	case "run":
		if mode == "runfail" {
			fmt.Fprintln(os.Stderr, "cannot connect to the docker daemon")
			os.Exit(1)
		}
		var hostdir, name string
		for i, a := range args {
			switch a {
			case "--volume":
				hostdir = strings.SplitN(args[i+1], ":", 2)[0]
			case "--name":
				name = args[i+1]
			}
		}
		if mode == "ok" {
			// Detach a listener process standing in for the containerized
			// plugin; it outlives this "docker run" invocation.
			exe, _ := os.Executable()
			sock := filepath.Join(hostdir, "plugin.sock")
			child := exec.Command(exe, "-test.run=^TestDockerHelper$", "--", "listen", sock)
			child.Env = append(os.Environ(), "FAKE_DOCKER_MODE=listen")
			if err := child.Start(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			os.WriteFile(pidfile(name), []byte(strconv.Itoa(child.Process.Pid)), 0666)
		}
		fmt.Println("deadbeefdeadbeef")
		os.Exit(0)

	case "listen":
		sock := args[1]
		l, err := net.Listen("unix", sock)
		if err != nil {
			os.Exit(1)
		}
		os.Chmod(sock, 0666)
		go func() {
			for {
				c, err := l.Accept()
				if err != nil {
					return
				}
				go io.Copy(c, c)
			}
		}()
		// Safety net if "rm" never kills us: exit when the socket is gone
		// or after a minute.
		for i := 0; i < 600; i++ {
			if _, err := os.Stat(sock); err != nil {
				os.Exit(0)
			}
			time.Sleep(100 * time.Millisecond)
		}
		os.Exit(0)

	case "inspect":
		name := args[len(args)-1]
		if _, err := os.Stat(pidfile(name)); err == nil {
			fmt.Println("true")
		} else {
			fmt.Println("false")
		}
		os.Exit(0)

	case "logs":
		fmt.Println("boom: plugin crashed on startup")
		os.Exit(0)

	case "rm":
		name := args[len(args)-1]
		if data, err := os.ReadFile(pidfile(name)); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
				if p, err := os.FindProcess(pid); err == nil {
					p.Kill()
				}
			}
			os.Remove(pidfile(name))
		}
		os.Exit(0)
	}
	os.Exit(1)
}
