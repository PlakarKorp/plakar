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
		// Once completed, quit so View() returns empty and bubbletea tears down
		// cleanly; the final summary is printed once by newApplication.
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
