package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func TestDynamicHandlerRootCallsList(t *testing.T) {
	listed := false
	h := NewDynamicSnapshotHandler(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			listed = true
			fmt.Fprint(w, "INDEX")
		},
		func(ctx context.Context, snap string) (fs.FS, error) {
			t.Fatalf("open should not be called for root")
			return nil, nil
		},
	)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.True(t, listed)
	require.Contains(t, rec.Body.String(), "INDEX")
}

func TestDynamicHandlerNotFound(t *testing.T) {
	h := NewDynamicSnapshotHandler(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {},
		func(ctx context.Context, snap string) (fs.FS, error) {
			return nil, fs.ErrNotExist
		},
	)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/abcd/", nil))
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDynamicHandlerOpenError(t *testing.T) {
	h := NewDynamicSnapshotHandler(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {},
		func(ctx context.Context, snap string) (fs.FS, error) {
			return nil, fmt.Errorf("boom")
		},
	)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/abcd/file", nil))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDynamicHandlerServesSnapshotFile(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	_ = ctx
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	defer snap.Close()

	vfs, err := snap.Filesystem()
	require.NoError(t, err)

	const snapID = "deadbeef"
	h := NewDynamicSnapshotHandler(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {},
		func(ctx context.Context, s string) (fs.FS, error) {
			require.Equal(t, snapID, s)
			return vfs, nil
		},
	)

	// The snapshot vfs is rooted at the import directory; resolve the absolute
	// path of dummy.txt within the vfs.
	backupDir := snap.Header.GetSource(0).Importer.Directory
	filePath := "/" + snapID + backupDir + "/subdir/dummy.txt"

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, filePath, nil))
	require.Equal(t, http.StatusOK, rec.Code)
	body, _ := io.ReadAll(rec.Body)
	require.Equal(t, "hello dummy", string(body))
}

// chroot serving: http.FileServer over the chroot vfs serves files directly
// (mirrors the chrootfs branch of ExecuteHTTP).
func TestChrootFileServing(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, _ := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	defer snap.Close()

	vfs, err := snap.Filesystem()
	require.NoError(t, err)

	backupDir := snap.Header.GetSource(0).Importer.Directory
	subPath := path.Join(backupDir, "subdir")
	chroot, err := fs.Sub(vfs, strings.TrimPrefix(subPath, "/"))
	require.NoError(t, err)

	handler := http.FileServer(http.FS(chroot))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/dummy.txt", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	body, _ := io.ReadAll(rec.Body)
	require.Equal(t, "hello dummy", string(body))
}
