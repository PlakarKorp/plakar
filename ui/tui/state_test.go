package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	tea "github.com/charmbracelet/bubbletea"
)

func TestStateUpdate_WorkflowStartSetsSnapshotID(t *testing.T) {
	s := newApplicationState()
	s.Update(Event{
		Type:     "workflow.start",
		Snapshot: objects.MAC{0xde, 0xad, 0xbe, 0xef},
	})
	if s.startTime.IsZero() {
		t.Fatal("startTime should be set")
	}
	if !strings.HasPrefix(s.snapshotID, "deadbeef") {
		t.Fatalf("snapshotID = %q, want deadbeef prefix", s.snapshotID)
	}
}

func TestStateUpdate_WorkflowEndIsNoOp(t *testing.T) {
	s := newApplicationState()
	s.Update(Event{Type: "workflow.start"})
	before := *s
	s.Update(Event{Type: "workflow.end"})
	if s.startTime != before.startTime {
		t.Fatal("workflow.end should not alter startTime")
	}
}

func TestStateUpdate_PathCounters(t *testing.T) {
	s := newApplicationState()
	s.Update(Event{Type: "path", Data: map[string]any{"path": "/etc"}})
	if s.countPath != 1 || s.lastItem != "/etc" {
		t.Fatalf("after path: %+v", s)
	}
	s.Update(Event{Type: "path.ok"})
	if s.countPathOk != 1 {
		t.Fatalf("countPathOk = %d", s.countPathOk)
	}
	s.Update(Event{Type: "path.cached"})
	if s.countPathCached != 1 {
		t.Fatalf("countPathCached = %d", s.countPathCached)
	}
	s.Update(Event{
		Type: "path.error",
		Data: map[string]any{"path": "/no", "error": "denied"},
	})
	if s.countPathError != 1 || len(s.errors) != 1 {
		t.Fatalf("expected one recorded error, got %+v", s.errors)
	}
}

func TestStateUpdate_DirectoryCounters(t *testing.T) {
	s := newApplicationState()
	for _, typ := range []string{"directory", "directory.ok", "directory.error", "directory.cached"} {
		s.Update(Event{Type: typ})
	}
	if s.countDir != 1 || s.countDirOk != 1 || s.countDirError != 1 || s.countDirCached != 1 {
		t.Fatalf("directory counters wrong: %+v", s)
	}
}

func TestStateUpdate_FileCountersAndSizes(t *testing.T) {
	s := newApplicationState()
	s.Update(Event{Type: "file"})

	fi := objects.FileInfo{Lsize: 1024}
	s.Update(Event{Type: "file.ok", Data: map[string]any{"fileinfo": fi}})
	if s.countFileOk != 1 || s.countFileSize != 1024 {
		t.Fatalf("file.ok: countFileOk=%d countFileSize=%d", s.countFileOk, s.countFileSize)
	}

	s.Update(Event{Type: "file.error"})
	if s.countFileError != 1 {
		t.Fatalf("file.error: countFileError=%d", s.countFileError)
	}

	s.Update(Event{Type: "file.cached", Data: map[string]any{"fileinfo": objects.FileInfo{Lsize: 2048}}})
	if s.countFileCached != 1 || s.countCachedSize != 2048 {
		t.Fatalf("file.cached: countFileCached=%d countCachedSize=%d", s.countFileCached, s.countCachedSize)
	}
}

func TestStateUpdate_CachedRebackupNoDoubleCount(t *testing.T) {
	// Regression: a re-backup of an unchanged file emits file.cached AND
	// file.ok for it. processedBytes() must report the file's size once, not
	// twice (the bug reported a 10 GiB file as 20 GiB).
	s := newApplicationState()
	const size = int64(10 << 30) // 10 GiB
	fi := objects.FileInfo{Lsize: size}

	s.Update(Event{Type: "file"})
	s.Update(Event{Type: "file.cached", Data: map[string]any{"fileinfo": fi}})
	s.Update(Event{Type: "file.ok", Data: map[string]any{"fileinfo": fi}})

	if got := s.processedBytes(); got != uint64(size) {
		t.Fatalf("processedBytes = %d, want %d (10 GiB counted once)", got, size)
	}
}

