package httpd

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/location"
	"github.com/PlakarKorp/kloset/objects"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// fakeStore is a minimal storage.Store that records the calls the httpd
// handlers make and returns canned responses. We only implement the methods
// the handlers actually call — Open, List, Put, Get, Delete — and panic on
// the rest so an accidentally widened handler surface is loud.
type fakeStore struct {
	openConfig []byte
	openErr    error

	listResources map[storage.StorageResource][]objects.MAC
	listErr       error

	getData map[storage.StorageResource]map[objects.MAC][]byte
	getErr  error

	deleteErr error
	putErr    error

	// recorded
	lastGetType storage.StorageResource
	lastGetMAC  objects.MAC
	lastGetRng  *storage.Range
	lastPutData []byte
	lastPutType storage.StorageResource
	lastPutMAC  objects.MAC
	lastDelType storage.StorageResource
	lastDelMAC  objects.MAC
}

func (f *fakeStore) Open(ctx context.Context) ([]byte, error) {
	return f.openConfig, f.openErr
}

func (f *fakeStore) List(ctx context.Context, typ storage.StorageResource) ([]objects.MAC, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listResources[typ], nil
}

func (f *fakeStore) Get(ctx context.Context, typ storage.StorageResource, mac objects.MAC, rg *storage.Range) (io.ReadCloser, error) {
	f.lastGetType, f.lastGetMAC, f.lastGetRng = typ, mac, rg
	if f.getErr != nil {
		return nil, f.getErr
	}
	data := f.getData[typ][mac]
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (f *fakeStore) Put(ctx context.Context, typ storage.StorageResource, mac objects.MAC, rd io.Reader) (int64, error) {
	f.lastPutType, f.lastPutMAC = typ, mac
	data, _ := io.ReadAll(rd)
	f.lastPutData = data
	return int64(len(data)), f.putErr
}

func (f *fakeStore) Delete(ctx context.Context, typ storage.StorageResource, mac objects.MAC) error {
	f.lastDelType, f.lastDelMAC = typ, mac
	return f.deleteErr
}

// Unused methods — panic so an accidentally widened surface is loud.
func (f *fakeStore) Create(context.Context, []byte) error { panic("unused") }
func (f *fakeStore) Ping(context.Context) error           { panic("unused") }
func (f *fakeStore) Origin() string                       { panic("unused") }
func (f *fakeStore) Type() string                         { panic("unused") }
func (f *fakeStore) Root() string                         { panic("unused") }
func (f *fakeStore) Flags() location.Flags                { panic("unused") }
func (f *fakeStore) Mode(context.Context) (storage.Mode, error) {
	panic("unused")
}
func (f *fakeStore) Size(context.Context) (int64, error) { panic("unused") }
func (f *fakeStore) Close(context.Context) error         { panic("unused") }

// makeMAC returns a 32-byte MAC whose first byte is the given value.
// 32 bytes is what objects.MAC and the httpd path validator require.
func makeMAC(first byte) objects.MAC {
	var m objects.MAC
	m[0] = first
	return m
}

// newTestMux wires up the same routes as Server() but against the supplied
// fakeStore so we can drive every handler via httptest without spinning up
// a real http.Server.
func newTestMux(store *fakeStore, noDelete bool) *http.ServeMux {
	s := &server{store: store, noDelete: noDelete}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.openRepository)
	mux.HandleFunc("GET /resources/{resource}", s.listResource)
	mux.HandleFunc("GET /resources/{resource}/{mac}", s.getResource)
	mux.HandleFunc("PUT /resources/{resource}/{mac}", s.putResource)
	mux.HandleFunc("DELETE /resources/{resource}/{mac}", s.deleteResource)
	return mux
}

// ---------- pure parser helpers ----------

