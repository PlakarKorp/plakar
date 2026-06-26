//go:build !windows

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsTerminalDumb(t *testing.T) {
	// TERM=dumb always reports "not a terminal".
	t.Setenv("TERM", "dumb")
	require.False(t, isTerminal())
}

func TestIsTerminalUnderTest(t *testing.T) {
	// In the test harness fd 1 is not a tty, so this returns false regardless of
	// TERM. We only assert it does not panic and returns a bool deterministically.
	t.Setenv("TERM", "xterm-256color")
	require.False(t, isTerminal())
}
