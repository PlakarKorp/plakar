package stdio

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/google/uuid"
)

// newCtxWithBufferedLogger returns a context whose logger writes to the given
// buffers — so tests can inspect what HandleEvent emits without touching
// os.Stdout / os.Stderr.
func newCtxWithBufferedLogger(t *testing.T, out, errBuf *bytes.Buffer) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.SetLogger(logging.NewLogger(out, errBuf))
	return ctx
}

func TestNewReturnsNonNilUI(t *testing.T) {
	ctx := appcontext.NewAppContext()
	u := New(ctx)
	if u == nil {
		t.Fatal("New returned nil")
	}
}

func TestStdoutStderrAccessors(t *testing.T) {
	ctx := appcontext.NewAppContext()
	u := New(ctx)
	if u.Stdout() != os.Stdout {
		t.Fatalf("Stdout = %v, want os.Stdout", u.Stdout())
	}
	if u.Stderr() != os.Stderr {
		t.Fatalf("Stderr = %v, want os.Stderr", u.Stderr())
	}
}

func TestStopAndSetRepositoryNoOps(t *testing.T) {
	ctx := appcontext.NewAppContext()
	u := New(ctx)
	u.Stop()             // no-op
	u.SetRepository(nil) // no-op
}

func TestRunDrainsEventsAndExitsOnBusClose(t *testing.T) {
	var out, errBuf bytes.Buffer
	ctx := newCtxWithBufferedLogger(t, &out, &errBuf)
	u := New(ctx)
	if err := u.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Publish at least one event so the goroutine loop body runs and dispatches
	// through HandleEvent before the bus is closed.
	emitter := ctx.Events().NewRepositoryEmitter(uuid.Nil, "test")
	emitter.PathOk("/x")

	ctx.Events().Close()

	done := make(chan error, 1)
	go func() { done <- u.Wait() }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return after bus close")
	}
}

func TestHandleEvent_Silent(t *testing.T) {
	var out, errBuf bytes.Buffer
	ctx := newCtxWithBufferedLogger(t, &out, &errBuf)
	ctx.Silent = true

	HandleEvent(ctx, &Event{Level: "info", Type: "path.ok", Data: map[string]any{"path": "/x"}})

	if out.Len() != 0 || errBuf.Len() != 0 {
		t.Fatalf("Silent should suppress all output, got out=%q err=%q", out.String(), errBuf.String())
	}
}

func TestHandleEvent_QuietSuppressesInfoOnly(t *testing.T) {
	var out, errBuf bytes.Buffer
	ctx := newCtxWithBufferedLogger(t, &out, &errBuf)
	ctx.Quiet = true

	HandleEvent(ctx, &Event{Level: "info", Type: "path.ok", Data: map[string]any{"path": "/x"}})
	if out.Len() != 0 {
		t.Fatalf("Quiet+info should suppress, got %q", out.String())
	}

	// error-level still emitted
	HandleEvent(ctx, &Event{
		Level: "error", Type: "path.error",
		Data: map[string]any{"path": "/y", "error": errors.New("boom")},
	})
	if errBuf.Len() == 0 {
		t.Fatal("Quiet should not suppress error-level events")
	}
}

func TestHandleEvent_PathOk(t *testing.T) {
	var out, errBuf bytes.Buffer
	ctx := newCtxWithBufferedLogger(t, &out, &errBuf)

	HandleEvent(ctx, &Event{
		Level: "info", Type: "path.ok",
		Data: map[string]any{"path": "/var/log/messages"},
	})
	if out.Len() == 0 {
		t.Fatal("path.ok should write to stdout")
	}
	if !bytes.Contains(out.Bytes(), []byte("/var/log/messages")) {
		t.Fatalf("expected path in stdout, got %q", out.String())
	}
}

func TestHandleEvent_PathError(t *testing.T) {
	var out, errBuf bytes.Buffer
	ctx := newCtxWithBufferedLogger(t, &out, &errBuf)

	HandleEvent(ctx, &Event{
		Level: "error", Type: "path.error",
		Data: map[string]any{"path": "/oops", "error": errors.New("denied")},
	})
	if errBuf.Len() == 0 {
		t.Fatal("path.error should write to stderr")
	}
	if !bytes.Contains(errBuf.Bytes(), []byte("/oops")) ||
		!bytes.Contains(errBuf.Bytes(), []byte("denied")) {
		t.Fatalf("expected path+error in stderr, got %q", errBuf.String())
	}
}

func TestHandleEvent_ObjectError(t *testing.T) {
	var out, errBuf bytes.Buffer
	ctx := newCtxWithBufferedLogger(t, &out, &errBuf)

	HandleEvent(ctx, &Event{
		Level: "error", Type: "object.error",
		Data: map[string]any{"mac": objects.MAC{0xde, 0xad}, "error": errors.New("missing")},
	})
	if errBuf.Len() == 0 {
		t.Fatal("object.error should write to stderr")
	}
	if !bytes.Contains(errBuf.Bytes(), []byte("missing")) {
		t.Fatalf("expected error message in stderr, got %q", errBuf.String())
	}
}

