package ui

import (
	"bytes"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	"github.com/PlakarKorp/plakar/subcommands"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func TestUiRegisteredFactory(t *testing.T) {
	cmd, _, _ := subcommands.Lookup([]string{"ui"})
	require.NotNil(t, cmd)
	require.IsType(t, &Ui{}, cmd)
}

func TestUiParse(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo

	cmd := &Ui{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-addr", "127.0.0.1:9999",
		"-cors", "-no-auth", "-no-spawn", "-no-refresh",
	}))
	require.Equal(t, "127.0.0.1:9999", cmd.Addr)
	require.True(t, cmd.Cors)
	require.True(t, cmd.NoAuth)
	require.True(t, cmd.NoSpawn)
	require.True(t, cmd.NoRefresh)
}

func TestUiParseTooManyArgs(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo

	cmd := &Ui{}
	err := cmd.Parse(ctx, []string{"extra"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many arguments")
}

func TestUiExecuteStartsAndShutsDown(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)

	cmd := &Ui{}
	// -no-spawn keeps it from launching a browser; -no-auth avoids minting a
	// token; a random port avoids collisions.
	require.NoError(t, cmd.Parse(ctx, []string{"-no-spawn", "-no-auth", "-no-refresh"}))

	done := make(chan struct{})
	go func() {
		cmd.Execute(ctx, repo)
		close(done)
	}()

	// Give the server a moment to bind, then cancel the context to trigger
	// graceful shutdown of the underlying v2 server.
	time.Sleep(200 * time.Millisecond)
	ctx.GetInner().Cancel(nil)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Ui.Execute did not return after context cancellation")
	}
}

func TestUiTokenFromEnvironment(t *testing.T) {
	t.Setenv("PLAKAR_UI_TOKEN", "pinned-token")

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())

	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	cmd := &Ui{}
	require.NoError(t, cmd.Parse(ctx, []string{"-addr", addr, "-no-spawn", "-no-refresh"}))

	done := make(chan struct{})
	go func() {
		cmd.Execute(ctx, repo)
		close(done)
	}()
	t.Cleanup(func() {
		ctx.GetInner().Cancel(nil)
		<-done
	})

	status := func(token string) int {
		req, err := http.NewRequest(http.MethodGet, "http://"+addr+"/api/info", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0
		}
		res.Body.Close()
		return res.StatusCode
	}

	require.Eventually(t, func() bool { return status("pinned-token") == http.StatusOK },
		5*time.Second, 20*time.Millisecond)
	require.Equal(t, http.StatusUnauthorized, status("another-token"))
}
