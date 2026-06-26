package cached

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// shortTempDir returns a short-lived temp dir with a short absolute path.
// t.TempDir() on macOS produces paths far longer than the 104-byte sun_path
// limit for unix sockets, so we make our own short one under os.TempDir().
func shortTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "cd")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// newTestCtx builds a minimal AppContext with a temp CacheDir and a real
// logger so the Trace() deferred calls in the Rebuild* helpers don't panic.
func newTestCtx(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.CacheDir = shortTempDir(t)
	ctx.SetLogger(logging.NewLogger(io.Discard, io.Discard))
	return ctx
}

// startFakeCachedServer listens on ctx.CacheDir/cached.sock and, for each
// accepted connection, performs the version handshake (echoing the client's
// version so it matches), reads one RequestPkt, then replies with resp and
// closes the connection. It records the last request it saw. The listener is
// closed via t.Cleanup.
func startFakeCachedServer(t *testing.T, ctx *appcontext.AppContext, resp ResponsePkt) *struct {
	mu  sync.Mutex
	req *RequestPkt
} {
	t.Helper()

	sockPath := filepath.Join(ctx.CacheDir, "cached.sock")
	ln, err := net.Listen("unix", sockPath)
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })

	state := &struct {
		mu  sync.Mutex
		req *RequestPkt
	}{}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				dec := msgpack.NewDecoder(conn)
				enc := msgpack.NewEncoder(conn)

				// handshake: read client version, echo it back
				var clientVers []byte
				if err := dec.Decode(&clientVers); err != nil {
					return
				}
				if err := enc.Encode([]byte(utils.GetVersion())); err != nil {
					return
				}

				// read the request packet
				var req RequestPkt
				if err := dec.Decode(&req); err != nil {
					return
				}
				state.mu.Lock()
				state.req = &req
				state.mu.Unlock()

				// reply with the canned response then close (EOF)
				_ = enc.Encode(&resp)
			}(conn)
		}
	}()

	return state
}

// ---------------------------------------------------------------------------
// newClient happy path over a real unix socket (no spawn, no version mismatch)
// ---------------------------------------------------------------------------

func TestNewClientConnectsAndHandshakes(t *testing.T) {
	ctx := newTestCtx(t)
	startFakeCachedServer(t, ctx, ResponsePkt{})

	sockPath := filepath.Join(ctx.CacheDir, "cached.sock")
	c, err := newClient(sockPath, false)
	require.NoError(t, err)
	require.NotNil(t, c)
	require.NoError(t, c.Close())
}

func TestNewClientVersionMismatch(t *testing.T) {
	ctx := newTestCtx(t)

	sockPath := filepath.Join(ctx.CacheDir, "cached.sock")
	ln, err := net.Listen("unix", sockPath)
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		dec := msgpack.NewDecoder(conn)
		enc := msgpack.NewEncoder(conn)
		var v []byte
		_ = dec.Decode(&v)
		_ = enc.Encode([]byte("0.0.0-bogus"))
	}()

	c, err := newClient(sockPath, false)
	require.Nil(t, c)
	require.ErrorIs(t, err, ErrWrongVersion)
}

// ---------------------------------------------------------------------------
// RebuildStateFromStateFile / RebuildStateFromStore drive rebuildStateRequest
// end-to-end against the fake server.
// ---------------------------------------------------------------------------

func TestRebuildStateFromStateFileSuccess(t *testing.T) {
	ctx := newTestCtx(t)
	ctx.SetSecret([]byte("topsecret"))
	state := startFakeCachedServer(t, ctx, ResponsePkt{ExitCode: 0})

	repoID := uuid.New()
	storeConfig := map[string]string{"location": "fs:///tmp/x"}

	var stateID [32]byte
	stateID[0] = 0xab

	code, err := RebuildStateFromStateFile(ctx, stateID, repoID, storeConfig, true)
	require.NoError(t, err)
	require.Equal(t, 0, code)

	// the server received the request we built
	state.mu.Lock()
	defer state.mu.Unlock()
	require.NotNil(t, state.req)
	require.Equal(t, repoID, state.req.RepoID)
	require.Equal(t, storeConfig, state.req.StoreConfig)
	require.Equal(t, []byte("topsecret"), state.req.Secret)
	require.True(t, state.req.FireAndForget)
	require.Equal(t, stateID, [32]byte(state.req.StateID))
}

func TestRebuildStateFromStoreSuccess(t *testing.T) {
	ctx := newTestCtx(t)
	state := startFakeCachedServer(t, ctx, ResponsePkt{ExitCode: 0})

	repoID := uuid.New()
	storeConfig := map[string]string{"location": "fs:///tmp/y"}

	code, err := RebuildStateFromStore(ctx, repoID, storeConfig, false)
	require.NoError(t, err)
	require.Equal(t, 0, code)

	state.mu.Lock()
	defer state.mu.Unlock()
	require.NotNil(t, state.req)
	require.Equal(t, repoID, state.req.RepoID)
	require.False(t, state.req.FireAndForget)
	// StateID left zero for a full rebuild
	require.Equal(t, [32]byte{}, [32]byte(state.req.StateID))
}

// The server returns a non-zero exit code and an error string: the client must
// surface both.
func TestRebuildStateFromStoreServerError(t *testing.T) {
	ctx := newTestCtx(t)
	startFakeCachedServer(t, ctx, ResponsePkt{ExitCode: 3, Err: "boom"})

	code, err := RebuildStateFromStore(ctx, uuid.New(), map[string]string{}, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
	require.Equal(t, 3, code)
}

// When the server hangs up before replying, the decode loop sees io.EOF and
// returns (0, nil).
func TestRebuildStateEOFNoResponse(t *testing.T) {
	ctx := newTestCtx(t)

	sockPath := filepath.Join(ctx.CacheDir, "cached.sock")
	ln, err := net.Listen("unix", sockPath)
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		dec := msgpack.NewDecoder(conn)
		enc := msgpack.NewEncoder(conn)
		var v []byte
		_ = dec.Decode(&v)
		_ = enc.Encode([]byte(utils.GetVersion()))
		// read the request then close without sending a response
		var req RequestPkt
		_ = dec.Decode(&req)
		conn.Close()
	}()

	code, err := RebuildStateFromStore(ctx, uuid.New(), map[string]string{}, false)
	require.NoError(t, err)
	require.Equal(t, 0, code)
}

// rebuildStateRequest returns an error when it can't reach a server and the
// lockfile parent does not exist (so newClient fails fast rather than spawning).
func TestRebuildStateRequestDialFailsFast(t *testing.T) {
	ctx := newTestCtx(t)
	// point CacheDir at a non-existent directory so both the socket dial and
	// the lockfile creation fail, making newClient return quickly.
	ctx.CacheDir = filepath.Join(shortTempDir(t), "does-not-exist")

	req := &RequestPkt{RepoID: uuid.New()}
	code, err := rebuildStateRequest(ctx, req)
	require.Error(t, err)
	require.Equal(t, 1, code)
}

// sanity: io is used (keep import even if a path changes)
var _ = io.EOF
