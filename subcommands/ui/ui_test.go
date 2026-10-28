package ui

import (
	"bytes"
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
		"-cors", "-no-spawn", "-no-refresh",
	}))
	require.Equal(t, "127.0.0.1:9999", cmd.Addr)
	require.True(t, cmd.Cors)
	require.False(t, cmd.NoAuth)
	require.True(t, cmd.NoSpawn)
	require.True(t, cmd.NoRefresh)
}

func TestUiParseCorsNoAuth(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"cors with auth", []string{"-cors"}, false},
		{"no-auth without cors", []string{"-no-auth"}, false},
		{"cors with no-auth", []string{"-cors", "-no-auth"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (&Ui{}).Parse(ctx, tt.args)
			if tt.wantErr {
				require.ErrorContains(t, err, "-cors cannot be used with -no-auth")
			} else {
				require.NoError(t, err)
			}
		})
	}
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
