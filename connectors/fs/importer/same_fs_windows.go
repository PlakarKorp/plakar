package fs

import (
	"io/fs"
	"os"
)

func dirDevice(info os.FileInfo) uint64 {
	return 0
}

func isSameFs(devno uint64, d fs.DirEntry) (bool, error) {
	return true, nil
}
