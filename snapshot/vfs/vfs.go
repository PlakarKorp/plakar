package vfs

import (
	"io/fs"
	"iter"
	"path"
	"strings"

	"github.com/PlakarKorp/plakar/btree"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/packfile"
	"github.com/PlakarKorp/plakar/repository"
)

const VERSION = 002

type Classification struct {
	Analyzer string   `msgpack:"analyzer" json:"analyzer"`
	Classes  []string `msgpack:"classes" json:"classes"`
}

type ExtendedAttribute struct {
	Name  string `msgpack:"name" json:"name"`
	Value []byte `msgpack:"value" json:"value"`
}

type CustomMetadata struct {
	Key   string `msgpack:"key" json:"key"`
	Value []byte `msgpack:"value" json:"value"`
}

type AlternateDataStream struct {
	Name    string `msgpack:"name" json:"name"`
	Content []byte `msgpack:"content" json:"content"`
}

type Filesystem struct {
	tree *btree.BTree[string, objects.Checksum, Entry]
	repo *repository.Repository
}

func PathCmp(a, b string) int {
	da := strings.Count(a, "/")
	db := strings.Count(b, "/")

	if da > db {
		return 1
	}
	if da < db {
		return -1
	}
	return strings.Compare(a, b)
}

// IsEntryBelow returns true when the entry string is a direct child
// of parent from a filesystem perspective.  Parent has to have a
// trailing slash.
func IsEntryBelow(parent, entry string) bool {
	if !strings.HasSuffix(parent, "/") {
		parent += "/"
	}

	if !strings.HasPrefix(entry, parent) {
		return false
	}
	if strings.Index(entry[len(parent):], "/") != -1 {
		return false
	}
	return true
}

func NewFilesystem(repo *repository.Repository, root objects.Checksum) (*Filesystem, error) {
	rd, err := repo.GetBlob(packfile.TYPE_VFS, root)
	if err != nil {
		return nil, err
	}

	storage := repository.NewRepositoryStore[string, Entry](repo, packfile.TYPE_VFS)
	tree, err := btree.Deserialize(rd, storage, PathCmp)
	if err != nil {
		return nil, err
	}

	fs := &Filesystem{
		tree: tree,
		repo: repo,
	}

	return fs, nil
}

func (fsc *Filesystem) lookup(entrypath string) (*Entry, error) {
	if !strings.HasPrefix(entrypath, "/") {
		entrypath = "/" + entrypath
	}
	entrypath = path.Clean(entrypath)

	if entrypath == "" {
		entrypath = "/"
	}

	entry, found, err := fsc.tree.Find(entrypath)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fs.ErrNotExist
	}
	return &entry, nil
}

func (fsc *Filesystem) Open(path string) (fs.File, error) {
	entry, err := fsc.lookup(path)
	if err != nil {
		return nil, err
	}

	return entry.Open(fsc, path), nil
}

func (fsc *Filesystem) ReadDir(path string) (entries []fs.DirEntry, err error) {
	fp, err := fsc.Open(path)
	if err != nil {
		return
	}
	dir, ok := fp.(fs.ReadDirFile)
	if !ok {
		return entries, fs.ErrInvalid
	}

	return dir.ReadDir(-1)
}

func (fsc *Filesystem) Files() iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		iter, err := fsc.tree.ScanAll()
		if err != nil {
			yield("", err)
			return
		}

		for iter.Next() {
			path, entry := iter.Current()
			if entry.FileInfo.Lmode.IsRegular() {
				if !yield(path, nil) {
					return
				}
			}
		}
		if err := iter.Err(); err != nil {
			yield("", err)
			return
		}
	}
}

func (fsc *Filesystem) Pathnames() iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		iter, err := fsc.tree.ScanAll()
		if err != nil {
			yield("", err)
			return
		}

		for iter.Next() {
			path, _ := iter.Current()
			if !yield(path, nil) {
				return
			}
		}

		if err := iter.Err(); err != nil {
			yield("", err)
			return
		}
	}
}

func (fsc *Filesystem) GetEntry(path string) (*Entry, error) {
	return fsc.lookup(path)
}

func (fsc *Filesystem) Children(path string) (iter.Seq2[string, error], error) {
	fp, err := fsc.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	dir, ok := fp.(fs.ReadDirFile)
	if !ok {
		return nil, fs.ErrInvalid
	}

	return func(yield func(string, error) bool) {
		for {
			entries, err := dir.ReadDir(16)
			if err != nil {
				yield("", err)
				return
			}
			for i := range entries {
				if !yield(entries[i].Name(), nil) {
					return
				}
			}
		}
	}, nil
}

func (fsc *Filesystem) VisitNodes(cb func(objects.Checksum, *btree.Node[string, objects.Checksum, Entry]) error) error {
	return fsc.tree.VisitDFS(cb)
}
