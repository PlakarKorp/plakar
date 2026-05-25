package testing

import (
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cached"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/vmihailenco/msgpack/v5"
)

// FakeCachedServer is an in-process replacement for the real cached daemon.
// Tests that exercise subcommands depending on cached (sync, backup, ...)
// can call StartFakeCachedServer to avoid the production code path that
// fork-execs `<plakar binary> cached`, which is impossible inside `go test`.
//
// Usage:
//
//	srv := ptesting.StartFakeCachedServer(t, ctx)
//	defer srv.Close()
//
// The server overrides ctx.CacheDir to a short-path tempdir (because macOS
// limits unix socket paths to 104 bytes and the default t.TempDir() is too
// long) and binds a unix socket at $CacheDir/cached.sock. It replies with
// ExitCode=0 to every request unless SetResponse is called.
type FakeCachedServer struct {
	listener net.Listener
	wg       sync.WaitGroup

	mu       sync.Mutex
	conns    []net.Conn
	requests []cached.RequestPkt
	response cached.ResponsePkt
}

// StartFakeCachedServer binds a unix socket at $CacheDir/cached.sock so the
// production client connects to us instead of fork-execing a real daemon.
//
// If the existing CacheDir would push the socket path past macOS' 104-byte
// limit, this function moves CacheDir to a short tempdir under /tmp. Beware:
// that re-points everything keyed off CacheDir, including the snapshot store
// layout. Tests that already populated CacheDir before calling this MUST keep
// their initial path short enough — fail fast otherwise.
func StartFakeCachedServer(t *testing.T, ctx *appcontext.AppContext) *FakeCachedServer {
	t.Helper()

	// Use the caller's CacheDir if the resulting socket path fits.
	sock := filepath.Join(ctx.CacheDir, "cached.sock")
	if ctx.CacheDir == "" || len(sock) > 104 {
		dir, err := os.MkdirTemp("/tmp", "plakar-cached-")
		if err != nil {
			t.Fatalf("MkdirTemp: %v", err)
		}
		t.Cleanup(func() { os.RemoveAll(dir) })
		ctx.CacheDir = dir
		sock = filepath.Join(dir, "cached.sock")
	}

	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen on %s: %v", sock, err)
	}
	s := &FakeCachedServer{listener: l}
	s.wg.Add(1)
	go s.serve()
	return s
}

// SetResponse overrides the ResponsePkt the fake server replies with.
func (s *FakeCachedServer) SetResponse(r cached.ResponsePkt) {
	s.mu.Lock()
	s.response = r
	s.mu.Unlock()
}

// Requests returns a copy of all RequestPkts seen by the fake server so far.
func (s *FakeCachedServer) Requests() []cached.RequestPkt {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]cached.RequestPkt, len(s.requests))
	copy(out, s.requests)
	return out
}

// Close stops accepting new connections and closes in-flight ones so handler
// goroutines blocked on Decode return promptly.
func (s *FakeCachedServer) Close() {
	s.listener.Close()
	s.mu.Lock()
	for _, c := range s.conns {
		_ = c.Close()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

func (s *FakeCachedServer) serve() {
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

func (s *FakeCachedServer) handle(conn net.Conn) {
	dec := msgpack.NewDecoder(conn)
	enc := msgpack.NewEncoder(conn)

	var clientVers []byte
	if err := dec.Decode(&clientVers); err != nil {
		return
	}
	if err := enc.Encode([]byte(utils.GetVersion())); err != nil {
		return
	}

	var pkt cached.RequestPkt
	if err := dec.Decode(&pkt); err != nil {
		return
	}
	s.mu.Lock()
	s.requests = append(s.requests, pkt)
	resp := s.response
	s.mu.Unlock()

	r := &resp
	_ = enc.Encode(&r)
}