func TestGetResource_KnownTypes(t *testing.T) {
	cases := map[string]storage.StorageResource{
		"packfiles":    storage.StorageResourcePackfile,
		"states":       storage.StorageResourceState,
		"locks":        storage.StorageResourceLock,
		"eccpackfiles": storage.StorageResourceECCPackfile,
		"eccstates":    storage.StorageResourceECCState,
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/resources/"+name, nil)
			req.SetPathValue("resource", name)
			got, err := getResource(req)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if got != want {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}

func TestGetResource_Invalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/resources/bogus", nil)
	req.SetPathValue("resource", "bogus")
	if _, err := getResource(req); !errors.Is(err, ErrInvalidResourceType) {
		t.Fatalf("err = %v, want ErrInvalidResourceType", err)
	}
}

func TestGetMac_ValidHex(t *testing.T) {
	mac := makeMAC(0xde)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.SetPathValue("mac", hex.EncodeToString(mac[:]))
	got, err := getMac(req)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != mac {
		t.Fatalf("got %x, want %x", got, mac)
	}
}

func TestGetMac_NonHex(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.SetPathValue("mac", "notvalidhex")
	if _, err := getMac(req); !errors.Is(err, ErrInvalidMAC) {
		t.Fatalf("err = %v, want ErrInvalidMAC", err)
	}
}

func TestGetMac_WrongLength(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.SetPathValue("mac", "deadbeef") // valid hex, 4 bytes — too short
	if _, err := getMac(req); !errors.Is(err, ErrInvalidMAC) {
		t.Fatalf("err = %v, want ErrInvalidMAC", err)
	}
}

func TestGetRange_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rng, err := getRange(req)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if rng != nil {
		t.Fatalf("range should be nil when Range header is absent, got %+v", rng)
	}
}

func TestGetRange_Valid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Range", "bytes=10-30")
	rng, err := getRange(req)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if rng == nil || rng.Offset != 10 || rng.Length != 20 {
		t.Fatalf("got %+v, want offset=10 length=20", rng)
	}
}

func TestGetRange_InvalidPrefix(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Range", "items=10-30")
	if _, err := getRange(req); !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("err = %v, want ErrInvalidRange", err)
	}
}

func TestGetRange_NoDash(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Range", "bytes=10")
	if _, err := getRange(req); !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("err = %v, want ErrInvalidRange", err)
	}
}

func TestGetRange_NonNumericStart(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Range", "bytes=x-30")
	if _, err := getRange(req); !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("err = %v", err)
	}
}

func TestGetRange_NonNumericEnd(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Range", "bytes=10-y")
	if _, err := getRange(req); !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("err = %v", err)
	}
}

func TestGetRange_EndBeforeStart(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Range", "bytes=100-50")
	if _, err := getRange(req); !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("err = %v", err)
	}
}

func TestGetRange_EndEqualStart(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Range", "bytes=50-50")
	if _, err := getRange(req); !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("err = %v", err)
	}
}

// ---------- handlers ----------

func TestOpenRepository_Ok(t *testing.T) {
	store := &fakeStore{openConfig: []byte("wrapped-config-bytes")}
	mux := newTestMux(store, false)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/octet-stream" {
		t.Fatalf("Content-Type = %q", got)
	}
	if rec.Body.String() != "wrapped-config-bytes" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestOpenRepository_StoreError(t *testing.T) {
	store := &fakeStore{openErr: errors.New("disk on fire")}
	mux := newTestMux(store, false)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "disk on fire") {
		t.Fatalf("body should include error, got %q", rec.Body.String())
	}
}

func TestListResource_Ok(t *testing.T) {
	mac := makeMAC(0xab)
	store := &fakeStore{
		listResources: map[storage.StorageResource][]objects.MAC{
			storage.StorageResourcePackfile: {mac},
		},
	}
	mux := newTestMux(store, false)

	req := httptest.NewRequest(http.MethodGet, "/resources/packfiles", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got []objects.MAC
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0] != mac {
		t.Fatalf("got %v, want [%v]", got, mac)
	}
}

