package journal

import (
	"fmt"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/google/uuid"
)

// State is read from another goroutine while events keep arriving; run under
// -race. Close must return a state that saw every event emitted before it.
func TestFoldStateWhileEventsArrive(t *testing.T) {
	ctx := appcontext.NewAppContext()
	defer ctx.Close()

	fold := NewFold(ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		emitter := ctx.Events().NewRepositoryEmitter(uuid.New(), "backup")
		for i := range 5000 {
			emitter.Path(fmt.Sprintf("/path-%d", i))
			emitter.PathOk(fmt.Sprintf("/path-%d", i))
		}
		emitter.Close()
	}()

	for {
		select {
		case <-done:
			state := fold.Close()
			if state.Paths.Total != 5000 || state.Paths.Ok != 5000 {
				t.Fatalf("paths = %+v, want 5000 total and ok", state.Paths)
			}
			return
		default:
			s := fold.State()
			if s.Paths.Ok > s.Paths.Total {
				t.Fatalf("ok %d > total %d", s.Paths.Ok, s.Paths.Total)
			}
		}
	}
}
