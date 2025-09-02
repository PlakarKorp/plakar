package agent

import (
	"os"
	"time"
)

type FileLock struct {
	Path     string
	Attempts int
	file     *os.File
}

func LockedFile(path string) (*FileLock, error) {
	lock := &FileLock{Path: path}
	err := lock.Lock()
	if err != nil {
		return nil, err
	}
	return lock, nil
}

func (lock *FileLock) Lock() error {
	file, err := os.OpenFile(lock.Path, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	n := lock.Attempts
	if n <= 0 {
		n = 1000
	}
	for range n {
		err = flock(file)
		if err == nil {
			lock.file = file
			return nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return err
}

func (lock *FileLock) Unlock() {
	os.Remove(lock.Path)
	lock.file.Close()
	lock.file = nil
}
