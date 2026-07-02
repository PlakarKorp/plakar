//go:build linux || darwin

package plakarfs

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
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
				if len(parent.readDirSnapshotMapping) == 0 {
					err := parent.getSnapshots()
					if err != nil {
						parent.readDirMutex.Unlock()
						return nil, syscall.ENOENT
					}
				}
				identifier := parent.readDirSnapshotMapping[pathname]
				parent.readDirMutex.Unlock()
				snap, err := snapshot.Load(pfs.repo, identifier)
				if err != nil {
					return nil, syscall.ENOENT
				}

				snapshotFS, err := snap.Filesystem()
				if err != nil {
					return nil, err
				}

				sourceRoot := path.Clean(snap.Header.GetSource(0).Importer.Directory)
				sourceRoot = strings.TrimPrefix(sourceRoot, "/")
				if sourceRoot == "" {
					sourceRoot = "."
				}
				subFS, err := fs.Sub(snapshotFS, sourceRoot)
				if err != nil {
					return nil, err
				}

				dir.snap = snap
				dir.vfs = subFS
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
		return NewDirectory(d.pfs, nil, d, name)
	}

	if len(d.readDirChildren) == 0 {
		err := d.loadDirectory()
		if err != nil {
			return nil, syscall.ENOENT
		}
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

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	d.readDirMutex.Lock()
	defer d.readDirMutex.Unlock()

	if d.vfs == nil {
		if !d.readDirLast.IsZero() && time.Since(d.readDirLast) < d.pfs.rootRefresh {
			return d.readDirChildren, nil
		}

		err := d.getSnapshots()
		if err != nil {
			return nil, err
		}
	} else {
		err := d.loadDirectory()
		if err != nil {
			return nil, err
		}
	}
	return d.readDirChildren, nil
}

func (d *Dir) Stat(name string) (fs.FileInfo, error) {
	if d.readDirEntries != nil {
		for _, de := range d.readDirEntries {
			if de.Name() == name {
				st, err := de.Info()
				if err != nil {
					return nil, err
				}
				return st, nil
			}
		}
	}
	return nil, fs.ErrNotExist
}

func (d *Dir) loadDirectory() error {
	if d.readDirLast.IsZero() {
		children, err := fs.ReadDir(d.vfs, path.Join(".", d.path))
		if err != nil {
			return err
		}
		d.readDirEntries = children

		d.readDirLast = time.Now()
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
	return nil
}

func (d *Dir) getSnapshots() error {
	now := time.Now()
	_, err := cached.RebuildStateFromStore(d.pfs.ctx, d.pfs.repo.Configuration().RepositoryID, d.pfs.ctx.StoreConfig, false)
	if err != nil {
		return err
	}
	snapshotIDs, err := locate.LocateSnapshotIDs(d.pfs.repo, d.pfs.locateOptions)
	if err != nil {
		return err
	}

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
	d.readDirLast = now
	return nil
}
