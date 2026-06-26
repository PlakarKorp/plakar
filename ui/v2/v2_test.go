package v2

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func TestCorsMiddleware(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	h := corsMiddleware(next)

	// Normal request passes through and gets CORS headers.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/foo", nil)
	h.ServeHTTP(rec, req)
	require.True(t, called)
	require.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, http.StatusOK, rec.Code)

	// Preflight OPTIONS short-circuits with 204 and does not call next.
	called = false
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodOptions, "/api/foo", nil)
	h.ServeHTTP(rec2, req2)
	require.False(t, called)
	require.Equal(t, http.StatusNoContent, rec2.Code)
}

// Drive the real Ui server: it serves the embedded frontend and the api routes,
// then shuts down when the app context is cancelled.
func TestUiServesFrontendAndShutsDown(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	listenAddr := fmt.Sprintf("localhost:%d", 47213)

	srvErr := make(chan error, 1)
	go func() {
		srvErr <- Ui(repo, ctx, listenAddr, &UiOptions{NoSpawn: true, Cors: true})
	}()

	base := "http://" + listenAddr

	// Poll until the server is accepting connections.
	client := &http.Client{Timeout: 2 * time.Second}
	var ready bool
	for i := 0; i < 100; i++ {
		resp, err := client.Get(base + "/")
		if err == nil {
			resp.Body.Close()
			ready = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.True(t, ready, "server did not come up")

	// Root path -> index.html fallback (path does not exist as a real file).
	resp, err := client.Get(base + "/some/spa/route")
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotEmpty(t, body)

	// A real embedded asset is served directly.
	resp2, err := client.Get(base + "/favicon.ico")
	require.NoError(t, err)
	resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	// CORS preflight handled by middleware.
	req, _ := http.NewRequest(http.MethodOptions, base+"/api/v1/info", nil)
	resp3, err := client.Do(req)
	require.NoError(t, err)
	resp3.Body.Close()
	require.Equal(t, http.StatusNoContent, resp3.StatusCode)

	// Output mentions the launch URL.
	require.Contains(t, bufOut.String(), "launching webUI at")

	// Cancel the app context to trigger graceful shutdown.
	ctx.Cancel(errors.New("test done"))

	select {
	case err := <-srvErr:
		// http.ErrServerClosed is the expected return after Shutdown.
		if err != nil {
			require.ErrorIs(t, err, http.ErrServerClosed)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down")
	}
}

// With a token configured, the launch URL embeds the token query param.
func TestUiTokenURL(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	listenAddr := "localhost:47214"
	srvErr := make(chan error, 1)
	go func() {
		srvErr <- Ui(repo, ctx, listenAddr, &UiOptions{NoSpawn: true, Token: "sekret"})
	}()

	client := &http.Client{Timeout: 2 * time.Second}
	for i := 0; i < 100; i++ {
		resp, err := client.Get("http://" + listenAddr + "/")
		if err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	require.Contains(t, bufOut.String(), "plakar_token=sekret")

	ctx.Cancel(errors.New("done"))
	select {
	case <-srvErr:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down")
	}
}
