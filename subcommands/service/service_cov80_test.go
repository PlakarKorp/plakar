package services

import (
	"bytes"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/stretchr/testify/require"
)

// cov80Ctx builds a hermetic AppContext. When withToken is set, a fresh cookie
// jar is seeded with an auth token so getClient succeeds and constructs a
// ServiceConnector WITHOUT touching the network. Otherwise the jar is empty so
// getClient fails with the login error.
func cov80Ctx(t *testing.T, withToken bool) *appcontext.AppContext {
	t.Helper()
	t.Setenv("PLAKAR_TOKEN", "")
	ctx := appcontext.NewAppContext()
	ctx.Stdout = bytes.NewBuffer(nil)
	ctx.Stderr = bytes.NewBuffer(nil)
	ctx.SetCookies(cookies.NewManager(t.TempDir()))
	if withToken {
		require.NoError(t, ctx.GetCookies().PutAuthToken("an-auth-token"))
	}
	return ctx
}

// TestCov80ServiceSetExecuteNoKeys covers ServiceSet.Execute's early-return
// branch: with a valid client but no key=value pairs, Execute returns success
// before any network call (the `len(cmd.Keys) == 0` short-circuit).
func TestCov80ServiceSetExecuteNoKeys(t *testing.T) {
	ctx := cov80Ctx(t, true)

	cmd := &ServiceSet{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	require.Empty(t, cmd.Keys)

	status, err := cmd.Execute(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// TestCov80ServiceUnsetExecuteNoKeys covers the same early-return branch in
// ServiceUnset.Execute: a valid client and an empty key slice returns success
// before any network call.
func TestCov80ServiceUnsetExecuteNoKeys(t *testing.T) {
	ctx := cov80Ctx(t, true)

	cmd := &ServiceUnset{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	require.Empty(t, cmd.Keys)

	status, err := cmd.Execute(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// TestCov80ServiceShowParseDefault covers the default (no-flag) Parse path of
// ServiceShow: neither -json nor -yaml set, single positional name accepted.
func TestCov80ServiceShowParseDefault(t *testing.T) {
	ctx := cov80Ctx(t, false)

	cmd := &ServiceShow{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	require.Equal(t, "alerting", cmd.Service)
	require.False(t, cmd.AsJson)
	require.False(t, cmd.AsYaml)
	require.False(t, cmd.ShowSecrets)
}

// TestCov80ServiceSetExecuteNoKeysWithErr confirms unset/set with keys but no
// token still fails at getClient (the login branch) rather than the no-key
// short-circuit, keeping both branches distinct.
func TestCov80ServiceSetExecuteWithKeysNoToken(t *testing.T) {
	ctx := cov80Ctx(t, false)

	cmd := &ServiceSet{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting", "k=v"}))
	status, err := cmd.Execute(ctx, nil)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "login")
}
