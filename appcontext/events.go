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

package appcontext

import (
	"sync"

	"github.com/PlakarKorp/kloset/events"
)

// subscriptionBuffer is the depth of each subscriber's channel. The bus
// itself is unbuffered, so this is the only slack between kloset and a
// consumer.
const subscriptionBuffer = 1024

// eventsTee is the single listener of the context's event bus. It copies
// every event to each subscription, in order, and closes them when the bus
// closes. A subscriber that stops reading blocks the tee, and through it
// the emitter, exactly as a slow renderer did when it was the bus's only
// listener: nothing is dropped.
type eventsTee struct {
	mu   sync.Mutex
	subs []*Subscription
}

// Subscription is one consumer's view of the event bus.
type Subscription struct {
	tee *eventsTee
	c   chan *events.Event
}

// C is the channel events arrive on. It closes when the subscription is
// closed or when the bus is.
func (s *Subscription) C() <-chan *events.Event {
	return s.c
}

// Close detaches the subscription from the tee and closes C once the
// events already copied to it have been read. Closing twice is a no-op.
func (s *Subscription) Close() {
	s.tee.mu.Lock()
	defer s.tee.mu.Unlock()
	for i, sub := range s.tee.subs {
		if sub == s {
			s.tee.subs = append(s.tee.subs[:i], s.tee.subs[i+1:]...)
			close(s.c)
			return
		}
	}
}

func (t *eventsTee) run(in <-chan *events.Event) {
	for e := range in {
		t.mu.Lock()
		for _, sub := range t.subs {
			sub.c <- e
		}
		t.mu.Unlock()
	}

	t.mu.Lock()
	for _, sub := range t.subs {
		close(sub.c)
	}
	t.subs = nil
	t.mu.Unlock()
}

func (t *eventsTee) subscribe() *Subscription {
	s := &Subscription{tee: t, c: make(chan *events.Event, subscriptionBuffer)}
	t.mu.Lock()
	t.subs = append(t.subs, s)
	t.mu.Unlock()
	return s
}

// Subscribe returns a subscription receiving every event published on this
// context's bus from now on. The first call starts the tee, which is the
// moment emitters stop dropping events; the renderers call it before any
// emitter exists.
func (c *AppContext) Subscribe() *Subscription {
	c.teeOnce.Do(func() {
		c.tee = &eventsTee{}
		go c.tee.run(c.Events().Listen())
	})
	return c.tee.subscribe()
}
