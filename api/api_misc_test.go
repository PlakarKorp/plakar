package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// --- api_errors.go ---

func TestApiErrorError(t *testing.T) {
	e := &ApiError{HttpCode: 400, ErrCode: "bad", Message: "boom"}
	require.Equal(t, "bad: boom", e.Error())
}

func TestParameterError(t *testing.T) {
	e := parameterError("field", InvalidArgument, errors.New("nope"))
	require.Equal(t, http.StatusBadRequest, e.HttpCode)
	require.Equal(t, "invalid_params", e.ErrCode)
	require.Contains(t, e.Params, "field")
	require.Equal(t, InvalidArgument, e.Params["field"].Code)
	require.Equal(t, "nope", e.Params["field"].Message)
}

func TestAuthErrorHelper(t *testing.T) {
	e := authError("denied")
	require.Equal(t, http.StatusUnauthorized, e.HttpCode)
	require.Equal(t, "bad_auth", e.ErrCode)
	require.Equal(t, "denied", e.Message)
}

// --- api.go: handleError mapping ---

func TestHandleError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"not readable", repository.ErrNotReadable, 400},
		{"blob not found", repository.ErrBlobNotFound, 404},
		{"packfile not found", repository.ErrPackfileNotFound, 404},
		{"fs not exist", fs.ErrNotExist, 404},
		{"snapshot not found", snapshot.ErrNotFound, 404},
		{"plain error -> 500", errors.New("whatever"), 500},
		{"api error passthrough", &ApiError{HttpCode: 418, ErrCode: "teapot", Message: "no"}, 418},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/x", nil)
			w := httptest.NewRecorder()
			handleError(w, req, c.err)
			require.Equal(t, c.want, w.Code)
			var body ApiErrorRes
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			require.NotNil(t, body.Error)
		})
	}
}

func TestJSONAPIViewServeHTTP(t *testing.T) {
	// success path: no error -> Content-Type set, body written by handler.
	view := JSONAPIView(func(w http.ResponseWriter, r *http.Request) error {
		_, err := w.Write([]byte(`{"ok":true}`))
		return err
	})
	req, _ := http.NewRequest("GET", "/api/x", nil)
	w := httptest.NewRecorder()
	view.ServeHTTP(w, req)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
	require.JSONEq(t, `{"ok":true}`, w.Body.String())

	// error path
	view2 := JSONAPIView(func(w http.ResponseWriter, r *http.Request) error {
		return &ApiError{HttpCode: 404, ErrCode: "not-found", Message: "x"}
	})
	w2 := httptest.NewRecorder()
	view2.ServeHTTP(w2, req)
	require.Equal(t, 404, w2.Code)
}

func TestAPIViewServeHTTP(t *testing.T) {
	// success path leaves headers to the handler.
	called := false
	view := APIView(func(w http.ResponseWriter, r *http.Request) error {
		called = true
		return nil
	})
	req, _ := http.NewRequest("GET", "/api/x", nil)
	w := httptest.NewRecorder()
	view.ServeHTTP(w, req)
	require.True(t, called)

	// error path sets Content-Type and serializes the error.
	view2 := APIView(func(w http.ResponseWriter, r *http.Request) error {
		return &ApiError{HttpCode: 400, ErrCode: "bad-request", Message: "x"}
	})
	w2 := httptest.NewRecorder()
	view2.ServeHTTP(w2, req)
	require.Equal(t, 400, w2.Code)
	require.Equal(t, "application/json", w2.Header().Get("Content-Type"))
}

func TestTokenAuthMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// empty token -> no-op, passes through
	mwNoop := TokenAuthMiddleware("")(next)
	req, _ := http.NewRequest("GET", "/api/x", nil)
	w := httptest.NewRecorder()
	mwNoop.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	mw := TokenAuthMiddleware("secret")(next)

	// missing header -> 401
	req2, _ := http.NewRequest("GET", "/api/x", nil)
	w2 := httptest.NewRecorder()
	mw.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusUnauthorized, w2.Code)

	// wrong token -> 401
	req3, _ := http.NewRequest("GET", "/api/x", nil)
	req3.Header.Set("Authorization", "Bearer wrong")
	w3 := httptest.NewRecorder()
	mw.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusUnauthorized, w3.Code)

	// correct token -> passes
	req4, _ := http.NewRequest("GET", "/api/x", nil)
	req4.Header.Set("Authorization", "Bearer secret")
	w4 := httptest.NewRecorder()
	mw.ServeHTTP(w4, req4)
	require.Equal(t, http.StatusOK, w4.Code)
}

// --- api_params.go ---

func TestQueryParamToString(t *testing.T) {
	req, _ := http.NewRequest("GET", "/?a=hello&b=", nil)
	v, ok, err := QueryParamToString(req, "a")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "hello", v)

	v, ok, err = QueryParamToString(req, "b")
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, "", v)

	v, ok, err = QueryParamToString(req, "missing")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestSnapshotPathParam(t *testing.T) {
	var bufOut, bufErr bytes.Buffer
	repo, _ := ptesting.GenerateRepository(t, &bufOut, &bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	t.Cleanup(func() { snap.Close() })
	id := snap.Header.GetIndexID()
	idHex := hex.EncodeToString(id[:])

	// empty id -> MissingArgument
	req, _ := http.NewRequest("GET", "/path", nil)
	req.SetPathValue("snapshot_path", "")
	_, _, err := SnapshotPathParam(req, repo, "snapshot_path")
	require.Error(t, err)

	// invalid id -> InvalidArgument (locate fails)
	req2, _ := http.NewRequest("GET", "/path", nil)
	req2.SetPathValue("snapshot_path", "zzzz:/foo")
	_, _, err = SnapshotPathParam(req2, repo, "snapshot_path")
	require.Error(t, err)

	// valid id prefix -> resolves on the real repo, returns path
	req3, _ := http.NewRequest("GET", "/path", nil)
	req3.SetPathValue("snapshot_path", idHex+":/subdir")
	mac, path, err := SnapshotPathParam(req3, repo, "snapshot_path")
	require.NoError(t, err)
	require.Equal(t, "/subdir", path)
	require.Equal(t, id[:], mac[:])
}

// --- api_snapshot.go: URL signer + downloader ---

func TestSnapshotReaderSignAndVerify(t *testing.T) {
	token := "sign-token"
	mux, idHex := buildRealRepoRouter(t, token)

	// Sign a URL for a real path.
	signReq, _ := http.NewRequest("POST", "/api/snapshot/reader-sign-url/"+idHex+":/subdir/dummy.txt", nil)
	signReq.Header.Set("Authorization", "Bearer "+token)
	signW := httptest.NewRecorder()
	mux.ServeHTTP(signW, signReq)
	require.Equal(t, http.StatusOK, signW.Code)

	var sresp Item[struct {
		Signature string `json:"signature"`
	}]
	require.NoError(t, json.Unmarshal(signW.Body.Bytes(), &sresp))
	require.NotEmpty(t, sresp.Item.Signature)

	// Use the signature to read content without the Authorization header.
	readReq, _ := http.NewRequest("GET", "/api/snapshot/reader/"+idHex+":/subdir/dummy.txt?render=text&signature="+sresp.Item.Signature, nil)
	readW := httptest.NewRecorder()
	mux.ServeHTTP(readW, readReq)
	require.Equal(t, http.StatusOK, readW.Code)
	require.Contains(t, readW.Body.String(), "hello dummy")
}

func TestSnapshotReaderInvalidSignature(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "sign-token")
	readReq, _ := http.NewRequest("GET", "/api/snapshot/reader/"+idHex+":/subdir/dummy.txt?signature=garbage", nil)
	readW := httptest.NewRecorder()
	mux.ServeHTTP(readW, readReq)
	require.Equal(t, http.StatusUnauthorized, readW.Code)
}

