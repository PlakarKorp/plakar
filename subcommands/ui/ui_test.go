package ui

import (
	"bytes"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func newCtx(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.CWD = t.TempDir()
	return ctx
}

// ---------------------------------------------------------------------------
// Parse
// ---------------------------------------------------------------------------

func TestUiParseDefaults(t *testing.T) {
	ctx := newCtx(t)
	cmd := &Ui{}
	err := cmd.Parse(ctx, []string{})
	require.NoError(t, err)

	require.Equal(t, "", cmd.Addr)
	require.False(t, cmd.Cors)
	require.False(t, cmd.NoAuth)
	require.False(t, cmd.NoSpawn)
	require.False(t, cmd.NoRefresh)
	require.Equal(t, "", cmd.Cert)
	require.Equal(t, "", cmd.Key)
}

func TestUiParseAllFlags(t *testing.T) {
	ctx := newCtx(t)
	cmd := &Ui{}
	err := cmd.Parse(ctx, []string{
		"-addr", "127.0.0.1:8080",
		"-cors",
		"-no-auth",
		"-no-spawn",
		"-no-refresh",
		"-cert", "/tmp/cert.pem",
		"-key", "/tmp/key.pem",
	})
	require.NoError(t, err)

	require.Equal(t, "127.0.0.1:8080", cmd.Addr)
	require.True(t, cmd.Cors)
	require.True(t, cmd.NoAuth)
	require.True(t, cmd.NoSpawn)
	require.True(t, cmd.NoRefresh)
	require.Equal(t, "/tmp/cert.pem", cmd.Cert)
	require.Equal(t, "/tmp/key.pem", cmd.Key)
}

func TestUiParseTooManyArgs(t *testing.T) {
	ctx := newCtx(t)
	cmd := &Ui{}
	err := cmd.Parse(ctx, []string{"unexpected"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Too many arguments")
}

func TestUiParseStoresSecret(t *testing.T) {
	ctx := newCtx(t)
	ctx.SetSecret([]byte("hunter2"))
	cmd := &Ui{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	require.Equal(t, []byte("hunter2"), cmd.RepositorySecret)
}

// ---------------------------------------------------------------------------
// Lookup wiring (registration)
// ---------------------------------------------------------------------------

func TestUiLookup(t *testing.T) {
	sub, _, args := subcommands.Lookup([]string{"ui", "-no-spawn"})
	require.NotNil(t, sub)
	_, ok := sub.(*Ui)
	require.True(t, ok)
	require.Equal(t, []string{"-no-spawn"}, args)
}

// ---------------------------------------------------------------------------
// Execute error path: an invalid listen address makes the underlying HTTP
// server fail immediately, so Execute returns (1, err) without blocking.
// NoSpawn avoids touching a browser; NoAuth avoids generating a token.
// ---------------------------------------------------------------------------

func TestUiExecuteInvalidAddr(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	cmd := &Ui{
		Addr:    "999.999.999.999:99999", // unresolvable -> ListenAndServe errors
		NoSpawn: true,
		NoAuth:  true,
	}

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, bufErr.String(), "ui:")
}

// With auth enabled and PLAKAR_UI_TOKEN set, the token is taken from the env;
// we still feed an invalid address so Execute returns rather than blocking.
func TestUiExecuteTokenFromEnv(t *testing.T) {
	t.Setenv("PLAKAR_UI_TOKEN", "fixed-token")

	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	cmd := &Ui{
		Addr:    "999.999.999.999:99999",
		NoSpawn: true,
	}

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}
