//go:build linux || darwin

package plakarfs

import (
	"context"
	"fmt"
	"io"
	"syscall"
	"time"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

var _ fs.Handle = (*fileHandle)(nil)
var _ fs.HandleReader = (*fileHandle)(nil) // must compile
var _ fs.Node = (*File)(nil)
var _ fs.NodeOpener = (*File)(nil)

type fileHandle struct {
	f   io.ReadCloser
	ino uint64
}

// File implements both Node and Handle for the hello file.
type File struct {
	fs       *FS
	parent   *Dir
	name     string
	fullpath string
	repo     *repository.Repository
	vfs      *vfs.Filesystem
	ino      uint64
}

func (f *File) Forget() {
	f.fs.muFiles.Lock()
	defer f.fs.muFiles.Unlock()

	if f.fs.files[f.ino] == f {
		delete(f.fs.files, f.ino)
	}
}

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	entry, err := f.vfs.GetEntry(f.fullpath)
	if err != nil {
		return syscall.ENOENT
	}
	if entry.Stat().IsDir() {
		panic(fmt.Sprintf("unexpected type %T", entry))
	}

	a.Inode = f.ino
	a.Valid = time.Minute
	a.Rdev = 0
	a.Mode = entry.Stat().Mode() // regular file bits (type=0) + perms
	a.Uid = uint32(entry.Stat().Uid())
	a.Gid = uint32(entry.Stat().Gid())
	a.Ctime = entry.Stat().ModTime()
	a.Mtime = entry.Stat().ModTime()
	a.Size = uint64(entry.Stat().Size()) // must be correct and >0 when applicable
	a.Nlink = uint32(entry.Stat().Nlink())

	return nil
}

func (h *File) Listxattr(_ context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	return nil
}

func (h *File) Getxattr(_ context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	return fuse.ErrNoXattr
}

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	resp.Flags |= fuse.OpenDirectIO
	resp.Flags |= fuse.OpenKeepCache

	rd, err := f.parent.snap.NewReader(f.fullpath)
	if err != nil {
		return nil, err
	}

	h := &fileHandle{f: rd, ino: f.ino}
	if _, ok := interface{}(h).(fs.HandleReader); !ok {
		panic("handle does not implement fs.HandleReader")
	}
	return h, nil
}

func (h *fileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	if rs, ok := h.f.(io.ReaderAt); !ok {
		return syscall.EIO
	} else {
		buf := make([]byte, req.Size)
		n, err := rs.ReadAt(buf, req.Offset)
		if err != nil && err != io.EOF {
			return err
		}
		resp.Data = buf[:n]
		return nil
	}
}

func (h *fileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	//log.Println("Releasing handle")
	return h.f.Close()
}

func (h *fileHandle) Access(ctx context.Context, req *fuse.AccessRequest) error {
	//log.Println("Access")
	return nil // allow
}
