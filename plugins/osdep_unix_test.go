//go:build !windows

package plugins

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

// A Ctrl-C from the terminal is sent to the whole foreground process group.
// The plugin must not share plakar's group, or it dies before plakar can shut
// it down.
func TestNativeRunnerPluginOutsideProcessGroup(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	t.Setenv("PLUGIN_PGID_HELPER", "1")

	conn, err := (&NativeRunner{
		Path: exe,
		Args: []string{"-test.run=^TestPluginPgidHelper$"},
	}).Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	out, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	pgid, err := strconv.Atoi(strings.Fields(string(out))[0])
	if err != nil {
		t.Fatalf("helper output %q: %v", out, err)
	}
	if pgid == unix.Getpgrp() {
		t.Fatalf("plugin shares our process group %d", pgid)
	}
}

// TestPluginPgidHelper is not a test: it is the plugin spawned by
// TestNativeRunnerPluginOutsideProcessGroup, and reports its process group.
func TestPluginPgidHelper(t *testing.T) {
	if os.Getenv("PLUGIN_PGID_HELPER") == "" {
		t.Skip("helper process for TestNativeRunnerPluginOutsideProcessGroup")
	}
	fmt.Println(unix.Getpgrp())
	os.Exit(0)
}
