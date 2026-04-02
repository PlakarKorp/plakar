package stdio

import (
	"fmt"
	"io"
	"os"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

type Event = events.Event

var (
	checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).SetString("✓")
	crossMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).SetString("✘")
)

type stdio struct {
	ctx  *appcontext.AppContext
	repo *repository.Repository
	done chan error
}

func New(ctx *appcontext.AppContext) ui.UI {
	return &stdio{
		ctx: ctx,
	}
}

func (stdio *stdio) Stdout() io.Writer {
	return os.Stdout
}

func (stdio *stdio) Stderr() io.Writer {
	return os.Stderr
}
func (stdio *stdio) Stop() {}

func (stdio *stdio) Wait() error {
	return <-stdio.done
}

func HandleEvent(ctx *appcontext.AppContext, e *Event) {
	if ctx.Silent {
		return
	}

	if ctx.Quiet && e.Level == "info" {
		return
	}

	switch e.Type {
	case "path", "directory", "file", "symlink":
		// ignore, displayed as either success or failure

	case "path.error":
		snapshotID := e.Snapshot
		pathname := e.Data["path"].(string)
		errorMessage := e.Data["error"].(error)
		ctx.GetLogger().Stderr("%x: KO %s %s: %s", snapshotID[:4], crossMark, pathname, errorMessage)

	case "path.ok":
		snapshotID := e.Snapshot
		pathname := e.Data["path"].(string)
		ctx.GetLogger().Stdout("%x: OK %s %s", snapshotID[:4], checkMark, pathname)

	case "object", "chunk":
		// ignore, too verbose for stdio

	case "object.ok", "chunk.ok":
		// ignore, too verbose for stdio

	case "object.error", "chunk.error":
		snapshotID := e.Snapshot
		mac := e.Data["mac"].(objects.MAC)
		errorMessage := e.Data["error"].(error)
		ctx.GetLogger().Stderr("%x: KO %s object=%x: %s", snapshotID[:4], crossMark, mac, errorMessage)

	case "result":
		snapshotID := e.Snapshot
		duration := e.Data["duration"]
		rbytes := humanize.IBytes(uint64(e.Data["rbytes"].(int64)))
		wbytes := humanize.IBytes(uint64(e.Data["wbytes"].(int64)))
		errors := e.Data["errors"].(uint64)

		var errorStr string
		if errors > 0 {
			errorWord := "errors"
			if errors == 1 {
				errorWord = "error"
			}
			errorStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).SetString(fmt.Sprintf("with %d %s", errors, errorWord)).String()
		} else {
			errorStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).SetString("without errors").String()
		}
		ctx.GetLogger().Stdout("%x: %s completed %s in %s (in: %s, out: %s)",
			snapshotID[:4], e.Workflow, errorStr, duration, rbytes, wbytes)

	default:
		//fmt.Printf("%T: %s\n", e, e.Type)
	}
}

func (stdio *stdio) Run() error {
	events := stdio.ctx.Events().Listen()
	stdio.done = make(chan error, 1)

	go func() {
		defer close(stdio.done)

		for e := range events {
			HandleEvent(stdio.ctx, e)
		}
	}()

	// wait function, as before
	return nil
}

func (stdio *stdio) SetRepository(repo *repository.Repository) {
	stdio.repo = repo
}
