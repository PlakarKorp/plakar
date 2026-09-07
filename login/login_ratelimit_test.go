package login

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunWrapsErrRateLimitedFromStatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	_, err := flow.Run("email", map[string]string{"email": "user@example.com"})
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v, want wrapping ErrRateLimited", err)
	}
}

func TestRunOtherErrorIsNotRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	_, err := flow.Run("email", map[string]string{"email": "user@example.com"})
	if err == nil {
		t.Fatal("err = nil, want a non-nil error")
	}
	if errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v, unexpectedly wraps ErrRateLimited", err)
	}
}
