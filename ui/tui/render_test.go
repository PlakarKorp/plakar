package tui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	ptesting "github.com/PlakarKorp/plakar/testing"
	tea "github.com/charmbracelet/bubbletea"
)

// renderModel builds an appModel wired to a freshly constructed *Application
// directly (no tea.Program), so View()/Update() can be exercised white-box
// without touching the live terminal.
func renderModel(s *State) appModel {
	return appModel{
		application: &Application{name: "import", state: s},
		progress:    progressBar(),
	}
}

// driveSummaryState returns a state in progress mode (gotSummary + total),
// having processed a handful of files/dirs.
func driveSummaryState() *State {
	s := newApplicationState()
	s.Update(Event{Type: "workflow.start", Snapshot: objects.MAC{0xde, 0xad, 0xbe, 0xef}})
	s.startTime = time.Now().Add(-5 * time.Second)
	s.Update(Event{Type: "snapshot.import.start"})
	s.Update(Event{
		Type: "fs.summary",
		Data: map[string]any{
			"files":       uint64(10),
			"directories": uint64(3),
			"symlinks":    uint64(1),
			"xattrs":      uint64(0),
			"size":        uint64(4096),
		},
	})
	s.Update(Event{Type: "directory", Data: map[string]any{}})
	s.Update(Event{Type: "directory.ok"})
	s.Update(Event{Type: "path", Data: map[string]any{"path": "/var/log/system.log"}})
	s.Update(Event{Type: "path.ok"})
	s.Update(Event{Type: "file.ok", Data: map[string]any{"fileinfo": objects.FileInfo{Lsize: 2048}}})
	return s
}

func TestRenderView_ProgressModeNonEmpty(t *testing.T) {
	t.Parallel()
	m := renderModel(driveSummaryState())
	m.width = 80
	m.height = 24
	out := m.View()
	if out == "" {
		t.Fatal("View() returned empty string in progress mode")
	}
	if !strings.Contains(out, "import") {
		t.Fatalf("expected app name in output, got:\n%s", out)
	}
	if !strings.Contains(out, "deadbeef") {
		t.Fatalf("expected snapshot id in output, got:\n%s", out)
	}
}

func TestRenderView_ForceQuitAborted(t *testing.T) {
	t.Parallel()
	m := renderModel(driveSummaryState())
	m.forceQuit = true
	out := m.View()
	if !strings.Contains(out, "aborted") {
		t.Fatalf("forceQuit View should say aborted, got: %q", out)
	}
	if !strings.Contains(out, "import") {
		t.Fatalf("forceQuit View should mention app name, got: %q", out)
	}
}

func TestRenderView_NonProgressMode(t *testing.T) {
	t.Parallel()
	s := newApplicationState()
	s.Update(Event{Type: "workflow.start", Snapshot: objects.MAC{0x01, 0x02, 0x03, 0x04}})
	s.startTime = time.Now().Add(-2 * time.Second)
	s.Update(Event{Type: "snapshot.import.start"})
	s.Update(Event{Type: "path", Data: map[string]any{"path": "/tmp/file.txt"}})
	s.Update(Event{Type: "path.ok"})
	// no fs.summary => non-progress branch
	m := renderModel(s)
	m.width = 100
	m.height = 30
	out := m.View()
	if out == "" {
		t.Fatal("non-progress View() should not be empty")
	}
	if strings.Contains(out, "ETA") {
		t.Fatalf("non-progress mode should not render ETA, got:\n%s", out)
	}
}

func TestRenderView_ZeroWidthPlainOutput(t *testing.T) {
	t.Parallel()
	s := driveSummaryState()
	m := renderModel(s)
	// width=0 forces the plain writeLine path and the height<=0 error fallback
	m.width = 0
	m.height = 0
	out := m.View()
	if out == "" {
		t.Fatal("zero-width View() should still produce output")
	}
}

func TestRenderView_WithErrorsAndHeight(t *testing.T) {
	t.Parallel()
	s := driveSummaryState()
	for i := 0; i < 8; i++ {
		s.Update(Event{
			Type: "path.error",
			Data: map[string]any{"path": "/bad/path", "error": "permission denied"},
		})
	}
	m := renderModel(s)
	m.width = 60
	m.height = 12
	out := m.View()
	if !strings.Contains(out, "permission denied") {
		t.Fatalf("expected errors rendered, got:\n%s", out)
	}
}

