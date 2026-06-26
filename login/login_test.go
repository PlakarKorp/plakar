package login

import (
	"context"
	"testing"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/stretchr/testify/require"
)

// newTestContext builds a hermetic AppContext with an on-disk cookies.Manager
// rooted in a temp dir, so nothing touches the real home directory.
func newTestContext(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.SetCookies(cookies.NewManager(t.TempDir()))
	t.Cleanup(ctx.Close)
	return ctx
}

func TestNewLoginFlow(t *testing.T) {
	ctx := newTestContext(t)

	flow, err := NewLoginFlow(ctx, false)
	require.NoError(t, err)
	require.NotNil(t, flow)
	require.Equal(t, ctx, flow.appCtx)
	require.False(t, flow.noSpawn)

	flow2, err := NewLoginFlow(ctx, true)
	require.NoError(t, err)
	require.True(t, flow2.noSpawn)
}

func TestLoginFlowClose(t *testing.T) {
	ctx := newTestContext(t)
	flow, err := NewLoginFlow(ctx, false)
	require.NoError(t, err)
	require.NoError(t, flow.Close())
}

// Poll with zero iterations never performs network I/O: the loop body never
// runs and it returns the "could not obtain token" exhaustion error.
func TestPollZeroIterations(t *testing.T) {
	ctx := newTestContext(t)
	flow, err := NewLoginFlow(ctx, false)
	require.NoError(t, err)

	token, err := flow.Poll("some-poll-id", 0, time.Second, func() {})
	require.Error(t, err)
	require.Empty(t, token)
	require.Contains(t, err.Error(), "could not obtain token after 0 iterations")
}

// Poll returns the context error when the context is already cancelled, before
// any HTTP request is attempted.
func TestPollContextCancelled(t *testing.T) {
	ctx := newTestContext(t)
	flow, err := NewLoginFlow(ctx, false)
	require.NoError(t, err)

	ctx.Cancel(context.Canceled)

	token, err := flow.Poll("some-poll-id", 5, time.Second, func() {})
	require.Error(t, err)
	require.Empty(t, token)
}

func TestRunUnsupportedProvider(t *testing.T) {
	ctx := newTestContext(t)
	flow, err := NewLoginFlow(ctx, false)
	require.NoError(t, err)

	token, err := flow.Run("unsupported", map[string]string{})
	require.Error(t, err)
	require.Empty(t, token)
	require.Contains(t, err.Error(), "unsupported provider: unsupported")
}

func TestRunUIUnsupportedProvider(t *testing.T) {
	ctx := newTestContext(t)
	flow, err := NewLoginFlow(ctx, false)
	require.NoError(t, err)

	url, err := flow.RunUI("unsupported", map[string]string{})
	require.Error(t, err)
	require.Empty(t, url)
	require.Contains(t, err.Error(), "unsupported provider: unsupported")
}

// DeriveToken bails out before any network call when there is no auth token
// available (no PLAKAR_TOKEN env var and no on-disk cookie).
func TestDeriveTokenNoAuthToken(t *testing.T) {
	t.Setenv("PLAKAR_TOKEN", "")
	ctx := newTestContext(t)

	token, err := DeriveToken(ctx)
	require.Error(t, err)
	require.Empty(t, token)
	require.Contains(t, err.Error(), "no auth token found")
}
