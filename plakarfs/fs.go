//go:build linux || darwin

package plakarfs

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/locate"
	"github.com/anacrolix/fuse/fs"
)

type FS struct {
	repo          *repository.Repository
	locateOptions *locate.LocateOptions
	snapshots     []string
	inodeSeed     uint64
}

func (fs *FS) initInodeSeed() {
	if fs.inodeSeed == 0 {
		// you can make this truly random if you want, or derived from repo UUID
		fs.inodeSeed = 0x6d5f_3b29_d4e1_9c07
	}
}

func (fs *FS) inodeFor(namespace string, parts ...string) uint64 {
	h := fnv.New64a()

	var seed [8]byte
	binary.LittleEndian.PutUint64(seed[:], fs.inodeSeed)
	h.Write(seed[:])

	h.Write([]byte{0})
	h.Write([]byte(namespace))

	for _, p := range parts {
		h.Write([]byte{0xff})
		h.Write([]byte(p))
	}

	ino := h.Sum64()
	if ino == 0 || ino == 1 {
		ino ^= 0x9e3779b97f4a7c15 // mix a constant
		if ino == 0 || ino == 1 {
			ino += 2
		}
	}
	return ino
}

func NewFS(repo *repository.Repository, locateOptions *locate.LocateOptions, snapshots []string, mountpoint string) *FS {
	fs := &FS{
		repo:          repo,
		locateOptions: locateOptions,
		snapshots:     snapshots,
	}
	return fs
}

func (f *FS) Root() (fs.Node, error) {
	fmt.Println("Root() called: Initializing root directory")
	return &Dir{name: "/", repo: f.repo, fs: f, ino: 1}, nil
}