func TestStateUpdate_ChunkBytesAdvanceWithinFile(t *testing.T) {
	s := newApplicationState()
	fi := objects.FileInfo{Lsize: 1000}

	// File starts streaming; chunks arrive before file.ok.
	s.Update(Event{Type: "file"})
	s.Update(Event{Type: "chunk.bytes", Data: map[string]any{"bytes": int64(300)}})
	if got := s.processedBytes(); got != 300 {
		t.Fatalf("mid-file processedBytes = %d, want 300", got)
	}
	s.Update(Event{Type: "chunk.bytes", Data: map[string]any{"bytes": int64(400)}})
	if got := s.processedBytes(); got != 700 {
		t.Fatalf("mid-file processedBytes = %d, want 700", got)
	}

	// file.ok commits the full size; the streamed bytes must reconcile so the
	// total doesn't jump past the file's real size.
	s.Update(Event{Type: "file.ok", Data: map[string]any{"fileinfo": fi}})
	if got := s.processedBytes(); got != 1000 {
		t.Fatalf("post-file.ok processedBytes = %d, want 1000 (no double count)", got)
	}

	// A cached file emits BOTH file.cached and file.ok (see processRecord in
	// kloset backup.go — file.ok fires for every file, cache hit or not). Its
	// size must be counted once, not twice.
	s.Update(Event{Type: "file.cached", Data: map[string]any{"fileinfo": objects.FileInfo{Lsize: 500}}})
	s.Update(Event{Type: "file.ok", Data: map[string]any{"fileinfo": objects.FileInfo{Lsize: 500}}})
	if got := s.processedBytes(); got != 1500 {
		t.Fatalf("after cached file processedBytes = %d, want 1500 (counted once, not double)", got)
	}

	// A second streaming file advances again from the committed floor.
	s.Update(Event{Type: "chunk.bytes", Data: map[string]any{"bytes": int64(250)}})
	if got := s.processedBytes(); got != 1750 {
		t.Fatalf("second file mid-stream processedBytes = %d, want 1750", got)
	}
}

func TestStateUpdate_XattrCounters(t *testing.T) {
	s := newApplicationState()
	s.Update(Event{Type: "xattr"})
	s.Update(Event{Type: "xattr.ok", Data: map[string]any{"size": int64(64)}})
	s.Update(Event{Type: "xattr.error"})
	s.Update(Event{Type: "xattr.cached", Data: map[string]any{"size": int64(128)}})

	if s.countXattr != 1 || s.countXattrOk != 1 || s.countXattrError != 1 || s.countXattrCached != 1 {
		t.Fatalf("xattr counters: %+v", s)
	}
	if s.countXattrSize != 64 {
		t.Fatalf("countXattrSize = %d, want 64", s.countXattrSize)
	}
	if s.countCachedSize != 128 {
		t.Fatalf("countCachedSize = %d, want 128", s.countCachedSize)
	}
}

func TestStateUpdate_SymlinkCounters(t *testing.T) {
	s := newApplicationState()
	for _, typ := range []string{"symlink", "symlink.ok", "symlink.error", "symlink.cached"} {
		s.Update(Event{Type: typ})
	}
	if s.countSymlink != 1 || s.countSymlinkOk != 1 ||
		s.countSymlinkError != 1 || s.countSymlinkCached != 1 {
		t.Fatalf("symlink counters: %+v", s)
	}
}

func TestStateUpdate_FsSummary(t *testing.T) {
	s := newApplicationState()
	s.Update(Event{
		Type: "fs.summary",
		Data: map[string]any{
			"files":       uint64(10),
			"directories": uint64(2),
			"symlinks":    uint64(1),
			"xattrs":      uint64(3),
			"size":        uint64(99999),
		},
	})
	if !s.gotSummary {
		t.Fatal("gotSummary should be true")
	}
	if s.summaryFile != 10 || s.summaryDirectory != 2 || s.summarySymlink != 1 || s.summaryXattr != 3 {
		t.Fatalf("summary fields wrong: %+v", s)
	}
	if s.summarySize != 99999 {
		t.Fatalf("summarySize = %d", s.summarySize)
	}
	if s.summaryPath != 10+2+1+3 {
		t.Fatalf("summaryPath = %d, want %d", s.summaryPath, 10+2+1+3)
	}
}

