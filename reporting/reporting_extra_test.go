package reporting

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func newCtx(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.SetLogger(logging.NewLogger(bytes.NewBuffer(nil), bytes.NewBuffer(nil)))
	cookiesManager, err := cookies.NewManager(t.TempDir())
	require.NoError(t, err)
	ctx.SetCookies(cookiesManager)
	return ctx
}

func TestNullEmitterEmit(t *testing.T) {
	e := &NullEmitter{}
	require.NoError(t, e.Emit(context.Background(), &Report{}))
}

func TestHttpEmitterEmitSuccess(t *testing.T) {
	var gotAuth, gotUA, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	e := &HttpEmitter{url: srv.URL, token: "tok"}
	require.NoError(t, e.Emit(context.Background(), &Report{}))
	require.Equal(t, "Bearer tok", gotAuth)
	require.Contains(t, gotUA, "plakar/")
	require.Equal(t, "application/json", gotCT)
}

func TestHttpEmitterEmitNoToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	e := &HttpEmitter{url: srv.URL}
	require.NoError(t, e.Emit(context.Background(), &Report{}))
	require.Empty(t, gotAuth, "no token must not set an Authorization header")
}

func TestHttpEmitterEmitNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	e := &HttpEmitter{url: srv.URL}
	err := e.Emit(context.Background(), &Report{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "request failed with status")
}

func TestHttpEmitterEmitNetworkError(t *testing.T) {
	e := &HttpEmitter{url: "http://127.0.0.1:0"} // unreachable
	require.Error(t, e.Emit(context.Background(), &Report{}))
}

func TestHttpEmitterEmitBadURL(t *testing.T) {
	e := &HttpEmitter{url: "http://\x7f"} // invalid: NewRequest fails
	require.Error(t, e.Emit(context.Background(), &Report{}))
}

func TestGetEmitterNoTokenReturnsNull(t *testing.T) {
	ctx := newCtx(t)
	r := NewReporter(ctx)
	defer r.StopAndWait()

	// No auth token configured: getEmitter falls back to NullEmitter.
	em := r.getEmitter()
	_, ok := em.(*NullEmitter)
	require.True(t, ok, "expected NullEmitter, got %T", em)
}

func TestGetEmitterCachesEmitter(t *testing.T) {
	ctx := newCtx(t)
	r := NewReporter(ctx)
	defer r.StopAndWait()

	first := r.getEmitter()
	// Second call within the timeout window returns the cached emitter.
	second := r.getEmitter()
	require.Same(t, first, second)
}

func TestGetEmitterAlertingDisabledReturnsNull(t *testing.T) {
	ctx := newCtx(t)
	require.NoError(t, ctx.GetCookies().PutAuthToken("tok"))

	// Point the services connector at a server that reports alerting disabled.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"enabled":false}`))
	}))
	defer srv.Close()
	t.Setenv("PLAKAR_API_URL", srv.URL)

	r := NewReporter(ctx)
	defer r.StopAndWait()

	// getEmitter calls GetServiceStatus against api.plakar.io (the services
	// connector default endpoint), which is unreachable in tests, so the
	// service-status lookup errors and getEmitter falls back to NullEmitter.
	em := r.getEmitter()
	_, ok := em.(*NullEmitter)
	require.True(t, ok, "expected NullEmitter when alerting can't be confirmed, got %T", em)
}

func TestReportTaskStartWarnsOnDoubleStart(t *testing.T) {
	var errBuf bytes.Buffer
	ctx := appcontext.NewAppContext()
	ctx.SetLogger(logging.NewLogger(bytes.NewBuffer(nil), &errBuf))
	ctx.GetLogger().EnableInfo()
	cookiesManager, err := cookies.NewManager(t.TempDir())
	require.NoError(t, err)
	ctx.SetCookies(cookiesManager)

	r := NewReporter(ctx)
	defer r.StopAndWait()

	report := r.NewReport()
	report.SetIgnore()
	report.TaskStart("kind", "name")
	report.TaskStart("kind", "name") // second start warns
	require.Contains(t, errBuf.String(), "already in a task")
	report.TaskDone()
}

