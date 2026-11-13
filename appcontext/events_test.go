package appcontext

import (
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/events"
	"github.com/google/uuid"
)

func publish(ctx *AppContext, typ string) {
	ctx.Events().Publish(&events.Event{Type: typ})
}

func recv(t *testing.T, c <-chan *events.Event) *events.Event {
	t.Helper()
	select {
	case e := <-c:
		return e
	case <-time.After(5 * time.Second):
		t.Fatal("no event")
		return nil
	}
}

func TestSubscribeFansOutInOrder(t *testing.T) {
	ctx := NewAppContext()
	a := ctx.Subscribe()
	b := ctx.Subscribe()

	publish(ctx, "one")
	publish(ctx, "two")

	for _, sub := range []*Subscription{a, b} {
		if e := recv(t, sub.C()); e.Type != "one" {
			t.Fatalf("got %q, want one", e.Type)
		}
		if e := recv(t, sub.C()); e.Type != "two" {
			t.Fatalf("got %q, want two", e.Type)
		}
	}
}

func TestSubscriptionCloseDetaches(t *testing.T) {
	ctx := NewAppContext()
	a := ctx.Subscribe()
	b := ctx.Subscribe()

	publish(ctx, "before")
	a.Close()
	a.Close() // idempotent
	publish(ctx, "after")

	// a got what was copied before it closed, then the close.
	if e := recv(t, a.C()); e.Type != "before" {
		t.Fatalf("got %q, want before", e.Type)
	}
	if e, ok := <-a.C(); ok {
		t.Fatalf("got %q after close", e.Type)
	}

	// b is unaffected.
	if e := recv(t, b.C()); e.Type != "before" {
		t.Fatalf("got %q, want before", e.Type)
	}
	if e := recv(t, b.C()); e.Type != "after" {
		t.Fatalf("got %q, want after", e.Type)
	}
}

func TestBusCloseClosesSubscriptions(t *testing.T) {
	ctx := NewAppContext()
	a := ctx.Subscribe()

	emitter := ctx.Events().NewRepositoryEmitter(uuid.New(), "test")
	emitter.Close()
	ctx.Close()

	var types []string
	for e := range a.C() {
		types = append(types, e.Type)
	}
	if len(types) != 2 || types[0] != "workflow.start" || types[1] != "workflow.end" {
		t.Fatalf("got %v, want workflow.start then workflow.end", types)
	}
}

func TestDerivedContextHasItsOwnTee(t *testing.T) {
	ctx := NewAppContext()
	parent := ctx.Subscribe()

	child := NewAppContextFrom(ctx)
	sub := child.Subscribe()
	publish(child, "child")

	if e := recv(t, sub.C()); e.Type != "child" {
		t.Fatalf("got %q, want child", e.Type)
	}
	select {
	case e := <-parent.C():
		t.Fatalf("parent received %q from the child bus", e.Type)
	case <-time.After(50 * time.Millisecond):
	}
}
