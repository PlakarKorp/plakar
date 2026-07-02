package tui

import (
	"fmt"
	"os"
	"sort"
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

	// I/O movement tracking: the last observed read/write totals and the last
	// time each advanced. Used to decide when a side is idle (show a status word)
	// versus actively transferring (show a rate). hasRead/hasWrite record whether
	// each side has ever moved, so no status shows before the first transfer.
	lastReadTotal, lastWriteTotal int64
	readMovedAt, writeMovedAt     time.Time
	hasRead, hasWrite             bool

	// Wall-clock throughput history per side, for the debug distribution. Each
	// entry is bytes/sec over one fixed real-time interval, sampled from the
	// cumulative totals. Sampling at a fixed cadence (rather than per I/O op) is
	// what makes it correct for bursty writers — packfile flushes cluster many
	// writes into a tiny window, which per-op bucketing would misread as an
	// enormous rate.
	readRates  rateSampler
	writeRates rateSampler

	lastItem string
	errors   []string
	logs     []string
}

// rateSampler keeps a bounded history of wall-clock throughput samples
// (bytes/sec), one per fixed real-time interval, computed from successive
// cumulative totals.
type rateSampler struct {
	lastTotal int64
	lastAt    time.Time
	rates     []float64
}

const (
	rateInterval   = 250 * time.Millisecond
	rateMaxSamples = 4096
)

// observe records the cumulative total at time now, appending a throughput
// sample once a full interval has elapsed since the previous one.
func (rs *rateSampler) observe(total int64, now time.Time) {
	if rs.lastAt.IsZero() {
		rs.lastTotal, rs.lastAt = total, now
		return
	}
	dt := now.Sub(rs.lastAt)
	if dt < rateInterval {
		return
	}
	if d := total - rs.lastTotal; d >= 0 {
		rs.rates = append(rs.rates, float64(d)/dt.Seconds())
		if len(rs.rates) > rateMaxSamples {
			rs.rates = rs.rates[len(rs.rates)-rateMaxSamples:]
		}
	}
	rs.lastTotal, rs.lastAt = total, now
}

// rateDist is a throughput distribution over the sampled wall-clock rates.
type rateDist struct {
	n                     int
	avg, min, max         float64
	median, p90, p95, p99 float64
}

func (rs *rateSampler) dist() rateDist {
	n := len(rs.rates)
	if n == 0 {
		return rateDist{}
	}
	sorted := make([]float64, n)
	copy(sorted, rs.rates)
	sort.Float64s(sorted)

	var sum float64
	for _, v := range sorted {
		sum += v
	}
	pct := func(p float64) float64 {
		if n == 1 {
			return sorted[0]
		}
		pos := p / 100 * float64(n-1)
		lo := int(pos)
		hi := lo + 1
		if hi >= n {
			return sorted[n-1]
		}
		w := pos - float64(lo)
		return sorted[lo]*(1-w) + sorted[hi]*w
	}
	return rateDist{
		n:      n,
		avg:    sum / float64(n),
		min:    sorted[0],
		max:    sorted[n-1],
		median: pct(50),
		p90:    pct(90),
		p95:    pct(95),
		p99:    pct(99),
	}
}

// ioActivity is a side's transfer status for display.
type ioActivity int

const (
	ioNone   ioActivity = iota // never transferred: show nothing
	ioMoving                   // actively transferring: show a rate
	ioIdle                     // transferred before, now idle: show a status word
)

// sampleIO records the current read/write totals at time now and returns each
// side's activity. A side is "moving" if its total advanced within the recent
// window, "idle" if it moved before but not lately, "none" if it never moved.
func (s *State) sampleIO(readTotal, writeTotal int64, now time.Time) (read, write ioActivity) {
	const window = 1500 * time.Millisecond

	if readTotal > s.lastReadTotal {
		s.lastReadTotal = readTotal
		s.readMovedAt = now
		s.hasRead = true
	}
	if writeTotal > s.lastWriteTotal {
		s.lastWriteTotal = writeTotal
		s.writeMovedAt = now
		s.hasWrite = true
	}

	activity := func(has bool, movedAt time.Time) ioActivity {
		switch {
		case !has:
			return ioNone
		case now.Sub(movedAt) <= window:
			return ioMoving
		default:
			return ioIdle
		}
	}
	return activity(s.hasRead, s.readMovedAt), activity(s.hasWrite, s.writeMovedAt)
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
		final, err := capp.prog.Run()
		if err != nil {
			capp.err = err
		}
		// The model renders nothing once finished (so bubbletea tears down
		// cleanly); print the completed summary exactly once here. Rendering
		// with finished=false reproduces the full final frame.
		if am, ok := final.(appModel); ok && am.finished {
			am.finished = false
			fmt.Fprint(os.Stdout, am.View())
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
		// Keep the first workflow ("import"/"export"), which identifies the
		// operation and drives scope selection. A nested workflow.start (the
		// kloset Builder emits its own "backup"/"export") must not overwrite it.
		if w, ok := e.Data["workflow"].(string); ok && s.workflow == "" {
			s.workflow = w
		}

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
