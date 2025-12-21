package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type backupModel struct {
	appContext *appcontext.AppContext
	events     <-chan Event

	phase string

	startTime time.Time
	lastLog   string
	forceQuit bool

	// counts (event-driven, no per-path memory)
	countFiles       uint64
	countFilesOk     uint64
	countFilesErrors uint64
	countDirs        uint64
	countDirsOk      uint64
	countDirsErrors  uint64

	// UI
	prog progress.Model
	spin spinner.Model
}

func newBackupModel(ctx *appcontext.AppContext, events <-chan Event) tea.Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	pr := progress.New(progress.WithDefaultGradient())

	return backupModel{
		appContext: ctx,
		events:     events,
		startTime:  time.Now(),
		prog:       pr,
		spin:       sp,
	}
}

func (m backupModel) Init() tea.Cmd {
	return tea.Batch(
		m.spin.Tick,
		tick(),
		waitForDoneEvent(m.events), // exactly one waiter in flight
	)
}

func (m backupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch event := msg.(type) {
	case tickMsg:
		// throttle percent updates
		const minFrame = 100 * time.Millisecond
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, tea.Batch(cmd, tick())

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case appMsg:
		e := Event(event)

		switch e.Type {
		case "workflow.start":
			m.startTime = time.Now()
			m.phase = "preparing backup..."

		case "workflow.end":
			m.phase = "done !"
			return m, tea.Quit

		case "directory":
			m.countDirs++

		case "file":
			m.countFiles++

		case "file.ok":
			m.countFilesOk++
			m.lastLog = fmt.Sprintf("%x: %s", e.Snapshot[:4], e.Data["path"].(string))

		case "directory.ok":
			m.countDirsOk++
			m.lastLog = fmt.Sprintf("%x: %s", e.Snapshot[:4], e.Data["path"].(string))

		case "file.error":
			m.countFilesErrors++

		case "directory.error":
			m.countDirsErrors++

		case "path.error":
			m.countFilesErrors++

		case "snapshot.import.start":
			m.phase = "importing resources..."

		case "snapshot.import.end":
			m.phase = "done importing resources..."

		case "snapshot.vfs.start":
			m.phase = "building virtual filesystem..."

		case "snapshot.vfs.end":
			m.phase = "done building virtual filesystem..."

		case "snapshot.index.start":
			m.phase = "indexing resources..."

		case "snapshot.index.end":
			m.phase = "done indexing resources..."

		case "snapshot.commit.start":
			m.phase = "committing snapshot..."

		case "snapshot.commit.end":
			m.phase = "done committing snapshot..."

		case "snapshot.backup.commit.done":
			m.lastLog = fmt.Sprintf("%x: created unsigned snapshot", e.Snapshot[:4])

		case "snapshot.backup.result":
			m.lastLog = fmt.Sprintf("%x: created unsigned snapshot", e.Snapshot[:4])
		}

		// re-arm exactly one next wait
		return m, waitForDoneEvent(m.events)

	case tea.KeyMsg:
		switch event.String() {
		case "ctrl+c":
			m.forceQuit = true
			m.appContext.Cancel(fmt.Errorf("aborted by user"))
			return m, tea.Quit
		}

	case tea.QuitMsg:
		m.lastLog = "Aborted"
		return m, tea.Quit
	}

	return m, nil
}

func (m backupModel) View() string {
	// Don't flash UI if nothing happened
	if m.countFilesOk == 0 && m.countFilesErrors == 0 && m.countDirsOk == 0 && m.countDirsErrors == 0 && m.lastLog == "" {
		return ""
	}

	if m.forceQuit {
		return fmt.Sprintf("[%s] backup: aborted !\n", humanDuration(time.Since(m.startTime)))
	}

	var s strings.Builder

	fmt.Fprintf(&s, "[%s] backup: %s\n", humanDuration(time.Since(m.startTime)), m.phase)

	fmt.Fprintf(&s, "Resources: %s %d", checkMark, m.countFilesOk)
	if m.countFilesErrors > 0 {
		fmt.Fprintf(&s, "  %s %d", crossMark, m.countFilesErrors)
	}
	fmt.Fprintf(&s, "\n")

	fmt.Fprintf(&s, "Structure: %s %d", checkMark, m.countDirsOk)
	if m.countDirsErrors > 0 {
		fmt.Fprintf(&s, "  %s %d", crossMark, m.countDirsErrors)
	}
	fmt.Fprintf(&s, "\n")

	fmt.Fprintf(&s, "%s\n", m.lastLog)
	return s.String()
}
