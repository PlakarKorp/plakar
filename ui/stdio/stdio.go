package stdio

import (
	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

type Event = events.Event

var (
	checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).SetString("✓")
	crossMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).SetString("✘")
)

func Run(ctx *appcontext.AppContext) func() {
	events := ctx.Events().Listen()
	done := make(chan error, 1)

	go func() {
		defer close(done)

		for {
			select {
			case <-ctx.Done():
				done <- ctx.Err()
				return

			case e, ok := <-events:
				if !ok {
					done <- nil
					return
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
					ctx.GetLogger().Stderr("%x: KO %s %s: %s", snapshotID[:4], crossMark, pathname, errorMessage)

				case "path.ok", "directory.ok", "file.ok", "symlink.ok":
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

				case "snapshot.backup.result":
					snapshotID := e.Snapshot
					totalSize := humanize.IBytes(e.Data["size"].(uint64))
					duration := e.Data["duration"]
					rbytes := humanize.IBytes(uint64(e.Data["rbytes"].(int64)))
					wbytes := humanize.IBytes(uint64(e.Data["wbytes"].(int64)))
					errors := e.Data["errors"].(uint64)
					ctx.GetLogger().Stdout("%x: created snapshot of logical size %s to %s in %s with %d errors (in: %s, out: %s)",
						snapshotID[:4], totalSize, e.Data["target"].(string), duration, errors, rbytes, wbytes)

				case "snapshot.restore.result":
					snapshotID := e.Snapshot
					totalSize := humanize.IBytes(e.Data["size"].(uint64))
					duration := e.Data["duration"]
					rbytes := humanize.IBytes(uint64(e.Data["rbytes"].(int64)))
					wbytes := humanize.IBytes(uint64(e.Data["wbytes"].(int64)))
					errors := e.Data["errors"].(uint64)
					ctx.GetLogger().Stdout("%x: restored snapshot of logical size %s to %s in %s with %d errors (in: %s, out: %s)",
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
					ctx.GetLogger().Stdout("%x: check %s snapshot of logical size %s in %s with %d errors (in: %s, out: %s)",
						snapshotID[:4], success, totalSize, duration, errors, rbytes, wbytes)

				default:
					//fmt.Printf("%T: %s\n", e, e.Type)
				}

			}
		}
	}()

	// wait function, as before
	return func() {
		<-done
	}
}