func TestListResource_BadType(t *testing.T) {
	store := &fakeStore{}
	mux := newTestMux(store, false)

	req := httptest.NewRequest(http.MethodGet, "/resources/bogus", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestListResource_StoreError(t *testing.T) {
	store := &fakeStore{listErr: errors.New("list failed")}
	mux := newTestMux(store, false)

	req := httptest.NewRequest(http.MethodGet, "/resources/packfiles", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestGetResourceHandler_Ok(t *testing.T) {
	mac := makeMAC(0x42)
	store := &fakeStore{
		getData: map[storage.StorageResource]map[objects.MAC][]byte{
			storage.StorageResourcePackfile: {mac: []byte("payload")},
		},
	}
	mux := newTestMux(store, false)

	req := httptest.NewRequest(http.MethodGet, "/resources/packfiles/"+hex.EncodeToString(mac[:]), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Body.String() != "payload" {
		t.Fatalf("body = %q", rec.Body.String())
	}
	if store.lastGetType != storage.StorageResourcePackfile || store.lastGetMAC != mac {
		t.Fatalf("store call args wrong: type=%v mac=%v", store.lastGetType, store.lastGetMAC)
	}
}

func TestGetResourceHandler_WithRange(t *testing.T) {
	mac := makeMAC(0x42)
	store := &fakeStore{
		getData: map[storage.StorageResource]map[objects.MAC][]byte{
			storage.StorageResourcePackfile: {mac: []byte("payload")},
		},
	}
	mux := newTestMux(store, false)

	req := httptest.NewRequest(http.MethodGet, "/resources/packfiles/"+hex.EncodeToString(mac[:]), nil)
	req.Header.Set("Range", "bytes=2-5")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if store.lastGetRng == nil || store.lastGetRng.Offset != 2 || store.lastGetRng.Length != 3 {
		t.Fatalf("range wrong: %+v", store.lastGetRng)
	}
}

func TestGetResourceHandler_BadType(t *testing.T) {
	mac := makeMAC(0x42)
	store := &fakeStore{}
	mux := newTestMux(store, false)

	req := httptest.NewRequest(http.MethodGet, "/resources/bogus/"+hex.EncodeToString(mac[:]), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetResourceHandler_BadMAC(t *testing.T) {
	store := &fakeStore{}
	mux := newTestMux(store, false)

	req := httptest.NewRequest(http.MethodGet, "/resources/packfiles/short", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetResourceHandler_BadRange(t *testing.T) {
	mac := makeMAC(0x42)
	store := &fakeStore{}
	mux := newTestMux(store, false)

	req := httptest.NewRequest(http.MethodGet, "/resources/packfiles/"+hex.EncodeToString(mac[:]), nil)
	req.Header.Set("Range", "items=garbage")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetResourceHandler_StoreError(t *testing.T) {
	mac := makeMAC(0x42)
	store := &fakeStore{getErr: errors.New("missing")}
	mux := newTestMux(store, false)

	req := httptest.NewRequest(http.MethodGet, "/resources/packfiles/"+hex.EncodeToString(mac[:]), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestPutResource_Ok(t *testing.T) {
	mac := makeMAC(0x42)
	store := &fakeStore{}
	mux := newTestMux(store, false)

	req := httptest.NewRequest(http.MethodPut,
		"/resources/packfiles/"+hex.EncodeToString(mac[:]), strings.NewReader("body-bytes"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if string(store.lastPutData) != "body-bytes" {
		t.Fatalf("put data = %q", store.lastPutData)
	}
}

func TestPutResource_BadType(t *testing.T) {
	mac := makeMAC(0x42)
	mux := newTestMux(&fakeStore{}, false)

	req := httptest.NewRequest(http.MethodPut,
		"/resources/bogus/"+hex.EncodeToString(mac[:]), strings.NewReader("x"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestPutResource_BadMAC(t *testing.T) {
	mux := newTestMux(&fakeStore{}, false)

	req := httptest.NewRequest(http.MethodPut,
		"/resources/packfiles/short", strings.NewReader("x"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestPutResource_StoreError(t *testing.T) {
	mac := makeMAC(0x42)
	mux := newTestMux(&fakeStore{putErr: errors.New("write failed")}, false)

	req := httptest.NewRequest(http.MethodPut,
		"/resources/packfiles/"+hex.EncodeToString(mac[:]), strings.NewReader("x"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestDeleteResource_Ok(t *testing.T) {
	mac := makeMAC(0x42)
	store := &fakeStore{}
	mux := newTestMux(store, false)

	req := httptest.NewRequest(http.MethodDelete,
		"/resources/packfiles/"+hex.EncodeToString(mac[:]), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if store.lastDelType != storage.StorageResourcePackfile || store.lastDelMAC != mac {
		t.Fatalf("delete call args wrong: type=%v mac=%v", store.lastDelType, store.lastDelMAC)
	}
}

func TestDeleteResource_Forbidden(t *testing.T) {
	mac := makeMAC(0x42)
	mux := newTestMux(&fakeStore{}, true) // noDelete=true

	req := httptest.NewRequest(http.MethodDelete,
		"/resources/packfiles/"+hex.EncodeToString(mac[:]), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestDeleteResource_BadType(t *testing.T) {
	mac := makeMAC(0x42)
	mux := newTestMux(&fakeStore{}, false)

	req := httptest.NewRequest(http.MethodDelete,
		"/resources/bogus/"+hex.EncodeToString(mac[:]), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestDeleteResource_BadMAC(t *testing.T) {
	mux := newTestMux(&fakeStore{}, false)

	req := httptest.NewRequest(http.MethodDelete, "/resources/packfiles/short", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestDeleteResource_StoreError(t *testing.T) {
	mac := makeMAC(0x42)
	mux := newTestMux(&fakeStore{deleteErr: errors.New("nope")}, false)

	req := httptest.NewRequest(http.MethodDelete,
		"/resources/packfiles/"+hex.EncodeToString(mac[:]), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

// freePort asks the kernel for an unused TCP port and returns it as a string.
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().(*net.TCPAddr)
	_ = l.Close()
	return fmt.Sprintf("127.0.0.1:%d", addr.Port)
}

func TestServer_StartServeAndShutdown(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, nil, nil, nil)

	addr := freePort(t)
	errCh := make(chan error, 1)
	go func() {
		// noDelete=false, no TLS — exercises the plain ListenAndServe path.
		errCh <- Server(ctx, repo, addr, false, "", "", "")
	}()

	// Wait until the server accepts connections, then make one request so the
	// wired routes are exercised through a real listener.
	base := "http://" + addr
	var resp *http.Response
	deadline := 2 * time.Second
	for waited := time.Duration(0); waited < deadline; waited += 20 * time.Millisecond {
		r, err := http.Get(base + "/")
		if err == nil {
			resp = r
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if resp == nil {
		t.Fatal("server never became reachable")
	}
	_ = resp.Body.Close()

	// Cancelling the context triggers the goroutine inside Server() to call
	// Shutdown, which makes ListenAndServe return.
	ctx.GetInner().Cancel(nil)

	select {
	case err := <-errCh:
		// http.ErrServerClosed is the expected return after a graceful Shutdown.
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("Server returned %v, want ErrServerClosed or nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not return after context cancellation")
	}
}

func TestServer_TLSConfigErrors(t *testing.T) {
	// With cert/key set but pointing at nonexistent files, ListenAndServeTLS
	// returns an error immediately — this covers the TLS branch of Server().
	repo, ctx := ptesting.GenerateRepository(t, nil, nil, nil)

	addr := freePort(t)
	err := Server(ctx, repo, addr, false, "", "/nonexistent/cert.pem", "/nonexistent/key.pem")
	if err == nil {
		t.Fatal("expected error from ListenAndServeTLS with missing cert/key")
	}
}

func TestServerToken(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, nil, nil, nil)

	token := "nestor"

	addr := freePort(t)
	errCh := make(chan error, 1)
	go func() {
		errCh <- Server(ctx, repo, addr, false, token, "", "")
	}()

	// Wait until the server accepts connections, then make one request so the
	// wired routes are exercised through a real listener.
	base := "http://" + addr
	var resp *http.Response
	deadline := 2 * time.Second
	for waited := time.Duration(0); waited < deadline; waited += 20 * time.Millisecond {
		r, err := http.Get(base + "/")
		if err == nil {
			resp = r
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if resp == nil {
		t.Fatal("server never became reachable")
	}
	_ = resp.Body.Close()

	require.Equal(t, 401, resp.StatusCode)

	req, err := http.NewRequest("GET", base+"/", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)

	// Cancelling the context triggers the goroutine inside Server() to call
	// Shutdown, which makes ListenAndServe return.
	ctx.GetInner().Cancel(nil)

	select {
	case err := <-errCh:
		// http.ErrServerClosed is the expected return after a graceful Shutdown.
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("Server returned %v, want ErrServerClosed or nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not return after context cancellation")
	}
}
