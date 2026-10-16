package login

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
)

// Every call of a sign-in presents plakar's User-Agent, which the API reads
// to tell which plakar a sign-in comes from.
func TestLoginSendsUserAgent(t *testing.T) {
	if !regexp.MustCompile(`^plakar/\S+ \([a-z0-9_]+/[a-z0-9_]+\)$`).MatchString(userAgent()) {
		t.Fatalf("userAgent() = %q", userAgent())
	}

	var mu sync.Mutex
	seen := map[string]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen[r.URL.Path] = r.Header.Get("User-Agent")
		mu.Unlock()
		switch r.URL.Path {
		case "/v1/auth/login/email":
			if ct := r.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("content type = %q", ct)
			}
			w.Write([]byte(`{"URL":"https://confirm","poll_id":"4ee1ee40-1026-4621-a8e3-996712dfc414"}`))
		default:
			w.Write([]byte(`{"token":"tok"}`))
		}
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL
	if tok, err := flow.Run("email", map[string]string{"email": "nestor@plakar.io"}); err != nil || tok != "tok" {
		t.Fatalf("Run = %q, %v", tok, err)
	}
	if _, err := flow.RunUI("email", map[string]string{"email": "nestor@plakar.io"}); err != nil {
		t.Fatalf("RunUI err = %v", err)
	}

	for _, path := range []string{"/v1/auth/login/email", "/v1/auth/poll/4ee1ee40-1026-4621-a8e3-996712dfc414"} {
		if got := seen[path]; got != userAgent() {
			t.Errorf("%s User-Agent = %q, want %q", path, got, userAgent())
		}
	}
}
