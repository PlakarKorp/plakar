//go:build linux || darwin

package plakarfs

import (
	"context"
	"io"
	"io/fs"
	"path"
	"sync"
	"syscall"

	"github.com/anacrolix/fuse"
	fusefs "github.com/anacrolix/fuse/fs"
)

// readBufPool reduces allocation pressure on Read. The kernel-side max read
// is typically 128KB; we use that as our nominal size.
const nominalReadSize = 128 * 1024

var readBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, nominalReadSize)
		return &b
	},
}

func getReadBuf(n int) *[]byte {
	if n <= nominalReadSize {
		b := readBufPool.Get().(*[]byte)
		return b
	}
	b := make([]byte, n)
	return &b
}

func putReadBuf(b *[]byte) {
	if cap(*b) >= nominalReadSize {
		*b = (*b)[:nominalReadSize]
		readBufPool.Put(b)
	}
}

var _ fusefs.Node = (*File)(nil)
var _ fusefs.NodeOpener = (*File)(nil)
var _ fusefs.NodeReadlinker = (*File)(nil)
var _ fusefs.NodeGetxattrer = (*File)(nil)
var _ fusefs.NodeListxattrer = (*File)(nil)

// fileHandle wraps the kloset vfile for the lifetime of a FUSE open.
//
// The underlying vfile is an io.ReadSeeker with a 4MB prefetch buffer that is
// invalidated on Seek. To get the most out of that buffer for sequential
// reads (the common case for `cat`, `cp`, indexers, etc.), we track the next
// expected offset and prefer sequential Read calls over ReaderAt when the
// request matches. Random-access reads fall back to ReadAt, which is
// stateless on the underlying reader.
//
// All access is serialized through mu — anacrolix/fuse can dispatch Read
// concurrently per handle, and Seek on the underlying reader is not safe to
// run alongside other operations on the same instance.
type fileHandle struct {
	f         io.ReadCloser
	rs        io.ReadSeeker // == f, when it implements ReadSeeker
	ra        io.ReaderAt   // == f, when it implements ReaderAt
	mu        sync.Mutex
	nextOff   int64 // expected offset for the next sequential Read
	seqActive bool  // true when nextOff reflects the current rs position
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
	}

	st, err := parent.Stat(path.Base(pathname))
	if err != nil {
		return nil, syscall.ENOENT
	}

	f := &File{
		pfs:      pfs,
		vfs:      vfs,
		path:     pathname,
		cacheKey: key,
		attr:     &fuse.Attr{Valid: pfs.kernelCacheTTL},
	}
	fillAttrFromFileInfo(f.attr, st)
	pfs.inodeCache.setFile(f.cacheKey, f)
	return f, nil
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
	// Snapshots are immutable, so let the kernel cache page contents
	// across opens. OpenDirectIO would conflict with this; do not set it.
	resp.Flags |= fuse.OpenKeepCache

	rd, err := f.vfs.Open(f.path)
	if err != nil {
		return nil, err
	}
	h := &fileHandle{f: rd}
	if rs, ok := rd.(io.ReadSeeker); ok {
		h.rs = rs
	}
	if ra, ok := rd.(io.ReaderAt); ok {
		h.ra = ra
	}
	return h, nil
}

func (h *fileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	bufp := getReadBuf(req.Size)
	defer putReadBuf(bufp)
	buf := (*bufp)[:req.Size]

	n, err := h.readLocked(buf, req.Offset)
	if err != nil && err != io.EOF {
		return err
	}
	// resp.Data must own its slice (the kernel reads from it after we return,
	// and we'll return our buffer to the pool).
	out := make([]byte, n)
	copy(out, buf[:n])
	resp.Data = out
	return nil
}

// readLocked must be called with h.mu held.
func (h *fileHandle) readLocked(buf []byte, off int64) (int, error) {
	// Prefer the sequential reader when the offset matches: that path
	// benefits from the kloset prefetch buffer (default 4MB).
	if h.rs != nil && h.seqActive && off == h.nextOff {
		n, err := io.ReadFull(h.rs, buf)
		if err == io.ErrUnexpectedEOF {
			err = io.EOF
		}
		h.nextOff += int64(n)
		return n, err
	}

	// Random offset (or first read on this handle). Try to seek; if that
	// works, we re-arm sequential reads from the new position. Otherwise
	// fall back to one-shot ReaderAt.
	if h.rs != nil {
		if _, err := h.rs.Seek(off, io.SeekStart); err == nil {
			n, err := io.ReadFull(h.rs, buf)
			if err == io.ErrUnexpectedEOF {
				err = io.EOF
			}
			h.nextOff = off + int64(n)
			h.seqActive = true
			return n, err
		}
		h.seqActive = false
	}
	if h.ra != nil {
		n, err := h.ra.ReadAt(buf, off)
		// ReadAt does not affect rs's cursor; sequential state is unchanged.
		return n, err
	}
	return 0, syscall.EIO
}

func (h *fileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return h.f.Close()
}

func (h *fileHandle) Access(ctx context.Context, req *fuse.AccessRequest) error {
	return nil
}

func (f *File) Readlink(ctx context.Context, req *fuse.ReadlinkRequest) (string, error) {
	target, err := readlink(f.vfs, f.path, f.attr.Mode)
	if err != nil {
		return "", syscall.EINVAL
	}
	return target, nil
}

func (f *File) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	return getxattr(f.vfs, f.path, req, resp)
}

func (f *File) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	return listxattr(f.vfs, f.path, resp)
}
