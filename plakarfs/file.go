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

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	fmt.Println("File.Attr() called on", f.fullpath, f.name)
	entry, err := f.vfs.GetEntry(f.fullpath)
	if err != nil {
		return syscall.ENOENT
	}

	if entry.Stat().IsDir() {
		panic(fmt.Sprintf("unexpected type %T", entry))
	}

	full := canon(f.fullpath)
	f.ino = f.fs.inodeFor("path", fmt.Sprintf("%x", f.parent.snap.Header.Identifier), full, f.name)
	a.Inode = f.ino
	a.Valid = time.Minute
	a.Rdev = 0
	a.Mode = entry.Stat().Mode()
	a.Uid = uint32(entry.Stat().Uid())
	a.Gid = uint32(entry.Stat().Gid())
	a.Ctime = entry.Stat().ModTime()
	a.Mtime = entry.Stat().ModTime()
	a.Size = uint64(entry.Stat().Size())
	a.Nlink = uint32(entry.Stat().Nlink())

	fmt.Printf("ATTR name=%q ino=%d mode=%#o size=%d full=%q\n",
		f.name, a.Inode, a.Mode, a.Size, f.fullpath,
	)
	return nil
}

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	fmt.Println("File.Open() called on", f.name, f.ino)
	resp.Flags |= fuse.OpenDirectIO
	resp.Flags |= fuse.OpenKeepCache

	rd, err := f.parent.snap.NewReader(f.fullpath)
	if err != nil {
		return nil, err
	}

	return &fileHandle{f: rd, ino: f.ino}, nil
}

func (h *fileHandle) ReadAll(ctx context.Context) ([]byte, error) {
	return io.ReadAll(h.f)
}

func (h *fileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	fmt.Println("File.Read()")
	b, err := io.ReadAll(h.f)
	if err != nil {
		return err
	}

	off := int(req.Offset)
	if off >= len(b) {
		resp.Data = nil
		return nil
	}
	end := off + req.Size
	if end > len(b) {
		end = len(b)
	}
	resp.Data = b[off:end]
	return nil
}

func (h *fileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	fmt.Println("Releasing handle")
	return h.f.Close()
}

func (h *fileHandle) Access(ctx context.Context, req *fuse.AccessRequest) error {
	fmt.Println("ACCESS")
	return nil // allow
}
