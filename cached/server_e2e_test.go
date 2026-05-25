package cached

import (
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

func testLogger() *logging.Logger {
	return logging.NewLogger(io.Discard, io.Discard)
}

// fakeCachedServer is an in-process replacement for the real cached daemon.
// It binds a unix socket inside ctx.CacheDir and speaks the same wire protocol
// (msgpack: version → version → RequestPkt → ResponsePkt) so the cached client
// in this package never enters the spawn/exec path.
type fakeCachedServer struct {
	t        *testing.T
	listener net.Listener
	wg       sync.WaitGroup

	mu          sync.Mutex
	conns       []net.Conn
	requests    []RequestPkt
	response    ResponsePkt
	serverVers  []byte
	closeBefore string // "" | "handshake" | "request" | "response"
}

func newFakeCachedServer(t *testing.T, cacheDir string) *fakeCachedServer {
	t.Helper()
	sockPath := filepath.Join(cacheDir, "cached.sock")
	// macOS unix socket path limit is 104 bytes; warn but don't fail here so the
	// caller can route around it.
	if len(sockPath) > 104 {
		t.Fatalf("socket path too long for this OS (%d bytes): %s", len(sockPath), sockPath)
	}
	l, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen on %s: %v", sockPath, err)
	}
	s := &fakeCachedServer{t: t, listener: l, serverVers: []byte(utils.GetVersion())}
	s.wg.Add(1)
	go s.serve()
	return s
}

func (s *fakeCachedServer) serve() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.conns = append(s.conns, conn)
		s.mu.Unlock()
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer conn.Close()
			s.handle(conn)
		}()
	}
}

func (s *fakeCachedServer) handle(conn net.Conn) {
	dec := msgpack.NewDecoder(conn)
	enc := msgpack.NewEncoder(conn)

	// 1) client → server: version
	var clientVers []byte
	if err := dec.Decode(&clientVers); err != nil {
		return
	}

	s.mu.Lock()
	closeBefore := s.closeBefore
	respCopy := s.response
	versCopy := append([]byte(nil), s.serverVers...)
	s.mu.Unlock()

	if closeBefore == "handshake" {
		return
	}

	// 2) server → client: version
	if err := enc.Encode(versCopy); err != nil {
		return
	}

	if closeBefore == "request" {
		return
	}

	// 3) client → server: RequestPkt
	var pkt RequestPkt
	if err := dec.Decode(&pkt); err != nil {
		return
	}
	s.mu.Lock()
	s.requests = append(s.requests, pkt)
	s.mu.Unlock()

	if closeBefore == "response" {
		return
	}

	// 4) server → client: ResponsePkt — production code encodes `&response`
	// where response is already a *ResponsePkt; mirror that so the wire format
	// matches.
	r := &respCopy
	_ = enc.Encode(&r)
}

func (s *fakeCachedServer) close() {
	s.listener.Close()
	// Close any in-flight connections so handler goroutines blocked on Decode
	// return promptly. Without this, handlers can sit forever waiting for a
	// message the client never sends (e.g. when the client aborts the
	// handshake).
	s.mu.Lock()
	for _, c := range s.conns {
		_ = c.Close()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

func (s *fakeCachedServer) Requests() []RequestPkt {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]RequestPkt, len(s.requests))
	copy(out, s.requests)
	return out
}

func (s *fakeCachedServer) SetResponse(r ResponsePkt) {
	s.mu.Lock()
	s.response = r
	s.mu.Unlock()
}

func (s *fakeCachedServer) SetServerVersion(v []byte) {
	s.mu.Lock()
	s.serverVers = v
	s.mu.Unlock()
}

func (s *fakeCachedServer) SetCloseBefore(stage string) {
	s.mu.Lock()
	s.closeBefore = stage
	s.mu.Unlock()
}

// shortTempDir creates a tempdir under /tmp to keep unix-socket paths under the
// 104-byte limit on macOS. t.TempDir() roots under /var/folders/... which is
// already ~80 chars and overflows when we append cached.sock.
func shortTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "cached-test-")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func newTestCtx(t *testing.T) (*appcontext.AppContext, string) {
	dir := shortTempDir(t)
	ctx := appcontext.NewAppContext()
	ctx.CacheDir = dir
	t.Cleanup(ctx.Close)
	return ctx, dir
}

