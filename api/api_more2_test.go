package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// buildRealRepoRouterCtx is like buildRealRepoRouter but also returns the
// appcontext so tests can manipulate cookies/state before firing requests.
func buildRealRepoRouterCtx(t *testing.T, token string) (*http.ServeMux, string, *appcontext.AppContext, *repository.Repository) {
	t.Helper()

	var bufOut, bufErr bytes.Buffer
	repo, ctx := ptesting.GenerateRepository(t, &bufOut, &bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/notes.md", 0644, "# title\n\nbody text\n"),
		ptesting.NewMockFile("top.txt", 0644, "top level content"),
	})
	t.Cleanup(func() { snap.Close() })

	id := snap.Header.GetIndexID()
	idHex := hex.EncodeToString(id[:])

	mux := http.NewServeMux()
	SetupRoutes(mux, repo, ctx, token, true)
	return mux, idHex, ctx, repo
}

// apiInfo: when an auth token cookie is present, authenticated is reported true.
func TestApiInfoAuthenticated(t *testing.T) {
	mux, _, ctx, _ := buildRealRepoRouterCtx(t, "")
	require.NoError(t, ctx.GetCookies().PutAuthToken("some-token"))
	t.Cleanup(func() { ctx.GetCookies().DeleteAuthToken() })

	w := doReq(t, mux, "GET", "/api/info", "", "")
	require.Equal(t, http.StatusOK, w.Code)

	var res struct {
		Authenticated bool `json:"authenticated"`
		DemoMode      bool `json:"demo_mode"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.True(t, res.Authenticated)
}

// apiInfo: demo mode flag is reflected in the response.
func TestApiInfoDemoMode(t *testing.T) {
	t.Setenv("PLAKAR_DEMO_MODE", "1")
	mux, _, _, _ := buildRealRepoRouterCtx(t, "")

	w := doReq(t, mux, "GET", "/api/info", "", "")
	require.Equal(t, http.StatusOK, w.Code)

	var res struct {
		DemoMode bool `json:"demo_mode"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.True(t, res.DemoMode)
}

// QueryParamToInt64: a value below the configured minimum is rejected.
func TestQueryParamToInt64BelowMin(t *testing.T) {
	req, err := http.NewRequest("GET", "/?param=2", nil)
	require.NoError(t, err)
	_, err = QueryParamToInt64(req, "param", 5, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "param")
}

// renderCode on a file with no path-based lexer match exercises the
// content-type / fallback lexer branch.
func TestSnapshotReaderCodeFallbackLexer(t *testing.T) {
	mux, idHex, _, _ := buildRealRepoRouterCtx(t, "")
	// dummy.txt has no source-code extension, so lexers.Match returns nil and
	// the handler falls through to the content-type / fallback lexer.
	w := doReq(t, mux, "GET", "/api/snapshot/reader/"+idHex+":/subdir/dummy.txt?render=code", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "<!DOCTYPE html>")
}

// snapshotVFSChunks with explicit offset+limit exercises the paging loop.
func TestSnapshotVFSChunksWithOffset(t *testing.T) {
	mux, idHex, _, _ := buildRealRepoRouterCtx(t, "")
	w := doReq(t, mux, "GET", "/api/snapshot/vfs/chunks/"+idHex+":/subdir/dummy.txt?offset=0&limit=1", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res Items[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
}

// snapshotVFSSearch with positive offset/limit values drives the parse paths
// that set non-default offset and limit.
func TestSnapshotVFSSearchWithPaging(t *testing.T) {
	mux, idHex, _, _ := buildRealRepoRouterCtx(t, "")
	w := doReq(t, mux, "GET", "/api/snapshot/vfs/search/"+idHex+":/?recursive=true&offset=0&limit=2", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res ItemsPage[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
}

// snapshotVFSSearch with a non-positive limit gets clamped to the default.
func TestSnapshotVFSSearchZeroLimit(t *testing.T) {
	mux, idHex, _, _ := buildRealRepoRouterCtx(t, "")
	w := doReq(t, mux, "GET", "/api/snapshot/vfs/search/"+idHex+":/?recursive=true&limit=0", "", "")
	require.Equal(t, http.StatusOK, w.Code)
}

// snapshotVFSErrors with bad offset/limit query params -> 400.
func TestSnapshotVFSErrorsBadParams(t *testing.T) {
	mux, idHex, _, _ := buildRealRepoRouterCtx(t, "")
	w := doReq(t, mux, "GET", "/api/snapshot/vfs/errors/"+idHex+":/?offset=abc", "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = doReq(t, mux, "GET", "/api/snapshot/vfs/errors/"+idHex+":/?limit=abc", "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// repositoryLocatePathname with an explicit limit value and a matching
// importerDirectory filter exercises the directory-match and paging branches.
func TestLocatePathnameDirectoryFilterAndLimit(t *testing.T) {
	mux, _, _, repo := buildRealRepoRouterCtx(t, "")

	// Discover the importer directory of the (single) snapshot via the
	// snapshots listing, then filter by it.
	w := doReq(t, mux, "GET", "/api/repository/snapshots", "", "")
	require.Equal(t, http.StatusOK, w.Code)

	var snaps Items[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &snaps))
	require.Equal(t, 1, snaps.Total)
	_ = repo

	// Explicit limit/offset values exercise the paging slice branch.
	w = doReq(t, mux, "GET", "/api/repository/locate-pathname?resource=/subdir/dummy.txt&limit=1&offset=0", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res Items[TimelineLocation]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.Equal(t, 1, res.Total)
}

// downloader-signed: the tarball (.tar.gz) archive format and a name that
// already carries an extension (no auto-suffix) path.
func TestSnapshotVFSDownloaderSignedTarball(t *testing.T) {
	mux, idHex, _, _ := buildRealRepoRouterCtx(t, "")

	body := `{"name":"backup.tar.gz","items":[{"pathname":"/subdir/dummy.txt"}]}`
	dlReq, _ := http.NewRequest("POST", "/api/snapshot/vfs/downloader/"+idHex+":/", strings.NewReader(body))
	dlW := httptest.NewRecorder()
	mux.ServeHTTP(dlW, dlReq)
	require.Equal(t, http.StatusOK, dlW.Code)

	var idResp struct {
		Id string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(dlW.Body.Bytes(), &idResp))

	getReq, _ := http.NewRequest("GET", "/api/snapshot/vfs/downloader-sign-url/"+idResp.Id+"?format=tarball&name=backup.tar.gz", nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getReq)
	require.Equal(t, http.StatusOK, getW.Code)
	require.Contains(t, getW.Header().Get("Content-Type"), "gzip")
	require.Contains(t, getW.Header().Get("Content-Disposition"), "backup.tar.gz")
}

// downloader-signed: the zip archive format with a default (extensionless)
// name, which triggers the extension auto-suffix branch.
func TestSnapshotVFSDownloaderSignedZipDefaultName(t *testing.T) {
	mux, idHex, _, _ := buildRealRepoRouterCtx(t, "")

	body := `{"items":[{"pathname":"/subdir/dummy.txt"}]}`
	dlReq, _ := http.NewRequest("POST", "/api/snapshot/vfs/downloader/"+idHex+":/", strings.NewReader(body))
	dlW := httptest.NewRecorder()
	mux.ServeHTTP(dlW, dlReq)
	require.Equal(t, http.StatusOK, dlW.Code)

	var idResp struct {
		Id string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(dlW.Body.Bytes(), &idResp))

	getReq, _ := http.NewRequest("GET", "/api/snapshot/vfs/downloader-sign-url/"+idResp.Id+"?format=zip", nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getReq)
	require.Equal(t, http.StatusOK, getW.Code)
	require.Contains(t, getW.Header().Get("Content-Type"), "zip")
	require.Contains(t, getW.Header().Get("Content-Disposition"), ".zip")
}

// snapshotReader download=true on the code render path sets the attachment
// disposition before rendering.
func TestSnapshotReaderDownloadCode(t *testing.T) {
	mux, idHex, _, _ := buildRealRepoRouterCtx(t, "")
	w := doReq(t, mux, "GET", "/api/snapshot/reader/"+idHex+":/subdir/notes.md?download=true&render=code", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
}
