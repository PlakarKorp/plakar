package httpd

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/objects"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// newTestServer builds an httpd server backed by a real (mock-backed) repo store.
func newTestServer(t *testing.T, noDelete bool) (*server, http.Handler) {
	t.Helper()
	var bufOut, bufErr bytes.Buffer
	repo, _ := ptesting.GenerateRepository(t, &bufOut, &bufErr, nil)

	s := &server{
		store:    repo.Store(),
		noDelete: noDelete,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.openRepository)
	mux.HandleFunc("GET /resources/{resource}", s.listResource)
	mux.HandleFunc("GET /resources/{resource}/{mac}", s.getResource)
	mux.HandleFunc("PUT /resources/{resource}/{mac}", s.putResource)
	mux.HandleFunc("DELETE /resources/{resource}/{mac}", s.deleteResource)
	return s, mux
}

func macHex() string {
	var m objects.MAC
	for i := range m {
		m[i] = byte(i)
	}
	return hex.EncodeToString(m[:])
}

func TestOpenRepository(t *testing.T) {
	_, h := newTestServer(t, false)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"))
	require.NotEmpty(t, rr.Body.Bytes())
}

func TestListResource(t *testing.T) {
	_, h := newTestServer(t, false)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/resources/packfiles", nil))
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	// must be valid JSON (an array, possibly empty/null)
	var macs []objects.MAC
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &macs))
}

func TestListResourceInvalidType(t *testing.T) {
	_, h := newTestServer(t, false)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/resources/bogus", nil))
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), ErrInvalidResourceType.Error())
}

func TestPutGetDeleteResourceRoundTrip(t *testing.T) {
	s, h := newTestServer(t, false)
	payload := []byte("hello packfile payload")
	mac := macHex()

	// PUT
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("PUT", "/resources/packfiles/"+mac, bytes.NewReader(payload)))
	require.Equal(t, http.StatusOK, rr.Code)

	// GET
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/resources/packfiles/"+mac, nil))
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"))
	require.Equal(t, payload, rr.Body.Bytes())

	// GET with a Range header
	rr = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/resources/packfiles/"+mac, nil)
	req.Header.Set("Range", "bytes=0-5")
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, payload[:5], rr.Body.Bytes())

	// DELETE
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("DELETE", "/resources/packfiles/"+mac, nil))
	require.Equal(t, http.StatusOK, rr.Code)

	_ = s
}

func TestGetResourceInvalidType(t *testing.T) {
	_, h := newTestServer(t, false)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/resources/bogus/"+macHex(), nil))
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetResourceInvalidMac(t *testing.T) {
	_, h := newTestServer(t, false)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/resources/packfiles/zzzz", nil))
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), ErrInvalidMAC.Error())
}

func TestGetResourceInvalidRange(t *testing.T) {
	_, h := newTestServer(t, false)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/resources/packfiles/"+macHex(), nil)
	req.Header.Set("Range", "items=0-5")
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), ErrInvalidRange.Error())
}

func TestGetResourceMissing(t *testing.T) {
	_, h := newTestServer(t, false)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/resources/packfiles/"+macHex(), nil))
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestPutResourceInvalidType(t *testing.T) {
	_, h := newTestServer(t, false)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("PUT", "/resources/bogus/"+macHex(), strings.NewReader("x")))
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestPutResourceInvalidMac(t *testing.T) {
	_, h := newTestServer(t, false)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("PUT", "/resources/packfiles/nothex", strings.NewReader("x")))
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeleteResourceForbidden(t *testing.T) {
	_, h := newTestServer(t, true) // noDelete
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("DELETE", "/resources/packfiles/"+macHex(), nil))
	require.Equal(t, http.StatusForbidden, rr.Code)
	require.Contains(t, rr.Body.String(), "not allowed to delete")
}

func TestDeleteResourceInvalidType(t *testing.T) {
	_, h := newTestServer(t, false)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("DELETE", "/resources/bogus/"+macHex(), nil))
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeleteResourceInvalidMac(t *testing.T) {
	_, h := newTestServer(t, false)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("DELETE", "/resources/packfiles/nothex", nil))
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetResourcePure(t *testing.T) {
	cases := map[string]struct {
		resource string
		want     storage.StorageResource
		wantErr  bool
	}{
		"packfiles":   {"packfiles", storage.StorageResourcePackfile, false},
		"states":      {"states", storage.StorageResourceState, false},
		"locks":       {"locks", storage.StorageResourceLock, false},
		"eccpackfiles": {"eccpackfiles", storage.StorageResourceECCPackfile, false},
		"eccstates":   {"eccstates", storage.StorageResourceECCState, false},
		"invalid":     {"nope", 0, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/resources/"+tc.resource, nil)
			req.SetPathValue("resource", tc.resource)
			got, err := getResource(req)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestGetMacPure(t *testing.T) {
	// valid
	req := httptest.NewRequest("GET", "/", nil)
	req.SetPathValue("mac", macHex())
	mac, err := getMac(req)
	require.NoError(t, err)
	require.Equal(t, byte(0), mac[0])
	require.Equal(t, byte(31), mac[31])

	// not hex
	req.SetPathValue("mac", "zz")
	_, err = getMac(req)
	require.ErrorIs(t, err, ErrInvalidMAC)

	// wrong length (valid hex but only 1 byte)
	req.SetPathValue("mac", "ab")
	_, err = getMac(req)
	require.ErrorIs(t, err, ErrInvalidMAC)
}

func TestGetRangePure(t *testing.T) {
	mk := func(v string) *http.Request {
		r := httptest.NewRequest("GET", "/", nil)
		if v != "" {
			r.Header.Set("Range", v)
		}
		return r
	}

	// no header -> nil, nil
	rg, err := getRange(mk(""))
	require.NoError(t, err)
	require.Nil(t, rg)

	// valid
	rg, err = getRange(mk("bytes=10-30"))
	require.NoError(t, err)
	require.NotNil(t, rg)
	require.Equal(t, uint64(10), rg.Offset)
	require.Equal(t, uint32(20), rg.Length)

	// missing bytes= prefix
	_, err = getRange(mk("items=0-5"))
	require.ErrorIs(t, err, ErrInvalidRange)

	// missing dash
	_, err = getRange(mk("bytes=10"))
	require.ErrorIs(t, err, ErrInvalidRange)

	// bad start
	_, err = getRange(mk("bytes=x-5"))
	require.ErrorIs(t, err, ErrInvalidRange)

	// bad end
	_, err = getRange(mk("bytes=5-y"))
	require.ErrorIs(t, err, ErrInvalidRange)

	// end <= offset
	_, err = getRange(mk("bytes=10-10"))
	require.ErrorIs(t, err, ErrInvalidRange)

	// length overflow uint32
	_, err = getRange(mk("bytes=0-4294967296"))
	require.ErrorIs(t, err, ErrInvalidRange)
}

// ensure io import is used even if a path changes
var _ = io.Discard