func TestHandleEvent_ChunkError(t *testing.T) {
	var out, errBuf bytes.Buffer
	ctx := newCtxWithBufferedLogger(t, &out, &errBuf)

	HandleEvent(ctx, &Event{
		Level: "error", Type: "chunk.error",
		Data: map[string]any{"mac": objects.MAC{0xbe, 0xef}, "error": errors.New("corrupted")},
	})
	if errBuf.Len() == 0 {
		t.Fatal("chunk.error should write to stderr")
	}
}

func TestHandleEvent_ResultNoErrors(t *testing.T) {
	var out, errBuf bytes.Buffer
	ctx := newCtxWithBufferedLogger(t, &out, &errBuf)

	HandleEvent(ctx, &Event{
		Level: "info", Type: "result", Workflow: "backup",
		Data: map[string]any{
			"duration": 2 * time.Second,
			"rbytes":   int64(1024),
			"wbytes":   int64(512),
			"errors":   uint64(0),
		},
	})
	if !bytes.Contains(out.Bytes(), []byte("backup")) {
		t.Fatalf("expected workflow name in stdout, got %q", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("without errors")) {
		t.Fatalf("expected 'without errors' marker in stdout, got %q", out.String())
	}
}

func TestHandleEvent_ResultWithErrorsSingular(t *testing.T) {
	var out, errBuf bytes.Buffer
	ctx := newCtxWithBufferedLogger(t, &out, &errBuf)

	HandleEvent(ctx, &Event{
		Level: "info", Type: "result", Workflow: "backup",
		Data: map[string]any{
			"duration": time.Second,
			"rbytes":   int64(100),
			"wbytes":   int64(50),
			"errors":   uint64(1),
		},
	})
	if !bytes.Contains(out.Bytes(), []byte("1 error")) ||
		bytes.Contains(out.Bytes(), []byte("1 errors")) {
		t.Fatalf("expected singular 'error' wording, got %q", out.String())
	}
}

func TestHandleEvent_ResultWithErrorsPlural(t *testing.T) {
	var out, errBuf bytes.Buffer
	ctx := newCtxWithBufferedLogger(t, &out, &errBuf)

	HandleEvent(ctx, &Event{
		Level: "info", Type: "result", Workflow: "backup",
		Data: map[string]any{
			"duration": time.Second,
			"rbytes":   int64(100),
			"wbytes":   int64(50),
			"errors":   uint64(3),
		},
	})
	if !bytes.Contains(out.Bytes(), []byte("3 errors")) {
		t.Fatalf("expected '3 errors' wording, got %q", out.String())
	}
}

func TestHandleEvent_IgnoredTypes(t *testing.T) {
	// These types are explicitly ignored — branch is reached but produces no output.
	var out, errBuf bytes.Buffer
	ctx := newCtxWithBufferedLogger(t, &out, &errBuf)

	for _, typ := range []string{"path", "directory", "file", "symlink", "object", "chunk", "object.ok", "chunk.ok"} {
		HandleEvent(ctx, &Event{Level: "info", Type: typ})
	}
	if out.Len() != 0 || errBuf.Len() != 0 {
		t.Fatalf("ignored types must not emit, got out=%q err=%q", out.String(), errBuf.String())
	}
}

func TestHandleEvent_UnknownTypeIsNoOp(t *testing.T) {
	var out, errBuf bytes.Buffer
	ctx := newCtxWithBufferedLogger(t, &out, &errBuf)

	HandleEvent(ctx, &Event{Level: "info", Type: "totally.unknown.type"})

	if out.Len() != 0 || errBuf.Len() != 0 {
		t.Fatalf("unknown type must hit default and be a no-op, got out=%q err=%q", out.String(), errBuf.String())
	}
}


func TestHandleEvent_PathsAreSanitized(t *testing.T) {
	var out, errBuf bytes.Buffer
	ctx := newCtxWithBufferedLogger(t, &out, &errBuf)

	evil := "/a\x1b[2K\x1b]0;pwned\x07b"
	snap := [32]byte{}

	HandleEvent(ctx, &Event{Type: "path.ok", Snapshot: snap, Data: map[string]any{"path": evil}})
	HandleEvent(ctx, &Event{Level: "error", Type: "path.error", Snapshot: snap,
		Data: map[string]any{"path": evil, "error": errors.New("bad " + evil)}})

	for name, b := range map[string]*bytes.Buffer{"stdout": &out, "stderr": &errBuf} {
		if b.Len() == 0 {
			t.Fatalf("%s: no output", name)
		}
		if bytes.ContainsAny(b.Bytes(), "\x1b\x07") {
			t.Fatalf("%s contains raw control bytes: %q", name, b.String())
		}
	}
}
