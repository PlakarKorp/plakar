package cached

import (
	"bytes"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/kloset/hashing"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// shortSocket returns a unix socket path short enough to satisfy the OS sun_path
// limit (108 on Linux, 104 on macOS), which t.TempDir() can exceed.
func shortSocket(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "cs")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "c.sock")
}

// wrapConfig serializes a storage.Configuration into the same wrapped on-disk
// format that storage.Open produces, so it can be fed to getSecret.
func wrapConfig(t *testing.T, config *storage.Configuration) []byte {
	t.Helper()
	hasher := hashing.GetHasher(storage.DEFAULT_HASHING_ALGORITHM)
	serialized, err := config.ToBytes()
	require.NoError(t, err)
	rd, err := storage.Serialize(hasher, resources.RT_CONFIG,
		versioning.GetCurrentVersion(resources.RT_CONFIG), bytes.NewReader(serialized))
	require.NoError(t, err)
	wrapped, err := io.ReadAll(rd)
	require.NoError(t, err)
	return wrapped
}

// ---------------------------------------------------------------------------
// Parse
// ---------------------------------------------------------------------------

func TestCachedParseForeground(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	cmd := &Cached{}
	err := cmd.Parse(ctx, []string{"-foreground"})
	require.NoError(t, err)

	// defaults / initialization
	require.Equal(t, 5*time.Second, cmd.teardown)
	require.Equal(t, filepath.Join(ctx.CacheDir, "cached.sock"), cmd.socketPath)
	require.NotNil(t, cmd.jobQueue)
	require.NotNil(t, cmd.runningJobs)
}

func TestCachedParseTeardown(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)

	cmd := &Cached{}
	err := cmd.Parse(ctx, []string{"-foreground", "-teardown", "42s"})
	require.NoError(t, err)
	require.Equal(t, 42*time.Second, cmd.teardown)
}

func TestCachedParseTooManyArgs(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)

	cmd := &Cached{}
	err := cmd.Parse(ctx, []string{"-foreground", "extra"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many arguments")
}

func TestCachedParseLogfileCreates(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)

	logfile := filepath.Join(t.TempDir(), "cached.log")
	cmd := &Cached{}
	err := cmd.Parse(ctx, []string{"-foreground", "-log", logfile})
	require.NoError(t, err)

	_, statErr := os.Stat(logfile)
	require.NoError(t, statErr)
}

func TestCachedParseLogfileError(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)

	// point the log at a path whose parent directory does not exist -> open fails
	logfile := filepath.Join(t.TempDir(), "missingdir", "cached.log")
	cmd := &Cached{}
	err := cmd.Parse(ctx, []string{"-foreground", "-log", logfile})
	require.Error(t, err)
}

// Parse uses REEXEC to skip the daemonize fork even without -foreground.
func TestCachedParseReexec(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)

	t.Setenv("REEXEC", "1")
	cmd := &Cached{}
	err := cmd.Parse(ctx, []string{})
	require.NoError(t, err)
	require.Equal(t, filepath.Join(ctx.CacheDir, "cached.sock"), cmd.socketPath)
}

// ---------------------------------------------------------------------------
// Close
// ---------------------------------------------------------------------------

func TestCachedCloseMissingSocket(t *testing.T) {
	cmd := &Cached{
		socketPath: filepath.Join(t.TempDir(), "does-not-exist.sock"),
	}
	// no listener, socket file absent -> Close should be a no-op (no error)
	require.NoError(t, cmd.Close())
}

func TestCachedClosePresentSocket(t *testing.T) {
	sock := shortSocket(t)

	listener, err := net.Listen("unix", sock)
	require.NoError(t, err)

	cmd := &Cached{
		socketPath: sock,
		listener:   listener,
	}
	require.NoError(t, cmd.Close())

	// the socket file must be gone
	_, statErr := os.Stat(sock)
	require.True(t, os.IsNotExist(statErr))
}

// ---------------------------------------------------------------------------
// Watcher
// ---------------------------------------------------------------------------

func TestWatcherTearsDownWhenIdle(t *testing.T) {
	sock := shortSocket(t)
	listener, err := net.Listen("unix", sock)
	require.NoError(t, err)

	cmd := &Cached{
		teardown:    20 * time.Millisecond,
		runningJobs: make(chan int),
	}

	go cmd.Watcher(listener)

	// With nothing in flight, the watcher should close the listener after the
	// teardown delay. A subsequent Accept must fail.
	done := make(chan error, 1)
	go func() {
		_, e := listener.Accept()
		done <- e
	}()

	select {
	case e := <-done:
		require.Error(t, e)
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not close the listener")
	}
}

func TestWatcherStaysAliveWhileInflight(t *testing.T) {
	sock := shortSocket(t)
	listener, err := net.Listen("unix", sock)
	require.NoError(t, err)
	defer listener.Close()

	cmd := &Cached{
		teardown:    20 * time.Millisecond,
		runningJobs: make(chan int),
	}

	go cmd.Watcher(listener)

	// Mark a job as in-flight: the watcher must not close the listener.
	cmd.runningJobs <- newJob

	accepted := make(chan error, 1)
	go func() {
		conn, e := listener.Accept()
		if conn != nil {
			conn.Close()
		}
		accepted <- e
	}()

	// Connect to prove the listener is still open after more than one teardown.
	time.Sleep(60 * time.Millisecond)
	conn, dialErr := net.Dial("unix", sock)
	require.NoError(t, dialErr)
	conn.Close()

	require.NoError(t, <-accepted)

	// release the job so the watcher goroutine can eventually exit cleanly
	cmd.runningJobs <- jobDone
}

// ---------------------------------------------------------------------------
// isDisconnectError
// ---------------------------------------------------------------------------

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestIsDisconnectError(t *testing.T) {
	require.True(t, isDisconnectError(io.EOF))
	require.True(t, isDisconnectError(io.ErrUnexpectedEOF))
	require.True(t, isDisconnectError(timeoutErr{}))
	require.False(t, isDisconnectError(io.ErrClosedPipe))
	require.False(t, isDisconnectError(nil))
}

// ---------------------------------------------------------------------------
// getSecret
// ---------------------------------------------------------------------------

func TestGetSecretNoEncryption(t *testing.T) {
	ctx := appcontext.NewAppContext()

	config := storage.NewConfiguration()
	config.Encryption = nil
	wrapped := wrapConfig(t, config)

	key, err := getSecret(ctx, []byte("ignored"), wrapped)
	require.NoError(t, err)
	require.Nil(t, key)
}

func TestGetSecretMalformedConfig(t *testing.T) {
	ctx := appcontext.NewAppContext()

	_, err := getSecret(ctx, nil, []byte("not a wrapped config"))
	require.Error(t, err)
}

func TestGetSecretWrongKey(t *testing.T) {
	ctx := appcontext.NewAppContext()

	// build an encrypted configuration with a canary derived from the real key
	config := storage.NewConfiguration()
	passphrase := []byte("correct horse battery staple")
	key, err := encryption.DeriveKey(config.Encryption.KDFParams, passphrase)
	require.NoError(t, err)
	canary, err := encryption.DeriveCanary(config.Encryption, key)
	require.NoError(t, err)
	config.Encryption.Canary = canary

	wrapped := wrapConfig(t, config)

	// correct key verifies
	got, err := getSecret(ctx, key, wrapped)
	require.NoError(t, err)
	require.Equal(t, key, got)

	// wrong key fails the canary check
	_, err = getSecret(ctx, []byte("the wrong key......the wrong key"), wrapped)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to verify key")
}
