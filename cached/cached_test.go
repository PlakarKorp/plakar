package cached

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

// shortCacheDir returns a temporary directory with a short path. Unix domain
// socket paths are limited (~104 bytes on macOS), and t.TempDir() can exceed
// that, so we mint our own short-named dir under the system temp root.
func shortCacheDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "pc")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// newTestContext returns an AppContext whose CacheDir points at a temporary
// directory and that has a no-op logger installed (the cached functions call
// ctx.GetLogger().Trace, which would panic on a nil logger).
func newTestContext(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.SetLogger(logging.NewLogger(io.Discard, io.Discard))
	ctx.CacheDir = shortCacheDir(t)
	t.Cleanup(ctx.Close)
	return ctx
}

// serverBehavior controls how the fake cached server responds on a connection.
type serverBehavior struct {
	// version the server reports during the handshake. If empty, the real
	// running version is used so the handshake succeeds.
	version string

	// closeAfterHandshake makes the server hang up right after the version
	// exchange, before reading the request (simulates an EOF on decode).
	closeAfterHandshake bool

	// closeBeforeHandshake makes the server hang up immediately on connect.
	closeBeforeHandshake bool

	// response is what the server sends back after receiving the request. If
	// nil and closeAfterRequest is false, a zero-value (success) response is
	// sent.
	response *ResponsePkt

	// closeAfterRequest hangs up after reading the request without sending a
	// response (client should observe EOF).
	closeAfterRequest bool

	// onRequest, if set, receives every decoded request packet.
	onRequest func(*RequestPkt)
}

// startFakeServer listens on <cacheDir>/cached.sock and serves connections
// according to the supplied behavior. It returns once the listener is up.
func startFakeServer(t *testing.T, ctx *appcontext.AppContext, b serverBehavior) {
	t.Helper()

	socketPath := filepath.Join(ctx.CacheDir, "cached.sock")
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen on %s: %v", socketPath, err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var conns []net.Conn

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
			wg.Add(1)
			go func(conn net.Conn) {
				defer wg.Done()
				defer conn.Close()
				serveConn(conn, b)
			}(conn)
		}
	}()

	t.Cleanup(func() {
		ln.Close()
		// Close any accepted connections so a serveConn goroutine blocked in
		// Read (e.g. waiting for a request the client never sends after a
		// failed handshake) unblocks instead of hanging until the test
		// timeout.
		mu.Lock()
		for _, c := range conns {
			c.Close()
		}
		mu.Unlock()
		wg.Wait()
	})
}

func serveConn(conn net.Conn, b serverBehavior) {
	if b.closeBeforeHandshake {
		return
	}

	enc := msgpack.NewEncoder(conn)
	dec := msgpack.NewDecoder(conn)

	// Read the client version.
	var clientvers []byte
	if err := dec.Decode(&clientvers); err != nil {
		return
	}

	// Reply with our version (or the configured one).
	vers := b.version
	if vers == "" {
		vers = utils.GetVersion()
	}
	if err := enc.Encode([]byte(vers)); err != nil {
		return
	}

	if b.closeAfterHandshake {
		return
	}

	// Read the request packet.
	pkt := &RequestPkt{}
	if err := dec.Decode(pkt); err != nil {
		return
	}
	if b.onRequest != nil {
		b.onRequest(pkt)
	}

	if b.closeAfterRequest {
		return
	}

	resp := b.response
	if resp == nil {
		resp = &ResponsePkt{}
	}
	_ = enc.Encode(resp)
}

