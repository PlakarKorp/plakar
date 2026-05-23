//go:build linux || darwin

package plakarfs

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/cached"
	"github.com/anacrolix/fuse"
	fusefs "github.com/anacrolix/fuse/fs"
)

type Dir struct {
	pfs    *plakarFS
	vfs    fs.FS
	parent *Dir

	snap    *snapshot.Snapshot
	snapKey string

	path string

	cacheKey string
	attr     *fuse.Attr

	readDirMutex           sync.Mutex
	readDirSnapshotMapping map[string]objects.MAC
	readDirLast            time.Time
	readDirEntries         []fs.DirEntry
	readDirChildren        []fuse.Dirent
}

func NewDirectory(pfs *plakarFS, vfs fs.FS, parent *Dir, pathname string) (*Dir, error) {
	var key string
	switch parent {
	case nil:
		key = stableKey(pathname)
	case parent.parent:
		key = stableKey("snapshot", pathname)
	default:
		key = stableKey("directory", parent.snapKey, pathname)
	}

	if child, ok := pfs.inodeCache.getDir(key); ok {
		return child, nil
	} else {
		dir := &Dir{
			pfs:      pfs,
			vfs:      vfs,
			parent:   parent,
			path:     pathname,
			cacheKey: key,
			attr: &fuse.Attr{
				Valid: pfs.kernelCacheTTL,
				Uid:   uint32(os.Geteuid()),
				Gid:   uint32(os.Getgid()),
				Nlink: 2,
				Mode:  os.ModeDir | 0o700,
			},
		}
		if parent == nil {
			dir.parent = dir
		}
		if parent != nil {
			dir.snapKey = parent.snapKey
			dir.vfs = parent.vfs
			dir.snap = parent.snap
		}
		if !dir.IsRoot() {
			if dir.vfs == nil {
				parent.readDirMutex.Lock()
				identifier := parent.readDirSnapshotMapping[pathname]
				parent.readDirMutex.Unlock()
				snap, err := snapshot.Load(pfs.repo, identifier)
				if err != nil {
					return nil, syscall.ENOENT
				}

				snapfs, err := snap.Filesystem()
				if err != nil {
					return nil, err
				}

				dir.snap = snap
				dir.vfs = snapfs
				dir.path = ""
				dir.snapKey = fmt.Sprintf("%x", dir.snap.Header.Identifier)

				dir.attr.Mode = os.ModeDir | 0o700
				ts := snap.Header.Timestamp
				dir.attr.Ctime, dir.attr.Mtime, dir.attr.Atime = ts, ts, ts
				dir.attr.Size = snap.Header.GetSource(0).Summary.Directory.Size + snap.Header.GetSource(0).Summary.Below.Size
			} else {
				st, err := parent.Stat(path.Base(pathname))
				if err != nil {
					return nil, err
				}

				dir.attr.Mode = st.Mode()
				//				dir.attr.Uid = uint32(entry.Stat().Uid())
				//				dir.attr.Gid = uint32(entry.Stat().Gid())
				dir.attr.Ctime = st.ModTime()
				dir.attr.Mtime = st.ModTime()
				dir.attr.Atime = st.ModTime()
				dir.attr.Size = uint64(st.Size())
			}
		}

		pfs.inodeCache.setDir(dir.cacheKey, dir)
		return dir, nil
	}
}

func (d *Dir) IsRoot() bool {
	return d.parent == d
}

func (d *Dir) IsSnapshotLister() bool {
	return d.IsRoot() && d.vfs == nil
}

func (d *Dir) Forget() {
	d.pfs.inodeCache.removeDir(d.cacheKey)
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	*a = *d.attr
	if !a.Mode.IsDir() {
		return syscall.ENOTDIR
	}
	return nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fusefs.Node, error) {
	if d.vfs == nil {
		if err := d.ensureSnapshotMapping(ctx); err != nil {
			return nil, err
		}
		d.readDirMutex.Lock()
		_, ok := d.readDirSnapshotMapping[name]
		d.readDirMutex.Unlock()
		if !ok {
			return nil, syscall.ENOENT
		}
		return NewDirectory(d.pfs, nil, d, name)
	}

	st, err := d.Stat(name)
	if err != nil {
		return nil, syscall.ENOENT
	}

	pathname := path.Clean(path.Join(d.path, name))
	if st.IsDir() {
		return NewDirectory(d.pfs, d.vfs, d, pathname)
	} else {
		return NewFile(d.pfs, d.vfs, d, pathname)
	}
}

func (d *Dir) ensureSnapshotMapping(ctx context.Context) error {
	d.readDirMutex.Lock()
	if !d.readDirLast.IsZero() && time.Since(d.readDirLast) < d.pfs.rootRefresh {
		d.readDirMutex.Unlock()
		return nil
	}
	d.readDirMutex.Unlock()
	_, err := d.ReadDirAll(ctx)
	return err
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	d.readDirMutex.Lock()
	defer d.readDirMutex.Unlock()

	now := time.Now()
	if d.vfs == nil {
		if !d.readDirLast.IsZero() && time.Since(d.readDirLast) < d.pfs.rootRefresh {
			return d.readDirChildren, nil
		}

		if _, err := cached.RebuildStateFromStore(d.pfs.ctx, d.pfs.repo.Configuration().RepositoryID, d.pfs.ctx.StoreConfig, false); err != nil {
			d.pfs.ctx.GetLogger().Warn("plakarfs: refreshing cached state failed: %v", err)
		}

		snapshotIDs, err := locate.LocateSnapshotIDs(d.pfs.repo, d.pfs.locateOptions)
		if err != nil {
			return nil, err
		}

		d.readDirLast = now
		readDirSnapshotMapping := make(map[string]objects.MAC)
		out := make([]fuse.Dirent, 0, len(snapshotIDs))
		for _, snapshotID := range snapshotIDs {
			name := fmt.Sprintf("%x", snapshotID[:4])
			readDirSnapshotMapping[name] = snapshotID
			out = append(out, fuse.Dirent{
				Name: name,
				Type: fuse.DT_Dir,
			})
		}
		d.readDirSnapshotMapping = readDirSnapshotMapping
		d.readDirChildren = out
	} else if d.readDirLast.IsZero() {
		children, err := fs.ReadDir(d.vfs, path.Join(".", d.path))
		if err != nil {
			return nil, err
		}
		d.readDirEntries = children

		d.readDirLast = now
		out := make([]fuse.Dirent, 0)
		for _, child := range children {
			de := fuse.Dirent{Name: child.Name(), Type: fuse.DT_File}
			if child.IsDir() {
				de.Type = fuse.DT_Dir
			}
			out = append(out, de)
		}
		d.readDirChildren = out
	}
	return d.readDirChildren, nil
}

func (d *Dir) Stat(name string) (fs.FileInfo, error) {
	d.readDirMutex.Lock()
	entries := d.readDirEntries
	d.readDirMutex.Unlock()
	for _, de := range entries {
		if de.Name() == name {
			return de.Info()
		}
	}
	if d.vfs == nil {
		return nil, fs.ErrNotExist
	}
	return fs.Stat(d.vfs, path.Join(".", d.path, name))
}
