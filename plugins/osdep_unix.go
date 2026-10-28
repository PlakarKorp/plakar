//go:build !windows

package plugins

import "syscall"

// A new session keeps terminal signals (Ctrl-C, Ctrl-\, Ctrl-Z) aimed at
// plakar's process group from reaching the plugin, and unlike Setpgid leaves
// it without a controlling terminal, so it is never stopped by SIGTTOU.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
