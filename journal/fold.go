/*
 * Copyright (c) 2026 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package journal

import (
	"sync"

	"github.com/PlakarKorp/plakar/appcontext"
)

// Fold subscribes to the context's events and folds them into a State in
// the background. State returns the current picture at any time; Close
// detaches from the bus, waits for the events already received to be
// folded, and returns the final one.
type Fold struct {
	sub  *appcontext.Subscription
	done chan struct{}

	mu    sync.Mutex
	state State
}

func NewFold(ctx *appcontext.AppContext) *Fold {
	f := &Fold{
		sub:  ctx.Subscribe(),
		done: make(chan struct{}),
	}
	go f.run()
	return f
}

func (f *Fold) run() {
	defer close(f.done)
	for e := range f.sub.C() {
		f.mu.Lock()
		f.state.Update(e)
		f.mu.Unlock()
	}
}

func (f *Fold) State() State {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state.Clone()
}

func (f *Fold) Close() State {
	f.sub.Close()
	<-f.done
	return f.State()
}