func TestRebuildStateFromStoreSuccess(t *testing.T) {
	ctx := newTestContext(t)

	var got *RequestPkt
	startFakeServer(t, ctx, serverBehavior{
		onRequest: func(p *RequestPkt) { got = p },
		response:  &ResponsePkt{ExitCode: 0},
	})

	repoID := uuid.New()
	storeConfig := map[string]string{"location": "fs:/tmp/repo"}
	ctx.SetSecret([]byte("s3cr3t"))

	code, err := RebuildStateFromStore(ctx, repoID, storeConfig, false)
	if err != nil {
		t.Fatalf("RebuildStateFromStore() error = %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	if got == nil {
		t.Fatal("server never received a request")
	}
	if got.RepoID != repoID {
		t.Errorf("RepoID = %v, want %v", got.RepoID, repoID)
	}
	if got.StoreConfig["location"] != "fs:/tmp/repo" {
		t.Errorf("StoreConfig = %v, want location set", got.StoreConfig)
	}
	if string(got.Secret) != "s3cr3t" {
		t.Errorf("Secret = %q, want %q", got.Secret, "s3cr3t")
	}
	if got.FireAndForget {
		t.Error("FireAndForget = true, want false")
	}
	// A full rebuild from the store carries the nil MAC.
	if got.StateID != (objects.MAC{}) {
		t.Errorf("StateID = %x, want zero", got.StateID)
	}
}

func TestRebuildStateFromStateFileSuccess(t *testing.T) {
	ctx := newTestContext(t)

	var got *RequestPkt
	startFakeServer(t, ctx, serverBehavior{
		onRequest: func(p *RequestPkt) { got = p },
	})

	repoID := uuid.New()
	var stateID objects.MAC
	for i := range stateID {
		stateID[i] = byte(i)
	}

	code, err := RebuildStateFromStateFile(ctx, stateID, repoID, map[string]string{}, true)
	if err != nil {
		t.Fatalf("RebuildStateFromStateFile() error = %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if got == nil {
		t.Fatal("server never received a request")
	}
	if got.StateID != stateID {
		t.Errorf("StateID = %x, want %x", got.StateID, stateID)
	}
	if !got.FireAndForget {
		t.Error("FireAndForget = false, want true")
	}
}

func TestRebuildStateServerReturnsError(t *testing.T) {
	ctx := newTestContext(t)

	startFakeServer(t, ctx, serverBehavior{
		response: &ResponsePkt{ExitCode: -1, Err: "boom"},
	})

	code, err := RebuildStateFromStore(ctx, uuid.New(), nil, false)
	if err == nil {
		t.Fatal("expected an error from the server response")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error = %v, want it to contain %q", err, "boom")
	}
	if code != -1 {
		t.Errorf("exit code = %d, want -1", code)
	}
}

func TestRebuildStateNonZeroExitNoError(t *testing.T) {
	ctx := newTestContext(t)

	// Exit code set but Err empty: the client should surface the code with a
	// nil error.
	startFakeServer(t, ctx, serverBehavior{
		response: &ResponsePkt{ExitCode: 42, Err: ""},
	})

	code, err := RebuildStateFromStore(ctx, uuid.New(), nil, false)
	if err != nil {
		t.Fatalf("unexpected error = %v", err)
	}
	if code != 42 {
		t.Errorf("exit code = %d, want 42", code)
	}
}

func TestRebuildStateServerClosesAfterRequest(t *testing.T) {
	ctx := newTestContext(t)

	// Server reads the request then hangs up without replying. The decode loop
	// hits io.EOF and returns success (code 0, nil err) per the current
	// implementation.
	startFakeServer(t, ctx, serverBehavior{closeAfterRequest: true})

	code, err := RebuildStateFromStore(ctx, uuid.New(), nil, false)
	if err != nil {
		t.Fatalf("unexpected error = %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestRebuildStateVersionMismatch(t *testing.T) {
	ctx := newTestContext(t)

	startFakeServer(t, ctx, serverBehavior{version: "v0.0.0-bogus-version"})

	code, err := RebuildStateFromStore(ctx, uuid.New(), nil, false)
	if err == nil {
		t.Fatal("expected a version mismatch error")
	}
	if !errorsIsWrongVersion(err) {
		t.Errorf("error = %v, want ErrWrongVersion", err)
	}
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func errorsIsWrongVersion(err error) bool {
	// rebuildStateRequest wraps the handshake error from newClient; check the
	// message since it is fmt.Errorf-wrapped with %w on ErrWrongVersion.
	for e := err; e != nil; {
		if e == ErrWrongVersion {
			return true
		}
		un, ok := e.(interface{ Unwrap() error })
		if !ok {
			break
		}
		e = un.Unwrap()
	}
	return strings.Contains(err.Error(), ErrWrongVersion.Error())
}

func TestRebuildStateContextCancelled(t *testing.T) {
	ctx := newTestContext(t)

	// Server completes the handshake then closes without replying. We cancel
	// the context first so the decode failure is reported as the context error.
	startFakeServer(t, ctx, serverBehavior{closeAfterHandshake: true})

	ctx.Cancel(nil)

	code, err := RebuildStateFromStore(ctx, uuid.New(), nil, false)
	if err == nil {
		t.Fatal("expected an error when the context is cancelled")
	}
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestNewClientHandshakeOK(t *testing.T) {
	ctx := newTestContext(t)
	startFakeServer(t, ctx, serverBehavior{closeAfterHandshake: true})

	socketPath := filepath.Join(ctx.CacheDir, "cached.sock")
	c, err := newClient(ctx, socketPath, false)
	if err != nil {
		t.Fatalf("newClient() error = %v", err)
	}
	if c == nil {
		t.Fatal("newClient() returned nil client")
	}
	if err := c.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestNewClientIgnoreVersion(t *testing.T) {
	ctx := newTestContext(t)
	startFakeServer(t, ctx, serverBehavior{
		version:             "totally-different",
		closeAfterHandshake: true,
	})

	socketPath := filepath.Join(ctx.CacheDir, "cached.sock")

	// ignoreVersion=false should fail.
	if c, err := newClient(ctx, socketPath, false); err == nil {
		c.Close()
		t.Fatal("expected version mismatch with ignoreVersion=false")
	}

	// ignoreVersion=true should succeed despite the mismatch.
	c, err := newClient(ctx, socketPath, true)
	if err != nil {
		t.Fatalf("newClient(ignoreVersion=true) error = %v", err)
	}
	c.Close()
}

func TestHandshakeServerClosesImmediately(t *testing.T) {
	ctx := newTestContext(t)
	startFakeServer(t, ctx, serverBehavior{closeBeforeHandshake: true})

	socketPath := filepath.Join(ctx.CacheDir, "cached.sock")
	if c, err := newClient(ctx, socketPath, false); err == nil {
		c.Close()
		t.Fatal("expected an error when the server closes before the handshake")
	}
}

// TestNewClientConcurrentConnections exercises several simultaneous clients
// against one server, ensuring the handshake path is safe to use concurrently.
func TestNewClientConcurrentConnections(t *testing.T) {
	ctx := newTestContext(t)
	startFakeServer(t, ctx, serverBehavior{response: &ResponsePkt{}})

	const clients = 10
	var wg sync.WaitGroup
	errs := make(chan error, clients)

	for i := 0; i < clients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := RebuildStateFromStore(ctx, uuid.New(), nil, false)
			errs <- err
		}()
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent RebuildStateFromStore() error = %v", err)
		}
	}
}

// TestRebuildStateRequestPropagatesPayload checks the request marshalling for a
// from-state-file rebuild that is not fire-and-forget.
func TestRebuildStateRequestPropagatesPayload(t *testing.T) {
	ctx := newTestContext(t)

	var got *RequestPkt
	startFakeServer(t, ctx, serverBehavior{
		onRequest: func(p *RequestPkt) { got = p },
		response:  &ResponsePkt{},
	})

	repoID := uuid.New()
	stateID := objects.MAC{0xaa, 0xbb, 0xcc}
	cfg := map[string]string{"a": "1", "b": "2"}
	ctx.SetSecret([]byte("key"))

	if _, err := RebuildStateFromStateFile(ctx, stateID, repoID, cfg, false); err != nil {
		t.Fatalf("RebuildStateFromStateFile() error = %v", err)
	}

	if got == nil {
		t.Fatal("no request received")
	}
	if got.RepoID != repoID || got.StateID != stateID {
		t.Errorf("ids mismatch: repo=%v state=%x", got.RepoID, got.StateID)
	}
	if got.StoreConfig["a"] != "1" || got.StoreConfig["b"] != "2" {
		t.Errorf("StoreConfig = %v", got.StoreConfig)
	}
	if got.FireAndForget {
		t.Error("FireAndForget should be false")
	}
}

// TestNewClientBlocksOnLockThenConnects exercises the lockfile branch of
// newClient: the first net.Dial fails (no server yet), so the client opens the
// .lock file and blocks on flock because the test already holds it. While the
// client is blocked we stand up the server and release the lock; the client
// then acquires the lock, retries the dial, and connects — all without ever
// spawning a child process.
func TestNewClientBlocksOnLockThenConnects(t *testing.T) {
	ctx := newTestContext(t)
	socketPath := filepath.Join(ctx.CacheDir, "cached.sock")

	// Pre-acquire the lock so the client blocks inside flock().
	held, err := os.OpenFile(socketPath+".lock", os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		t.Fatalf("open lock: %v", err)
	}
	if err := flock(held); err != nil {
		t.Fatalf("flock: %v", err)
	}

	type result struct {
		c   *Client
		err error
	}
	done := make(chan result, 1)
	go func() {
		c, err := newClient(ctx, socketPath, false)
		done <- result{c, err}
	}()

	// Give the client time to fail its first dial and start blocking on the
	// lock, then bring the server up and release the lock.
	time.Sleep(50 * time.Millisecond)
	startFakeServer(t, ctx, serverBehavior{closeAfterHandshake: true})
	held.Close() // releases the flock

	select {
	case res := <-done:
		if res.err != nil {
			t.Fatalf("newClient() error = %v", res.err)
		}
		res.c.Close()
	case <-time.After(10 * time.Second):
		t.Fatal("newClient() did not return after lock release and server start")
	}
}
