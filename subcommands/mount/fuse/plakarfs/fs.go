//go:build linux || darwin

package plakarfs

import (
	"sync"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/anacrolix/fuse/fs"
)

func stableIno(parts ...string) uint64 {
	// Any stable 64-bit hash is fine. FNV-1a:
	const off = 1469598103934665603
	const prime = 1099511628211
	h := uint64(off)
	for _, s := range parts {
		for i := 0; i < len(s); i++ {
			h ^= uint64(s[i])
			h *= prime
		}
	}
	if h < 2 {
		h += 2 // never zero
	}
	return h
}

type FS struct {
	repo          *repository.Repository
	locateOptions *locate.LocateOptions

	muFiles sync.Mutex
	files   map[uint64]*File // ino to node

	muDirectories sync.Mutex
	directories   map[uint64]*Dir // ino to node
}

func NewFS(repo *repository.Repository, locateOptions *locate.LocateOptions, mountpoint string) *FS {
	fs := &FS{
		repo:          repo,
		locateOptions: locateOptions,
		files:         make(map[uint64]*File),
		directories:   make(map[uint64]*Dir),
	}
	return fs
}

func (f *FS) Root() (fs.Node, error) {
	f.muDirectories.Lock()
	defer f.muDirectories.Unlock()

	//fmt.Println("Root() called: Initializing root directory")
	if root, exists := f.directories[1]; exists {
		return root, nil
	} else {
		root := &Dir{name: "/", repo: f.repo, fs: f, ino: 1}
		f.directories[root.ino] = root
		return root, nil
	}
}
