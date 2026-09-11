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
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func newCachedCtx(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.Stdout = bytes.NewBuffer(nil)
	ctx.Stderr = bytes.NewBuffer(nil)
	// Use a short temp path: macOS sun_path is limited to 104 bytes.
	dir, err := os.MkdirTemp("", "cd")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	ctx.CacheDir = dir
	ctx.SetLogger(logging.NewLogger(ctx.Stdout, ctx.Stderr))
	return ctx
}

// ---------------------------------------------------------------------------
// Cached.Parse
// ---------------------------------------------------------------------------

func TestCachedParseForeground(t *testing.T) {
	ctx := newCachedCtx(t)
	cmd := &Cached{}
	require.NoError(t, cmd.Parse(ctx, []string{"-foreground"}))
	require.Equal(t, filepath.Join(ctx.CacheDir, "cached.sock"), cmd.socketPath)
	require.NotNil(t, cmd.jobQueue)
	require.NotNil(t, cmd.runningJobs)
	// default teardown is 5s
	require.Equal(t, 5*time.Second, cmd.teardown)
}

func TestCachedParseTeardownFlag(t *testing.T) {
	ctx := newCachedCtx(t)
	cmd := &Cached{}
	require.NoError(t, cmd.Parse(ctx, []string{"-foreground", "-teardown", "250ms"}))
	require.Equal(t, 250*time.Millisecond, cmd.teardown)
}

func TestCachedParseReexec(t *testing.T) {
	ctx := newCachedCtx(t)
	t.Setenv("REEXEC", "1")
	// With REEXEC set, Parse does not daemonize even without -foreground.
	// syslog setup may fail in CI; that path falls back to io.Discard and
	// still returns nil.
	cmd := &Cached{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	require.Equal(t, filepath.Join(ctx.CacheDir, "cached.sock"), cmd.socketPath)
}

func TestCachedParseLogfile(t *testing.T) {
	ctx := newCachedCtx(t)
	logfile := filepath.Join(ctx.CacheDir, "cached.log")
	cmd := &Cached{}
	require.NoError(t, cmd.Parse(ctx, []string{"-foreground", "-log", logfile}))
	_, err := os.Stat(logfile)
	require.NoError(t, err)
}

func TestCachedParseLogfileError(t *testing.T) {
	ctx := newCachedCtx(t)
	// Point -log at a path whose parent does not exist -> open fails.
	bad := filepath.Join(ctx.CacheDir, "missing-dir", "cached.log")
	cmd := &Cached{}
	err := cmd.Parse(ctx, []string{"-foreground", "-log", bad})
	require.Error(t, err)
}

func TestCachedParseTooManyArgs(t *testing.T) {
	ctx := newCachedCtx(t)
	err := (&Cached{}).Parse(ctx, []string{"-foreground", "extra"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many arguments")
}

// ---------------------------------------------------------------------------
// Cached.Close
// ---------------------------------------------------------------------------

func TestCachedCloseMissingSocket(t *testing.T) {
	ctx := newCachedCtx(t)
	cmd := &Cached{}
	require.NoError(t, cmd.Parse(ctx, []string{"-foreground"}))
	// No socket file exists yet; Close should tolerate ENOENT.
	require.NoError(t, cmd.Close())
}

func TestCachedClosePresentSocket(t *testing.T) {
	ctx := newCachedCtx(t)
	cmd := &Cached{}
	require.NoError(t, cmd.Parse(ctx, []string{"-foreground"}))

	// Create a real listening unix socket at socketPath, store the listener.
	ln, err := net.Listen("unix", cmd.socketPath)
	require.NoError(t, err)
	cmd.listener = ln

	require.NoError(t, cmd.Close())
	// Close removes the socket file.
	_, statErr := os.Stat(cmd.socketPath)
	require.True(t, os.IsNotExist(statErr))
}

// ---------------------------------------------------------------------------
// isDisconnectError
// ---------------------------------------------------------------------------

type timeoutErr struct{ to bool }

func (e timeoutErr) Error() string   { return "timeout" }
func (e timeoutErr) Timeout() bool   { return e.to }
func (e timeoutErr) Temporary() bool { return false }

func TestIsDisconnectError(t *testing.T) {
	require.True(t, isDisconnectError(io.EOF))
	require.True(t, isDisconnectError(io.ErrUnexpectedEOF))
	require.True(t, isDisconnectError(timeoutErr{to: true}))

	require.False(t, isDisconnectError(nil))
	require.False(t, isDisconnectError(io.ErrClosedPipe))
	require.False(t, isDisconnectError(timeoutErr{to: false}))
}

// ---------------------------------------------------------------------------
// getSecret
// ---------------------------------------------------------------------------

// wrapConfig serializes + wraps a storage.Configuration the same way the
// repository does, producing the bytes getSecret expects.
func wrapConfig(t *testing.T, config *storage.Configuration, key []byte) []byte {
	t.Helper()
	serialized, err := config.ToBytes()
	require.NoError(t, err)

	var hasher = hashing.GetHasher(storage.DEFAULT_HASHING_ALGORITHM)
	if key != nil {
		hasher = hashing.GetMACHasher(storage.DEFAULT_HASHING_ALGORITHM, key)
	}
	rd, err := storage.Serialize(hasher, resources.RT_CONFIG, versioning.GetCurrentVersion(resources.RT_CONFIG), bytes.NewReader(serialized))
	require.NoError(t, err)
	wrapped, err := io.ReadAll(rd)
	require.NoError(t, err)
	return wrapped
}

func TestGetSecretNoEncryption(t *testing.T) {
	ctx := newCachedCtx(t)
	config := storage.NewConfiguration()
	config.Encryption = nil

	wrapped := wrapConfig(t, config, nil)
	key, err := getSecret(ctx, nil, wrapped)
	require.NoError(t, err)
	require.Nil(t, key)
}

func TestGetSecretMalformedConfig(t *testing.T) {
	ctx := newCachedCtx(t)
	_, err := getSecret(ctx, nil, []byte("not a wrapped config"))
	require.Error(t, err)
}

func TestGetSecretCorrectAndWrongKey(t *testing.T) {
	ctx := newCachedCtx(t)
	config := storage.NewConfiguration()

	passphrase := []byte("correct horse battery staple")
	key, err := encryption.DeriveKey(config.Encryption.KDFParams, passphrase)
	require.NoError(t, err)
	canary, err := encryption.DeriveCanary(config.Encryption, key)
	require.NoError(t, err)
	config.Encryption.Canary = canary

	wrapped := wrapConfig(t, config, key)

	// Correct key verifies the canary and is returned unchanged.
	got, err := getSecret(ctx, key, wrapped)
	require.NoError(t, err)
	require.Equal(t, key, got)

	// Wrong key fails canary verification.
	wrongKey := make([]byte, len(key))
	copy(wrongKey, key)
	wrongKey[0] ^= 0xff
	_, err = getSecret(ctx, wrongKey, wrapped)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to verify key")
}