func TestSnapshotReaderNoSignatureRequiresAuth(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "sign-token")
	// no signature, no auth header -> falls back to token middleware -> 401
	readReq, _ := http.NewRequest("GET", "/api/snapshot/reader/"+idHex+":/subdir/dummy.txt", nil)
	readW := httptest.NewRecorder()
	mux.ServeHTTP(readW, readReq)
	require.Equal(t, http.StatusUnauthorized, readW.Code)
}

func TestSnapshotVFSDownloaderFlow(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	body := `{"name":"dl","items":[{"pathname":"/subdir/dummy.txt"}]}`
	dlReq, _ := http.NewRequest("POST", "/api/snapshot/vfs/downloader/"+idHex+":/", strings.NewReader(body))
	dlW := httptest.NewRecorder()
	mux.ServeHTTP(dlW, dlReq)
	require.Equal(t, http.StatusOK, dlW.Code)

	var idResp struct {
		Id string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(dlW.Body.Bytes(), &idResp))
	require.NotEmpty(t, idResp.Id)

	// Now fetch the signed download as a tar archive.
	getReq, _ := http.NewRequest("GET", "/api/snapshot/vfs/downloader-sign-url/"+idResp.Id+"?format=tar", nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getReq)
	require.Equal(t, http.StatusOK, getW.Code)
	require.Contains(t, getW.Header().Get("Content-Type"), "tar")
	require.Contains(t, getW.Header().Get("Content-Disposition"), "attachment")
}

func TestSnapshotVFSDownloaderBadBody(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")
	dlReq, _ := http.NewRequest("POST", "/api/snapshot/vfs/downloader/"+idHex+":/", strings.NewReader("not json"))
	dlW := httptest.NewRecorder()
	mux.ServeHTTP(dlW, dlReq)
	require.Equal(t, http.StatusBadRequest, dlW.Code)
}

func TestSnapshotVFSDownloaderSignedNotFound(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")
	getReq, _ := http.NewRequest("GET", "/api/snapshot/vfs/downloader-sign-url/nonexistent-id?format=tar", nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getReq)
	require.Equal(t, http.StatusNotFound, getW.Code)
}

func TestSnapshotVFSDownloaderSignedBadFormat(t *testing.T) {
	mux, idHex := buildRealRepoRouter(t, "")

	body := `{"items":[{"pathname":"/subdir/dummy.txt"}]}`
	dlReq, _ := http.NewRequest("POST", "/api/snapshot/vfs/downloader/"+idHex+":/", strings.NewReader(body))
	dlW := httptest.NewRecorder()
	mux.ServeHTTP(dlW, dlReq)
	require.Equal(t, http.StatusOK, dlW.Code)
	var idResp struct {
		Id string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(dlW.Body.Bytes(), &idResp))

	getReq, _ := http.NewRequest("GET", "/api/snapshot/vfs/downloader-sign-url/"+idResp.Id+"?format=bogus", nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getReq)
	require.Equal(t, http.StatusBadRequest, getW.Code)
}

// Bad snapshot id format for path-param handlers -> 400.
func TestSnapshotHandlersBadID(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")
	cases := []string{
		"/api/snapshot/notahexid",
		"/api/snapshot/vfs/notahexid:/",
		"/api/snapshot/vfs/children/notahexid:/",
		"/api/snapshot/vfs/chunks/notahexid:/x",
		"/api/snapshot/vfs/search/notahexid:/",
		"/api/snapshot/vfs/errors/notahexid:/",
	}
	for _, target := range cases {
		t.Run(target, func(t *testing.T) {
			w := doReq(t, mux, "GET", target, "", "")
			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// Valid-but-unknown snapshot id -> 404 (not found) for the header handler.
func TestSnapshotHeaderNotFound(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")
	missing := hex.EncodeToString(make([]byte, 32))
	// flip a byte so it's a syntactically valid but unknown id
	missing = "7e0e6e24a6e29faf11d022dca77826fe8b8a000aff5ea27e16650d03acefc93c"
	w := doReq(t, mux, "GET", "/api/snapshot/"+missing, "", "")
	require.Equal(t, http.StatusNotFound, w.Code)
}
