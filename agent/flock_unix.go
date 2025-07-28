//go:build !windows

package agent

import (
	"os"

	"golang.org/x/sys/unix"
)

func flock(fp *os.File) error {
	return unix.Flock(int(fp.Fd()), unix.LOCK_EX)
}
