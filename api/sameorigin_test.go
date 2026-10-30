package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSameOriginMiddleware(t *testing.T) {
	const host = "localhost:1234"

	tests := []struct {
		name    string
		method  string
		headers map[string]string
		body    string
		want    int
	}{
		{"GET is not checked", "GET", map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example"}, "", http.StatusOK},
		{"preflight is not checked", "OPTIONS", map[string]string{"Origin": "https://evil.example"}, "", http.StatusOK},
		{"non-browser client", "POST", nil, "", http.StatusOK},
		{"non-browser client with json", "POST", map[string]string{"Content-Type": "application/json"}, "{}", http.StatusOK},
		{"same origin json", "POST", map[string]string{
			"Sec-Fetch-Site": "same-origin", "Origin": "http://" + host, "Content-Type": "application/json",
		}, "{}", http.StatusOK},
		{"json with charset", "PUT", map[string]string{"Content-Type": "application/json; charset=utf-8"}, "{}", http.StatusOK},
		{"same origin without body", "DELETE", map[string]string{"Sec-Fetch-Site": "same-origin", "Origin": "http://" + host}, "", http.StatusOK},
		{"direct navigation", "POST", map[string]string{"Sec-Fetch-Site": "none"}, "", http.StatusOK},
		{"cross-site fetch", "POST", map[string]string{"Sec-Fetch-Site": "cross-site"}, "", http.StatusForbidden},
		{"same-site fetch", "POST", map[string]string{"Sec-Fetch-Site": "same-site"}, "", http.StatusForbidden},
		{"foreign origin", "POST", map[string]string{"Origin": "https://evil.example"}, "", http.StatusForbidden},
		{"null origin", "DELETE", map[string]string{"Origin": "null"}, "", http.StatusForbidden},
		{"same host other port", "POST", map[string]string{"Origin": "http://localhost:9999"}, "", http.StatusForbidden},
		{"text/plain form post", "POST", map[string]string{"Content-Type": "text/plain"}, "{}", http.StatusUnsupportedMediaType},
		{"urlencoded form post", "POST", map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "a=b", http.StatusUnsupportedMediaType},
		{"body without content type", "PUT", nil, "{}", http.StatusUnsupportedMediaType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			h := SameOriginMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			}))

			req := httptest.NewRequest(tt.method, "/api/x", strings.NewReader(tt.body))
			req.Host = host
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
			require.Equal(t, tt.want == http.StatusOK, called)
		})
	}
}
