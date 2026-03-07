package json

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/ui"
	"github.com/google/uuid"
)

type jsonRenderer struct {
	ctx     *appcontext.AppContext
	repo    *repository.Repository
	done    chan error
	encoder *json.Encoder
}

type jsonEvent struct {
	Version    int            `json:"version"`
	Timestamp  time.Time      `json:"timestamp"`
	Repository uuid.UUID      `json:"repository"`
	Snapshot   objects.MAC    `json:"snapshot"`
	Level      string         `json:"level"`
	Workflow   string         `json:"workflow"`
	Job        uuid.UUID      `json:"job"`
	Type       string         `json:"type"`
	Data       map[string]any `json:"data,omitempty"`
}

func New(ctx *appcontext.AppContext) ui.UI {
	return &jsonRenderer{
		ctx:     ctx,
		encoder: json.NewEncoder(os.Stdout),
	}
}

func (jr *jsonRenderer) Stdout() io.Writer {
	return io.Discard
}

func (jr *jsonRenderer) Stderr() io.Writer {
	return os.Stderr
}

func (jr *jsonRenderer) SetRepository(repo *repository.Repository) {
	jr.repo = repo
}

func (jr *jsonRenderer) Wait() error {
	return <-jr.done
}

func (jr *jsonRenderer) Run() error {
	ch := jr.ctx.Events().Listen()
	jr.done = make(chan error, 1)

	go func() {
		defer close(jr.done)

		for e := range ch {
			jr.handleEvent(e)
		}
	}()

	return nil
}

func (jr *jsonRenderer) handleEvent(e *events.Event) {
	if jr.ctx.Silent {
		return
	}
	if jr.ctx.Quiet && e.Level == "info" {
		return
	}

	out := jsonEvent{
		Version:    e.Version,
		Timestamp:  e.Timestamp,
		Repository: e.Repository,
		Snapshot:   e.Snapshot,
		Level:      e.Level,
		Workflow:   e.Workflow,
		Job:        e.Job,
		Type:       e.Type,
	}
	if len(e.Data) > 0 {
		out.Data = sanitizeData(e.Data)
	}
	if err := jr.encoder.Encode(out); err != nil {
		return // stop on write errors (e.g., broken pipe)
	}
}

// sanitizeData converts values that don't serialize cleanly to JSON.
func sanitizeData(data map[string]any) map[string]any {
	out := make(map[string]any, len(data))
	for k, v := range data {
		switch val := v.(type) {
		case error:
			out[k] = val.Error()
		case time.Duration:
			out[k] = val.Milliseconds()
		default:
			out[k] = val
		}
	}
	return out
}
