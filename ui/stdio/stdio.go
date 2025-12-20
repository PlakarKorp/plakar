package stdio

import (
	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/charmbracelet/lipgloss"
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

				//	if ctx.Silent {
				// even if silent, we still need to drive TUIs
				// so only skip printing/logging here
				//	}

				switch e.Type {
				case "snapshot.backup.path.error", "snapshot.backup.directory.error", "snapshot.backup.file.error":
					snapshotID := e.Data["snapshot_id"].([]byte)
					pathname := e.Data["path"].(string)
					errorMessage := e.Data["error"].(string)
					ctx.GetLogger().Stderr("%x: KO %s %s: %s", snapshotID[:4], crossMark, pathname, errorMessage)

				case "snapshot.backup.directory.ok", "snapshot.backup.file.ok":
					snapshotID := e.Data["snapshot_id"].([]byte)
					pathname := e.Data["path"].(string)
					ctx.GetLogger().Stdout("%x: OK %s %s", snapshotID[:4], checkMark, pathname)

				case "snapshot.backup.commit.done":
					snapshotID := e.Data["snapshot_id"].([]byte)
					ctx.GetLogger().Stdout("%x: created unsigned snapshot", snapshotID[:4])

				// CHECK / RESTORE etc -> keep your old cases here if you want
				case "snapshot.check.directory.ok", "snapshot.check.file.ok":
					snapshotID := e.Data["snapshot_id"].(objects.MAC)
					pathname := e.Data["path"].(string)
					ctx.GetLogger().Stdout("%x: OK %s %s", snapshotID[:4], checkMark, pathname)

				case "snapshot.restore.path.error", "snapshot.restore.directory.error", "snapshot.restore.file.error", "snapshot.restore.symlink.error":
					snapshotID := e.Data["snapshot_id"].(objects.MAC)
					pathname := e.Data["path"].(string)
					errorMessage := e.Data["error"].(string)
					ctx.GetLogger().Stderr("%x: KO %s %s: %s", snapshotID[:4], crossMark, pathname, errorMessage)

				case "snapshot.restore.directory.ok", "snapshot.restore.file.ok", "snapshot.restore.symlink.ok":
					snapshotID := e.Data["snapshot_id"].(objects.MAC)
					pathname := e.Data["path"].(string)
					ctx.GetLogger().Stdout("%x: OK %s %s", snapshotID[:4], checkMark, pathname)

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
