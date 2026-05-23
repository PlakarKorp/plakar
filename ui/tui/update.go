package tui

import (
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
		m.forceQuit = true
		return m, tea.Quit

	case tickMsg:
		var cmd tea.Cmd

		state := m.application.state

		now := time.Now()
		if m.lastETAAt.IsZero() {
			m.lastETAAt = now
		} else {
			dt := now.Sub(m.lastETAAt).Seconds()
			if dt > 0.2 { // ~5Hz max
				resDone := state.countFileOk + state.countFileError + state.countSymlinkOk + state.countSymlinkError + state.countXattrOk + state.countXattrError
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

		// IMPORTANT: do NOT re-arm waitForBatch here; keep exactly one waiter in flight
		// (armed after processing each appBatchMsg)
		return m, tea.Batch(cmd, tick())

	case tea.KeyMsg:
		switch event.String() {
		case "ctrl+c":
			m.forceQuit = true
			return m, tea.Interrupt
		}

	case tea.QuitMsg:
		return m, tea.Quit
	}

	return m, nil
}
