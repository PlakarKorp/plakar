package login

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// gatedServer answers the poll with 401 until the expected code arrives on the
// X-Completion-Code header, then 200. The 401 body is caller-supplied so a test
// can drive either the new `code` discriminator or the old prose fallback.
func gatedServer(t *testing.T, wantCode, unauthorizedBody string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Completion-Code") == wantCode {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"token":"tok-ok"}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(unauthorizedBody))
	}))
}

// pollWithPrompt runs the internal poll with a prompt that feeds answers in
// order and records how many times it was asked.
func pollWithPrompt(t *testing.T, flow *loginFlow, answers ...string) (string, error, *int32) {
	t.Helper()
	var calls int32
	i := 0
	prompt := func(retry bool) (string, error) {
		atomic.AddInt32(&calls, 1)
		a := answers[i]
		if i < len(answers)-1 {
			i++
		}
		return a, nil
	}
	tok, err := flow.poll("abc", 20, time.Millisecond, func() {}, prompt)
	return tok, err, &calls
}

// TestCompletionCodeGateNewServer: the api sends the stable `code`, the flow
// prompts and retries with the header, and gets the token.
func TestCompletionCodeGateNewServer(t *testing.T) {
	srv := gatedServer(t, "GOOD-CODE", `{"code":"completion_code_required","error":"invalid completion code"}`)
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	tok, err, calls := pollWithPrompt(t, flow, "GOOD-CODE")
	if err != nil {
		t.Fatalf("poll err = %v", err)
	}
	if tok != "tok-ok" {
		t.Fatalf("token = %q, want tok-ok", tok)
	}
	if *calls == 0 {
		t.Fatal("prompt was never called")
	}
}

// TestCompletionCodeGateOldServer: an api with no `code` field is still
// recognized via the prose fallback.
func TestCompletionCodeGateOldServer(t *testing.T) {
	srv := gatedServer(t, "GOOD-CODE", `{"error":"invalid completion code"}`)
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	tok, _, _ := pollWithPrompt(t, flow, "wrong", "GOOD-CODE")
	if tok != "tok-ok" {
		t.Fatalf("token = %q, want tok-ok (prose fallback + retry)", tok)
	}
}

// TestPollSecretMismatchIsFatal: a 401 that is NOT the completion-code gate
// (here a poll-secret mismatch) must not prompt — it ends the poll with an
// error.
func TestPollSecretMismatchIsFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code":"poll_secret_required","error":"invalid poll secret"}`))
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	var prompted int32
	_, err := flow.poll("abc", 3, time.Millisecond, func() {}, func(retry bool) (string, error) {
		atomic.AddInt32(&prompted, 1)
		return "irrelevant", nil
	})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v, want a 401 failure", err)
	}
	if prompted != 0 {
		t.Fatal("prompted on a poll-secret 401; that 401 is not recoverable")
	}
}

// TestUnauthorizedWithoutPromptIsFatal: the legacy poll path (promptCode nil)
// treats any 401 as fatal, unchanged.
func TestUnauthorizedWithoutPromptIsFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code":"completion_code_required","error":"invalid completion code"}`))
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	if _, err := flow.Poll("abc", 3, time.Millisecond, func() {}); err == nil {
		t.Fatal("Poll (no prompt) should fail on 401, not prompt")
	}
}
