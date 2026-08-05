package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestProxyAlertingGetNoAuth covers the no-auth-token 401 branch of
// servicesGetAlertingServiceConfiguration (no network access
// required).
func TestProxyAlertingGetNoAuth(t *testing.T) {
	mux, _, snap, _ := server(t, "")
	defer snap.Close()

	w := get(t, mux, "/api/proxy/v1/account/services/alerting")
	require.Equal(t, http.StatusUnauthorized, w.Code, "body=%s", w.Body.String())
	require.Contains(t, w.Body.String(), "authorization_error")
}

// TestAlertingSetNoAuth covers the no-auth-token 401 branch of
// servicesSetAlertingServiceConfiguration.
func TestAlertingSetNoAuth(t *testing.T) {
	mux, _, snap, _ := server(t, "")
	defer snap.Close()

	req, err := http.NewRequest("PUT", "/api/proxy/v1/account/services/alerting", bytes.NewBufferString(`{"enabled":true}`))
	require.NoError(t, err)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, "body=%s", w.Body.String())
}

func TestProxyAlertingGetUnauthorized(t *testing.T) {
	mux, _, snap, _ := server(t, "")
	defer snap.Close()

	// No auth token in the cookie jar -> handler returns a 401 JSON body.
	w := get(t, mux, "/api/proxy/v1/account/services/alerting")
	require.Equal(t, http.StatusUnauthorized, w.Code, "body=%s", w.Body.String())

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "authorization_error", resp["error"])
}

func TestProxyAlertingSetUnauthorized(t *testing.T) {
	mux, _, snap, _ := server(t, "")
	defer snap.Close()

	body := `{"enabled":true,"email_report":true}`
	req, _ := http.NewRequest("PUT", "/api/proxy/v1/account/services/alerting", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, "body=%s", w.Body.String())

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "authorization_error", resp["error"])
}

func TestProxyIntegrationPathNotImplemented(t *testing.T) {
	mux, _, snap, _ := server(t, "")
	defer snap.Close()

	w := get(t, mux, "/api/proxy/v1/integration/some-id/some/path")
	// Returns a plain error -> mapped to 500 by handleError.
	require.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
}

// TestServicesProxyBadEndpoint points the proxy at an unparsable
// endpoint URL, exercising the `targetBase, err := url.Parse(...); if
// err != nil` branch of servicesProxy.
func TestProxyBadEndpoint(t *testing.T) {
	t.Setenv("PLAKAR_SERVICE_ENDPOINT", "://this is not a url")
	mux, _, snap, _ := server(t, "")
	defer snap.Close()

	w := get(t, mux, "/api/proxy/v1/account/me")
	require.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
}

// TestServicesProxyUnreachable points the proxy at a closed local port so
// the outbound http.DefaultClient.Do fails with connection-refused, exercising
// the `resp, err := http.DefaultClient.Do(req); if err != nil` branch. Hermetic:
// nothing listens on 127.0.0.1:0-equivalent unreachable port.
func TestProxyUnreachable(t *testing.T) {
	t.Setenv("PLAKAR_SERVICE_ENDPOINT", "http://127.0.0.1:1")
	mux, _, snap, _ := server(t, "")
	defer snap.Close()

	w := get(t, mux, "/api/proxy/v1/account/me")
	require.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
}
