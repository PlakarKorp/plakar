//go:build linux || darwin

package plakarfs

import (
	"io"
	"io/fs"
	"os"
	"path"

	"github.com/PlakarKorp/kloset/objects"
	klosetvfs "github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/anacrolix/fuse"
)

// fillAttrFromFileInfo populates a fuse.Attr from a generic fs.FileInfo.
// If the FileInfo's Sys() exposes an objects.FileInfo (as kloset's VFS does),
// uid/gid/nlink/inode are taken from there; otherwise they fall back to the
// process's effective uid/gid and Nlink=1.
func fillAttrFromFileInfo(a *fuse.Attr, fi fs.FileInfo, ttl, dirTTL uint32) {
	a.Mode = fi.Mode()
	a.Size = uint64(fi.Size())
	a.Ctime = fi.ModTime()
	a.Mtime = fi.ModTime()
	a.Atime = fi.ModTime()
	a.Nlink = 1

	if kfi, ok := fi.Sys().(objects.FileInfo); ok {
		a.Uid = uint32(kfi.Uid())
		a.Gid = uint32(kfi.Gid())
		if n := kfi.Nlink(); n > 0 {
			a.Nlink = uint32(n)
		}
		if ino := kfi.Ino(); ino != 0 {
			a.Inode = ino
		}
	} else {
		a.Uid = uint32(os.Geteuid())
		a.Gid = uint32(os.Getgid())
	}

	if fi.IsDir() && a.Nlink < 2 {
		a.Nlink = 2
	}
}

// lookupKlosetEntry returns the underlying kloset Entry for a given path on
// the given fs.FS, if the fs.FS is (or wraps) a *klosetvfs.Filesystem.
// Returns (nil, nil, false) when the underlying filesystem is not a kloset
// VFS (e.g., when a chrootfs from fs.Sub is in use).
func lookupKlosetEntry(vfs fs.FS, p string) (*klosetvfs.Filesystem, *klosetvfs.Entry, bool) {
	kf, ok := vfs.(*klosetvfs.Filesystem)
	if !ok {
		return nil, nil, false
	}
	entry, err := kf.GetEntryNoFollow(path.Join("/", p))
	if err != nil {
		return nil, nil, false
	}
	return kf, entry, true
}

func getxattr(vfs fs.FS, p string, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	kf, entry, ok := lookupKlosetEntry(vfs, p)
	if !ok {
		return fuse.ErrNoXattr
	}
	rs, err := entry.Xattr(kf, req.Name)
	if err != nil {
		return fuse.ErrNoXattr
	}
	data, err := io.ReadAll(rs)
	if err != nil {
		return err
	}
	if int(req.Position) >= len(data) {
		resp.Xattr = nil
		return nil
	}
	data = data[req.Position:]
	if req.Size > 0 && uint32(len(data)) > req.Size {
		data = data[:req.Size]
	}
	resp.Xattr = data
	return nil
}

func listxattr(vfs fs.FS, p string, resp *fuse.ListxattrResponse) error {
	_, entry, ok := lookupKlosetEntry(vfs, p)
	if !ok {
		return nil
	}
	resp.Append(entry.ExtendedAttributes...)
	return nil
}

func readlink(vfs fs.FS, p string, mode fs.FileMode) (string, error) {
	if mode&fs.ModeSymlink == 0 {
		return "", fs.ErrInvalid
	}
	_, entry, ok := lookupKlosetEntry(vfs, p)
	if !ok {
		return "", fs.ErrInvalid
	}
	return entry.SymlinkTarget, nil
}
