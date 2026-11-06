package main

import "sync/atomic"

// interrupter is to handle graceful termination
// interrupt once and it will wait, more than once and it exits,
// this is the behaviour you would expect from ctrl+c.
type interrupter struct {
	cancel func(error)
	exit   func()
	fired  atomic.Bool
}

func (i *interrupter) interrupt(cause error) {
	if i.fired.Swap(true) {
		i.exit()
		return
	}
	i.cancel(cause)
}
