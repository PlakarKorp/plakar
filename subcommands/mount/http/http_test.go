package http

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestDynamicHandlerRootCallsList(t *testing.T) {
	called := false
	h := NewDynamicSnapshotHandler(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			called = true
			fmt.Fprint(w, "INDEX")
		},
		func(ctx context.Context, id string) (fs.FS, error) {
			t.Fatal("open should not be called for /")
			return nil, nil
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	require.True(t, called, "list should be called for the root path")
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "INDEX")
}

func TestDynamicHandlerEmptyPathNormalizesToRoot(t *testing.T) {
	// path.Clean("") -> "." -> "/" so list is invoked.
	called := false
	h := NewDynamicSnapshotHandler(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) { called = true },
		func(ctx context.Context, id string) (fs.FS, error) { return nil, nil },
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.URL.Path = "" // force the "." normalization branch
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.True(t, called)
}

func TestDynamicHandlerServesSnapshotFile(t *testing.T) {
	memFS := fstest.MapFS{
		"hello.txt": &fstest.MapFile{Data: []byte("world")},
	}
	var gotID string
	h := NewDynamicSnapshotHandler(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {},
		func(ctx context.Context, id string) (fs.FS, error) {
			gotID = id
			return memFS, nil
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/abc123/hello.txt", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	require.Equal(t, "abc123", gotID, "the first path segment is the snapshot id")
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "world")
}

func driveLetterHandler(t *testing.T) func(target string) *httptest.ResponseRecorder {
	t.Helper()

	memFS := fstest.MapFS{
		"C:/tmp/hello.txt": &fstest.MapFile{Data: []byte("world")},
		"plain/x.txt":      &fstest.MapFile{Data: []byte("x")},
	}
	h := NewDynamicSnapshotHandler(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {},
		func(ctx context.Context, id string) (fs.FS, error) { return memFS, nil },
	)

	return func(target string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
		return w
	}
}

func TestDynamicHandlerRedirectsDriveLetterDirectoryToAbsolutePath(t *testing.T) {
	serve := driveLetterHandler(t)

	w := serve("/abc123/C:")
	require.Equal(t, http.StatusMovedPermanently, w.Code)
	require.Equal(t, "/abc123/C:/", w.Header().Get("Location"))

	w = serve("/abc123/C:?k=v")
	require.Equal(t, http.StatusMovedPermanently, w.Code)
	require.Equal(t, "/abc123/C:/?k=v", w.Header().Get("Location"))
}

func TestDynamicHandlerKeepsRelativeRedirectForColonlessDirectory(t *testing.T) {
	serve := driveLetterHandler(t)

	w := serve("/abc123/plain")
	require.Equal(t, http.StatusMovedPermanently, w.Code)
	require.Equal(t, "plain/", w.Header().Get("Location"))
}

func TestDynamicHandlerServesDriveLetterSubtree(t *testing.T) {
	serve := driveLetterHandler(t)

	w := serve("/abc123/C:/")
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "tmp/")

	w = serve("/abc123/C:/tmp/hello.txt")
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "world")
}

func TestDynamicHandlerNotFound(t *testing.T) {
	h := NewDynamicSnapshotHandler(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {},
		func(ctx context.Context, id string) (fs.FS, error) {
			return nil, fs.ErrNotExist
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/missing/x", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDynamicHandlerOpenError(t *testing.T) {
	h := NewDynamicSnapshotHandler(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {},
		func(ctx context.Context, id string) (fs.FS, error) {
			return nil, errors.New("boom")
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/bad/x", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().(*net.TCPAddr)
	_ = l.Close()
	return fmt.Sprintf("127.0.0.1:%d", addr.Port)
}

func TestExecuteHTTPChrootStartServeShutdown(t *testing.T) {
	// With a chroot fs, ExecuteHTTP serves a plain file server and never touches
	// the cached daemon / dynamic handler. Drive start -> request -> shutdown.
	repo, ctx := ptesting.GenerateRepository(t, nil, nil, nil)

	chroot := fstest.MapFS{
		"file.txt": &fstest.MapFile{Data: []byte("chroot-content")},
	}

	addr := freePort(t)
	errCh := make(chan struct {
		status int
		err    error
	}, 1)
	go func() {
		status, err := ExecuteHTTP(ctx, repo, "http://"+addr, nil, chroot)
		errCh <- struct {
			status int
			err    error
		}{status, err}
	}()

	base := "http://" + addr
	var reached bool
	for waited := time.Duration(0); waited < 3*time.Second; waited += 25 * time.Millisecond {
		if resp, err := http.Get(base + "/file.txt"); err == nil {
			require.Equal(t, http.StatusOK, resp.StatusCode)
			_ = resp.Body.Close()
			reached = true
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	require.True(t, reached, "server never became reachable")

	// Cancel the app context to trigger graceful shutdown.
	ctx.GetInner().Cancel(nil)

	select {
	case res := <-errCh:
		require.NoError(t, res.err)
		require.Equal(t, 0, res.status)
	case <-time.After(8 * time.Second):
		t.Fatal("ExecuteHTTP did not return after context cancellation")
	}
}

func TestExecuteHTTPListenError(t *testing.T) {
	// A malformed address makes ListenAndServe fail immediately, returning
	// status 1.
	repo, ctx := ptesting.GenerateRepository(t, nil, nil, nil)

	chroot := fstest.MapFS{}
	status, err := ExecuteHTTP(ctx, repo, "http://256.256.256.256:99999", nil, chroot)
	require.Error(t, err)
	require.Equal(t, 1, status)
}
