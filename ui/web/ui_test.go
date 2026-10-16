package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorsMiddleware_SetsHeadersOnGET(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = io.WriteString(w, "ok")
	})

	handler := corsMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler should have been called for GET")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Allow-Origin = %q, want *", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("Allow-Methods should be set")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("Allow-Headers should be set")
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q, want ok", rec.Body.String())
	}
}

func TestCorsMiddleware_OptionsShortCircuits(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := corsMiddleware(next)

	req := httptest.NewRequest(http.MethodOptions, "/anything", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("next handler must NOT be called for preflight OPTIONS")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Allow-Origin = %q, want *", got)
	}
}
