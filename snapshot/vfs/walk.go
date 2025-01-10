package vfs

import (
	"io/fs"
)

type WalkDirFunc func(path string, entry *Entry, err error) error

func (fsc *Filesystem) walkdir(entry *Entry, fn WalkDirFunc) error {
	path := entry.Path()
	if err := fn(path, entry, nil); err != nil {
		return err
	}

	if !entry.FileInfo.Mode().IsDir() {
		return nil
	}

	iter, err := entry.Getdents(fsc)
	if err != nil {
		return fn(path, nil, err)
	}

	for entry, err := range iter {
		if err != nil {
			return fn(path, nil, err)
		}

		if err := fsc.walkdir(entry, fn); err != nil {
			if err == fs.SkipDir {
				continue
			}
			return err
		}
	}

	return nil
}

func (fsc *Filesystem) WalkDir(root string, fn WalkDirFunc) error {
	entry, err := fsc.GetEntry(root)
	if err != nil {
		return err
	}

	if err = fsc.walkdir(entry, fn); err != nil {
		if err == fs.SkipDir || err == fs.SkipAll {
			err = nil
		}
	}
	return err
}
