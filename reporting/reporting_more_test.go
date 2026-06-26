package reporting

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestNullEmitter(t *testing.T) {
	e := &NullEmitter{}
	require.NoError(t, e.Emit(context.Background(), &Report{}))
}

func TestHttpEmitterSuccess(t *testing.T) {
	var gotBody []byte
	var gotAuth, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = readAll(r)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	e := &HttpEmitter{url: srv.URL, token: "secret"}
	report := &Report{
		Task: &ReportTask{Type: "backup", Name: "n", Status: StatusOK},
	}
	require.NoError(t, e.Emit(context.Background(), report))

	require.Equal(t, "Bearer secret", gotAuth)
	require.Equal(t, "application/json", gotCT)

	// body must be the JSON-serialized report
	var decoded Report
	require.NoError(t, json.Unmarshal(gotBody, &decoded))
	require.NotNil(t, decoded.Task)
	require.Equal(t, "backup", decoded.Task.Type)
	require.Equal(t, StatusOK, decoded.Task.Status)
}

func TestHttpEmitterNoToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Empty(t, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	e := &HttpEmitter{url: srv.URL}
	require.NoError(t, e.Emit(context.Background(), &Report{}))
}

func TestHttpEmitterBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	e := &HttpEmitter{url: srv.URL}
	err := e.Emit(context.Background(), &Report{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "request failed with status")
}

func TestHttpEmitterBadURL(t *testing.T) {
	e := &HttpEmitter{url: "http://\x7f"}
	err := e.Emit(context.Background(), &Report{})
	require.Error(t, err)
}

func TestHttpEmitterConnRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	e := &HttpEmitter{url: url}
	err := e.Emit(context.Background(), &Report{})
	require.Error(t, err)
}

func TestReportTaskDone(t *testing.T) {
	ctx := newReporterCtx(t)
	reporter := NewReporter(ctx)
	defer reporter.StopAndWait()

	r := reporter.NewReport()
	r.SetIgnore() // do not actually emit
	r.TaskStart("backup", "mybackup")
	require.NotNil(t, r.Task)
	require.Equal(t, "backup", r.Task.Type)
	require.Equal(t, "mybackup", r.Task.Name)

	r.TaskDone()
	require.Equal(t, StatusOK, r.Task.Status)
	require.True(t, r.Task.Duration >= 0)
}

func TestReportTaskWarning(t *testing.T) {
	ctx := newReporterCtx(t)
	reporter := NewReporter(ctx)
	defer reporter.StopAndWait()

	r := reporter.NewReport()
	r.SetIgnore()
	r.TaskStart("check", "c")
	r.TaskWarning("careful: %d", 7)
	require.Equal(t, StatusWarning, r.Task.Status)
	require.Equal(t, "careful: 7", r.Task.ErrorMessage)
}

func TestReportTaskFailed(t *testing.T) {
	ctx := newReporterCtx(t)
	reporter := NewReporter(ctx)
	defer reporter.StopAndWait()

	r := reporter.NewReport()
	r.SetIgnore()
	r.TaskStart("restore", "r")
	r.TaskFailed(TaskErrorCode(42), "boom")
	require.Equal(t, StatusFailed, r.Task.Status)
	require.Equal(t, TaskErrorCode(42), r.Task.ErrorCode)
	require.Equal(t, "boom", r.Task.ErrorMessage)
}

func TestReportTaskStartTwiceWarns(t *testing.T) {
	ctx := newReporterCtx(t)
	reporter := NewReporter(ctx)
	defer reporter.StopAndWait()

	r := reporter.NewReport()
	r.SetIgnore()
	r.TaskStart("a", "1")
	r.TaskStart("b", "2") // logs a warning, overwrites
	require.Equal(t, "b", r.Task.Type)
	r.TaskDone()
}

func TestReportWithRepositoryName(t *testing.T) {
	ctx := newReporterCtx(t)
	reporter := NewReporter(ctx)
	defer reporter.StopAndWait()

	r := reporter.NewReport()
	r.SetIgnore()
	r.WithRepositoryName("myrepo")
	require.NotNil(t, r.Repository)
	require.Equal(t, "myrepo", r.Repository.Name)

	// second call warns but overwrites
	r.WithRepositoryName("other")
	require.Equal(t, "other", r.Repository.Name)

	// Every NewReport must be published, otherwise StopAndWait blocks.
	r.TaskStart("backup", "n")
	r.TaskDone()
}

func TestReportWithRepositoryAndSnapshot(t *testing.T) {
	var bufOut, bufErr bytes.Buffer
	repo, _ := ptesting.GenerateRepository(t, &bufOut, &bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("d"),
		ptesting.NewMockFile("d/f.txt", 0644, "hello"),
	})
	defer snap.Close()

	ctx := newReporterCtx(t)
	reporter := NewReporter(ctx)
	defer reporter.StopAndWait()

	r := reporter.NewReport()
	r.SetIgnore()
	r.WithRepositoryName("repo")
	r.WithRepository(repo)
	require.NotNil(t, r.Repository.Storage)

	r.WithSnapshot(snap)
	require.NotNil(t, r.Snapshot)
	require.Equal(t, snap.Header.Identifier, r.Snapshot.Header.Identifier)
	r.TaskStart("backup", "n")
	r.TaskDone()

	// WithSnapshotID loads from the repo and populates Snapshot
	r2 := reporter.NewReport()
	r2.SetIgnore()
	r2.WithRepositoryName("repo")
	r2.WithRepository(repo)
	r2.WithSnapshotID(snap.Header.GetIndexID())
	require.NotNil(t, r2.Snapshot)
	r2.TaskStart("backup", "n2")
	r2.TaskDone()
}

func TestReportWithSnapshotIDLoadError(t *testing.T) {
	var bufOut, bufErr bytes.Buffer
	repo, _ := ptesting.GenerateRepository(t, &bufOut, &bufErr, nil)

	ctx := newReporterCtx(t)
	reporter := NewReporter(ctx)
	defer reporter.StopAndWait()

	r := reporter.NewReport()
	r.SetIgnore()
	r.WithRepositoryName("repo")
	r.WithRepository(repo)

	// A snapshot id that does not exist -> load fails, Snapshot stays nil.
	var bogus [32]byte
	bogus[0] = 0xab
	r.WithSnapshotID(bogus)
	require.Nil(t, r.Snapshot)
	r.TaskStart("backup", "n")
	r.TaskDone()
}

// getEmitter: with no auth token the emitter must be the NullEmitter.
func TestGetEmitterNullWhenNoToken(t *testing.T) {
	ctx := newReporterCtx(t)
	reporter := NewReporter(ctx)
	defer reporter.StopAndWait()

	e := reporter.getEmitter()
	_, ok := e.(*NullEmitter)
	require.True(t, ok)

	// cached path: second call returns the same emitter while not timed out.
	reporter.emitter_timeout = time.Now().Add(time.Minute)
	e2 := reporter.getEmitter()
	require.Same(t, e, e2)
}

// Process with an ignored report returns immediately (no emit attempt).
func TestProcessIgnored(t *testing.T) {
	ctx := newReporterCtx(t)
	reporter := NewReporter(ctx)
	defer reporter.StopAndWait()

	reporter.Process(&Report{ignore: true})
}

// Full publish path through the reporter goroutine using a NullEmitter.
func TestPublishThroughReporter(t *testing.T) {
	ctx := newReporterCtx(t)
	reporter := NewReporter(ctx)

	r := reporter.NewReport()
	r.TaskStart("backup", "b")
	r.TaskDone() // publishes -> goroutine processes via NullEmitter

	reporter.StopAndWait()
}
