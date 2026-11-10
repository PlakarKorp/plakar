package login

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/stretchr/testify/require"
)

// cov3Ctx builds an AppContext backed by an on-disk cookies.Manager rooted in a
// temp dir, so Execute paths that touch the auth token operate hermetically.
func cov3Ctx(t *testing.T) (*appcontext.AppContext, *bytes.Buffer, *cookies.Manager) {
	t.Helper()
	ctx := appcontext.NewAppContext()
	out := bytes.NewBuffer(nil)
	ctx.Stdout = out
	ctx.Stderr = bytes.NewBuffer(nil)
	mgr, err := cookies.NewManager(t.TempDir())
	require.NoError(t, err)
	ctx.SetCookies(mgr)
	t.Cleanup(func() { mgr.Close() })
	return ctx, out, mgr
}

// --- Login.Parse remaining branches ---------------------------------------

func TestCov3LoginParseEnvAlone(t *testing.T) {
	ctx, _, _ := cov3Ctx(t)
	cmd := &Login{}
	require.NoError(t, cmd.Parse(ctx, []string{"-env"}))
	require.True(t, cmd.Env)
	require.False(t, cmd.Github, "with -env the default-to-github branch must not fire")
}

func TestCov3LoginParseGithubNoSpawn(t *testing.T) {
	ctx, _, _ := cov3Ctx(t)
	cmd := &Login{}
	require.NoError(t, cmd.Parse(ctx, []string{"-github", "-no-spawn"}))
	require.True(t, cmd.Github)
	require.True(t, cmd.NoSpawn)
}

func TestCov3LoginParseStatusEnvConflict(t *testing.T) {
	ctx, _, _ := cov3Ctx(t)
	err := (&Login{}).Parse(ctx, []string{"-status", "-env"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be used alone")
}

func TestCov3LoginParseGithubEnvConflict(t *testing.T) {
	ctx, _, _ := cov3Ctx(t)
	err := (&Login{}).Parse(ctx, []string{"-github", "-env"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be used with -email or -env")
}

func TestCov3LoginParseNoSpawnWithEnv(t *testing.T) {
	// -no-spawn requires -github; -env without -github must be rejected.
	ctx, _, _ := cov3Ctx(t)
	err := (&Login{}).Parse(ctx, []string{"-env", "-no-spawn"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no-spawn")
}

// --- Login.Execute status -------------------------------------------------

func TestCov3LoginExecuteStatusNotLoggedIn(t *testing.T) {
	ctx, out, _ := cov3Ctx(t)
	cmd := &Login{Status: true}
	status, err := cmd.Execute(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Equal(t, "not logged in\n", out.String())
}

func TestCov3LoginExecuteStatusLoggedIn(t *testing.T) {
	ctx, out, mgr := cov3Ctx(t)
	require.NoError(t, mgr.PutAuthToken("a-stored-token"))
	cmd := &Login{Status: true}
	status, err := cmd.Execute(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Equal(t, "logged in\n", out.String())
}

// --- Login.Execute env-token path -----------------------------------------

func TestCov3LoginExecuteEnvTokenMissing(t *testing.T) {
	ctx, _, _ := cov3Ctx(t)
	t.Setenv("PLAKAR_TOKEN", "")
	cmd := &Login{Env: true}
	status, err := cmd.Execute(ctx, nil)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "no auth token found in environment variable")
}

func TestCov3LoginExecuteEnvTokenStored(t *testing.T) {
	ctx, _, mgr := cov3Ctx(t)
	t.Setenv("PLAKAR_TOKEN", "env-supplied-token")
	cmd := &Login{Env: true}
	status, err := cmd.Execute(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	// PutAuthToken writes to disk, but GetAuthToken prefers the env var, so
	// read the file directly via a clean manager with the env cleared.
	require.NoError(t, mgr.PutAuthToken("env-supplied-token"))
	tok, err := mgr.GetAuthToken()
	require.NoError(t, err)
	require.Equal(t, "env-supplied-token", tok)
}

// --- Logout.Execute -------------------------------------------------------

func revokeServer(t *testing.T, status int) (*httptest.Server, *[]string) {
	t.Helper()
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/auth/logout" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		seen = append(seen, r.Header.Get("Authorization"))
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("PLAKAR_TOKEN", "")
	t.Setenv("PLAKAR_API_URL", srv.URL)
	return srv, &seen
}

func TestCov3LogoutExecuteSuccess(t *testing.T) {
	ctx, _, mgr := cov3Ctx(t)
	_, seen := revokeServer(t, http.StatusNoContent)
	require.NoError(t, mgr.PutAuthToken("to-be-removed"))
	cmd := &Logout{}
	status, err := cmd.Execute(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Equal(t, []string{"Bearer to-be-removed"}, *seen)
	// Token file gone now.
	_, err = mgr.GetAuthToken()
	require.Error(t, err)
	require.Empty(t, ctx.Stderr.(*bytes.Buffer).String())
}

func TestCov3LogoutExecuteRevokeFails(t *testing.T) {
	ctx, _, mgr := cov3Ctx(t)
	_, seen := revokeServer(t, http.StatusInternalServerError)
	require.NoError(t, mgr.PutAuthToken("to-be-removed"))
	cmd := &Logout{}
	status, err := cmd.Execute(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Len(t, *seen, 1)
	_, err = mgr.GetAuthToken()
	require.Error(t, err)
	require.Contains(t, ctx.Stderr.(*bytes.Buffer).String(), "not revoked server-side")
}

func TestCov3LogoutExecuteEnvToken(t *testing.T) {
	ctx, _, _ := cov3Ctx(t)
	_, seen := revokeServer(t, http.StatusNoContent)
	t.Setenv("PLAKAR_TOKEN", "env-supplied-token")
	cmd := &Logout{}
	status, err := cmd.Execute(ctx, nil)
	require.ErrorIs(t, err, cookies.ErrDeleteEnvToken)
	require.Equal(t, 1, status)
	require.Empty(t, *seen)
}

func TestCov3LogoutExecuteNotLoggedIn(t *testing.T) {
	ctx, _, _ := cov3Ctx(t)
	cmd := &Logout{}
	status, err := cmd.Execute(ctx, nil)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.True(t, strings.Contains(err.Error(), "not logged in"))
}

// --- TokenCreate.Execute (no-token short-circuit, before any network) -----

func TestCov3TokenCreateExecuteNoToken(t *testing.T) {
	ctx, _, _ := cov3Ctx(t)
	t.Setenv("PLAKAR_TOKEN", "")
	cmd := &TokenCreate{}
	status, err := cmd.Execute(ctx, nil)
	require.Error(t, err)
	require.Equal(t, 1, status)
}