func TestRenderView_WithErrorsZeroHeightFallback(t *testing.T) {
	t.Parallel()
	// non-progress + errors + height 0 => the 5-line fallback path
	s := newApplicationState()
	s.Update(Event{Type: "workflow.start", Snapshot: objects.MAC{0xaa}})
	s.startTime = time.Now()
	for i := 0; i < 10; i++ {
		s.Update(Event{Type: "path.error", Data: map[string]any{"path": "/x", "error": "boom"}})
	}
	m := renderModel(s)
	m.width = 40
	m.height = 0
	out := m.View()
	if !strings.Contains(out, "boom") {
		t.Fatalf("expected fallback errors rendered, got:\n%s", out)
	}
}

func TestRenderView_WithLogs(t *testing.T) {
	t.Parallel()
	s := driveSummaryState()
	s.logs = append(s.logs, "[stdout] hello from log")
	m := renderModel(s)
	m.width = 80
	m.height = 24
	out := m.View()
	if !strings.Contains(out, "hello from log") {
		t.Fatalf("expected last log line rendered, got:\n%s", out)
	}
}

func TestRenderView_CompletedPhase(t *testing.T) {
	t.Parallel()
	s := driveSummaryState()
	s.Update(Event{
		Type: "result",
		Data: map[string]any{
			"size":     uint64(1024 * 1024),
			"errors":   uint64(0),
			"duration": 3 * time.Second,
		},
	})
	m := renderModel(s)
	m.width = 80
	m.height = 24
	out := m.View()
	if !strings.Contains(out, "completed") {
		t.Fatalf("expected completed phase in output, got:\n%s", out)
	}
}

func TestRenderView_NarrowWidthTruncatesPath(t *testing.T) {
	t.Parallel()
	s := driveSummaryState()
	s.lastItem = "/very/deep/nested/directory/structure/with/a/long/file/name.txt"
	m := renderModel(s)
	m.width = 20
	m.height = 10
	out := m.View()
	if out == "" {
		t.Fatal("narrow width View() should produce output")
	}
}

func TestRenderView_WithRealRepoStoreSummary(t *testing.T) {
	t.Parallel()
	var out, errb bytes.Buffer
	repo, _ := ptesting.GenerateRepository(t, &out, &errb, nil)

	s := driveSummaryState()
	m := renderModel(s)
	m.repo = repo
	m.width = 100
	m.height = 24
	// debounceStat is zero => the store-summary branch computes IOStats.
	rendered := m.View()
	if !strings.Contains(rendered, "store:") {
		t.Fatalf("expected store summary line with non-nil repo, got:\n%s", rendered)
	}
}

// --- appModel.Update message dispatch ---

func TestRenderUpdate_WindowSize(t *testing.T) {
	t.Parallel()
	m := renderModel(newApplicationState())
	next, cmd := m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	got := next.(appModel)
	if got.width != 200 || got.height != 50 {
		t.Fatalf("geometry = %dx%d", got.width, got.height)
	}
	if cmd != nil {
		t.Fatal("WindowSizeMsg should return nil cmd")
	}
}

func TestRenderUpdate_TickComputesRateAndRearms(t *testing.T) {
	t.Parallel()
	s := driveSummaryState()
	// boost result counters so resDone > lastDone yields a positive rate
	s.countFileOk = 100
	s.countSymlinkOk = 5
	s.countXattrOk = 2
	m := renderModel(s)
	// pre-seed lastETAAt in the past so dt > 0.2 path runs
	m.lastETAAt = time.Now().Add(-1 * time.Second)
	m.lastDone = 0
	next, cmd := m.Update(tickMsg{})
	got := next.(appModel)
	if got.rateEMA <= 0 {
		t.Fatalf("tick should compute a positive rateEMA, got %v", got.rateEMA)
	}
	if cmd == nil {
		t.Fatal("tickMsg should re-arm the tick command")
	}
}

