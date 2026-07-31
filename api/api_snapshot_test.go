package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// TestSnapshot drives most snapshot endpoints against a single repository.
func TestSnapshot(t *testing.T) {
	mux, repo, snap, _ := server(t, "")
	defer snap.Close()
	id := snapid(snap)

	t.Run("path param missing id", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/x/{snapshot_path}", nil)
		req.SetPathValue("snapshot_path", "")
		_, _, err := SnapshotPathParam(req, repo, "snapshot_path")
		require.Error(t, err)
		apierr, ok := err.(*ApiError)
		require.True(t, ok)
		require.Equal(t, http.StatusBadRequest, apierr.HttpCode)
	})

	t.Run("header", func(t *testing.T) {
		w := get(t, mux, "/api/snapshot/"+id)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	})

	t.Run("header not found", func(t *testing.T) {
		// mangle the snapid so that it does not match
		snapid := snap.Header.GetIndexID()

		// cheating a bit, but the meaning here is to just mangle the
		// id to generate a non-existent one.
		if snapid[1] = snapid[1] + 1; snapid[1] > '9' {
			snapid[1] = '0'
		}

		// Valid hex, valid length, but no such snapshot -> 404.
		w := get(t, mux, "/api/snapshot/"+hex.EncodeToString(snapid[:]))
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
	})

	t.Run("header bad param", func(t *testing.T) {
		w := get(t, mux, "/api/snapshot/zz")
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	})

	t.Run("vfs search has next", func(t *testing.T) {
		// "d" holds four .txt files; recursive search with limit=1 forces the
		// "one extra item -> HasNext" branch (limit is incremented internally).
		w := get(t, mux, "/api/snapshot/vfs/search/"+id+":/subdir?recursive=true&limit=1&pattern=txt")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		var page ItemsPage[json.RawMessage]
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
		require.True(t, page.HasNext)
		require.Len(t, page.Items, 1)
	})

	t.Run("vfs search mime filter", func(t *testing.T) {
		// A single mime filter exercises the Mimes pass-through (not the >20 cap).
		w := get(t, mux, "/api/snapshot/vfs/search/"+id+":/subdir?recursive=true&mime=text/plain")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), "has_next")
	})

	t.Run("vfs search mime filter keeps partial name match", func(t *testing.T) {
		w := get(t, mux, "/api/snapshot/vfs/search/"+id+":/?recursive=true&pattern=CODE&mime=text/plain&limit=10")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), searchFixtureConductFile)
	})

	t.Run("vfs search non recursive", func(t *testing.T) {
		// non-recursive search of a directory.
		w := get(t, mux, "/api/snapshot/vfs/search/"+id+":/subdir?limit=10")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), "has_next")

		// limit<=0 is normalized to 50 (covers that branch).
		w = get(t, mux, "/api/snapshot/vfs/search/"+id+":/subdir?limit=0")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	})

	t.Run("vfs search too many mimes", func(t *testing.T) {
		// More than 20 mime params -> 400.
		url := "/api/snapshot/vfs/search/" + id + ":/subdir?"
		for i := range 21 {
			if i > 0 {
				url += "&"
			}
			url += "mime=text/plain"
		}
		w := get(t, mux, url)
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	})

	t.Run("vfs search variants", func(t *testing.T) {
		// recursive search with offset/limit and a name pattern.
		w := get(t, mux, "/api/snapshot/vfs/search/"+id+":/subdir?recursive=true&offset=0&limit=1&pattern=txt")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), "has_next")

		// bad offset -> 400.
		w = get(t, mux, "/api/snapshot/vfs/search/"+id+":/subdir?offset=abc")
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())

		// bad limit -> 400.
		w = get(t, mux, "/api/snapshot/vfs/search/"+id+":/subdir?limit=abc")
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	})

	t.Run("vfs browse root and file", func(t *testing.T) {
		// Root directory (path empty -> "/").
		w := get(t, mux, "/api/snapshot/vfs/"+id+":/")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// directory -> loadEntrySummaries path runs.
		w = get(t, mux, "/api/snapshot/vfs/"+id+":/subdir")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// A regular file entry.
		w = get(t, mux, "/api/snapshot/vfs/"+id+":/top.txt")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// Non-existent path -> error (not 200).
		w = get(t, mux, "/api/snapshot/vfs/"+id+":/does/not/exist")
		require.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("vfs browse unknown prefix", func(t *testing.T) {
		// A well-formed but unmatched snapshot prefix -> LocateSnapshotByPrefix
		// error surfaced as a 400 invalid_params.
		w := get(t, mux, "/api/snapshot/vfs/deadbeef:/subdir")
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	})

	t.Run("vfs children root", func(t *testing.T) {
		// The root directory prepends no ".." entry (fsinfo.Path() == "/").
		w := get(t, mux, "/api/snapshot/vfs/children/"+id+":/")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		var items Items[json.RawMessage]
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
		require.Greater(t, len(items.Items), 0)
	})

	t.Run("vfs children limit decrement to zero", func(t *testing.T) {
		// On a non-root directory, page 0 prepends "..", which decrements limit. With
		// limit=1 this drives limit to 0 and hits the "replace with child count"
		// branch in snapshotVFSChildren.
		w := get(t, mux, "/api/snapshot/vfs/children/"+id+":/subdir?offset=0&limit=1")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		var items Items[json.RawMessage]
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
		require.GreaterOrEqual(t, len(items.Items), 1)
	})

	t.Run("vfs children paging", func(t *testing.T) {
		// Default listing of a directory with several entries.
		w := get(t, mux, "/api/snapshot/vfs/children/"+id+":/subdir")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		var items Items[json.RawMessage]
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
		require.Greater(t, len(items.Items), 0)

		// offset>0 path: exercises the ".." offset-decrement branch.
		w = get(t, mux, "/api/snapshot/vfs/children/"+id+":/subdir?offset=1&limit=2")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// Explicit sort key.
		w = get(t, mux, "/api/snapshot/vfs/children/"+id+":/subdir?sort=Name")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	})

	t.Run("vfs children errors", func(t *testing.T) {
		// children of a regular file -> 400 "not a directory".
		w := get(t, mux, "/api/snapshot/vfs/children/"+id+":/top.txt")
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())

		// invalid sort key -> 400.
		w = get(t, mux, "/api/snapshot/vfs/children/"+id+":/subdir?sort=Bogus")
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())

		// invalid offset -> error.
		w = get(t, mux, "/api/snapshot/vfs/children/"+id+":/subdir?offset=abc")
		require.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("vfs errors paging", func(t *testing.T) {
		// Exercise the offset/limit window arithmetic on a clean (no-error) dir.
		w := get(t, mux, "/api/snapshot/vfs/errors/"+id+":/subdir?offset=0&limit=1")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// bad offset -> error from QueryParamToInt64.
		w = get(t, mux, "/api/snapshot/vfs/errors/"+id+":/subdir?offset=-1")
		require.NotEqual(t, http.StatusOK, w.Code)

		// errors on a missing dir -> 404.
		w = get(t, mux, "/api/snapshot/vfs/errors/"+id+":/no-such-dir")
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
	})

	t.Run("vfs errors handler", func(t *testing.T) {
		// happy path on a dir, with a paging window.
		w := get(t, mux, "/api/snapshot/vfs/errors/"+id+":/subdir?offset=0&limit=10")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), "total")

		// invalid sort key -> 400.
		w = get(t, mux, "/api/snapshot/vfs/errors/"+id+":/?sort=Bogus")
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())

		// -Name sort key is accepted.
		w = get(t, mux, "/api/snapshot/vfs/errors/"+id+":/?sort=-Name")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// errors of a regular file -> 400 not a directory.
		w = get(t, mux, "/api/snapshot/vfs/errors/"+id+":/top.txt")
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	})

	t.Run("vfs chunks offset", func(t *testing.T) {
		// offset beyond the chunk count -> empty Items but valid Total.
		w := get(t, mux, "/api/snapshot/vfs/chunks/"+id+":/subdir/dummy.txt?offset=1000&limit=10")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), "total")

		// chunks on a missing path returns 200 with empty body (early nil return).
		w = get(t, mux, "/api/snapshot/vfs/chunks/"+id+":/dir/missing.txt")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	})

	t.Run("vfs chunks paging", func(t *testing.T) {
		w := get(t, mux, "/api/snapshot/vfs/chunks/"+id+":/subdir/dummy.txt?offset=0&limit=10")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), "total")

		// bad limit -> error.
		w = get(t, mux, "/api/snapshot/vfs/chunks/"+id+":/subdir/dummy.txt?limit=abc")
		require.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("reader render variants", func(t *testing.T) {
		base := "/api/snapshot/reader/" + id + ":"

		for _, render := range []string{"", "auto", "text", "text_styled", "code"} {
			u := base + "/subdir/dummy.txt"
			if render != "" {
				u += "?render=" + render
			}
			w := get(t, mux, u)
			require.Equal(t, http.StatusOK, w.Code, "render=%s body=%s", render, w.Body.String())

			switch render {
			case "", "auto":
				require.Contains(t, w.Header().Get("Content-type"), "text/plain")
				require.Contains(t, w.Body.String(), "hello dummy")
			case "text":
				require.Contains(t, w.Header().Get("Content-type"), "text/plain")
				require.Contains(t, w.Body.String(), "hello dummy")
			case "text_styled":
				require.Contains(t, w.Header().Get("Content-type"), "text/html")
				require.Contains(t, w.Body.String(), "<pre>")
			case "code":
				require.Contains(t, w.Header().Get("Content-type"), "text/html")
				require.Contains(t, w.Body.String(), "<!DOCTYPE html>")
			}

			// "noext" has no file extension
			w = get(t, mux, strings.Replace(u, "dummy.txt", "noext", 1))
			require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		}

		base += "/subdir/dummy.txt"

		// download=true sets a Content-Disposition attachment header naming the file.
		w := get(t, mux, base+"?download=true")
		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
		require.Contains(t, w.Header().Get("Content-Disposition"), "dummy.txt")

		// an invalid render value is rejected.
		w = get(t, mux, base+"?render=bogus")
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("reader missing file not found", func(t *testing.T) {
		// GetEntry on a missing path bubbles up an fs.ErrNotExist which handleError
		// maps to a 404.
		w := get(t, mux, "/api/snapshot/reader/"+id+":/missing.txt")
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
	})

	t.Run("reader sign url", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/snapshot/reader-sign-url/"+id+":/subdir/dummy.txt", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		var resp map[string]map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.NotEmpty(t, resp["item"]["signature"])
	})

	// postDownload registers a download bundle and returns its (single-use) id.
	postDownload := func(t *testing.T) string {
		t.Helper()
		body := `{"name":"dl","items":[{"pathname":"/subdir/dummy.txt"}]}`
		req, _ := http.NewRequest("POST", "/api/snapshot/vfs/downloader/"+id+":/subdir", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		var resp struct {
			Id string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.NotEmpty(t, resp.Id)
		return resp.Id
	}

	t.Run("downloader signed default name", func(t *testing.T) {
		// No name query param -> handler synthesizes "snapshot-<id>-<ts>" + ext.
		w := get(t, mux, "/api/snapshot/vfs/downloader-sign-url/"+postDownload(t)+"?format=zip")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Header().Get("Content-Disposition"), "snapshot-")
		require.Contains(t, w.Header().Get("Content-Disposition"), ".zip")
	})

	t.Run("downloader signed custom name", func(t *testing.T) {
		// name already carries an extension -> the ext-appending branch is skipped.
		w := get(t, mux, "/api/snapshot/vfs/downloader-sign-url/"+postDownload(t)+"?format=zip&name=custom.zip")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Header().Get("Content-Disposition"), "custom.zip")
	})

	t.Run("downloader signed not found", func(t *testing.T) {
		// Unknown download id -> 404.
		w := get(t, mux, "/api/snapshot/vfs/downloader-sign-url/does-not-exist?format=zip")
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
	})

	t.Run("downloader formats", func(t *testing.T) {
		// tar format.
		w := get(t, mux, "/api/snapshot/vfs/downloader-sign-url/"+postDownload(t)+"?format=tar")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Header().Get("Content-Type"), "tar")

		// tarball format.
		w = get(t, mux, "/api/snapshot/vfs/downloader-sign-url/"+postDownload(t)+"?format=tarball")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// unknown format -> 400.
		w = get(t, mux, "/api/snapshot/vfs/downloader-sign-url/"+postDownload(t)+"?format=bogus")
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	})

	t.Run("downloader flow", func(t *testing.T) {
		// Without a format the signed endpoint rejects the request.
		w := get(t, mux, "/api/snapshot/vfs/downloader-sign-url/"+postDownload(t))
		require.Equal(t, http.StatusBadRequest, w.Code)

		// With a valid archive format it serves the bundle. (Re-POST first, as the
		// id is single-use once consumed.)
		w = get(t, mux, "/api/snapshot/vfs/downloader-sign-url/"+postDownload(t)+"?format=zip")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	})

	t.Run("downloader bad body", func(t *testing.T) {
		// malformed JSON body -> 400.
		req, _ := http.NewRequest("POST", "/api/snapshot/vfs/downloader/"+id+":/subdir", bytes.NewBufferString("{not-json"))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	})

	t.Run("downloader bad snapshot id", func(t *testing.T) {
		// Empty snapshot id segment -> SnapshotPathParam returns a 400.
		body := `{"name":"dl","items":[{"pathname":"/subdir/dummy.txt"}]}`
		req, _ := http.NewRequest("POST", "/api/snapshot/vfs/downloader/:/subdir", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	})

	t.Run("handle error passes ApiError", func(t *testing.T) {
		// An existing *ApiError is forwarded verbatim (its HttpCode and ErrCode
		// preserved) rather than remapped.
		req, _ := http.NewRequest("GET", "/x", nil)
		w := httptest.NewRecorder()
		handleError(w, req, &ApiError{HttpCode: http.StatusTeapot, ErrCode: "teapot", Message: "short and stout"})
		require.Equal(t, http.StatusTeapot, w.Code)

		var body ApiErrorRes
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.Equal(t, "teapot", body.Error.ErrCode)
	})
}

