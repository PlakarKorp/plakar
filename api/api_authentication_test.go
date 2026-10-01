package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLoginRejectsBadInput covers the checks the login endpoints make before
// reaching the auth api: a bad request is a 400, never a 500.
func TestLoginRejectsBadInput(t *testing.T) {
	mux, _, snap, _ := server(t, "")
	defer snap.Close()

	tests := []struct {
		name string
		path string
		body string
	}{
		{"email missing", "/api/authentication/login/email", `{}`},
		{"email invalid", "/api/authentication/login/email", `{"email":"nestor"}`},
		{"email bad json", "/api/authentication/login/email", `{not-json`},
		{"poll bad json", "/api/authentication/login/poll", `{not-json`},
		{"poll missing id", "/api/authentication/login/poll", `{"completion_code":"M4TP-K7WQ"}`},
		{"poll non-uuid id", "/api/authentication/login/poll", `{"poll_id":"../account/me"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", tt.path, bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
		})
	}
}
