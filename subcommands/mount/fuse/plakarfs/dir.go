//go:build linux || darwin

package plakarfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
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

func (d *Dir) Forget() {
	d.fs.muDirectories.Lock()
	defer d.fs.muDirectories.Unlock()
	if d.fs.directories[d.ino] == d {
		delete(d.fs.files, d.ino)
	}
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	//log.Println("Dir.Attr() called on", d.fullpath)
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
	//log.Println("Dir.Lookup() called on", name)
	if d.name == "/" {
		d.fs.muDirectories.Lock()
		defer d.fs.muDirectories.Unlock()
		if child, ok := d.fs.directories[stableIno("snapdir", name)]; ok {
			return child, nil
		} else {
			child := &Dir{parent: d, name: name, repo: d.repo, fs: d.fs}
			child.fullpath = "/" // top of that snapshot's vfs
			child.ino = stableIno("snapdir", name)
			d.fs.directories[child.ino] = child
			return child, nil
		}
	} else if d.parent != nil && d.parent.name == "/" {
		d.fs.muDirectories.Lock()
		defer d.fs.muDirectories.Unlock()

		if child, ok := d.fs.directories[stableIno("dir", d.fullpath, name)]; ok {
			return child, nil
		} else {
			// within a snapshot; this `name` is a path component
			child := &Dir{parent: d, name: name, fs: d.fs}
			child.ino = stableIno("dir", d.fullpath, name)
			d.fs.directories[child.ino] = child
			return child, nil
		}
	}

	cleanpath := filepath.Clean(d.fullpath + "/" + name)
	entry, err := d.vfs.GetEntry(cleanpath)
	if err != nil {
		return nil, err
	}

	if entry.Stat().IsDir() {
		d.fs.muDirectories.Lock()
		defer d.fs.muDirectories.Unlock()
		if dir, ok := d.fs.directories[stableIno("dir", cleanpath)]; ok {
			return dir, nil
		} else {
			dir := &Dir{parent: d, name: name, fs: d.fs}
			dir.ino = stableIno("dir", cleanpath)
			d.fs.directories[dir.ino] = dir
			return dir, nil
		}
	}

	d.fs.muFiles.Lock()
	defer d.fs.muFiles.Unlock()
	if f, ok := d.fs.files[stableIno("file", cleanpath)]; ok {
		return f, nil
	} else {
		//log.Println("Dir.Lookup(): file not in cache, creating new")
		f := &File{
			parent:   d,
			name:     name,
			fs:       d.fs,
			repo:     d.repo,
			vfs:      d.vfs,
			fullpath: cleanpath,
		}
		f.ino = stableIno("file", cleanpath)
		f.fs.files[f.ino] = f
		return f, nil
	}
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	//log.Println("Dir.ReadDirAll() called on")
	if d.name == "/" {

		d.repo.RebuildState()

		snapshotIDs, err := locate.LocateSnapshotIDs(d.repo, d.fs.locateOptions)
		if err != nil {
			return nil, err
		}
		snapshots := append([]objects.MAC{}, snapshotIDs...)

		dirDirs := make([]fuse.Dirent, 0)
		for _, snapshotID := range snapshots {
			idHex := fmt.Sprintf("%x", snapshotID) // preferably full id
			dirDirs = append(dirDirs, fuse.Dirent{
				Name: idHex[:8], // your display choice
				Type: fuse.DT_Dir,
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
			Name: entry.Name(),
			Type: fuse.DT_File,
		}
		if entry.Stat().IsDir() {
			dirEnt.Type = fuse.DT_Dir
		}

		dirDirs = append(dirDirs, dirEnt)
	}
	return dirDirs, nil
}
