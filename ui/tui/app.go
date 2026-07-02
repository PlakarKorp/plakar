package tui

import (
	"fmt"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
)

var applications = map[string]func(*appcontext.AppContext, *Application, *repository.Repository) tea.Model{
	"import": newGenericModel,
	"export": newGenericModel,
}

// displayNames maps a workflow key to the label shown to the user. plakar has
// no standalone "export" command — the export workflow is only ever driven by
// `restore` — so present it as "restore".
var displayNames = map[string]string{
	"import": "backup",
	"export": "restore",
}

func displayName(workflow string) string {
	if n, ok := displayNames[workflow]; ok {
		return n
	}
	return workflow
}

type Application struct {
	ctx   *appcontext.AppContext
	job   uuid.UUID
	name  string
	state *State
	done  chan struct{} // closed when Bubbletea program exits
	prog  *tea.Program
	err   error
}

// ioScopeStat is the latest sampled read/write figures for one iostat scope.
// Totals are cumulative bytes. Rates are wall-clock throughput (bytes/sec)
// derived from the delta between consecutive sampler ticks — the sampler's own
// "overall" is computed over active (in-read) time, which is near-zero when
// data comes from the OS page cache and yields absurd rates, so we don't use
// it for display. sampledAt is the wall-clock time of this snapshot, used to
// compute the next delta.
type ioScopeStat struct {
	readTotal  int64
	writeTotal int64
	readRate   float64
	writeRate  float64
	sampledAt  time.Time
}

type State struct {
	startTime  time.Time
	snapshotID string

	timerBegin time.Time
	timerDone  bool

	phase  string
	detail string

	gotSummary       bool
	summaryPath      uint64
	summaryFile      uint64
	summaryDirectory uint64
	summarySymlink   uint64
	summaryXattr     uint64
	summarySize      uint64

	// counts (event-driven, no per-path memory)
	countPath       uint64
	countPathOk     uint64
	countPathError  uint64
	countPathCached uint64

	countDir       uint64
	countDirOk     uint64
	countDirError  uint64
	countDirCached uint64

	countFile       uint64
	countFileOk     uint64
	countFileError  uint64
	countFileCached uint64

	countXattr       uint64
	countXattrOk     uint64
	countXattrError  uint64
	countXattrCached uint64

	countSymlink       uint64
	countSymlinkOk     uint64
	countSymlinkError  uint64
	countSymlinkCached uint64

	countFileSize   int64
	countXattrSize  int64
	countCachedSize uint64

	// streamedBytes is the running total of payload bytes read from the source
	// at chunk granularity (the "chunk.bytes" event), so progress can advance
	// *within* a large file rather than jumping only when it completes.
	// reconciledBytes is the portion of streamedBytes already folded into
	// countFileSize/countXattrSize by a completed file/xattr; subtracting it
	// leaves just the bytes of files still in flight, which avoids
	// double-counting and is safe under concurrent chunking.
	streamedBytes   uint64
	reconciledBytes uint64

	workflow string

	// ioRate is the latest throughput (bytes/sec) sampled by the kloset
	// iostat sampler for the scope that tracks payload progress: the source
	// read rate for a backup, the destination write rate for an export. It
	// drives the ETA. ioRateAt records when it was last updated so a stalled
	// stream doesn't keep a stale rate forever.
	ioRate   float64
	ioRateAt time.Time

	// ioScopes holds the latest sampled read/write stats per scope (e.g.
	// "source", "storage") as reported by the kloset iostat sampler. Unlike
	// the repository's raw counters, these are emitted on a fixed cadence and
	// decoupled from packfile flush timing, so they move smoothly during a
	// backup. Used to render the throughput/totals summary line.
	ioScopes map[string]ioScopeStat

	lastItem string
	errors   []string
	logs     []string
}

// processedBytes returns the number of bytes handled so far: bytes of
// completed files/xattrs plus the in-flight bytes of files still being chunked.
// It is measured against summarySize (the total scanned size) to drive the
// progress bar and ETA.
//
// Cached files are NOT added separately: every file emits file.ok with its full
// size (see processRecord in kloset backup.go), whether or not it was a cache
// hit, so countFileSize already accounts for them. A cache hit additionally
// emits file.cached, but folding countCachedSize in here as well would
// double-count it — a re-backup of an unchanged 10 GiB file would report 20 GiB.
func (s *State) processedBytes() uint64 {
	done := uint64(s.countFileSize) + uint64(s.countXattrSize)
	// Bytes streamed for files that haven't emitted their .ok yet.
	if s.streamedBytes > s.reconciledBytes {
		done += s.streamedBytes - s.reconciledBytes
	}
	return done
}

