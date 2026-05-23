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
		state := m.application.state
		now := time.Now()
		const alpha = 0.2

		if m.lastETAAt.IsZero() {
			m.lastETAAt = now
		} else {
			dt := now.Sub(m.lastETAAt).Seconds()
			if dt > 0.2 { // ~5 Hz max
				// item-level rate for ETA
				resDone := state.countFileOk + state.countFileError +
					state.countSymlinkOk + state.countSymlinkError +
					state.countXattrOk + state.countXattrError
				resRate := float64(resDone-m.lastDone) / dt
				if resRate > 0 {
					if m.rateEMA == 0 {
						m.rateEMA = resRate
					} else {
						m.rateEMA = alpha*resRate + (1-alpha)*m.rateEMA
					}
				}
				m.lastETAAt = now
				m.lastDone = resDone

				// byte-level throughput rates from granular events
				if !state.lastRateAt.IsZero() {
					dTransfer := state.transferBytes - state.lastTransfer
					transferRate := float64(dTransfer) / dt
					if transferRate >= 0 {
						if state.transferRate == 0 {
							state.transferRate = transferRate
						} else {
							state.transferRate = alpha*transferRate + (1-alpha)*state.transferRate
						}
					}

					dStoreWrite := state.storeWriteBytes - state.lastStoreWrite
					storeWriteRate := float64(dStoreWrite) / dt
					if storeWriteRate >= 0 {
						if state.storeWriteRate == 0 {
							state.storeWriteRate = storeWriteRate
						} else {
							state.storeWriteRate = alpha*storeWriteRate + (1-alpha)*state.storeWriteRate
						}
					}

					dStoreRead := state.storeReadBytes - state.lastStoreRead
					storeReadRate := float64(dStoreRead) / dt
					if storeReadRate >= 0 {
						if state.storeReadRate == 0 {
							state.storeReadRate = storeReadRate
						} else {
							state.storeReadRate = alpha*storeReadRate + (1-alpha)*state.storeReadRate
						}
					}
				}
				state.lastRateAt = now
				state.lastTransfer = state.transferBytes
				state.lastStoreWrite = state.storeWriteBytes
				state.lastStoreRead = state.storeReadBytes
			}
		}

		return m, tea.Batch(tick())

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
