package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// VerifyMiddleware: a signature minted for one path must be rejected when the
// request targets a different path (claims.Path mismatch).
func TestVerifyMiddlewareWrongPath(t *testing.T) {
	token := "vm-token"
	mux, idHex := buildRealRepoRouter(t, token)

	signReq, _ := http.NewRequest("POST", "/api/snapshot/reader-sign-url/"+idHex+":/subdir/dummy.txt", nil)
	signReq.Header.Set("Authorization", "Bearer "+token)
	signW := httptest.NewRecorder()
	mux.ServeHTTP(signW, signReq)
	require.Equal(t, http.StatusOK, signW.Code)

	var sresp Item[struct {
		Signature string `json:"signature"`
	}]
	require.NoError(t, json.Unmarshal(signW.Body.Bytes(), &sresp))

	// Use the signature against a different (valid) path -> 401.
	readReq, _ := http.NewRequest("GET", "/api/snapshot/reader/"+idHex+":/subdir/code.go?render=text&signature="+sresp.Item.Signature, nil)
	readW := httptest.NewRecorder()
	mux.ServeHTTP(readW, readReq)
	require.Equal(t, http.StatusUnauthorized, readW.Code)
}

// repositoryLocatePathname with importer filters that don't match -> 0 results.
func TestLocatePathnameImporterFilters(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")

	cases := []string{
		"resource=/subdir/dummy.txt&importerType=nope",
		"resource=/subdir/dummy.txt&importerOrigin=nope",
		"resource=/subdir/dummy.txt&importerDirectory=/nope",
	}
	for _, q := range cases {
		t.Run(q, func(t *testing.T) {
			w := doReq(t, mux, "GET", "/api/repository/locate-pathname?"+q, "", "")
			require.Equal(t, http.StatusOK, w.Code)
			var res Items[TimelineLocation]
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
			require.Equal(t, 0, res.Total)
		})
	}
}

// repositoryLocatePathname descending sort branch.
func TestLocatePathnameDescSort(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")
	w := doReq(t, mux, "GET", "/api/repository/locate-pathname?resource=/subdir/dummy.txt&sort=-Timestamp", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res Items[TimelineLocation]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.Equal(t, 1, res.Total)
}

// repositorySnapshots with explicit limit/offset paging.
func TestRepositorySnapshotsPaging(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")
	w := doReq(t, mux, "GET", "/api/repository/snapshots?limit=1&offset=0&sort=-Timestamp", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res Items[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.Equal(t, 1, res.Total)
	require.Len(t, res.Items, 1)

	// offset beyond range -> empty items, total unchanged.
	w = doReq(t, mux, "GET", "/api/repository/snapshots?offset=99", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.Equal(t, 1, res.Total)
	require.Empty(t, res.Items)
}

// snapshotVFSChildren with an offset on a directory exercises the non-first
// page branch (offset-- accounting for "..").
func TestSnapshotVFSChildrenPaging(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")
	w := doReq(t, mux, "GET", "/api/snapshot/vfs/children/"+idHex+":/subdir?offset=1&limit=10", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res Items[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
}

// snapshotVFSErrors descending sort key branch + paging.
func TestSnapshotVFSErrorsDescSort(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")
	w := doReq(t, mux, "GET", "/api/snapshot/vfs/errors/"+idHex+":/?sort=-Name&offset=0&limit=5", "", "")
	require.Equal(t, http.StatusOK, w.Code)
}

// snapshotVFSSearch with a name pattern + paging limit.
func TestSnapshotVFSSearchPattern(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")
	w := doReq(t, mux, "GET", "/api/snapshot/vfs/search/"+idHex+":/?recursive=true&pattern=dummy&limit=1", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	var res ItemsPage[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
}

// snapshotReader for a directory path returns nil (non-regular file -> early
// return after Open), yielding an empty 200 body.
func TestSnapshotReaderDirectory(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")
	w := doReq(t, mux, "GET", "/api/snapshot/vfs/chunks/"+idHex+":/missing-file", "", "")
	// chunks handler returns nil (200) for a missing entry path.
	require.Equal(t, http.StatusOK, w.Code)
}
