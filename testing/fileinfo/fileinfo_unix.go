//go:build !windows

package fileinfo

import "syscall"

func newSys() any { return &syscall.Stat_t{} }
