package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func exchange(t *testing.T, mux *http.ServeMux, body string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest("POST", "/api/authentication/exchange", bytes.NewBufferString(body))
	require.NoError(t, err)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

// TestAuthenticationExchange covers the route the spawned browser uses to
// trade its one-time code for the token: it needs no bearer, hands out the
// token exactly once, and refuses anything else.
func TestAuthenticationExchange(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	mux := http.NewServeMux()
	SetupRoutes(mux, repo, ctx, "secret-token", "launch-code", true)

	require.Equal(t, http.StatusBadRequest, exchange(t, mux, `{not-json`).Code)
	require.Equal(t, http.StatusBadRequest, exchange(t, mux, `{}`).Code)
	require.Equal(t, http.StatusUnauthorized, exchange(t, mux, `{"code":"nope"}`).Code)

	w := exchange(t, mux, `{"code":"launch-code"}`)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	var resp TokenResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "secret-token", resp.Token)

	require.Equal(t, http.StatusUnauthorized, exchange(t, mux, `{"code":"launch-code"}`).Code, "the code is spent after one use")
}

func TestLaunchCodeExpires(t *testing.T) {
	code := newLaunchCode("launch-code")
	code.expires = time.Now().Add(-time.Second)
	require.False(t, code.redeem("launch-code"))
}

// Without a launch code, or without a token to hand out, the route is absent.
func TestAuthenticationExchangeAbsent(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	for name, c := range map[string]struct{ token, code string }{
		"no code":  {"secret-token", ""},
		"no token": {"", "launch-code"},
	} {
		t.Run(name, func(t *testing.T) {
			mux := http.NewServeMux()
			SetupRoutes(mux, repo, ctx, c.token, c.code, true)
			require.Equal(t, http.StatusNotFound, exchange(t, mux, `{"code":"launch-code"}`).Code)
		})
	}
}
