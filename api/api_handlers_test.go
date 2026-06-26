package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// buildRealRepoRouter creates a real fs-backed repository populated with a
// single snapshot and returns a router wired with norefresh=true (so the
// handlers don't try to rebuild state from a store config we don't have) along
// with the snapshot id (hex) for use in request paths.
func buildRealRepoRouter(t *testing.T, token string) (*http.ServeMux, string) {
	t.Helper()

	var bufOut, bufErr bytes.Buffer
	repo, ctx := ptesting.GenerateRepository(t, &bufOut, &bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/code.go", 0644, "package main\n\nfunc main() {}\n"),
		ptesting.NewMockFile("top.txt", 0644, "top level content"),
	})
	t.Cleanup(func() { snap.Close() })

	id := snap.Header.GetIndexID()
	idHex := hex.EncodeToString(id[:])

	mux := http.NewServeMux()
	// norefresh=true: avoids cached.RebuildStateFromStore which needs StoreConfig.
	SetupRoutes(mux, repo, ctx, token, true)

	return mux, idHex
}

func doReq(t *testing.T, mux *http.ServeMux, method, target, auth string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest(method, target, strings.NewReader(body))
	require.NoError(t, err)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestApiInfo(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/info", "", "")
	require.Equal(t, http.StatusOK, w.Code)

	var res struct {
		RepositoryId  string `json:"repository_id"`
		Authenticated bool   `json:"authenticated"`
		Version       string `json:"version"`
		Browsable     bool   `json:"browsable"`
		DemoMode      bool   `json:"demo_mode"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.NotEmpty(t, res.RepositoryId)
	require.NotEmpty(t, res.Version)
	require.False(t, res.Authenticated)
	require.False(t, res.DemoMode)
}

func TestRepositoryInfo(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/repository/info", "", "")
	require.Equal(t, http.StatusOK, w.Code)

	var res Item[RepositoryInfoResponse]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.NotEmpty(t, res.Item.Location)
	require.Equal(t, 1, res.Item.Snapshots.Total)
	require.NotEmpty(t, res.Item.OS)
	require.NotEmpty(t, res.Item.Arch)
}

func TestRepositorySnapshots(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/repository/snapshots", "", "")
	require.Equal(t, http.StatusOK, w.Code)

	var res Items[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.Equal(t, 1, res.Total)
	require.Len(t, res.Items, 1)
}

func TestRepositorySnapshotsFilters(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/repository/snapshots?importer=doesnotexist", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res Items[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.Equal(t, 0, res.Total)
	require.Empty(t, res.Items)
}

func TestRepositorySnapshotsBadParams(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")

	cases := []struct {
		name   string
		query  string
		status int
	}{
		{"bad offset", "offset=abc", http.StatusBadRequest},
		{"bad limit", "limit=abc", http.StatusBadRequest},
		{"bad sort", "sort=NotAKey", http.StatusBadRequest},
		{"bad since", "since=not-a-date", http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := doReq(t, mux, "GET", "/api/repository/snapshots?"+c.query, "", "")
			require.Equal(t, c.status, w.Code)
		})
	}
}

func TestRepositorySnapshotsSinceValid(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/repository/snapshots?since=2999-01-01T00:00:00Z", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res Items[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.Equal(t, 0, res.Total)
}

func TestRepositoryImporterTypes(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/repository/importer-types", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res Items[struct {
		Name string `json:"name"`
	}]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.GreaterOrEqual(t, res.Total, 1)
}

func TestRepositoryLocatePathname(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/repository/locate-pathname?resource=/subdir/dummy.txt", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res Items[TimelineLocation]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.Equal(t, 1, res.Total)
	require.Len(t, res.Items, 1)
}

func TestRepositoryLocatePathnameNoMatch(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/repository/locate-pathname?resource=/nope/missing.txt", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res Items[TimelineLocation]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.Equal(t, 0, res.Total)
}

func TestRepositoryLocatePathnameBadParams(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")
	w := doReq(t, mux, "GET", "/api/repository/locate-pathname?offset=abc", "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = doReq(t, mux, "GET", "/api/repository/locate-pathname?sort=NotAKey", "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSnapshotHeaderOK(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/"+idHex, "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res Item[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.NotEmpty(t, res.Item)
}

func TestSnapshotVFSBrowse(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/vfs/"+idHex+":/", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res Item[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.NotEmpty(t, res.Item)
}

func TestSnapshotVFSBrowseFile(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/vfs/"+idHex+":/subdir/dummy.txt", "", "")
	require.Equal(t, http.StatusOK, w.Code)
}

func TestSnapshotVFSBrowseNotFound(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/vfs/"+idHex+":/does/not/exist", "", "")
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSnapshotVFSChildren(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/vfs/children/"+idHex+":/", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res Items[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.GreaterOrEqual(t, len(res.Items), 1)
}

func TestSnapshotVFSChildrenSubdir(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/vfs/children/"+idHex+":/subdir", "", "")
	require.Equal(t, http.StatusOK, w.Code)
}

func TestSnapshotVFSChildrenNotDir(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/vfs/children/"+idHex+":/top.txt", "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSnapshotVFSChildrenBadParams(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/vfs/children/"+idHex+":/?offset=abc", "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = doReq(t, mux, "GET", "/api/snapshot/vfs/children/"+idHex+":/?sort=NotAKey", "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSnapshotVFSChunks(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/vfs/chunks/"+idHex+":/subdir/dummy.txt", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res Items[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
}

func TestSnapshotVFSChunksBadParams(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")
	w := doReq(t, mux, "GET", "/api/snapshot/vfs/chunks/"+idHex+":/subdir/dummy.txt?limit=abc", "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSnapshotVFSSearch(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/vfs/search/"+idHex+":/?recursive=true", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res ItemsPage[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
}

func TestSnapshotVFSSearchBadParams(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")
	w := doReq(t, mux, "GET", "/api/snapshot/vfs/search/"+idHex+":/?offset=abc", "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = doReq(t, mux, "GET", "/api/snapshot/vfs/search/"+idHex+":/?limit=abc", "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSnapshotVFSSearchTooManyMimes(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")
	var sb strings.Builder
	for i := 0; i < 21; i++ {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(fmt.Sprintf("mime=type%d", i))
	}
	w := doReq(t, mux, "GET", "/api/snapshot/vfs/search/"+idHex+":/?"+sb.String(), "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSnapshotVFSErrors(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/vfs/errors/"+idHex+":/", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res Items[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
}

func TestSnapshotVFSErrorsNotDir(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/vfs/errors/"+idHex+":/top.txt", "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSnapshotVFSErrorsBadSort(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/vfs/errors/"+idHex+":/?sort=Bogus", "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSnapshotReaderText(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/reader/"+idHex+":/subdir/dummy.txt?render=text", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "hello dummy")
}

func TestSnapshotReaderTextStyled(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/reader/"+idHex+":/subdir/dummy.txt?render=text_styled", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "hello dummy")
	require.Contains(t, w.Body.String(), "<pre>")
}

func TestSnapshotReaderCode(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/reader/"+idHex+":/subdir/code.go?render=code", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "<!DOCTYPE html>")
}

func TestSnapshotReaderAuto(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/reader/"+idHex+":/subdir/dummy.txt", "", "")
	require.Equal(t, http.StatusOK, w.Code)
}

func TestSnapshotReaderDownload(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/reader/"+idHex+":/subdir/dummy.txt?download=true&render=text", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
}

func TestSnapshotReaderBadRender(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	w := doReq(t, mux, "GET", "/api/snapshot/reader/"+idHex+":/subdir/dummy.txt?render=bogus", "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
}
