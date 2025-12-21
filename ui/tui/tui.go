package tui

import (
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/plakar/appcontext"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
)

type commandApp struct {
	events chan Event    // events we feed into the Bubbletea model
	done   chan struct{} // closed when Bubbletea program exits
}

var apps = map[string]func(*appcontext.AppContext, <-chan Event) tea.Model{
	"backup":  newBackupModel,
	"restore": newRestoreModel,
	"check":   newCheckModel,
}

func Run(ctx *appcontext.AppContext) func() {
	events := ctx.Events().Listen()
	done := make(chan error, 1)

	go func() {
		defer close(done)

		var app *commandApp

		for {
			select {
			case <-ctx.Done():
				if app != nil {
					close(app.events)
					<-app.done
					app = nil
				}
				done <- ctx.Err()
				return

			case e, ok := <-events:
				if !ok {
					if app != nil {
						close(app.events)
						<-app.done
						app = nil
					}

					done <- nil
					return
				}

				if app != nil {
					app.events <- *e
					if e.Type == "workflow.end" {
						close(app.events)
						<-app.done
						app = nil
					}
					continue
				}

				if e.Type == "workflow.start" {
					app = startApp(ctx, e.Data["workflow"].(string))
					if app != nil {
						app.events <- *e
						continue
					}
				}

				if ctx.Silent {
					continue
				}

				if ctx.Quiet && e.Level == "info" {
					continue
				}

				switch e.Type {
				case "path", "directory", "file", "symlink":
					// ignore, displayed as either success or failure

				case "path.error", "directory.error", "file.error", "symlink.error":
					snapshotID := e.Snapshot
					pathname := e.Data["path"].(string)
					errorMessage := e.Data["error"].(error)
					ctx.GetLogger().Stderr("TUI: %x: KO %s %s: %s", snapshotID[:4], crossMark, pathname, errorMessage)

				case "path.ok", "directory.ok", "file.ok", "symlink.ok":
					snapshotID := e.Snapshot
					pathname := e.Data["path"].(string)
					ctx.GetLogger().Stdout("TUI: %x: OK %s %s", snapshotID[:4], checkMark, pathname)

				case "object", "chunk":
					// ignore, too verbose for stdio

				case "object.ok", "chunk.ok":
					// ignore, too verbose for stdio

				case "object.error", "chunk.error":
					snapshotID := e.Snapshot
					mac := e.Data["mac"].(objects.MAC)
					errorMessage := e.Data["error"].(error)
					ctx.GetLogger().Stderr("TUI: %x: KO %s object=%x: %s", snapshotID[:4], crossMark, mac, errorMessage)

				case "snapshot.backup.result":
					snapshotID := e.Snapshot
					totalSize := humanize.IBytes(e.Data["size"].(uint64))
					duration := e.Data["duration"]
					rbytes := humanize.IBytes(uint64(e.Data["rbytes"].(int64)))
					wbytes := humanize.IBytes(uint64(e.Data["wbytes"].(int64)))
					errors := e.Data["errors"].(uint64)
					ctx.GetLogger().Stdout("TUI: %x: created snapshot of logical size %s to %s in %s with %d errors (in: %s, out: %s)",
						snapshotID[:4], totalSize, e.Data["target"].(string), duration, errors, rbytes, wbytes)

				case "snapshot.restore.result":
					snapshotID := e.Snapshot
					totalSize := humanize.IBytes(e.Data["size"].(uint64))
					duration := e.Data["duration"]
					rbytes := humanize.IBytes(uint64(e.Data["rbytes"].(int64)))
					wbytes := humanize.IBytes(uint64(e.Data["wbytes"].(int64)))
					errors := e.Data["errors"].(uint64)
					ctx.GetLogger().Stdout("TUI: %x: restored snapshot of logical size %s to %s in %s with %d errors (in: %s, out: %s)",
						snapshotID[:4], totalSize, e.Data["target"].(string), duration, errors, rbytes, wbytes)

				case "snapshot.check.result":
					snapshotID := e.Snapshot
					totalSize := humanize.IBytes(e.Data["size"].(uint64))
					duration := e.Data["duration"]
					rbytes := humanize.IBytes(uint64(e.Data["rbytes"].(int64)))
					wbytes := humanize.IBytes(uint64(e.Data["wbytes"].(int64)))
					errors := e.Data["errors"].(uint64)
					success := "succeeded"
					if errors != 0 {
						success = "failed"
					}
					ctx.GetLogger().Stdout("TUI: %x: check %s snapshot of logical size %s in %s with %d errors (in: %s, out: %s)",
						snapshotID[:4], success, totalSize, duration, errors, rbytes, wbytes)

				default:
					//fmt.Printf("%T: %s\n", e, e.Type)
				}

			}
		}
	}()

	// wait function, as before
	return func() {
		ctx.GetLogger().Warn("TUI: waiting for completion...")
		<-done
	}
}

func startApp(ctx *appcontext.AppContext, app string) *commandApp {
	events := make(chan Event, 256) // larger buffer to avoid stalls without dropping
	done := make(chan struct{})

	if modelFunc, ok := apps[app]; !ok {
		return nil
	} else {
		m := modelFunc(ctx, events)
		p := tea.NewProgram(m) // optionally: tea.WithOutput(ctx.Stdout)

		go func() {
			defer close(done)
			if _, err := p.Run(); err != nil {
				ctx.GetLogger().Stderr("backup TUI error: %v", err)
			}
		}()
	}

	return &commandApp{
		events: events,
		done:   done,
	}
}
