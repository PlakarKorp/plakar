package api

import (
	"bytes"
	"net/http"
	"os"
	"testing"

	"github.com/PlakarKorp/pkg"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// buildRouterWithPkg builds a real repo router and attaches a FlatBackend-backed
// pkg manager pointing at empty temp dirs (no installed plugins, no network for
// local queries).
func buildRouterWithPkg(t *testing.T, token string) *http.ServeMux {
	t.Helper()

	var bufOut, bufErr bytes.Buffer
	repo, ctx := ptesting.GenerateRepository(t, &bufOut, &bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("a.txt", 0644, "a"),
	})
	t.Cleanup(func() { snap.Close() })

	pkgDir := t.TempDir()
	cacheDir := t.TempDir()
	backend, err := pkg.NewFlatBackend(ctx.GetInner(), pkgDir, cacheDir, &pkg.FlatBackendOptions{})
	require.NoError(t, err)
	mgr, err := pkg.New(backend, &pkg.Options{})
	require.NoError(t, err)
	ctx.SetPkgManager(mgr)

	mux := http.NewServeMux()
	SetupRoutes(mux, repo, ctx, token, true)
	return mux
}

// --- api_authentication.go ---

func TestServicesLogoutNotLoggedIn(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")
	// No auth token stored -> DeleteAuthToken returns ErrNotLoggedIn, handler
	// swallows it and returns nil -> 200.
	w := doReq(t, mux, "POST", "/api/authentication/logout", "", "")
	require.Equal(t, http.StatusOK, w.Code)
}

func TestServicesLoginGithubBadBody(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")
	w := doReq(t, mux, "POST", "/api/authentication/login/github", "", "not json")
	// JSON decode failure -> wrapped error -> 500
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestServicesLoginEmailBadBody(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")
	w := doReq(t, mux, "POST", "/api/authentication/login/email", "", "not json")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- api_proxy.go: alerting config requires an auth token ---

func TestGetAlertingNoAuth(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")
	w := doReq(t, mux, "GET", "/api/proxy/v1/account/services/alerting", "", "")
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "authorization_error")
}

func TestSetAlertingNoAuth(t *testing.T) {
	mux, _ := buildRealRepoRouter(t, "")
	w := doReq(t, mux, "PUT", "/api/proxy/v1/account/services/alerting", "", `{"enabled":true,"email_report":false}`)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "authorization_error")
}

// --- api_proxy.go: integration listing via pkg manager (empty/local) ---

func TestServicesGetIntegrationBadParams(t *testing.T) {
	mux := buildRouterWithPkg(t, "")
	// offset/limit validation happens before the pkg manager Query, so these
	// 400 without touching the (network-backed) catalog.
	w := doReq(t, mux, "GET", "/api/proxy/v1/integration?offset=abc", "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = doReq(t, mux, "GET", "/api/proxy/v1/integration?limit=abc", "", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServicesGetIntegrationPathNotImplemented(t *testing.T) {
	mux := buildRouterWithPkg(t, "")
	w := doReq(t, mux, "GET", "/api/proxy/v1/integration/foo/some/path", "", "")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- api_integrations.go ---

func TestIntegrationsInstallBadBody(t *testing.T) {
	mux := buildRouterWithPkg(t, "")
	w := doReq(t, mux, "POST", "/api/integrations/install", "", "not json")
	// Handler always returns a JSON body with status 200, but marks the
	// operation as failed in the body.
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "failed")
}

func TestIntegrationsUninstallUnknown(t *testing.T) {
	mux := buildRouterWithPkg(t, "")
	w := doReq(t, mux, "DELETE", "/api/integrations/nonexistent", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	// Del is idempotent for the flat backend: removing an unknown plugin
	// succeeds. We assert the handler produced a well-formed response.
	require.Contains(t, w.Body.String(), "pkg_uninstall")
}

// --- api_integrations.go: response helpers (pure) ---

func TestIntegrationsResponseHelpers(t *testing.T) {
	resp := NewIntegrationsResponse("pkg_test")
	require.Equal(t, "pkg_test", resp.Type)
	require.Equal(t, "completed", resp.Status)
	require.Empty(t, resp.Messages)

	resp.AddMessage("hello")
	require.Len(t, resp.Messages, 1)
	require.Equal(t, "hello", resp.Messages[0].Message)
}

// --- demo mode wiring: write endpoints disabled when PLAKAR_DEMO_MODE=1 ---

func TestDemoModeDisablesWriteEndpoints(t *testing.T) {
	t.Setenv("PLAKAR_DEMO_MODE", "1")

	var bufOut, bufErr bytes.Buffer
	repo, ctx := ptesting.GenerateRepository(t, &bufOut, &bufErr, nil)
	mux := http.NewServeMux()
	SetupRoutes(mux, repo, ctx, "", true)

	// logout is a write endpoint; in demo mode it is not registered, so the
	// catch-all /api/ handler returns 404.
	w := doReq(t, mux, "POST", "/api/authentication/logout", "", "")
	require.Equal(t, http.StatusNotFound, w.Code)

	os.Unsetenv("PLAKAR_DEMO_MODE")
}
