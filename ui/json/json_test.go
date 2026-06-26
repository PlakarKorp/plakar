package json

import (
	"bytes"
	enc "encoding/json"
	"errors"
	"io"
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

func newCtx() *appcontext.AppContext {
	ctx := appcontext.NewAppContext()
	logger := logging.NewLogger(bytes.NewBuffer(nil), bytes.NewBuffer(nil))
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
	ctx := newCtx()
	defer ctx.Close()

	u := New(ctx)
	require.NotNil(t, u)
	require.Equal(t, io.Discard, u.Stdout())
	require.Equal(t, os.Stderr, u.Stderr())
	u.Stop()
	u.SetRepository(nil)
}

func TestSanitizeData(t *testing.T) {
	in := map[string]any{
		"err":      errors.New("boom"),
		"duration": 2 * time.Second,
		"str":      "hello",
		"num":      42,
	}
	out := sanitizeData(in)
	require.Equal(t, "boom", out["err"])
	require.Equal(t, int64(2000), out["duration"])
	require.Equal(t, "hello", out["str"])
	require.Equal(t, 42, out["num"])
}

func TestHandleEventEncodesJSON(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	jr := &jsonRenderer{
		ctx:     newCtx(),
		encoder: enc.NewEncoder(buf),
	}
	defer jr.ctx.Close()

	e := &events.Event{
		Version:    1,
		Timestamp:  time.Unix(0, 0).UTC(),
		Repository: uuid.Nil,
		Snapshot:   mac(0xab),
		Level:      "warn",
		Workflow:   "backup",
		Type:       "path.error",
		Data:       map[string]any{"path": "/x", "error": errors.New("nope")},
	}
	jr.handleEvent(e)

	var decoded jsonEvent
	require.NoError(t, enc.Unmarshal(buf.Bytes(), &decoded))
	require.Equal(t, "path.error", decoded.Type)
	require.Equal(t, "backup", decoded.Workflow)
	require.Equal(t, "warn", decoded.Level)
	require.Equal(t, "/x", decoded.Data["path"])
	require.Equal(t, "nope", decoded.Data["error"])
}

func TestHandleEventEmptyData(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	jr := &jsonRenderer{ctx: newCtx(), encoder: enc.NewEncoder(buf)}
	defer jr.ctx.Close()

	jr.handleEvent(&events.Event{Type: "result", Snapshot: mac(1)})
	require.Contains(t, buf.String(), `"type":"result"`)
	// no data key when empty
	require.NotContains(t, buf.String(), `"data"`)
}

func TestHandleEventSilentAndQuiet(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	ctx := newCtx()
	ctx.Silent = true
	jr := &jsonRenderer{ctx: ctx, encoder: enc.NewEncoder(buf)}
	defer ctx.Close()
	jr.handleEvent(&events.Event{Type: "path.ok", Snapshot: mac(1)})
	require.Empty(t, buf.String())

	buf2 := bytes.NewBuffer(nil)
	ctx2 := newCtx()
	ctx2.Quiet = true
	jr2 := &jsonRenderer{ctx: ctx2, encoder: enc.NewEncoder(buf2)}
	defer ctx2.Close()
	jr2.handleEvent(&events.Event{Type: "path.ok", Level: "info", Snapshot: mac(1)})
	require.Empty(t, buf2.String())
}

// Run/Wait drive an event through the bus.
func TestRunAndWait(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	ctx := newCtx()
	jr := &jsonRenderer{ctx: ctx, encoder: enc.NewEncoder(buf)}

	require.NoError(t, jr.Run())

	emitter := ctx.Events().NewSnapshotEmitter(uuid.Nil, mac(0xcd), "backup")
	emitter.PathError("/served", errors.New("x"))

	time.Sleep(50 * time.Millisecond)
	ctx.Events().Close()

	require.NoError(t, jr.Wait())
	require.Contains(t, buf.String(), "/served")
}
