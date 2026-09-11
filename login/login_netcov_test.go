package login

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
)

// newTestFlow builds a login flow whose appCtx has an on-disk cookies Manager
// rooted in t.TempDir(), with noSpawn=true so no browser is launched.
func newTestFlow(t *testing.T) *loginFlow {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.SetCookies(cookies.NewManager(t.TempDir()))
	flow, err := NewLoginFlow(ctx, true)
	if err != nil {
		t.Fatalf("NewLoginFlow err = %v", err)
	}
	return flow
}

func TestNetPollReturnsTokenOn200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v1/auth/poll/") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"token":"tok-200"}`))
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	token, err := flow.Poll("abc", 3, time.Millisecond, func() { t.Fatal("progress should not be called") })
	if err != nil {
		t.Fatalf("Poll err = %v", err)
	}
	if token != "tok-200" {
		t.Fatalf("token = %q, want tok-200", token)
	}
}

func TestNetPollUnknownIDOn404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	_, err := flow.Poll("missing", 3, time.Millisecond, func() {})
	if err == nil || !strings.Contains(err.Error(), "unknown ID") {
		t.Fatalf("err = %v, want 'unknown ID'", err)
	}
}

func TestNetPollAcceptedThenToken(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n < 3 {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"token":"tok-eventual"}`))
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	var progress int32
	token, err := flow.Poll("p", 10, time.Millisecond, func() { atomic.AddInt32(&progress, 1) })
	if err != nil {
		t.Fatalf("Poll err = %v", err)
	}
	if token != "tok-eventual" {
		t.Fatalf("token = %q, want tok-eventual", token)
	}
	if atomic.LoadInt32(&progress) < 2 {
		t.Fatalf("progress = %d, want >=2", progress)
	}
}

func TestNetPollAcceptedExhausts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	var progress int32
	_, err := flow.Poll("p", 4, time.Millisecond, func() { atomic.AddInt32(&progress, 1) })
	if err == nil || !strings.Contains(err.Error(), "could not obtain token after") {
		t.Fatalf("err = %v, want exhaustion error", err)
	}
	if atomic.LoadInt32(&progress) != 4 {
		t.Fatalf("progress = %d, want 4", progress)
	}
}

func TestNetPollUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	_, err := flow.Poll("p", 3, time.Millisecond, func() {})
	if err == nil || !strings.Contains(err.Error(), "unexpected status code") {
		t.Fatalf("err = %v, want unexpected status code", err)
	}
}

func TestNetRunGithubSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/auth/login/github":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"URL":"https://example/login","poll_id":"gh1"}`))
		case strings.HasPrefix(r.URL.Path, "/v1/auth/poll/"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"token":"gh-token"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	token, err := flow.Run("github", map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatalf("Run github err = %v", err)
	}
	if token != "gh-token" {
		t.Fatalf("token = %q, want gh-token", token)
	}
}

func TestNetRunEmailSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/auth/login/email":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"poll_id":"em1"}`))
		case strings.HasPrefix(r.URL.Path, "/v1/auth/poll/"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"token":"em-token"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	token, err := flow.Run("email", map[string]string{"email": "a@b.c"})
	if err != nil {
		t.Fatalf("Run email err = %v", err)
	}
	if token != "em-token" {
		t.Fatalf("token = %q, want em-token", token)
	}
}

func TestNetRunNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad input"))
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	_, err := flow.Run("github", nil)
	if err == nil || !strings.Contains(err.Error(), "unexpected status code") {
		t.Fatalf("err = %v, want unexpected status code", err)
	}
}

func TestNetRunUnsupportedProvider(t *testing.T) {
	flow := newTestFlow(t)
	_, err := flow.Run("gitlab", nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported provider") {
		t.Fatalf("err = %v, want unsupported provider", err)
	}
}

func TestNetRunUIGithubSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/auth/login/github":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"URL":"https://ui/login","poll_id":"ghui"}`))
		case strings.HasPrefix(r.URL.Path, "/v1/auth/poll/"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"token":"ui-token"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	url, err := flow.RunUI("github", nil)
	if err != nil {
		t.Fatalf("RunUI github err = %v", err)
	}
	if url != "https://ui/login" {
		t.Fatalf("url = %q, want https://ui/login", url)
	}

	// The goroutine polls and stores the token via cookies; give it a moment.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if tok, _ := flow.appCtx.GetCookies().GetAuthToken(); tok == "ui-token" {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("auth token was not stored by RunUI goroutine")
}

func TestNetRunUIEmailSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/auth/login/email":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"poll_id":"emui"}`))
		case strings.HasPrefix(r.URL.Path, "/v1/auth/poll/"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"token":"ui-email-token"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	url, err := flow.RunUI("email", nil)
	if err != nil {
		t.Fatalf("RunUI email err = %v", err)
	}
	if url != "" {
		t.Fatalf("url = %q, want empty", url)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if tok, _ := flow.appCtx.GetCookies().GetAuthToken(); tok == "ui-email-token" {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("auth token was not stored by RunUI email goroutine")
}

func TestNetRunUINon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	flow := newTestFlow(t)
	flow.baseURL = srv.URL

	_, err := flow.RunUI("email", nil)
	if err == nil || !strings.Contains(err.Error(), "unexpected status code") {
		t.Fatalf("err = %v, want unexpected status code", err)
	}
}

func TestNetDeriveTokenSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/account/derive-token" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer auth-tok" {
			t.Errorf("Authorization = %q, want Bearer auth-tok", got)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"token":"derived-tok"}`))
	}))
	defer srv.Close()

	ctx := appcontext.NewAppContext()
	mgr := cookies.NewManager(t.TempDir())
	if err := mgr.PutAuthToken("auth-tok"); err != nil {
		t.Fatalf("PutAuthToken err = %v", err)
	}
	ctx.SetCookies(mgr)

	t.Setenv("PLAKAR_TOKEN", "") // ensure file token is used
	t.Setenv("PLAKAR_API_URL", srv.URL)

	token, err := DeriveToken(ctx)
	if err != nil {
		t.Fatalf("DeriveToken err = %v", err)
	}
	if token != "derived-tok" {
		t.Fatalf("token = %q, want derived-tok", token)
	}
}

func TestNetDeriveTokenNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	ctx := appcontext.NewAppContext()
	mgr := cookies.NewManager(t.TempDir())
	if err := mgr.PutAuthToken("auth-tok"); err != nil {
		t.Fatalf("PutAuthToken err = %v", err)
	}
	ctx.SetCookies(mgr)

	t.Setenv("PLAKAR_TOKEN", "")
	t.Setenv("PLAKAR_API_URL", srv.URL)

	_, err := DeriveToken(ctx)
	if err == nil || !strings.Contains(err.Error(), "request failed with status") {
		t.Fatalf("err = %v, want request failed with status", err)
	}
}

func TestNetDeriveTokenNoAuthToken(t *testing.T) {
	ctx := appcontext.NewAppContext()
	ctx.SetCookies(cookies.NewManager(t.TempDir()))

	t.Setenv("PLAKAR_TOKEN", "")

	_, err := DeriveToken(ctx)
	if err == nil {
		t.Fatal("expected error when no auth token present, got nil")
	}
}
