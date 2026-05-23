package tui

import (
	"io"
	"os"
	"sync"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/ui"
	"github.com/PlakarKorp/plakar/ui/stdio"
)

type tui struct {
	ctx  *appcontext.AppContext
	repo *repository.Repository

	mu      sync.Mutex
	app     *Application
	silent  bool // true after user abort; suppresses all further output

	done chan error
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

func (t *tui) getApp() *Application {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.app
}

func (t *tui) setApp(app *Application) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.app = app
}

func (t *tui) isSilent() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.silent
}

func (t *tui) setSilent() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.silent = true
}

func (t *tui) Stop() {
	t.mu.Lock()
	app := t.app
	t.app = nil
	t.mu.Unlock()

	if app != nil {
		app.Stop()
	}
}

func (t *tui) Wait() error {
	return <-t.done
}

func (t *tui) SetRepository(repo *repository.Repository) {
	t.repo = repo
}

func (t *tui) Run() error {
	events := t.ctx.Events().Listen()
	t.done = make(chan error, 1)

	go func() {
		var result error

		for e := range events {
			app := t.getApp()

			if app != nil {
				app.state.Update(*e)

				// Check if app exited (non-blocking)
				select {
				case <-app.done:
					if app.aborted {
						result = ui.ErrUserAbort
						t.setSilent()
					} else if app.err != nil {
						result = app.err
					}
					// App is done; clear it so fallback kicks in
					t.setApp(nil)
					app = nil
				default:
				}

				if app != nil {
					// Close app on matching workflow.end
					if e.Type == "workflow.end" && e.Job == app.job {
						app.Stop()
						t.setApp(nil)
					}
					continue
				}
			}

			// No active app: start one when workflow.start matches a known model
			if e.Type == "workflow.start" {
				newApp := newApplication(t.ctx, e.Data["workflow"].(string), t.repo)
				if newApp != nil {
					newApp.job = e.Job
					newApp.state.Update(*e)
					t.setApp(newApp)
					continue
				}
			}

			// Default fallback for unhandled events (suppressed after user abort)
			if !t.isSilent() {
				stdio.HandleEvent(t.ctx, e)
			}
		}

		// Drain: if an app is still running when events close, stop it
		app := t.getApp()
		if app != nil {
			app.Stop()
			t.setApp(nil)
			if result == nil {
				if app.aborted {
					result = ui.ErrUserAbort
				} else if app.err != nil {
					result = app.err
				}
			}
		}

		t.done <- result
	}()

	return nil
}