// reconcile folds the size of a just-completed file/xattr into reconciledBytes,
// clamped so it never exceeds streamedBytes. The declared size can differ from
// the bytes actually streamed for that file (chunk sums vs. scanned size);
// clamping keeps the in-flight term (streamedBytes-reconciledBytes) from going
// negative and, crucially, from borrowing against the next file's bytes.
func (s *State) reconcile(size uint64) {
	s.reconciledBytes += size
	if s.reconciledBytes > s.streamedBytes {
		s.reconciledBytes = s.streamedBytes
	}
}

// payloadScope returns the iostats scope and direction ("r" or "w") whose
// throughput tracks how fast the snapshot payload is being processed for the
// current workflow, along with whether a scope is known. Backup ingests the
// source (read); export emits to the destination (write).
func (s *State) payloadScope() (scope, dir string, ok bool) {
	switch s.workflow {
	case "import":
		return "source", "r", true
	case "export":
		return "destination", "w", true
	default:
		return "", "", false
	}
}

// iostatTotal reads the cumulative "total" bytes from a sampled stats map.
func iostatTotal(m map[string]any) int64 {
	if m == nil {
		return 0
	}
	switch v := m["total"].(type) {
	case int64:
		return v
	case uint64:
		return int64(v)
	case float64:
		return int64(v)
	}
	return 0
}

// updateIOStats records the latest sampler snapshot: the per-scope read/write
// totals plus wall-clock rates (for the summary line), and the payload-scope
// rate that drives the ETA. Rates are derived from the byte delta since the
// previous sample for the same scope, divided by elapsed wall-clock time.
func (s *State) updateIOStats(e Event) {
	s.updateIOStatsAt(e, time.Now())
}

// updateIOStatsAt is updateIOStats with an injectable clock for testing the
// delta-based rate computation.
func (s *State) updateIOStatsAt(e Event, now time.Time) {
	scope, _ := e.Data["scope"].(string)
	if scope == "" {
		return
	}
	r, _ := e.Data["r"].(map[string]any)
	w, _ := e.Data["w"].(map[string]any)

	rTotal := iostatTotal(r)
	wTotal := iostatTotal(w)

	if s.ioScopes == nil {
		s.ioScopes = make(map[string]ioScopeStat)
	}

	cur := ioScopeStat{readTotal: rTotal, writeTotal: wTotal, sampledAt: now}
	if prev, ok := s.ioScopes[scope]; ok && !prev.sampledAt.IsZero() {
		if dt := now.Sub(prev.sampledAt).Seconds(); dt > 0 {
			if d := rTotal - prev.readTotal; d > 0 {
				cur.readRate = float64(d) / dt
			}
			if d := wTotal - prev.writeTotal; d > 0 {
				cur.writeRate = float64(d) / dt
			}
		}
	}
	s.ioScopes[scope] = cur

	// Feed the ETA from the payload scope's active direction.
	if want, dir, ok := s.payloadScope(); ok && scope == want {
		rate := cur.readRate
		if dir == "w" {
			rate = cur.writeRate
		}
		if rate > 0 {
			s.ioRate = rate
			s.ioRateAt = now
		}
	}
}

func newApplicationState() *State {
	return &State{
		lastItem: "",
		errors:   []string{},
		logs:     []string{},
	}
}

func newApplication(ctx *appcontext.AppContext, name string, repo *repository.Repository) *Application {
	done := make(chan struct{})

	modelFunc, ok := applications[name]
	if !ok {
		return nil
	}

	capp := &Application{
		ctx:   ctx,
		name:  displayName(name),
		done:  done,
		state: newApplicationState(),
	}
	capp.prog = tea.NewProgram(modelFunc(ctx, capp, repo))

	go func() {
		defer close(done)
		_, err := capp.prog.Run()
		if err != nil {
			capp.err = err
		}
	}()

	return capp
}

