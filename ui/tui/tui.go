package tui

import (
	"errors"
	"io"
	"os"
	"sync/atomic"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/ui"
	"github.com/PlakarKorp/plakar/ui/stdio"
	tea "github.com/charmbracelet/bubbletea"
)

type tui struct {
	ctx  *appcontext.AppContext
	repo *repository.Repository

	app *Application

	done chan error

	cancel atomic.Bool
}

func New(ctx *appcontext.AppContext) ui.UI {
	return &tui{
		ctx: ctx,
	}
}

func (t *tui) Stdout() io.Writer {
	return &switchWriter{
		tui:      t,
		stream:   "stdout",
		fallback: os.Stdout,
	}
}

func (t *tui) Stderr() io.Writer {
	return &switchWriter{
		tui:      t,
		stream:   "stderr",
		fallback: os.Stderr,
	}
}

func (t *tui) Stop() {
	t.cancel.Store(true)
	if t.app != nil {
		t.app.Stop()
		t.app = nil
	}
}

func (tui *tui) Wait() error {
	return <-tui.done
}

func (tui *tui) SetRepository(repo *repository.Repository) {
	tui.repo = repo
}

func (tui *tui) Run() error {
	events := tui.ctx.Events().Listen()
	tui.done = make(chan error, 1)

	go func() {
		defer close(tui.done)

		hasSeenStop := false
		for e := range events {
			// If app is running, forward event (non-blocking / cancel-safe)
			if !tui.cancel.Load() {
				if tui.app != nil {
					tui.app.state.Update(*e)
					select {
					case <-tui.app.done:
						if tui.app.err != nil {
							if errors.Is(tui.app.err, tea.ErrInterrupted) {
								if !hasSeenStop {
									hasSeenStop = true
									tui.done <- ui.ErrUserAbort
								}
							} else {
								if !hasSeenStop {
									hasSeenStop = true
									tui.done <- tui.app.err
								}
							}
						}
					default:
					}

					// Close app on matching workflow.end
					if e.Type == "workflow.end" && e.Job == tui.app.job {
						tui.app.Stop()
						tui.app = nil
					}
					continue
				}

				// No app: start when workflow.start matches a known model
				if e.Type == "workflow.start" {
					tui.app = newApplication(tui.ctx, e.Data["workflow"].(string), tui.repo)
					if tui.app != nil {
						tui.app.job = e.Job
						tui.app.state.Update(*e)
						continue
					}
				}
			}

			// default fallback
			stdio.HandleEvent(tui.ctx, e)
		}
		if tui.app != nil {
			tui.app.Stop()
			tui.done <- nil
			return
		}
	}()

	return nil
}
