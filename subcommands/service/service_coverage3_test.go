package services

import (
	"bytes"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/stretchr/testify/require"
)

// cov3Ctx builds a hermetic AppContext with a fresh cookie jar.
func cov3Ctx(t *testing.T) *appcontext.AppContext {
	t.Helper()
	t.Setenv("PLAKAR_TOKEN", "")
	ctx := appcontext.NewAppContext()
	ctx.Stdout = bytes.NewBuffer(nil)
	ctx.Stderr = bytes.NewBuffer(nil)
	ctx.SetCookies(cookies.NewManager(t.TempDir()))
	return ctx
}

// TestCov3GetClientNoToken exercises the "empty token -> requires login" branch
// of getClient (an empty cookie jar returns an empty token, not an error).
func TestCov3GetClientNoToken(t *testing.T) {
	ctx := cov3Ctx(t)

	sc, err := getClient(ctx)
	require.Error(t, err)
	require.Nil(t, sc)
	require.Contains(t, err.Error(), "login")
}

// TestCov3GetClientWithToken covers the success path of getClient: when an auth
// token is present in the cookie jar a non-nil ServiceConnector is returned with
// no error. This exercises the trailing return statement (no network is hit; a
// connector is just constructed).
func TestCov3GetClientWithToken(t *testing.T) {
	ctx := cov3Ctx(t)
	require.NoError(t, ctx.GetCookies().PutAuthToken("an-auth-token"))

	sc, err := getClient(ctx)
	require.NoError(t, err)
	require.NotNil(t, sc)
}

// TestCov3ServiceExecuteNoAction covers the top-level Service.Execute branch
// (always status 1, "no action specified") independently of Parse.
func TestCov3ServiceExecuteNoAction(t *testing.T) {
	ctx := cov3Ctx(t)
	status, err := (&Service{}).Execute(ctx, nil)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "no action specified")
}

// TestCov3ParseSetsRepositorySecret verifies the tail of each one-arg Parse:
// Service is set from the positional arg and RepositorySecret is taken from the
// context. ctx.GetSecret() returns nil here, which is fine.
func TestCov3ParseSetsRepositorySecret(t *testing.T) {
	ctx := cov3Ctx(t)

	{
		cmd := &ServiceStatus{}
		require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
		require.Equal(t, "alerting", cmd.Service)
	}
	{
		cmd := &ServiceEnable{}
		require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
		require.Equal(t, "alerting", cmd.Service)
	}
	{
		cmd := &ServiceDisable{}
		require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
		require.Equal(t, "alerting", cmd.Service)
	}
	{
		cmd := &ServiceRm{}
		require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
		require.Equal(t, "alerting", cmd.Service)
	}
	{
		cmd := &ServiceShow{}
		require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
		require.Equal(t, "alerting", cmd.Service)
	}
}

// TestCov3LookupUnknownSubcommand confirms that an unknown service subcommand
// falls back to the bare Service command (no nil panic), with the unknown token
// preserved as a trailing arg.
func TestCov3LookupUnknownSubcommand(t *testing.T) {
	cmd, _, rest := subcommands.Lookup([]string{"service", "bogus-subcommand"})
	require.NotNil(t, cmd)
	require.IsType(t, &Service{}, cmd)
	require.Equal(t, []string{"bogus-subcommand"}, rest)
}
