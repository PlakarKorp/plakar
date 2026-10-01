package login

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRunUIRequestsCompletionCode: the api refuses a ui sign-in without the
// completion code opt-in, and refuses a redirect alongside it.
func TestRunUIRequestsCompletionCode(t *testing.T) {
	for _, provider := range []string{"github", "email"} {
		t.Run(provider, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/auth/login/"+provider {
					t.Errorf("unexpected path %s", r.URL.Path)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				if body["completion_code"] != true {
					t.Errorf("completion_code = %v, want true", body["completion_code"])
				}
				if _, ok := body["redirect"]; ok {
					t.Errorf("redirect must not be sent with completion_code")
				}
				w.Write([]byte(`{"URL":"https://confirm","poll_id":"4ee1ee40-1026-4621-a8e3-996712dfc414","completion_code":true}`))
			}))
			defer srv.Close()

			flow := newTestFlow(t)
			flow.baseURL = srv.URL

			res, err := flow.RunUI(provider, map[string]string{"email": "nestor@plakar.io"})
			if err != nil {
				t.Fatalf("RunUI err = %v", err)
			}
			if res.PollID != "4ee1ee40-1026-4621-a8e3-996712dfc414" {
				t.Fatalf("poll_id = %q", res.PollID)
			}
			if res.URL != "https://confirm" {
				t.Fatalf("URL = %q", res.URL)
			}
		})
	}
}

func TestRunUIMissingPollID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"URL":"https://confirm"}`))
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	if _, err := flow.RunUI("github", nil); err == nil {
		t.Fatal("RunUI accepted a response without poll_id")
	}
}

func TestRunUIRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	if _, err := flow.RunUI("email", nil); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v, want ErrRateLimited", err)
	}
}

func TestPollOnce(t *testing.T) {
	codeRequired := `{"code":"completion_code_required","error":"invalid completion code"}`
	tests := []struct {
		name       string
		status     int
		body       string
		code       string
		wantHeader string
		wantToken  string
		wantStatus PollStatus
		wantErr    error
	}{
		{name: "pending", status: http.StatusAccepted, wantStatus: PollPending},
		{name: "code required", status: http.StatusUnauthorized, body: codeRequired, wantStatus: PollCodeRequired},
		{name: "code sent", status: http.StatusOK, body: `{"token":"tok"}`, code: "M4TP-K7WQ", wantHeader: "M4TP-K7WQ", wantToken: "tok", wantStatus: PollDone},
		{name: "unknown", status: http.StatusNotFound, wantErr: ErrUnknownPollID},
		{name: "rate limited", status: http.StatusTooManyRequests, wantErr: ErrRateLimited},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("X-Completion-Code"); got != tt.wantHeader {
					t.Errorf("X-Completion-Code = %q, want %q", got, tt.wantHeader)
				}
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			flow := newTestFlow(t)
			flow.baseURL = srv.URL

			token, status, err := flow.PollOnce("abc", tt.code)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if token != tt.wantToken || status != tt.wantStatus {
				t.Fatalf("got (%q, %v), want (%q, %v)", token, status, tt.wantToken, tt.wantStatus)
			}
		})
	}
}

// TestPollOnceFatalUnauthorized: a 401 that is not about the completion code
// (a poll-secret mismatch) is not something retrying with a code can fix.
func TestPollOnceFatalUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code":"poll_secret_required"}`))
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	if _, _, err := flow.PollOnce("abc", ""); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v, want a 401 error", err)
	}
}

// TestPollOnceEscapesPollID: the poll ID comes from the browser; it must not
// be able to reach another api path.
func TestPollOnceEscapesPollID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/v1/auth/poll/..%2Faccount" {
			t.Errorf("path = %q", r.URL.EscapedPath())
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	if _, _, err := flow.PollOnce("../account", ""); err != nil {
		t.Fatalf("err = %v", err)
	}
}
