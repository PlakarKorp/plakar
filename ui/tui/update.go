package tui

import (
	"fmt"
	"time"

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
		// Context was cancelled externally (e.g. signal handler); quit cleanly.
		m.forceQuit = true
		return m, tea.Quit

	case tickMsg:
		state := m.application.state

		now := time.Now()
		if m.lastETAAt.IsZero() {
			m.lastETAAt = now
		} else {
			dt := now.Sub(m.lastETAAt).Seconds()
			if dt > 0.2 { // ~5 Hz max
				resDone := state.countFileOk + state.countFileError +
					state.countSymlinkOk + state.countSymlinkError +
					state.countXattrOk + state.countXattrError
				resRate := float64(resDone-m.lastDone) / dt

				const alpha = 0.2
				if resRate > 0 {
					if m.rateEMA == 0 {
						m.rateEMA = resRate
					} else {
						m.rateEMA = alpha*resRate + (1-alpha)*m.rateEMA
					}
				}

				m.lastETAAt = now
				m.lastDone = resDone
			}
		}

		return m, tick()

	case tea.KeyMsg:
		switch event.String() {
		case "ctrl+c":
			m.application.aborted = true
			m.forceQuit = true
			// Cancel the context immediately so the running command stops
			// without waiting for the event channel to drain and close.
			m.application.ctx.Cancel(fmt.Errorf("aborted"))
			return m, tea.Quit
		}

	case tea.QuitMsg:
		return m, tea.Quit
	}

	return m, nil
}
