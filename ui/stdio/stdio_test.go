package stdio

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func newCtx(bufOut, bufErr *bytes.Buffer) *appcontext.AppContext {
	ctx := appcontext.NewAppContext()
	logger := logging.NewLogger(bufOut, bufErr)
	logger.EnableInfo()
	ctx.SetLogger(logger)
	return ctx
}

func mac(b byte) objects.MAC {
	var m objects.MAC
	for i := range m {
		m[i] = b
	}
	return m
}

func TestNewAndAccessors(t *testing.T) {
	ctx := newCtx(bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	defer ctx.Close()

	u := New(ctx)
	require.NotNil(t, u)
	require.Equal(t, os.Stdout, u.Stdout())
	require.Equal(t, os.Stderr, u.Stderr())
	u.Stop() // no-op, should not panic

	u.SetRepository(nil) // exercises SetRepository
}

func TestHandleEventPathOK(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	ctx := newCtx(bufOut, bufErr)
	defer ctx.Close()

	e := &events.Event{
		Type:     "path.ok",
		Snapshot: mac(0xab),
		Data:     map[string]any{"path": "/etc/passwd"},
	}
	HandleEvent(ctx, e)
	require.Contains(t, bufOut.String(), "/etc/passwd")
	require.Contains(t, bufOut.String(), "OK")
}

func TestHandleEventPathError(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	ctx := newCtx(bufOut, bufErr)
	defer ctx.Close()

	e := &events.Event{
		Type:     "path.error",
		Snapshot: mac(0x12),
		Data:     map[string]any{"path": "/bad", "error": errors.New("boom")},
	}
	HandleEvent(ctx, e)
	require.Contains(t, bufErr.String(), "/bad")
	require.Contains(t, bufErr.String(), "boom")
	require.Contains(t, bufErr.String(), "KO")
}

func TestHandleEventObjectError(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	ctx := newCtx(bufOut, bufErr)
	defer ctx.Close()

	e := &events.Event{
		Type:     "object.error",
		Snapshot: mac(0x34),
		Data:     map[string]any{"mac": mac(0x99), "error": errors.New("corrupt")},
	}
	HandleEvent(ctx, e)
	require.Contains(t, bufErr.String(), "corrupt")
	require.Contains(t, bufErr.String(), "object=")
}

func TestHandleEventResultWithErrors(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	ctx := newCtx(bufOut, bufErr)
	defer ctx.Close()

	e := &events.Event{
		Type:     "result",
		Snapshot: mac(0x56),
		Workflow: "backup",
		Data: map[string]any{
			"duration": 5 * time.Second,
			"rbytes":   int64(1024),
			"wbytes":   int64(2048),
			"errors":   uint64(2),
		},
	}
	HandleEvent(ctx, e)
	out := bufOut.String()
	require.Contains(t, out, "backup")
	require.Contains(t, out, "completed")
	require.Contains(t, out, "with 2 errors")
}

func TestHandleEventResultSingularErrorAndNoErrors(t *testing.T) {
	// singular "error"
	bufOut := bytes.NewBuffer(nil)
	ctx := newCtx(bufOut, bytes.NewBuffer(nil))
	defer ctx.Close()
	HandleEvent(ctx, &events.Event{
		Type:     "result",
		Snapshot: mac(0x01),
		Workflow: "check",
		Data: map[string]any{
			"duration": time.Second, "rbytes": int64(1), "wbytes": int64(1),
			"errors": uint64(1),
		},
	})
	require.Contains(t, bufOut.String(), "with 1 error")

	// no errors path
	bufOut2 := bytes.NewBuffer(nil)
	ctx2 := newCtx(bufOut2, bytes.NewBuffer(nil))
	defer ctx2.Close()
	HandleEvent(ctx2, &events.Event{
		Type:     "result",
		Snapshot: mac(0x02),
		Workflow: "restore",
		Data: map[string]any{
			"duration": time.Second, "rbytes": int64(1), "wbytes": int64(1),
			"errors": uint64(0),
		},
	})
	require.Contains(t, bufOut2.String(), "without errors")
}

func TestHandleEventSilentAndQuiet(t *testing.T) {
	// Silent: nothing emitted
	bufOut := bytes.NewBuffer(nil)
	ctx := newCtx(bufOut, bytes.NewBuffer(nil))
	ctx.Silent = true
	defer ctx.Close()
	HandleEvent(ctx, &events.Event{Type: "path.ok", Snapshot: mac(1), Data: map[string]any{"path": "/x"}})
	require.Empty(t, bufOut.String())

	// Quiet + info-level: suppressed
	bufOut2 := bytes.NewBuffer(nil)
	ctx2 := newCtx(bufOut2, bytes.NewBuffer(nil))
	ctx2.Quiet = true
	defer ctx2.Close()
	HandleEvent(ctx2, &events.Event{Type: "path.ok", Level: "info", Snapshot: mac(1), Data: map[string]any{"path": "/x"}})
	require.Empty(t, bufOut2.String())
}

func TestHandleEventIgnoredTypes(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	ctx := newCtx(bufOut, bufErr)
	defer ctx.Close()
	for _, typ := range []string{"path", "directory", "file", "symlink", "object", "chunk", "object.ok", "chunk.ok", "unknown.type"} {
		HandleEvent(ctx, &events.Event{Type: typ, Snapshot: mac(1)})
	}
	require.Empty(t, bufOut.String())
	require.Empty(t, bufErr.String())
}

// Run/Wait: drive an event through the bus and ensure the listener goroutine
// processes it then exits cleanly when the bus closes.
func TestRunProcessesEventsAndWait(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	ctx := newCtx(bufOut, bytes.NewBuffer(nil))

	u := New(ctx)
	require.NoError(t, u.Run())

	emitter := ctx.Events().NewSnapshotEmitter(uuid.Nil, mac(0xaa), "backup")
	emitter.PathOk("/served")

	// Give the goroutine a moment, then close the bus to drain Wait.
	time.Sleep(50 * time.Millisecond)
	ctx.Events().Close()

	require.NoError(t, u.Wait())
	require.Contains(t, bufOut.String(), "/served")
}
