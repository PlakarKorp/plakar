package agent

import (
	"os"

	"golang.org/x/sys/windows"
)

const (
	reserved = 0
	allBytes = ^uint32(0)
)

func flock(fp *os.File) error {
	ol := new(windows.Overlapped)
	return windows.LockFileEx(windows.Handle(fp.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK,
		reserved, allBytes, allBytes, ol)
}
