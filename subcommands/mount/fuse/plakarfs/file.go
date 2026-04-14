//go:build linux || darwin

package plakarfs

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path"
	"syscall"

	"github.com/anacrolix/fuse"
	fusefs "github.com/anacrolix/fuse/fs"
)

var _ fusefs.Node = (*File)(nil)
var _ fusefs.NodeOpener = (*File)(nil)

type fileHandle struct {
	f      io.ReadCloser
	offset int64
}

var _ fusefs.Handle = (*fileHandle)(nil)
var _ fusefs.HandleReader = (*fileHandle)(nil)

type File struct {
	pfs *plakarFS
	vfs fs.FS

	path string

	cacheKey string
	attr     *fuse.Attr
}

func NewFile(pfs *plakarFS, vfs fs.FS, parent *Dir, pathname string) (*File, error) {
	key := stableKey("file", parent.snapKey, pathname)
	if f, ok := pfs.inodeCache.getFile(key); ok {
		return f, nil
	} else {
		st, err := parent.Stat(path.Base(pathname))
		if err != nil {
			return nil, syscall.ENOENT
		}

		f := &File{
			pfs:      pfs,
			vfs:      vfs,
			path:     pathname,
			cacheKey: key,
			attr: &fuse.Attr{
				Valid: pfs.kernelCacheTTL,
				Mode:  st.Mode(),
				//Uid:   uint32(entry.Stat().Uid()),
				//Gid:   uint32(entry.Stat().Gid()),
				Uid:   uint32(os.Geteuid()),
				Gid:   uint32(os.Getgid()),
				Ctime: st.ModTime(),
				Mtime: st.ModTime(),
				Atime: st.ModTime(),
				Size:  uint64(st.Size()),
				//Nlink: uint32(st.Nlink()),
			},
		}
		pfs.inodeCache.setFile(f.cacheKey, f)
		return f, nil
	}
}

func (f *File) Forget() {
	f.pfs.inodeCache.removeFile(f.cacheKey)
}

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	*a = *f.attr
	if a.Mode.IsDir() {
		return syscall.EISDIR
	}
	return nil
}

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fusefs.Handle, error) {
	resp.Flags |= fuse.OpenDirectIO
	resp.Flags |= fuse.OpenKeepCache

	rd, err := f.vfs.Open(f.path)
	if err != nil {
		return nil, err
	}
	return &fileHandle{f: rd}, nil
}

func (h *fileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	if req.Offset != h.offset {
		s, ok := h.f.(io.Seeker)
		if !ok {
			return syscall.EIO
		}
		if _, err := s.Seek(req.Offset, io.SeekStart); err != nil {
			return err
		}
		h.offset = req.Offset
	}

	buf := make([]byte, req.Size)
	n, err := io.ReadFull(h.f, buf)
	if err == io.ErrUnexpectedEOF {
		err = nil
	}
	if err != nil && err != io.EOF {
		return err
	}
	h.offset += int64(n)
	resp.Data = buf[:n]
	return nil
}

func (h *fileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return h.f.Close()
}

func (h *fileHandle) Access(ctx context.Context, req *fuse.AccessRequest) error {
	return nil
}
