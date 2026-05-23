package tui

import (
	"time"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type Event = events.Event

type tickMsg struct{}

func tick() tea.Cmd {
	return tea.Tick(10*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

type eventsClosedMsg struct{}

type cancelledMsg struct{ err error }

func waitForCancel(ctx *appcontext.AppContext) tea.Cmd {
	return func() tea.Msg {
		<-ctx.Done() // adapt if your AppContext exposes Done() differently
		return cancelledMsg{err: ctx.Err()}
	}
}

type appModel struct {
	application *Application
	repo        *repository.Repository

	forceQuit bool

	// geometry
	width  int
	height int

	// UI
	barPrefix string
	progress  progress.Model

	// ETA calculation
	lastETAAt time.Time
	lastDone  uint64
	rateEMA   float64 // items/secec
}

func newGenericModel(ctx *appcontext.AppContext, application *Application, repo *repository.Repository) tea.Model {
	return appModel{
		repo:        repo,
		application: application,
		progress:    progressBar(),
	}
}

func (m appModel) Init() tea.Cmd {
	const batchMax = 1024 // tune: 128/256/512/1024 depending on workload

	return tea.Batch(
		tick(),
		waitForCancel(m.application.ctx),
	)
}
