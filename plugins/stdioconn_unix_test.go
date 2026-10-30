//go:build !windows

package plugins

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// runStdioHelper starts the test binary as a plugin running
// TestStdioHelper in the given mode, and returns the conn and the pid the
// helper reports on its first line.
func runStdioHelper(t *testing.T, mode string, stderr io.Writer) (*StdioConn, int) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	t.Setenv("PLUGIN_STDIO_HELPER", mode)

	conn, err := (&NativeRunner{
		Path:   exe,
		Args:   []string{"-test.run=^TestStdioHelper$"},
		Stderr: stderr,
	}).Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read helper pid: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatalf("helper pid %q: %v", line, err)
	}
	return conn.(*StdioConn), pid
}

func TestStdioConnCloseKillsPluginIgnoringEOF(t *testing.T) {
	old := exitGrace
	exitGrace = 200 * time.Millisecond
	t.Cleanup(func() { exitGrace = old })

	conn, pid := runStdioHelper(t, "ignore-eof", nil)

	start := time.Now()
	err := conn.Close()
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("Close took %s on a plugin ignoring EOF", elapsed)
	}
	if err == nil {
		t.Fatal("expected Close to report the kill")
	}
	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("plugin %d still exists after Close: %v", pid, err)
	}
}

func TestStdioConnCloseNotHeldByOrphanedStderr(t *testing.T) {
	// A non-file Stderr makes Wait copy the pipe, which a leftover child of
	// the plugin keeps open.
	conn, child := runStdioHelper(t, "orphan-stderr", &bytes.Buffer{})
	t.Cleanup(func() { syscall.Kill(child, syscall.SIGKILL) })

	start := time.Now()
	conn.Close()
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("Close took %s, held by the plugin's child", elapsed)
	}
}

// TestStdioHelper is not a test: it is the plugin spawned by the StdioConn
// tests. It reports a pid on its first line, then behaves per mode.
func TestStdioHelper(t *testing.T) {
	mode := os.Getenv("PLUGIN_STDIO_HELPER")
	switch mode {
	case "":
		t.Skip("helper process for the StdioConn tests")

	case "ignore-eof":
		fmt.Println(os.Getpid())
		io.Copy(io.Discard, os.Stdin)
		time.Sleep(time.Minute)

	case "orphan-stderr":
		child := exec.Command("sleep", "60")
		child.Stderr = os.Stderr
		if err := child.Start(); err != nil {
			os.Exit(1)
		}
		fmt.Println(child.Process.Pid)
		io.Copy(io.Discard, os.Stdin)
	}
	os.Exit(0)
}
