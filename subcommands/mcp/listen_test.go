package mcp

import (
	"bytes"
	"context"
	"encoding/hex"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestMcpParseListenDefaultsToStdio(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	cmd := &Mcp{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	require.Empty(t, cmd.ListenAddr)
}

func TestMcpParseListenLoopbackNeedsNoToken(t *testing.T) {
	for _, addr := range []string{"localhost:9877", "127.0.0.1:9877", "[::1]:9877"} {
		t.Run(addr, func(t *testing.T) {
			_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
			defer ctx.Close()

			cmd := &Mcp{}
			require.NoError(t, cmd.Parse(ctx, []string{"-listen", addr}))
			require.Equal(t, addr, cmd.ListenAddr)
		})
	}
}

// A bind reachable from off the host must not serve the repository to anyone
// who can route to it just because -token was left off.
func TestMcpParseListenPublicRequiresToken(t *testing.T) {
	for _, addr := range []string{"0.0.0.0:9877", ":9877", "192.0.2.1:9877", "example.com:9877"} {
		t.Run(addr, func(t *testing.T) {
			_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
			defer ctx.Close()

			cmd := &Mcp{}
			err := cmd.Parse(ctx, []string{"-listen", addr})
			require.ErrorContains(t, err, "beyond this host")

			cmd = &Mcp{}
			require.NoError(t, cmd.Parse(ctx, []string{"-listen", addr, "-token", "s3cret"}))
		})
	}
}

// -listen https://host is the natural mistake; it has to name the fix.
func TestMcpParseListenRejectsURL(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	cmd := &Mcp{}
	err := cmd.Parse(ctx, []string{"-listen", "https://example.com:9877"})
	require.ErrorContains(t, err, "not a URL")
}

func TestMcpParseListenRejectsBadAddr(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	cmd := &Mcp{}
	require.Error(t, cmd.Parse(ctx, []string{"-listen", "localhost"}))
}

func TestMcpParseHTTPFlagsRequireListen(t *testing.T) {
	for _, args := range [][]string{
		{"-token", "s3cret"},
		{"-cert", "/nonexistent.pem", "-key", "/nonexistent.key"},
	} {
		_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
		cmd := &Mcp{}
		require.ErrorContains(t, cmd.Parse(ctx, args), "require")
		ctx.Close()
	}
}

func TestMcpParseCertAndKeyGoTogether(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	cmd := &Mcp{}
	err := cmd.Parse(ctx, []string{"-listen", "localhost:9877", "-cert", "/nonexistent.pem"})
	require.ErrorContains(t, err, "together")
}

// Drive a real session over HTTP with the SDK client: the transport is wired
// up, the tools are reachable, and the repository answers through it.
func TestServeHTTPServesTools(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())
	snap.Close()

	cmd := &Mcp{MaxFileSize: 1 << 20, ListenAddr: "127.0.0.1:0", Token: "s3cret"}
	url, wait := startTestServer(t, ctx, cmd, repo)

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v1"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: &http.Client{Transport: bearer{token: "s3cret"}},
	}

	session, err := client.Connect(context.Background(), transport, nil)
	require.NoError(t, err)

	tools, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, tools.Tools)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_snapshots",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	require.False(t, res.IsError)

	var text string
	for _, content := range res.Content {
		if tc, ok := content.(*mcp.TextContent); ok {
			text += tc.Text
		}
	}
	require.Contains(t, text, snapID)

	// Close the session first: the drain then finishes on its own rather than
	// falling through to the shutdown deadline.
	require.NoError(t, session.Close())

	ctx.Cancel(context.Canceled)
	require.NoError(t, <-wait)
}

// Without the bearer token the session never gets off the ground.
func TestServeHTTPRequiresToken(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	cmd := &Mcp{MaxFileSize: 1 << 20, ListenAddr: "127.0.0.1:0", Token: "s3cret"}
	url, wait := startTestServer(t, ctx, cmd, repo)

	resp, err := http.Post(url, "application/json", bytes.NewReader(nil))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	ctx.Cancel(context.Canceled)
	require.NoError(t, <-wait)
}

// startTestServer serves cmd on an ephemeral port and returns its URL plus a
// channel carrying the serve error once the context is cancelled.
func startTestServer(t *testing.T, ctx *appcontext.AppContext, cmd *Mcp, repo *repository.Repository) (string, <-chan error) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	cmd.ListenAddr = addr
	wait := make(chan error, 1)
	go func() {
		_, err := cmd.serveHTTP(ctx, cmd.newServer(ctx, repo))
		wait <- err
	}()

	url := "http://" + addr
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if c, err := net.Dial("tcp", addr); err == nil {
			c.Close()
			return url, wait
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server did not come up on %s", addr)
	return "", nil
}

type bearer struct{ token string }

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}

// An operator interrupting the server while a client still holds its session
// open must get the process back, not wait on a stream that never ends.
func TestServeHTTPShutsDownWithClientAttached(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	cmd := &Mcp{MaxFileSize: 1 << 20, Token: "s3cret"}
	url, wait := startTestServer(t, ctx, cmd, repo)

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: &http.Client{Transport: bearer{token: "s3cret"}},
	}, nil)
	require.NoError(t, err)
	defer session.Close()

	// Deliberately leave the session open across the cancellation.
	ctx.Cancel(context.Canceled)

	select {
	case err := <-wait:
		require.NoError(t, err)
	case <-time.After(shutdownGrace + 15*time.Second):
		t.Fatal("serveHTTP did not return with a client attached")
	}
}

// The write gates are a property of the server, not of the transport: exposing
// it over HTTP must not hand a client tools that stdio would have withheld.
func TestServeHTTPKeepsWriteToolsGated(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	cmd := &Mcp{MaxFileSize: 1 << 20, Token: "s3cret"}
	url, wait := startTestServer(t, ctx, cmd, repo)

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: &http.Client{Transport: bearer{token: "s3cret"}},
	}, nil)
	require.NoError(t, err)

	tools, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)

	served := make(map[string]bool, len(tools.Tools))
	for _, tool := range tools.Tools {
		served[tool.Name] = true
	}

	require.True(t, served["list_snapshots"], "read-only tools stay available")
	for _, name := range []string{"create_backup", "remove_snapshots", "prune_snapshots", "restore_files", "sync_snapshots"} {
		require.False(t, served[name], "%s must not be served without its flag", name)
	}

	require.NoError(t, session.Close())
	ctx.Cancel(context.Canceled)
	require.NoError(t, <-wait)
}
