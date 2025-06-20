//go:build !windows

package fs

import (
	"io/fs"
	"os"
	"syscall"
)

func dirDevice(info os.FileInfo) uint64 {
	if sb, ok := info.Sys().(*syscall.Stat_t); ok {
		return uint64(sb.Dev)
	}
	return 0
}

func isSameFs(devno uint64, d fs.DirEntry) (bool, error) {
	info, err := d.Info()
	if err != nil {
		return false, err
	}

	if sb, ok := info.Sys().(*syscall.Stat_t); ok {
		return uint64(sb.Dev) == devno, nil
	}

	return true, nil
}
