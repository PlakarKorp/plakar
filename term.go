//go:build !windows

package main

import "github.com/charmbracelet/x/term"

func isTerminal() bool {
	return term.IsTerminal(1)
}
