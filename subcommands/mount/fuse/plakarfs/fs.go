//go:build linux || darwin

package plakarfs

import (
	"io/fs"
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	fusefs "github.com/anacrolix/fuse/fs"
)

type plakarFS struct {
	ctx *appcontext.AppContext

	repo          *repository.Repository
	locateOptions *locate.LocateOptions
	chrootfs      fs.FS

	// rootRefresh is how often we re-list snapshots at the FUSE root.
	rootRefresh time.Duration
	// rootCacheTTL is the kernel-side attr/entry cache TTL at the FUSE
	// root. Kept short so newly created snapshots show up promptly.
	rootCacheTTL time.Duration
	// kernelCacheTTL is the kernel-side attr/entry cache TTL inside
	// snapshots, which are immutable — so we can be generous.
	kernelCacheTTL time.Duration
	inodeCache     *inodeCache
}

func NewFS(ctx *appcontext.AppContext, repo *repository.Repository, locateOptions *locate.LocateOptions, chrootfs fs.FS) *plakarFS {
	return &plakarFS{
		ctx:            ctx,
		repo:           repo,
		locateOptions:  locateOptions,
		chrootfs:       chrootfs,
		rootRefresh:    10 * time.Second,
		rootCacheTTL:   5 * time.Second,
		kernelCacheTTL: time.Hour,
		inodeCache:     newInodeCache(),
	}
}

func (fs *plakarFS) Root() (fusefs.Node, error) {
	return NewDirectory(fs, fs.chrootfs, nil, "")
}
