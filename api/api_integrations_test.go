package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	"github.com/stretchr/testify/require"
)

// TestIntegrations drives the integration install/uninstall endpoints against a
// single fixture server.
func TestIntegrations(t *testing.T) {
	mux, _, snap, ctx := server(t, "")
	defer snap.Close()

	t.Run("install bad body", func(t *testing.T) {
		// Malformed body fails the JSON decode before any package manager call.
		// The handler still encodes a (failed) IntegrationsResponse with 200.
		req, _ := http.NewRequest("POST", "/api/integrations/install", bytes.NewBufferString("{not-json"))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		var resp IntegrationsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, "pkg_install", resp.Type)
		require.Equal(t, "failed", resp.Status)
		require.NotEmpty(t, resp.Messages)
	})

	t.Run("uninstall", func(t *testing.T) {
		attachPkgManager(t, ctx)

		req, _ := http.NewRequest("DELETE", "/api/integrations/some-plugin", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		var resp IntegrationsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, "pkg_uninstall", resp.Type)
		require.NotEmpty(t, resp.Messages)
	})

	t.Run("response helpers", func(t *testing.T) {
		resp := NewIntegrationsResponse("pkg_install")
		require.Equal(t, "pkg_install", resp.Type)
		require.Equal(t, "completed", resp.Status)
		require.False(t, resp.StartedAt.IsZero())
		require.Empty(t, resp.Messages)

		resp.AddMessage("first")
		resp.AddMessage("second")
		require.Len(t, resp.Messages, 2)
		require.Equal(t, "first", resp.Messages[0].Message)
		require.Equal(t, "second", resp.Messages[1].Message)
		require.False(t, resp.Messages[0].Date.IsZero())

		// The struct round-trips through JSON (the handlers encode it to the wire).
		b, err := json.Marshal(resp)
		require.NoError(t, err)
		require.Contains(t, string(b), "pkg_install")
		require.Contains(t, string(b), "first")
	})
}