func TestRebuildStateRequestSuccess(t *testing.T) {
	ctx, dir := newTestCtx(t)
	srv := newFakeCachedServer(t, dir)
	defer srv.close()
	srv.SetResponse(ResponsePkt{ExitCode: 0})

	repoID := uuid.New()
	req := &RequestPkt{
		Secret:      []byte("topsecret"),
		RepoID:      repoID,
		StoreConfig: map[string]string{"location": "fs:///x"},
		StateID:     objects.MAC{1, 2, 3},
	}

	exitCode, err := rebuildStateRequest(ctx, req)
	if err != nil {
		t.Fatalf("rebuildStateRequest: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}

	got := srv.Requests()
	if len(got) != 1 {
		t.Fatalf("server saw %d requests, want 1", len(got))
	}
	if string(got[0].Secret) != "topsecret" {
		t.Fatalf("server.Secret = %q", got[0].Secret)
	}
	if got[0].RepoID != repoID {
		t.Fatalf("server.RepoID = %v, want %v", got[0].RepoID, repoID)
	}
	if got[0].StoreConfig["location"] != "fs:///x" {
		t.Fatalf("server.StoreConfig = %+v", got[0].StoreConfig)
	}
	if got[0].StateID != (objects.MAC{1, 2, 3}) {
		t.Fatalf("server.StateID = %v", got[0].StateID)
	}
}

func TestRebuildStateRequestServerError(t *testing.T) {
	ctx, dir := newTestCtx(t)
	srv := newFakeCachedServer(t, dir)
	defer srv.close()
	srv.SetResponse(ResponsePkt{ExitCode: 42, Err: "remote bust"})

	exitCode, err := rebuildStateRequest(ctx, &RequestPkt{})
	if err == nil || err.Error() != "remote bust" {
		t.Fatalf("err = %v, want 'remote bust'", err)
	}
	if exitCode != 42 {
		t.Fatalf("exitCode = %d, want 42", exitCode)
	}
}

func TestRebuildStateRequestWrongServerVersion(t *testing.T) {
	ctx, dir := newTestCtx(t)
	srv := newFakeCachedServer(t, dir)
	defer srv.close()
	srv.SetServerVersion([]byte("v0.0.0-bogus"))

	exitCode, err := rebuildStateRequest(ctx, &RequestPkt{})
	if err == nil {
		t.Fatal("expected version-mismatch error, got nil")
	}
	if !errors.Is(err, ErrWrongVersion) {
		t.Fatalf("err = %v, want errors.Is(err, ErrWrongVersion)", err)
	}
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
}

func TestRebuildStateRequestServerHangsUpAfterHandshake(t *testing.T) {
	// Production behavior: when the server hangs up before sending a response,
	// the client's Decode loop hits io.EOF and breaks, returning (0, nil). This
	// pins down that contract so we notice if it changes.
	ctx, dir := newTestCtx(t)
	srv := newFakeCachedServer(t, dir)
	defer srv.close()
	srv.SetCloseBefore("response")

	exitCode, err := rebuildStateRequest(ctx, &RequestPkt{})
	if err != nil {
		t.Fatalf("err = %v, want nil (EOF should be treated as clean close)", err)
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
}

func TestRebuildStateRequestServerSendsGarbage(t *testing.T) {
	// When the server completes the handshake but then sends invalid msgpack,
	// Decode returns a non-EOF error and the client should surface it.
	ctx, dir := newTestCtx(t)
	sockPath := filepath.Join(dir, "cached.sock")
	l, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		dec := msgpack.NewDecoder(conn)
		enc := msgpack.NewEncoder(conn)
		var v []byte
		_ = dec.Decode(&v)
		_ = enc.Encode([]byte(utils.GetVersion()))
		var pkt RequestPkt
		_ = dec.Decode(&pkt)
		// Garbage bytes that aren't a valid msgpack ResponsePkt envelope.
		_, _ = conn.Write([]byte{0xff, 0xff, 0xff, 0xff, 0xff})
	}()

	exitCode, err := rebuildStateRequest(ctx, &RequestPkt{})
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
}

func TestRebuildStateFromStoreSendsNilStateID(t *testing.T) {
	ctx, dir := newTestCtx(t)
	srv := newFakeCachedServer(t, dir)
	defer srv.close()

	// Logger must be set because RebuildStateFromStore traces on completion.
	// kcontext.GetLogger() returns nil otherwise; the trace call would NPE.
	ctx.SetLogger(testLogger())

	ctx.SetSecret([]byte("k"))
	repoID := uuid.New()
	_, err := RebuildStateFromStore(ctx, repoID, map[string]string{"location": "x"}, false)
	if err != nil {
		t.Fatalf("RebuildStateFromStore: %v", err)
	}
	reqs := srv.Requests()
	if len(reqs) != 1 {
		t.Fatalf("server saw %d requests, want 1", len(reqs))
	}
	if reqs[0].StateID != (objects.MAC{}) {
		t.Fatalf("StateID = %v, want zero (NilMac)", reqs[0].StateID)
	}
	if reqs[0].RepoID != repoID {
		t.Fatalf("RepoID = %v, want %v", reqs[0].RepoID, repoID)
	}
	if string(reqs[0].Secret) != "k" {
		t.Fatalf("Secret = %q", reqs[0].Secret)
	}
	if reqs[0].FireAndForget {
		t.Fatal("FireAndForget should be false")
	}
}

func TestRebuildStateFromStateFileSendsStateID(t *testing.T) {
	ctx, dir := newTestCtx(t)
	srv := newFakeCachedServer(t, dir)
	defer srv.close()
	ctx.SetLogger(testLogger())
	ctx.SetSecret([]byte("k"))

	repoID := uuid.New()
	wantState := objects.MAC{9, 9, 9, 9}
	_, err := RebuildStateFromStateFile(ctx, wantState, repoID, map[string]string{}, true)
	if err != nil {
		t.Fatalf("RebuildStateFromStateFile: %v", err)
	}
	reqs := srv.Requests()
	if len(reqs) != 1 {
		t.Fatalf("server saw %d requests, want 1", len(reqs))
	}
	if reqs[0].StateID != wantState {
		t.Fatalf("StateID = %v, want %v", reqs[0].StateID, wantState)
	}
	if !reqs[0].FireAndForget {
		t.Fatal("FireAndForget should be true (passed through from caller)")
	}
}
