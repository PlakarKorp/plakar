//go:build linux || darwin

package plakarfs

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"syscall"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/plakar/locate"
	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Dir struct {
	fs       *FS
	parent   *Dir
	name     string
	fullpath string
	repo     *repository.Repository
	snap     *snapshot.Snapshot
	vfs      *vfs.Filesystem
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	if d.name == "/" {
		d.fullpath = d.name
		a.Inode = 1
		a.Mode = os.ModeDir | 0o700
		a.Uid = uint32(os.Geteuid())
		a.Gid = uint32(os.Getgid())
	} else if d.parent.name == "/" {
		snap, _, err := locate.OpenSnapshotByPath(d.repo, d.name)
		if err != nil {
			return err
		}
		snapfs, err := snap.Filesystem()
		if err != nil {
			return err
		}

		d.snap = snap
		d.repo = d.parent.repo
		d.vfs = snapfs
		d.fullpath = "/"

		a.Inode = rand.Uint64()
		a.Mode = os.ModeDir | 0o700
		a.Uid = uint32(os.Geteuid())
		a.Gid = uint32(os.Getgid())
		a.Ctime = snap.Header.Timestamp
		a.Mtime = snap.Header.Timestamp
		a.Atime = snap.Header.Timestamp
		a.Size = snap.Header.GetSource(0).Summary.Directory.Size + snap.Header.GetSource(0).Summary.Below.Size
	} else {
		d.snap = d.parent.snap
		d.repo = d.parent.repo
		d.vfs = d.parent.vfs
		d.fullpath = d.parent.fullpath + "/" + d.name

		d.fullpath = filepath.Clean(d.fullpath)

		fi, err := d.vfs.GetEntry(d.fullpath)
		if err != nil {
			return syscall.ENOENT
		}

		if !fi.Stat().IsDir() {
			panic(fmt.Sprintf("unexpected type %T", fi))
		}

		a.Rdev = uint32(fi.Stat().Dev())
		a.Inode = fi.Stat().Ino()
		a.Mode = fi.Stat().Mode()
		a.Uid = uint32(fi.Stat().Uid())
		a.Gid = uint32(fi.Stat().Gid())
		a.Ctime = fi.Stat().ModTime()
		a.Mtime = fi.Stat().ModTime()
		a.Size = uint64(fi.Stat().Size())
	}
	return nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if d.name == "/" {
		return &Dir{parent: d, name: name, repo: d.repo, fs: d.fs}, nil
	} else if d.parent.name == "/" {
		return &Dir{parent: d, name: name, fs: d.fs}, nil
	} else {
		cleanpath := filepath.Clean(d.fullpath + "/" + name)
		entry, err := d.vfs.GetEntry(cleanpath)
		if err != nil {
			return nil, err
		}

		if entry.Stat().IsDir() {
			return &Dir{parent: d, name: name, fs: d.fs}, nil
		}
		return &File{parent: d, name: name, fs: d.fs}, nil
	}
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	if d.name == "/" {

		d.repo.RebuildState()

		var snapshots []objects.MAC
		if len(d.fs.snapshots) == 0 {
			snapshotIDs, err := locate.LocateSnapshotIDs(d.repo, d.fs.locateOptions)
			if err != nil {
				return nil, err
			}
			snapshots = append(snapshots, snapshotIDs...)
		} else {
			for _, prefix := range d.fs.snapshots {
				snapshotID, err := locate.LocateSnapshotByPrefix(d.repo, prefix)
				if err != nil {
					continue
				}
				snapshots = append(snapshots, snapshotID)
			}
		}

		dirDirs := make([]fuse.Dirent, 0)
		for idx, snapshotID := range snapshots {
			dirDirs = append(dirDirs, fuse.Dirent{
				Inode: uint64(idx),
				Name:  fmt.Sprintf("%x", snapshotID[:4]),
				Type:  fuse.DT_Dir,
			})
		}
		return dirDirs, nil
	}

	children, err := d.vfs.Children(d.fullpath)
	if err != nil {
		return nil, err
	}

	dirDirs := make([]fuse.Dirent, 0)
	for entry, err := range children {
		if err != nil {
			return nil, err
		}

		dirEnt := fuse.Dirent{
			Inode: entry.Stat().Ino(),
			Name:  entry.Name(),
			Type:  fuse.DT_File,
		}
		if entry.Stat().IsDir() {
			dirEnt.Type = fuse.DT_Dir
		}

		dirDirs = append(dirDirs, dirEnt)
	}
	return dirDirs, nil
}
