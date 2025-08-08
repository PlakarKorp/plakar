//go:build linux || darwin

package plakarfs

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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
	ino      uint64
}

func canon(p string) string {
	// ensure leading slash and no duplicate separators
	return path.Clean("/" + strings.TrimPrefix(p, "/"))
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	fmt.Println("Dir.Attr() called on", d.fullpath)
	if d.name == "/" {
		d.fullpath = d.name
		a.Valid = time.Minute
		a.Inode = d.ino
		a.Mode = os.ModeDir | 0o700
		a.Uid = uint32(os.Geteuid())
		a.Gid = uint32(os.Getgid())
		a.Nlink = 2
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

		a.Valid = time.Minute
		a.Inode = d.fs.inodeFor("snapRoot", fmt.Sprintf("%x", d.snap.Header.Identifier))
		a.Mode = os.ModeDir | 0o700
		a.Uid = uint32(os.Geteuid())
		a.Gid = uint32(os.Getgid())
		a.Ctime = snap.Header.Timestamp
		a.Mtime = snap.Header.Timestamp
		a.Atime = snap.Header.Timestamp
		a.Size = snap.Header.GetSource(0).Summary.Directory.Size + snap.Header.GetSource(0).Summary.Below.Size
		a.Nlink = 2
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

		a.Valid = time.Minute
		a.Inode = d.fs.inodeFor("path", fmt.Sprintf("%x", d.snap.Header.Identifier), filepath.Clean(d.fullpath))
		a.Mode = fi.Stat().Mode()
		a.Uid = uint32(fi.Stat().Uid())
		a.Gid = uint32(fi.Stat().Gid())
		a.Ctime = fi.Stat().ModTime()
		a.Mtime = fi.Stat().ModTime()
		a.Size = uint64(fi.Stat().Size())
		a.Nlink = 2
	}
	return nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	fmt.Println("Dir.Lookup() called on", name)
	if d.name == "/" {
		return &Dir{parent: d, name: name, repo: d.repo, fs: d.fs, ino: 1}, nil
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
		return &File{
			parent:   d,
			name:     name,
			fs:       d.fs,
			repo:     d.repo,
			vfs:      d.vfs,
			fullpath: filepath.Clean(d.fullpath + "/" + name),
		}, nil
	}
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fmt.Println("Dir.ReadDirAll() called on")
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
		for _, snapshotID := range snapshots {
			idHex := fmt.Sprintf("%x", snapshotID) // preferably full id
			dirDirs = append(dirDirs, fuse.Dirent{
				Inode: d.fs.inodeFor("snapRoot", idHex), // must match Dir.Attr
				Name:  idHex[:8],                        // your display choice
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

		childFull := canon(d.fullpath + "/" + entry.Name())
		ino := d.fs.inodeFor("path", fmt.Sprintf("%x", d.snap.Header.Identifier), childFull)
		fmt.Println("DIR ATTR inode", ino, "name", entry.Name())
		dirEnt := fuse.Dirent{
			Inode: ino,
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
