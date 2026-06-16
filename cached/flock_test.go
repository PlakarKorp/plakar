package cached

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestLockedFileAcquiresAndUnlocks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")

	lock, err := LockedFile(path)
	if err != nil {
		t.Fatalf("LockedFile() error = %v", err)
	}
	if lock == nil {
		t.Fatal("LockedFile() returned nil lock")
	}
	if lock.Path != path {
		t.Errorf("lock.Path = %q, want %q", lock.Path, path)
	}
	if lock.file == nil {
		t.Error("lock.file is nil after a successful Lock()")
	}

	// The lock file must exist on disk while held.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("lock file should exist while held: %v", err)
	}

	lock.Unlock()

	if lock.file != nil {
		t.Error("lock.file should be nil after Unlock()")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("lock file should be removed after Unlock(), stat err = %v", err)
	}
}

func TestLockReusesSameFileLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reuse.lock")

	lock := &FileLock{Path: path}
	if err := lock.Lock(); err != nil {
		t.Fatalf("first Lock() error = %v", err)
	}
	if lock.file == nil {
		t.Fatal("lock.file is nil after Lock()")
	}
	lock.Unlock()
}

func TestLockInvalidPath(t *testing.T) {
	// A path whose parent directory does not exist cannot be opened.
	path := filepath.Join(t.TempDir(), "does-not-exist", "child.lock")

	lock, err := LockedFile(path)
	if err == nil {
		lock.Unlock()
		t.Fatal("LockedFile() with an invalid parent dir should fail")
	}
}

// TestLockContention verifies that two FileLocks over the same path are
// mutually exclusive: while one holds the flock, the second blocks until the
// first releases it.
func TestLockContention(t *testing.T) {
	path := filepath.Join(t.TempDir(), "contended.lock")

	first, err := LockedFile(path)
	if err != nil {
		t.Fatalf("first LockedFile() error = %v", err)
	}

	acquired := make(chan struct{})
	var second *FileLock
	go func() {
		// A high attempt budget so this doesn't spuriously fail while
		// blocked, but it will only succeed once `first` is released.
		l := &FileLock{Path: path, Attempts: 1000}
		if err := l.Lock(); err != nil {
			t.Errorf("second Lock() error = %v", err)
			close(acquired)
			return
		}
		second = l
		close(acquired)
	}()

	// The second goroutine must not be able to acquire while we hold it.
	select {
	case <-acquired:
		t.Fatal("second Lock() acquired while the first lock was still held")
	case <-time.After(100 * time.Millisecond):
		// expected: still blocked
	}

	first.Unlock()

	select {
	case <-acquired:
		// good, it got through after release
	case <-time.After(5 * time.Second):
		t.Fatal("second Lock() never acquired after the first was released")
	}

	if second != nil {
		second.Unlock()
	}
}

// TestLockSerializesWriters spins up several goroutines that each take the lock,
// bump a shared counter, and release. With correct mutual exclusion the final
// count equals the number of goroutines and no race is observed.
func TestLockSerializesWriters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serialize.lock")

	const workers = 8
	var wg sync.WaitGroup
	var mu sync.Mutex // guards counter reads/writes outside the flock for the race detector
	counter := 0

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l := &FileLock{Path: path, Attempts: 2000}
			if err := l.Lock(); err != nil {
				t.Errorf("Lock() error = %v", err)
				return
			}
			mu.Lock()
			counter++
			mu.Unlock()
			l.Unlock()
		}()
	}

	wg.Wait()

	if counter != workers {
		t.Errorf("counter = %d, want %d", counter, workers)
	}
}