func TestRenderUpdate_TickFirstSeedsETAClock(t *testing.T) {
	t.Parallel()
	m := renderModel(driveSummaryState())
	if !m.lastETAAt.IsZero() {
		t.Fatal("precondition: lastETAAt should start zero")
	}
	next, cmd := m.Update(tickMsg{})
	got := next.(appModel)
	if got.lastETAAt.IsZero() {
		t.Fatal("first tick should seed lastETAAt")
	}
	if cmd == nil {
		t.Fatal("tickMsg should always re-arm")
	}
}

func TestRenderUpdate_CtrlCSetsForceQuit(t *testing.T) {
	t.Parallel()
	m := renderModel(newApplicationState())
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !next.(appModel).forceQuit {
		t.Fatal("ctrl+c should set forceQuit")
	}
	if cmd == nil {
		t.Fatal("ctrl+c should return an interrupt command")
	}
}

func TestRenderUpdate_OtherKeyIsNoOp(t *testing.T) {
	t.Parallel()
	m := renderModel(newApplicationState())
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if next.(appModel).forceQuit {
		t.Fatal("plain 'q' should not force quit in this model")
	}
	if cmd != nil {
		t.Fatal("unhandled key should yield nil cmd")
	}
}

func TestRenderUpdate_EventsClosedQuits(t *testing.T) {
	t.Parallel()
	m := renderModel(newApplicationState())
	_, cmd := m.Update(eventsClosedMsg{})
	if cmd == nil {
		t.Fatal("eventsClosedMsg should return tea.Quit")
	}
}

func TestRenderUpdate_CancelledForceQuits(t *testing.T) {
	t.Parallel()
	m := renderModel(newApplicationState())
	next, cmd := m.Update(cancelledMsg{err: context_errStub()})
	if !next.(appModel).forceQuit {
		t.Fatal("cancelledMsg should set forceQuit")
	}
	if cmd == nil {
		t.Fatal("cancelledMsg should return a quit cmd")
	}
}

func TestRenderUpdate_QuitMsg(t *testing.T) {
	t.Parallel()
	m := renderModel(newApplicationState())
	_, cmd := m.Update(tea.QuitMsg{})
	if cmd == nil {
		t.Fatal("QuitMsg should return tea.Quit")
	}
}

func context_errStub() error { return errStubError{} }

type errStubError struct{}

func (errStubError) Error() string { return "stub cancelled" }

// --- pure helpers ---

func TestHelperHumanDuration(t *testing.T) {
	t.Parallel()
	if got := humanDuration(90 * time.Second); got != "01:30" {
		t.Fatalf("humanDuration(90s) = %q, want 01:30", got)
	}
	if got := humanDuration(3661 * time.Second); got != "01:01:01" {
		t.Fatalf("humanDuration(3661s) = %q, want 01:01:01", got)
	}
	if got := humanDuration(-5 * time.Second); got != "00:00" {
		t.Fatalf("humanDuration(neg) = %q, want 00:00", got)
	}
}

func TestHelperFmtETA(t *testing.T) {
	t.Parallel()
	if got := fmtETA(0); got != "" {
		t.Fatalf("fmtETA(0) = %q, want empty", got)
	}
	if got := fmtETA(45 * time.Second); got != "45s" {
		t.Fatalf("fmtETA(45s) = %q, want 45s", got)
	}
	if got := fmtETA(90 * time.Second); got != "1m30s" {
		t.Fatalf("fmtETA(90s) = %q, want 1m30s", got)
	}
	if got := fmtETA(3*time.Hour + 5*time.Minute); got != "3h05m" {
		t.Fatalf("fmtETA(3h05m) = %q, want 3h05m", got)
	}
}

func TestHelperFormatBytes(t *testing.T) {
	t.Parallel()
	if got := formatBytes(0); got != "0 B" {
		t.Fatalf("formatBytes(0) = %q", got)
	}
	if got := formatBytes(-1); got != "0 B" {
		t.Fatalf("formatBytes(-1) = %q", got)
	}
	if got := formatBytes(1024); !strings.Contains(got, "KiB") {
		t.Fatalf("formatBytes(1024) = %q, want KiB", got)
	}
}