func TestStateUpdate_SnapshotPhases(t *testing.T) {
	s := newApplicationState()

	s.Update(Event{Type: "snapshot.import.start"})
	if s.phase != "processing" {
		t.Fatalf("phase after import.start = %q", s.phase)
	}
	if s.timerBegin.IsZero() {
		t.Fatal("timerBegin should be set on import.start")
	}

	s.Update(Event{Type: "snapshot.import.done"})
	if !s.timerDone {
		t.Fatal("timerDone should be true after import.done")
	}

	s.Update(Event{Type: "snapshot.vfs.start"})
	if s.phase != "building VFS" {
		t.Fatalf("phase after vfs.start = %q", s.phase)
	}

	s.Update(Event{Type: "snapshot.vfs.end"})
	if s.phase != "" {
		t.Fatalf("phase after vfs.end = %q", s.phase)
	}

	s.Update(Event{Type: "snapshot.index.start"})
	if s.phase != "indexing" {
		t.Fatalf("phase after index.start = %q", s.phase)
	}

	s.Update(Event{Type: "snapshot.index.end"})
	if s.phase != "" {
		t.Fatalf("phase after index.end = %q", s.phase)
	}

	s.Update(Event{Type: "snapshot.commit.start"})
	if s.phase != "committing" {
		t.Fatalf("phase after commit.start = %q", s.phase)
	}
}

func TestStateUpdate_Result(t *testing.T) {
	s := newApplicationState()
	s.Update(Event{
		Type: "result",
		Data: map[string]any{
			"size":     uint64(1024 * 1024),
			"errors":   uint64(2),
			"duration": 5 * time.Second,
		},
	})
	if s.phase != "completed" {
		t.Fatalf("phase = %q, want completed", s.phase)
	}
	if !strings.Contains(s.detail, "errors=2") {
		t.Fatalf("detail should include error count: %q", s.detail)
	}
	if !strings.Contains(s.detail, "5s") {
		t.Fatalf("detail should include duration: %q", s.detail)
	}
}

func TestStateUpdate_UnknownTypeIsNoOp(t *testing.T) {
	s := newApplicationState()
	before := *s
	s.Update(Event{Type: "totally.unknown.event"})
	// Anything beyond no-op would change one of these fields:
	if s.countPath != before.countPath || s.countFile != before.countFile ||
		s.phase != before.phase || s.lastItem != before.lastItem {
		t.Fatalf("unknown type altered state: %+v", s)
	}
}

// newAppModelForTest returns an appModel with just enough plumbing for
// the message-dispatch tests below. We don't construct a real Application
// (that spawns a bubbletea program) — we set application to nil and only
// invoke branches that don't dereference it.
func newAppModelForTest() appModel {
	return appModel{}
}

func TestAppModelUpdate_WindowSizeStoresGeometry(t *testing.T) {
	m := newAppModelForTest()
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	got := next.(appModel)
	if got.width != 120 || got.height != 40 {
		t.Fatalf("dimensions = %dx%d, want 120x40", got.width, got.height)
	}
}

func TestAppModelUpdate_CancelledSetsForceQuit(t *testing.T) {
	m := newAppModelForTest()
	next, cmd := m.Update(cancelledMsg{})
	if !next.(appModel).forceQuit {
		t.Fatal("cancelledMsg should set forceQuit")
	}
	if cmd == nil {
		t.Fatal("cancelledMsg should return a quit command")
	}
}

func TestAppModelUpdate_EventsClosedReturnsQuit(t *testing.T) {
	m := newAppModelForTest()
	_, cmd := m.Update(eventsClosedMsg{})
	if cmd == nil {
		t.Fatal("eventsClosedMsg should return tea.Quit")
	}
}

func TestAppModelUpdate_UnhandledMsgIsNoOp(t *testing.T) {
	m := newAppModelForTest()
	type bogus struct{}
	next, cmd := m.Update(bogus{})
	if next.(appModel).forceQuit {
		t.Fatal("unhandled message should not set forceQuit")
	}
	if cmd != nil {
		t.Fatal("unhandled message should return nil cmd")
	}
}
