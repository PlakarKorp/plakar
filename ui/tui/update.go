package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch event := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = event.Width
		m.height = event.Height
		return m, nil

	case eventsClosedMsg:
		return m, tea.Quit

	case cancelledMsg:
		m.forceQuit = true
		return m, tea.Quit

	case tickMsg:
		// The ETA rate is fed by the kloset iostat sampler (see
		// State.updateIOStats); the tick only drives repaints.
		//
		// Once the workflow has completed, stop painting and quit: View() then
		// returns empty so bubbletea's inline renderer tears down cleanly, and
		// the final summary is printed once by newApplication after Run()
		// returns. This avoids both the duplicated frame and the diff-renderer
		// leaving stale blank lines.
		if m.application.state.phase == "completed" {
			m.finished = true
			return m, tea.Quit
		}
		return m, tick()

	case tea.KeyMsg:
		switch event.String() {
		case "ctrl+c":
			m.forceQuit = true
			return m, tea.Interrupt
		case "d":
			m.debug = !m.debug
			return m, nil
		}

	case tea.QuitMsg:
		return m, tea.Quit
	}

	return m, nil
}
