package cached

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/stretchr/testify/require"
)

// newCov80Ctx builds an isolated AppContext with a short temp CacheDir (macOS
// sun_path is limited to 104 bytes). Distinct helper name so it does not clash
// with the team's newCachedCtx.
func newCov80Ctx(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.Stdout = bytes.NewBuffer(nil)
	ctx.Stderr = bytes.NewBuffer(nil)
	dir, err := os.MkdirTemp("", "c8")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	ctx.CacheDir = dir
	ctx.SetLogger(logging.NewLogger(ctx.Stdout, ctx.Stderr))
	return ctx
}

// shortSocket returns a short unix socket path under a fresh temp dir.
func shortSocket(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "c8s")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "s.sock")
}

// ---------------------------------------------------------------------------
// Watcher
// ---------------------------------------------------------------------------

// TestCov80WatcherTearsDownWhenIdle verifies that, once inflight drops back to
// zero and the teardown interval elapses with nothing running, the Watcher
// closes the listener.
func TestCov80WatcherTearsDownWhenIdle(t *testing.T) {
	t.Parallel()

	sock := shortSocket(t)
	ln, err := net.Listen("unix", sock)
	require.NoError(t, err)

	cmd := &Cached{
		teardown:    40 * time.Millisecond,
		runningJobs: make(chan int),
	}

	go cmd.Watcher(ln)

	// Simulate a job starting then finishing: inflight goes 1 -> 0.
	cmd.runningJobs <- newJob
	cmd.runningJobs <- jobDone

	// After teardown elapses with inflight == 0 the listener is closed, so a
	// subsequent Accept returns an error promptly.
	done := make(chan error, 1)
	go func() {
		_, aerr := ln.Accept()
		done <- aerr
	}()

	select {
	case aerr := <-done:
		require.Error(t, aerr)
	case <-time.After(3 * time.Second):
		t.Fatal("watcher did not close the listener after teardown")
	}
}

// TestCov80WatcherStaysAliveWhileBusy verifies that while a job is inflight the
// Watcher does NOT close the listener even after several teardown intervals.
func TestCov80WatcherStaysAliveWhileBusy(t *testing.T) {
	t.Parallel()

	sock := shortSocket(t)
	ln, err := net.Listen("unix", sock)
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })

	cmd := &Cached{
		teardown:    20 * time.Millisecond,
		runningJobs: make(chan int),
	}

	go cmd.Watcher(ln)

	// One job stays inflight (no matching jobDone).
	cmd.runningJobs <- newJob

	// Let several teardown intervals pass; the listener must remain open.
	time.Sleep(120 * time.Millisecond)

	accepted := make(chan struct{})
	go func() {
		c, derr := net.Dial("unix", sock)
		if derr == nil {
			c.Close()
			close(accepted)
		}
	}()

	select {
	case <-accepted:
		// Dial succeeded -> listener is still open as expected.
	case <-time.After(2 * time.Second):
		t.Fatal("listener appears closed while a job was still inflight")
	}

	// Now drain the inflight job so the watcher can eventually tear down and the
	// goroutine is not leaked indefinitely beyond the test.
	cmd.runningJobs <- jobDone
}

// ---------------------------------------------------------------------------
// ListenAndServe: "already running" early return (no server started)
// ---------------------------------------------------------------------------

// TestCov80ListenAndServeAlreadyRunning pre-binds the socket so the dial inside
// ListenAndServe succeeds, driving the "cached already running" early-return
// branch without ever entering the accept loop.
func TestCov80ListenAndServeAlreadyRunning(t *testing.T) {
	t.Parallel()

	ctx := newCov80Ctx(t)
	sock := shortSocket(t)

	// A live listener on the socket makes net.Dial succeed.
	existing, err := net.Listen("unix", sock)
	require.NoError(t, err)
	t.Cleanup(func() { existing.Close() })

	cmd := &Cached{
		socketPath:  sock,
		runningJobs: make(chan int),
	}

	err = cmd.ListenAndServe(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already running")
}

// ---------------------------------------------------------------------------
// Execute: crash-log open failure (returns before ListenAndServe)
// ---------------------------------------------------------------------------

// TestCov80ExecuteCrashLogOpenError points CacheDir at a path that cannot hold
// the crash log file (a parent that does not exist), so os.OpenFile fails and
// Execute returns (1, err) without ever reaching the long-lived ListenAndServe.
func TestCov80ExecuteCrashLogOpenError(t *testing.T) {
	t.Parallel()

	ctx := newCov80Ctx(t)
	// CacheDir whose parent directory does not exist -> Join produces a path
	// under a missing dir, so creating crash-cached.log fails with ENOENT.
	ctx.CacheDir = filepath.Join(ctx.CacheDir, "missing-subdir")

	c := &Cached{
		socketPath:  shortSocket(t),
		runningJobs: make(chan int),
	}

	status, err := c.Execute(ctx, nil)
	require.Error(t, err)
	require.Equal(t, 1, status)
}
