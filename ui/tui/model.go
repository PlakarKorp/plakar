package tui

import (
	"time"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
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
	finished  bool // completed; render nothing so tea tears down clean
	debug     bool // 'd' toggles extra iostat detail

	// geometry
	width  int
	height int
}

func newGenericModel(ctx *appcontext.AppContext, application *Application, repo *repository.Repository) tea.Model {
	return appModel{
		repo:        repo,
		application: application,
	}
}

func (m appModel) Init() tea.Cmd {
	return tea.Batch(
		tick(),
		waitForCancel(m.application.ctx),
	)
}
