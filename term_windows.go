//go:build windows

package main

import "golang.org/x/sys/windows"

func isTerminal() bool {
	var st uint32
	return windows.GetConsoleMode(windows.Stdout, &st) == nil

}
