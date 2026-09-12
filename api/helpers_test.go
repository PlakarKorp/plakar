package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// server builds a real repository with a richer snapshot than newAPIServer
// (nested dirs, multiple files) and wires SetupRoutes with norefresh=true.
// Distinct name to avoid clashing with the team's newAPIServer helper.
func server(t *testing.T, token string) (*http.ServeMux, *repository.Repository, *snapshot.Snapshot, *appcontext.AppContext) {
	t.Helper()
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/noext", 0644, "hello noext"),
		ptesting.NewMockDir("subdir/nested"),
		ptesting.NewMockFile("subdir/nested/deep.txt", 0644, "deep content"),
		ptesting.NewMockFile("top.txt", 0644, "top level"),
		ptesting.NewMockFile("CODE_OF_CONDUCT", 0644, "project conduct"),
	})
	mux := http.NewServeMux()
	SetupRoutes(mux, repo, ctx, token, true /* norefresh */)
	return mux, repo, snap, ctx
}

func get(t *testing.T, mux *http.ServeMux, url string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func snapid(snap *snapshot.Snapshot) string {
	id := snap.Header.GetIndexID()
	return hex.EncodeToString(id[:])
}

// signReader posts to the sign endpoint (auth via Bearer token) and returns the
// JWT signature for a snapshot path.
func signReader(t *testing.T, mux *http.ServeMux, token, snapPath string) string {
	t.Helper()
	req, _ := http.NewRequest("POST", "/api/snapshot/reader-sign-url/"+snapPath, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	var resp struct {
		Item struct {
			Signature string `json:"signature"`
		} `json:"item"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Item.Signature)
	return resp.Item.Signature
}

// attachPkgManager installs a real (but empty, hermetic) pkg.Manager onto ctx so
// the integration/uninstall handlers have a backend that resolves locally
// against an empty temp directory without touching the network.
func attachPkgManager(t *testing.T, ctx *appcontext.AppContext) {
	t.Helper()
	dir := t.TempDir()
	backend, err := pkg.NewFlatBackend(ctx.GetInner(),
		filepath.Join(dir, "plugins"), filepath.Join(dir, "cache"), &pkg.FlatBackendOptions{})
	require.NoError(t, err)
	mgr, err := pkg.New(backend, &pkg.Options{})
	require.NoError(t, err)
	ctx.SetPkgManager(mgr)
}