func TestHelperErrAndFmtNewReuse(t *testing.T) {
	t.Parallel()
	if got := err(3); !strings.Contains(got, "3") {
		t.Fatalf("err(3) = %q, want to contain 3", got)
	}
	if got := fmtNewReuse(5, 10, false); !strings.Contains(got, "5") || strings.Contains(got, "/") {
		t.Fatalf("fmtNewReuse no-progress = %q", got)
	}
	if got := fmtNewReuse(5, 10, true); !strings.Contains(got, "5") || !strings.Contains(got, "10") {
		t.Fatalf("fmtNewReuse progress = %q", got)
	}
}

func TestHelperTruncateLeft(t *testing.T) {
	t.Parallel()
	if got := truncateLeft("hello", 0); got != "" {
		t.Fatalf("truncateLeft maxW=0 = %q", got)
	}
	if got := truncateLeft("hello", 10); got != "hello" {
		t.Fatalf("truncateLeft fits = %q", got)
	}
	if got := truncateLeft("hello world", 2); got != ".." {
		t.Fatalf("truncateLeft maxW<=3 = %q, want ..", got)
	}
	got := truncateLeft("abcdefghij", 6)
	if !strings.HasPrefix(got, "...") {
		t.Fatalf("truncateLeft = %q, want ... prefix", got)
	}
}

func TestHelperShortenPathTailMax(t *testing.T) {
	t.Parallel()
	if got := shortenPathTailMax("", 10); got != "" {
		t.Fatalf("empty path = %q", got)
	}
	if got := shortenPathTailMax("/a/b/c", 0); got != "" {
		t.Fatalf("maxW=0 = %q", got)
	}
	if got := shortenPathTailMax("/a/b", 100); got != "/a/b" {
		t.Fatalf("fits = %q, want /a/b", got)
	}
	// must drop leading components -> ".../" prefix
	got := shortenPathTailMax("/very/long/path/to/some/file.txt", 15)
	if !strings.Contains(got, "...") {
		t.Fatalf("shortenPathTailMax narrow = %q, want ... prefix", got)
	}
	// only file fits, filename itself must truncate
	got2 := shortenPathTailMax("/dir/averylongfilenamehere.txt", 8)
	if got2 == "" {
		t.Fatalf("shortenPathTailMax tiny width returned empty")
	}
}

// --- app.go: newApplication / Stop ---

func TestNewApplicationUnknownNameReturnsNil(t *testing.T) {
	t.Parallel()
	var out, errb bytes.Buffer
	_, ctx := ptesting.GenerateRepository(t, &out, &errb, nil)
	if app := newApplication(ctx, "totally-unknown", nil); app != nil {
		t.Fatal("unknown application name should return nil")
	}
}

func TestApplicationStopNil(t *testing.T) {
	t.Parallel()
	var app *Application
	// Should be a safe no-op.
	app.Stop()
}

// --- model.go: newGenericModel / Init ---

func TestNewGenericModelAndInit(t *testing.T) {
	t.Parallel()
	var out, errb bytes.Buffer
	repo, ctx := ptesting.GenerateRepository(t, &out, &errb, nil)

	app := &Application{name: "import", ctx: ctx, state: newApplicationState()}
	model := newGenericModel(ctx, app, repo)
	am, ok := model.(appModel)
	if !ok {
		t.Fatalf("newGenericModel returned %T, want appModel", model)
	}
	if am.application != app || am.repo != repo {
		t.Fatal("newGenericModel did not wire application/repo")
	}
	cmd := am.Init()
	if cmd == nil {
		t.Fatal("Init() should return a batched command")
	}
}

func TestTickReturnsCommand(t *testing.T) {
	t.Parallel()
	if tick() == nil {
		t.Fatal("tick() should return a non-nil cmd")
	}
}

func TestWaitForCancelFiresOnCtxDone(t *testing.T) {
	t.Parallel()
	var out, errb bytes.Buffer
	_, ctx := ptesting.GenerateRepository(t, &out, &errb, nil)

	cmd := waitForCancel(ctx)
	if cmd == nil {
		t.Fatal("waitForCancel should return a cmd")
	}

	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	// cancel the context so the waiter unblocks
	ctx.Cancel(errStubError{})

	select {
	case msg := <-done:
		if _, ok := msg.(cancelledMsg); !ok {
			t.Fatalf("waitForCancel produced %T, want cancelledMsg", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waitForCancel did not fire after context cancel")
	}
}