func TestReportWithRepositoryName(t *testing.T) {
	ctx := newCtx(t)
	r := NewReporter(ctx)
	defer r.StopAndWait()

	report := r.NewReport()
	report.SetIgnore()
	report.WithRepositoryName("myrepo")
	require.NotNil(t, report.Repository)
	require.Equal(t, "myrepo", report.Repository.Name)
	// A second call warns but still works.
	report.WithRepositoryName("other")

	report.TaskStart("k", "n")
	report.TaskDone()
}

func TestReportWithRepositoryAndSnapshot(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bytes.NewBuffer(nil), nil)
	cookiesManager, err := cookies.NewManager(t.TempDir())
	require.NoError(t, err)
	ctx.SetCookies(cookiesManager)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("a.txt", 0644, "a"),
	})
	defer snap.Close()

	r := NewReporter(ctx)
	defer r.StopAndWait()

	report := r.NewReport()
	report.SetIgnore()
	report.WithRepositoryName("myrepo")
	report.WithRepository(repo)
	require.NotNil(t, report.Repository.Storage)

	report.WithSnapshot(snap)
	require.NotNil(t, report.Snapshot)
	// Double snapshot warns but still works.
	report.WithSnapshot(snap)

	// WithSnapshotID loads by MAC.
	report.WithSnapshotID(snap.Header.GetIndexID())

	report.TaskStart("k", "n")
	report.TaskDone()
}

func TestReportWithSnapshotIDLoadError(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bytes.NewBuffer(nil), nil)
	cookiesManager, err := cookies.NewManager(t.TempDir())
	require.NoError(t, err)
	ctx.SetCookies(cookiesManager)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("a.txt", 0644, "a"),
	})
	defer snap.Close()

	r := NewReporter(ctx)
	defer r.StopAndWait()

	report := r.NewReport()
	report.SetIgnore()
	report.WithRepositoryName("myrepo")
	report.WithRepository(repo)

	// An unknown snapshot ID makes snapshot.Load fail; WithSnapshotID returns
	// silently without setting Snapshot.
	var missing [32]byte
	missing[0] = 0xff
	report.WithSnapshotID(missing)
	require.Nil(t, report.Snapshot)

	report.TaskStart("k", "n")
	report.TaskDone()
}

func TestReportTaskWarningAndFailed(t *testing.T) {
	ctx := newCtx(t)
	r := NewReporter(ctx)
	defer r.StopAndWait()

	warn := r.NewReport()
	warn.SetIgnore()
	warn.TaskStart("k", "n")
	warn.TaskWarning("careful: %s", "watch out")
	require.Equal(t, StatusWarning, warn.Task.Status)
	require.Equal(t, "careful: watch out", warn.Task.ErrorMessage)

	fail := r.NewReport()
	fail.SetIgnore()
	fail.TaskStart("k", "n")
	fail.TaskFailed(TaskErrorCode(42), "broke")
	require.Equal(t, StatusFailed, fail.Task.Status)
	require.Equal(t, TaskErrorCode(42), fail.Task.ErrorCode)
	require.Equal(t, "broke", fail.Task.ErrorMessage)
}

func TestProcessEmitsThroughEmitter(t *testing.T) {
	// Drive a non-ignored report all the way through Process -> getEmitter ->
	// NullEmitter.Emit (no token configured), which succeeds on the first try.
	ctx := newCtx(t)
	r := NewReporter(ctx)

	report := r.NewReport()
	report.TaskStart("k", "n")
	report.TaskDone() // Publish() -> queued -> Process() in the goroutine

	r.StopAndWait()
}

func TestProcessIgnoredReportIsNoOp(t *testing.T) {
	ctx := newCtx(t)
	r := NewReporter(ctx)
	defer r.StopAndWait()

	report := &Report{ignore: true}
	r.Process(report) // returns immediately, no emitter touched
}
