package backup

import (
	"fmt"
	"os"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/events"
	tea "github.com/charmbracelet/bubbletea"
)

type tickMsg struct{}

// tick command sends a message after 500ms to update the elapsed time
func tick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

type Model struct {
	forceQuit bool

	startTime time.Time
	elapsed   time.Duration

	lastLog string

	countFilesOk     uint64
	countFilesErrors uint64

	countDirsOk     uint64
	countDirsErrors uint64
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch event := msg.(type) {
	case tickMsg:
		m.elapsed = time.Since(m.startTime)
		return m, tick()

	case events.FileOK:
		m.countFilesOk++
		m.lastLog = fmt.Sprintf("%x: %s", event.SnapshotID[:4], event.Pathname)

	case events.FileError, events.PathError:
		m.countFilesErrors++

	case events.DirectoryOK:
		m.countDirsOk++
		m.lastLog = fmt.Sprintf("%x: %s", event.SnapshotID[:4], event.Pathname)

	case events.DirectoryError:
		m.countDirsErrors++

	case tea.KeyMsg:
		switch event.String() {
		case "ctrl+c":
			m.forceQuit = true
			return m, tea.Quit
		}

	case tea.QuitMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) View() string {
	s := ""

	s += fmt.Sprintf("Duration: %ds\n", int64(m.elapsed.Seconds()))

	s += fmt.Sprintf("Directories: %s %d", checkMark, m.countDirsOk)
	if m.countDirsErrors > 0 {
		s += fmt.Sprintf(" %s %d", crossMark, m.countDirsErrors)
	}
	s += "\n"

	s += fmt.Sprintf("Files: %s %d", checkMark, m.countFilesOk)
	if m.countFilesErrors > 0 {
		s += fmt.Sprintf(" %s %d\n", crossMark, m.countFilesErrors)
	}
	s += "\n"

	s += m.lastLog

	s += "\n"
	return s
}

type eventsProcessorInteractive struct {
	ctx     *appcontext.AppContext
	program *tea.Program
}

func NewEventsProcessorInteractive(ctx *appcontext.AppContext) eventsProcessorInteractive {
	return eventsProcessorInteractive{
		ctx: ctx,
		program: tea.NewProgram(Model{
			startTime: time.Now(),
		}),
	}
}

func (ep eventsProcessorInteractive) Start() {
	// Start a goroutine that listens for events and sends them to the bubble Tea program.
	go func() {
		for event := range ep.ctx.Events().Listen() {
			ep.program.Send(event)
		}
	}()

	// Start the Bubble Tea program in the background.
	go func() {
		model, err := ep.program.Run()
		if err != nil {
			ep.ctx.GetLogger().Error("error starting Bubble Tea: %v", err)
			return
		}

		// In case of ctrl+c, forward the signal to the main process now that
		// the Bubble Tea program has exited and terminal control has been
		// restored.
		if model.(Model).forceQuit {
			p := os.Process{Pid: os.Getpid()}
			_ = p.Signal(os.Interrupt)
		}
	}()
}

func (ep eventsProcessorInteractive) Close() {
	ep.program.Quit()
}
