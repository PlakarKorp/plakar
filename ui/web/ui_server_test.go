package web

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().(*net.TCPAddr)
	_ = l.Close()
	return fmt.Sprintf("127.0.0.1:%d", addr.Port)
}

func TestUiStartServeShutdown(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)

	addr := freePort(t)
	errCh := make(chan error, 1)
	go func() {
		errCh <- Ui(repo, ctx, addr, &UiOptions{NoSpawn: true, NoRefresh: true})
	}()

	// Wait until the server is reachable, then make a request: an API route and
	// a static path that falls back to index.html.
	base := "http://" + addr
	var reached bool
	for waited := time.Duration(0); waited < 3*time.Second; waited += 25 * time.Millisecond {
		if resp, err := http.Get(base + "/api/info"); err == nil {
			_ = resp.Body.Close()
			reached = true
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	require.True(t, reached, "server never became reachable")

	// Unknown static asset falls back to index.html (served from the embedded FS).
	resp, err := http.Get(base + "/some/spa/route")
	require.NoError(t, err)
	_ = resp.Body.Close()

	// Cancelling the context triggers graceful shutdown.
	ctx.GetInner().Cancel(nil)

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Fatalf("Ui returned %v, want ErrServerClosed or nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Ui did not return after context cancellation")
	}
}

func TestUiCorsAndRandomPort(t *testing.T) {
	// addr == "" exercises the random-port path; Cors=true wraps the handler.
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)

	errCh := make(chan error, 1)
	go func() {
		errCh <- Ui(repo, ctx, "", &UiOptions{NoSpawn: true, NoRefresh: true, Cors: true, Token: "tok"})
	}()

	// Give the server a moment to bind, then shut it down. We don't know the
	// random port, so just confirm the goroutine exits cleanly on cancel.
	time.Sleep(200 * time.Millisecond)
	ctx.GetInner().Cancel(nil)

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Fatalf("Ui returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Ui did not return after cancellation")
	}
}

func TestUiQuitStopsServer(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)

	addr := freePort(t)
	errCh := make(chan error, 1)
	go func() {
		errCh <- Ui(repo, ctx, addr, &UiOptions{NoSpawn: true, NoRefresh: true, Token: "tok"})
	}()

	base := "http://" + addr
	var reached bool
	for waited := time.Duration(0); waited < 3*time.Second; waited += 25 * time.Millisecond {
		if resp, err := http.Get(base + "/"); err == nil {
			_ = resp.Body.Close()
			reached = true
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	require.True(t, reached, "server never became reachable")

	req, err := http.NewRequest("POST", base+"/api/quit", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Ui did not return after /api/quit")
	}
	require.Error(t, ctx.Err())
}
