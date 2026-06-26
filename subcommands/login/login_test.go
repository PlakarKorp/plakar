package login

import (
	"bytes"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/stretchr/testify/require"
)

// newTestContext builds a hermetic AppContext backed by an on-disk
// cookies.Manager rooted in a temp dir. Stdout is wired to a buffer.
func newTestContext(t *testing.T) (*appcontext.AppContext, *bytes.Buffer) {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.SetCookies(cookies.NewManager(t.TempDir()))
	bufOut := bytes.NewBuffer(nil)
	ctx.Stdout = bufOut
	t.Cleanup(ctx.Close)
	return ctx, bufOut
}

func TestLoginParseStatusAlone(t *testing.T) {
	ctx, _ := newTestContext(t)

	cmd := &Login{}
	require.NoError(t, cmd.Parse(ctx, []string{"-status"}))
	require.True(t, cmd.Status)

	// -status combined with another option is rejected.
	cmd = &Login{}
	err := cmd.Parse(ctx, []string{"-status", "-github"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be used alone")
}

func TestLoginParseConflicts(t *testing.T) {
	ctx, _ := newTestContext(t)

	cmd := &Login{}
	err := cmd.Parse(ctx, []string{"-github", "-env"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "the -github option cannot be used with -email or -env")

	cmd = &Login{}
	err = cmd.Parse(ctx, []string{"-email", "foo@example.com", "-env"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "the -email option cannot be used with -env")

	// -no-spawn combined with -email (a non-github method) is rejected.
	cmd = &Login{}
	err = cmd.Parse(ctx, []string{"-no-spawn", "-email", "user@example.com"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "the -no-spawn option is only valid with -github")

	cmd = &Login{}
	err = cmd.Parse(ctx, []string{"extra-arg"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many arguments")
}

func TestLoginParseEmailValidation(t *testing.T) {
	ctx, _ := newTestContext(t)

	cmd := &Login{}
	require.NoError(t, cmd.Parse(ctx, []string{"-email", "user@example.com"}))
	require.Equal(t, "user@example.com", cmd.Email)

	cmd = &Login{}
	err := cmd.Parse(ctx, []string{"-email", "not-an-email"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid email address")
}

func TestLoginParseDefaultsToGithub(t *testing.T) {
	ctx, _ := newTestContext(t)

	cmd := &Login{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	require.True(t, cmd.Github)
}

func TestLoginExecuteStatusNotLoggedIn(t *testing.T) {
	t.Setenv("PLAKAR_TOKEN", "")
	ctx, bufOut := newTestContext(t)

	cmd := &Login{Status: true}
	status, err := cmd.Execute(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, bufOut.String(), "not logged in")
}

func TestLoginExecuteStatusLoggedIn(t *testing.T) {
	t.Setenv("PLAKAR_TOKEN", "")
	ctx, bufOut := newTestContext(t)
	require.NoError(t, ctx.GetCookies().PutAuthToken("a-token"))

	cmd := &Login{Status: true}
	status, err := cmd.Execute(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	out := bufOut.String()
	require.Contains(t, out, "logged in")
	require.NotContains(t, out, "not logged in")
}

func TestLoginExecuteEnvWithToken(t *testing.T) {
	t.Setenv("PLAKAR_TOKEN", "env-token-value")
	ctx, _ := newTestContext(t)

	cmd := &Login{Env: true}
	status, err := cmd.Execute(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// The env token should have been persisted to the cookies store.
	got, err := ctx.GetCookies().GetAuthToken()
	require.NoError(t, err)
	require.Equal(t, "env-token-value", got)
}

func TestLoginExecuteEnvWithoutToken(t *testing.T) {
	t.Setenv("PLAKAR_TOKEN", "")
	ctx, _ := newTestContext(t)

	cmd := &Login{Env: true}
	status, err := cmd.Execute(ctx, nil)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "no auth token found in environment variable PLAKAR_TOKEN")
}

func TestLogoutParseTooManyArgs(t *testing.T) {
	ctx, _ := newTestContext(t)
	cmd := &Logout{}
	err := cmd.Parse(ctx, []string{"extra"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many arguments")
}

func TestLogoutExecuteNotLoggedIn(t *testing.T) {
	t.Setenv("PLAKAR_TOKEN", "")
	ctx, _ := newTestContext(t)

	cmd := &Logout{}
	status, err := cmd.Execute(ctx, nil)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestLogoutExecuteSuccess(t *testing.T) {
	t.Setenv("PLAKAR_TOKEN", "")
	ctx, _ := newTestContext(t)
	require.NoError(t, ctx.GetCookies().PutAuthToken("a-token"))

	cmd := &Logout{}
	status, err := cmd.Execute(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Token should be gone now.
	_, err = ctx.GetCookies().GetAuthToken()
	require.Error(t, err)
}

func TestTokenCreateParseTooManyArgs(t *testing.T) {
	ctx, _ := newTestContext(t)
	cmd := &TokenCreate{}
	err := cmd.Parse(ctx, []string{"extra"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Too many arguments")
}

// DeriveToken bails out before any network call when no auth token is available.
func TestTokenCreateExecuteNoToken(t *testing.T) {
	t.Setenv("PLAKAR_TOKEN", "")
	ctx, _ := newTestContext(t)

	cmd := &TokenCreate{}
	status, err := cmd.Execute(ctx, nil)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "no auth token found")
}
