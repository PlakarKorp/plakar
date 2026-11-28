package main

import (
	"errors"
	"testing"
)

func TestInterrupterCancelsThenExits(t *testing.T) {
	var cancelled []error
	exits := 0
	i := &interrupter{
		cancel: func(err error) { cancelled = append(cancelled, err) },
		exit:   func() { exits++ },
	}

	first := errors.New("interrupted")
	i.interrupt(first)
	if len(cancelled) != 1 || cancelled[0] != first || exits != 0 {
		t.Fatalf("after first interrupt: cancelled=%v exits=%d, want one cancel and no exit", cancelled, exits)
	}

	i.interrupt(errors.New("interrupted"))
	if len(cancelled) != 1 || exits != 1 {
		t.Fatalf("after second interrupt: cancelled=%v exits=%d, want no new cancel and one exit", cancelled, exits)
	}
}