// The signed-reader cases need routes wired with a signing token, which is
// a property of SetupRoutes and cannot be changed after the fact; hence a
// second fixture for this group only.
func TestSignedReader(t *testing.T) {
	const token = "verify-token"
	mux, _, snap, _ := server(t, token)
	defer snap.Close()
	id := snapid(snap)

	t.Run("valid signature", func(t *testing.T) {
		sig := signReader(t, mux, token, id+":/subdir/dummy.txt")

		// A valid signature lets the (otherwise token-protected) reader through.
		w := get(t, mux, "/api/snapshot/reader/"+id+":/subdir/dummy.txt?signature="+sig)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), "hello dummy")
	})

	t.Run("tampered path", func(t *testing.T) {
		sig := signReader(t, mux, token, id+":/subdir/dummy.txt")

		// Same valid signature but requesting a different path -> rejected.
		w := get(t, mux, "/api/snapshot/reader/"+id+":/top.txt?signature="+sig)
		require.Equal(t, http.StatusUnauthorized, w.Code, "body=%s", w.Body.String())
	})

	t.Run("bad signature", func(t *testing.T) {
		// Garbage signature -> JWT parse failure -> 401.
		w := get(t, mux, "/api/snapshot/reader/"+id+":/subdir/dummy.txt?signature=not-a-jwt")
		require.Equal(t, http.StatusUnauthorized, w.Code, "body=%s", w.Body.String())
	})

	t.Run("expired", func(t *testing.T) {
		// Hand-craft an already-expired token signed with the right key.
		now := time.Now().Add(-3 * time.Hour)
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, SnapshotSignedURLClaims{
			SnapshotID: id,
			Path:       "/subdir/dummy.txt",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(now),
				Issuer:    "plakar-api",
			},
		})
		sig, err := jwtToken.SignedString([]byte(token))
		require.NoError(t, err)

		w := get(t, mux, "/api/snapshot/reader/"+id+":/subdir/dummy.txt?signature="+sig)
		require.Equal(t, http.StatusUnauthorized, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), "expired")
	})

	t.Run("wrong signing method", func(t *testing.T) {
		// A token with the "none" alg should be rejected by the method check.
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodNone, SnapshotSignedURLClaims{
			SnapshotID: id,
			Path:       "/subdir/dummy.txt",
		})
		sig, err := jwtToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		w := get(t, mux, "/api/snapshot/reader/"+id+":/subdir/dummy.txt?signature="+sig)
		require.Equal(t, http.StatusUnauthorized, w.Code, "body=%s", w.Body.String())
	})

	t.Run("no signature requires token", func(t *testing.T) {
		// No signature and no Authorization header -> token middleware rejects.
		w := get(t, mux, "/api/snapshot/reader/"+id+":/subdir/dummy.txt")
		require.Equal(t, http.StatusUnauthorized, w.Code, "body=%s", w.Body.String())

		// With the correct Bearer token it succeeds.
		req, _ := http.NewRequest("GET", "/api/snapshot/reader/"+id+":/subdir/dummy.txt", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	})

	t.Run("sign nonexistent path", func(t *testing.T) {
		// Signing a path that does not exist in the snapshot returns an error.
		req, _ := http.NewRequest("POST", "/api/snapshot/reader-sign-url/"+id+":/no/such/file", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	})

	t.Run("sign bad snapshot id", func(t *testing.T) {
		// Empty snapshot id segment -> SnapshotPathParam missing-arg error.
		req, _ := http.NewRequest("POST", "/api/snapshot/reader-sign-url/:/subdir/dummy.txt", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	})
}
