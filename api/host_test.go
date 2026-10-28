package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHostMiddleware(t *testing.T) {
	tests := []struct {
		name string
		addr string
		host string
		want int
	}{
		{"localhost", "localhost:1234", "localhost:1234", http.StatusOK},
		{"localhost uppercase", "localhost:1234", "LOCALHOST:1234", http.StatusOK},
		{"loopback v4", "localhost:1234", "127.0.0.1:1234", http.StatusOK},
		{"loopback v6", "localhost:1234", "[::1]:1234", http.StatusOK},
		{"loopback v6 no port", "localhost:1234", "[::1]", http.StatusOK},
		{"lan ip", "0.0.0.0:1234", "192.168.1.5:1234", http.StatusOK},
		{"configured name", "backup.lan:9000", "backup.lan:9000", http.StatusOK},
		{"rebinding name", "localhost:1234", "evil.example:1234", http.StatusForbidden},
		{"localhost suffix", "localhost:1234", "localhost.evil.example:1234", http.StatusForbidden},
		{"other name than configured", "backup.lan:9000", "other.lan:9000", http.StatusForbidden},
		{"empty host", "localhost:1234", "", http.StatusForbidden},
		{"wildcard addr does not allow names", ":1234", "evil.example:1234", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			h := HostMiddleware(tt.addr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			}))

			req := httptest.NewRequest(http.MethodGet, "/api/info", nil)
			req.Host = tt.host
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
			require.Equal(t, tt.want == http.StatusOK, called)
		})
	}
}
