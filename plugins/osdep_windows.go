package plugins

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// Console Ctrl-C and Ctrl-Break go to every process attached to the console.
// CREATE_NEW_PROCESS_GROUP alone still lets Ctrl-Break through; with no
// console at all, the plugin gets neither.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
	}
}
