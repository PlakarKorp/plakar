package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func postQuit(t *testing.T, mux *http.ServeMux, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest("POST", "/api/quit", nil)
	require.NoError(t, err)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestQuit(t *testing.T) {
	mux, _, snap, ctx := server(t, "secret-token")
	defer snap.Close()

	for _, c := range []struct {
		name    string
		headers map[string]string
		want    int
	}{
		{"missing token", nil, http.StatusUnauthorized},
		{"cross-site", map[string]string{
			"Authorization":  "Bearer secret-token",
			"Sec-Fetch-Site": "cross-site",
		}, http.StatusForbidden},
		{"same-site", map[string]string{
			"Authorization":  "Bearer secret-token",
			"Sec-Fetch-Site": "same-site",
		}, http.StatusForbidden},
	} {
		t.Run(c.name+" is refused", func(t *testing.T) {
			w := postQuit(t, mux, c.headers)
			require.Equal(t, c.want, w.Code, "body=%s", w.Body.String())
			require.NoError(t, ctx.Err())
		})
	}

	t.Run("same-origin with token stops", func(t *testing.T) {
		w := postQuit(t, mux, map[string]string{
			"Authorization":  "Bearer secret-token",
			"Sec-Fetch-Site": "same-origin",
		})
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
			t.Fatal("the context was not cancelled")
		}
		require.ErrorIs(t, context.Cause(ctx.Context), ErrQuit)
	})
}

func TestQuitDemoMode(t *testing.T) {
	t.Setenv("PLAKAR_DEMO_MODE", "true")
	mux, _, snap, ctx := server(t, "")
	defer snap.Close()

	w := postQuit(t, mux, nil)
	require.Equal(t, http.StatusNotFound, w.Code)
	require.NoError(t, ctx.Err())
}
