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
					ema := func(current float64, delta int64) float64 {
						rate := float64(delta) / dt
						if rate < 0 {
							return current
						}
						if current == 0 {
							return rate
						}
						return alpha*rate + (1-alpha)*current
					}

					state.sourceReadRate = ema(state.sourceReadRate, state.sourceReadBytes-state.lastSourceRead)
					state.sourceWriteRate = ema(state.sourceWriteRate, state.sourceWriteBytes-state.lastSourceWrite)
					state.storeReadRate = ema(state.storeReadRate, state.storeReadBytes-state.lastStoreRead)
					state.storeWriteRate = ema(state.storeWriteRate, state.storeWriteBytes-state.lastStoreWrite)
				}
				state.lastRateAt = now
				state.lastSourceRead = state.sourceReadBytes
				state.lastSourceWrite = state.sourceWriteBytes
				state.lastStoreRead = state.storeReadBytes
				state.lastStoreWrite = state.storeWriteBytes
			}
		}

		return m, tea.Batch(tick())

	case tea.KeyMsg:
		switch event.String() {
		case "ctrl+c":
			m.forceQuit = true
			return m, tea.Interrupt
		case "d", "D":
			// toggle debug overlay
			if m.application != nil && m.application.tui != nil {
				t := m.application.tui
				t.debug.Store(!t.debug.Load())
			}
			return m, nil
		}

	case tea.QuitMsg:
		return m, tea.Quit
	}

	return m, nil
}
