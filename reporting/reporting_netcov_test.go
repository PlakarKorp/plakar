package reporting

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/stretchr/testify/require"
)

// netcovCtx builds a hermetic AppContext with a logger and a fresh cookie jar.
// Tests in this file use t.Setenv (process-global), so they must NOT call
// t.Parallel().
func netcovReportingCtx(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.SetLogger(logging.NewLogger(bytes.NewBuffer(nil), bytes.NewBuffer(nil)))
	ctx.SetCookies(cookies.NewManager(t.TempDir()))
	return ctx
}

// TestNetcovGetEmitterAlertingEnabledReturnsHttp covers the success branch of
// getEmitter that the existing tests miss: a logged-in user whose "alerting"
// service reports enabled=true gets an *HttpEmitter wired to PLAKAR_API_URL.
//
// PLAKAR_API_URL is honored by BOTH the services connector (for the alerting
// status check) and the resulting HttpEmitter (for the report POST), so a single
// mock server answers both the status GET and any report POST.
func TestNetcovGetEmitterAlertingEnabledReturnsHttp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/account/services/alerting" {
			_, _ = w.Write([]byte(`{"enabled":true}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	t.Setenv("PLAKAR_API_URL", srv.URL)

	ctx := netcovReportingCtx(t)
	require.NoError(t, ctx.GetCookies().PutAuthToken("tok"))

	r := NewReporter(ctx)
	defer r.StopAndWait()

	em := r.getEmitter()
	he, ok := em.(*HttpEmitter)
	require.True(t, ok, "expected *HttpEmitter when alerting is enabled, got %T", em)
	require.Equal(t, srv.URL, he.url)
	require.Equal(t, "tok", he.token)
}

// TestNetcovProcessEmitsToHttpEmitter drives a real report through Process ->
// getEmitter (alerting enabled) -> HttpEmitter.Emit against the mock, exercising
// the end-to-end happy path including the report POST.
func TestNetcovProcessEmitsToHttpEmitter(t *testing.T) {
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/account/services/alerting":
			_, _ = w.Write([]byte(`{"enabled":true}`))
		case r.Method == "POST":
			posts.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()
	t.Setenv("PLAKAR_API_URL", srv.URL)

	ctx := netcovReportingCtx(t)
	require.NoError(t, ctx.GetCookies().PutAuthToken("tok"))

	r := NewReporter(ctx)

	report := r.NewReport()
	report.TaskStart("backup", "nightly")
	report.TaskDone() // Publish -> queued -> Process -> Emit

	r.StopAndWait() // drains the queue, guaranteeing Process ran

	require.GreaterOrEqual(t, posts.Load(), int32(1), "expected at least one report POST")
}
