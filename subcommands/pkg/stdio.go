package pkg

import (
	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/charmbracelet/lipgloss"
)

var (
	checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).SetString("✓")
	crossMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).SetString("✘")
)

type eventsProcessorStdio struct {
	done chan struct{}
}

func startEventsProcessorStdio(ctx *appcontext.AppContext, quiet bool) eventsProcessorStdio {
	done := make(chan struct{})
	ep := eventsProcessorStdio{done: done}

	go func() {
		for event := range ctx.Events().Listen() {
			switch event := event.(type) {
			case events.PathError:
				ctx.GetLogger().Stderr("%x: KO %s %s: %s", event.SnapshotID[:4], crossMark, event.Pathname, event.Message)
			case events.DirectoryOK:
				if !quiet {
					ctx.GetLogger().Stdout("%x: OK %s %s", event.SnapshotID[:4], checkMark, event.Pathname)
				}
			case events.FileOK:
				if !quiet {
					ctx.GetLogger().Stdout("%x: OK %s %s", event.SnapshotID[:4], checkMark, event.Pathname)
				}
			case events.DirectoryError:
				ctx.GetLogger().Stderr("%x: KO %s %s: %s", event.SnapshotID[:4], crossMark, event.Pathname, event.Message)
			case events.FileError:
				ctx.GetLogger().Stderr("%x: KO %s %s: %s", event.SnapshotID[:4], crossMark, event.Pathname, event.Message)
			case events.Done:
				done <- struct{}{}
			default:
				//ctx.GetLogger().Warn("unknown event: %T", event)
			}
		}
	}()

	return ep
}

func (ep eventsProcessorStdio) Close() {
	<-ep.done
}
