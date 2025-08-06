//go:build linux || darwin

package plakarfs

import (
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/locate"
	"github.com/anacrolix/fuse/fs"
)

type FS struct {
	repo          *repository.Repository
	locateOptions *locate.LocateOptions
	snapshots     []string
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
	return &Dir{name: "/", repo: f.repo, fs: f}, nil
}