func (app *Application) Stop() {
	if app == nil {
		return
	}

	if app.prog != nil {
		app.prog.Quit()
	}

	<-app.done
}

func (s *State) Update(e Event) {
	switch e.Type {
	case "workflow.start":
		s.startTime = time.Now()
		s.snapshotID = fmt.Sprintf("%x", e.Snapshot[0:4])
		if w, ok := e.Data["workflow"].(string); ok {
			s.workflow = w
		}

	case "iostats":
		s.updateIOStats(e)

	case "workflow.end":

	case "path":
		if p, ok := e.Data["path"].(string); ok {
			s.lastItem = p
		}
		s.countPath++

	case "path.ok":
		s.countPathOk++

	case "path.error":
		if p, ok := e.Data["path"].(string); ok {
			s.errors = append(s.errors, fmt.Sprintf("%s %s: %s", crossMark, p, e.Data["error"]))
		}
		s.countPathError++

	case "path.cached":
		s.countPathCached++

	case "directory":
		s.countDir++

	case "directory.ok":
		s.countDirOk++

	case "directory.error":
		s.countDirError++

	case "directory.cached":
		s.countDirCached++

	case "file":
		s.countFile++

	case "chunk.bytes":
		if n, ok := e.Data["bytes"].(int64); ok && n > 0 {
			s.streamedBytes += uint64(n)
		}

	case "file.ok":
		s.countFileOk++
		fileinfo := e.Data["fileinfo"].(objects.FileInfo)
		size := fileinfo.Size()
		s.countFileSize += size
		// This file's streamed bytes are now committed in countFileSize;
		// reconcile so they aren't double-counted by processedBytes().
		if size > 0 {
			s.reconcile(uint64(size))
		}

	case "file.error":
		s.countFileError++

	case "file.cached":
		s.countFileCached++
		fileinfo := e.Data["fileinfo"].(objects.FileInfo)
		s.countCachedSize += uint64(fileinfo.Size())

	case "xattr":
		s.countXattr++

	case "xattr.ok":
		s.countXattrOk++
		size := e.Data["size"].(int64)
		s.countXattrSize += size
		// Xattrs are chunked through the same path as files, so reconcile
		// their streamed bytes too.
		if size > 0 {
			s.reconcile(uint64(size))
		}

	case "xattr.error":
		s.countXattrError++

	case "xattr.cached":
		s.countXattrCached++
		size := e.Data["size"].(int64)
		s.countCachedSize += uint64(size)

	case "symlink":
		s.countSymlink++

	case "symlink.ok":
		s.countSymlinkOk++

	case "symlink.error":
		s.countSymlinkError++

	case "symlink.cached":
		s.countSymlinkCached++

	case "fs.summary":
		s.gotSummary = true
		s.summaryFile = e.Data["files"].(uint64)
		s.summaryDirectory = e.Data["directories"].(uint64)
		s.summarySymlink = e.Data["symlinks"].(uint64)
		s.summaryXattr = e.Data["xattrs"].(uint64)
		s.summarySize = e.Data["size"].(uint64)
		s.summaryPath = s.summaryFile + s.summaryDirectory + s.summarySymlink + s.summaryXattr

	case "snapshot.import.start":
		s.phase = "processing"
		s.timerBegin = time.Now()

	case "snapshot.import.done":
		s.detail = ""
		s.timerDone = true

	case "snapshot.vfs.start":
		s.lastItem = ""
		s.phase = "building VFS"

	case "snapshot.vfs.end":
		s.phase = ""
		s.detail = ""
		s.lastItem = ""

	case "snapshot.index.start":
		s.lastItem = ""
		s.phase = "indexing"

	case "snapshot.index.end":
		s.lastItem = ""
		s.phase = ""
		s.detail = ""

	case "snapshot.commit.start":
		s.lastItem = ""
		s.phase = "committing"

	case "result":
		s.lastItem = ""
		s.phase = "completed"
		s.detail = fmt.Sprintf(
			"size=%s errors=%d duration=%s",
			humanize.IBytes(e.Data["size"].(uint64)),
			e.Data["errors"].(uint64),
			e.Data["duration"].(time.Duration),
		)
	}
}
