/*
 * Copyright (c) 2023 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package fs

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/PlakarKorp/kloset/location"
	"github.com/PlakarKorp/kloset/snapshot/importer"
)

type FSImporter struct {
	fsType   string
	ctx      context.Context
	opts     *importer.Options
	rootDir  string
	realpath string

	uidToName map[uint64]string
	gidToName map[uint64]string
	mu        sync.RWMutex

	nocrossfs bool
	devno     uint64

	snapshotter Snapshotter
}

func init() {
	importer.Register("fs", location.FLAG_LOCALFS, NewFSImporter)
	importer.Register("btrfs", location.FLAG_LOCALFS, NewBTRFSImporter)
}

func NewBTRFSImporter(appCtx context.Context, opts *importer.Options, name string, config map[string]string) (importer.Importer, error) {
	rootDir := strings.TrimPrefix(config["location"], "btrfs://")
	return fsImporter("btrfs", rootDir, appCtx, opts, name, config, &BtrfsSnapshotter{})
}

func NewFSImporter(appCtx context.Context, opts *importer.Options, name string, config map[string]string) (importer.Importer, error) {
	rootDir := strings.TrimPrefix(config["location"], "fs://")
	return fsImporter("fs", rootDir, appCtx, opts, name, config, &NOOPSnapshotter{})
}

func fsImporter(fsType string, rootDir string, appCtx context.Context, opts *importer.Options, name string, config map[string]string, snapshotter Snapshotter) (importer.Importer, error) {
	location := config["location"]

	if !filepath.IsAbs(rootDir) {
		return nil, fmt.Errorf("not an absolute path %s", location)
	}

	rootDir = filepath.Clean(rootDir)

	nocrossfs, _ := strconv.ParseBool(config["dont_traverse_fs"])

	realpath, devno, err := realpathFollow(rootDir)
	if err != nil {
		return nil, err
	}

	rootDir, err = snapshotter.Create(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot for %s: %w", rootDir, err)
	}

	return &FSImporter{
		fsType:      fsType,
		ctx:         appCtx,
		opts:        opts,
		rootDir:     rootDir,
		realpath:    realpath,
		uidToName:   make(map[uint64]string),
		gidToName:   make(map[uint64]string),
		nocrossfs:   nocrossfs,
		devno:       devno,
		snapshotter: snapshotter,
	}, nil
}

func (p *FSImporter) Origin() string {
	return p.opts.Hostname
}

func (p *FSImporter) Type() string {
	return p.fsType
}

func (p *FSImporter) Scan() (<-chan *importer.ScanResult, error) {
	results := make(chan *importer.ScanResult, 1000)
	go p.walkDir_walker(results, 256)
	return results, nil
}

func (f *FSImporter) walkDir_walker(results chan<- *importer.ScanResult, numWorkers int) {
	jobs := make(chan string, 1000) // Buffered channel to feed paths to workers
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go f.walkDir_worker(jobs, results, &wg)
	}

	// Add prefix directories first
	walkDir_addPrefixDirectories(f.realpath, results)
	if f.realpath != f.rootDir {
		jobs <- f.rootDir
		walkDir_addPrefixDirectories(f.rootDir, results)
	}

	err := filepath.WalkDir(f.realpath, func(path string, d fs.DirEntry, err error) error {
		if f.ctx.Err() != nil {
			return err
		}

		if err != nil {
			results <- importer.NewScanError(path, err)
			return nil
		}

		if d.IsDir() && f.nocrossfs {
			same, err := isSameFs(f.devno, d)
			if err != nil {
				results <- importer.NewScanError(path, err)
				return nil
			}
			if !same {
				return filepath.SkipDir
			}
		}

		jobs <- path
		return nil
	})
	if err != nil {
		results <- importer.NewScanError(f.realpath, err)
	}

	close(jobs)
	wg.Wait()
	close(results)
}

func (p *FSImporter) lookupIDs(uid, gid uint64) (uname, gname string) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if name, ok := p.uidToName[uid]; !ok {
		if u, err := user.LookupId(fmt.Sprint(uid)); err == nil {
			uname = u.Username

			p.mu.RUnlock()
			p.mu.Lock()
			p.uidToName[uid] = uname
			p.mu.Unlock()
			p.mu.RLock()
		}
	} else {
		uname = name
	}

	if name, ok := p.gidToName[gid]; !ok {
		if g, err := user.LookupGroupId(fmt.Sprint(gid)); err == nil {
			gname = g.Name

			p.mu.RUnlock()
			p.mu.Lock()
			p.gidToName[gid] = name
			p.mu.Unlock()
			p.mu.RLock()
		}
	} else {
		gname = name
	}

	return
}

func realpathFollow(path string) (resolved string, dev uint64, err error) {
	info, err := os.Lstat(path)
	if err != nil {
		return
	}

	if info.Mode()&os.ModeDir != 0 {
		dev = dirDevice(info)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		realpath, err := os.Readlink(path)
		if err != nil {
			return "", 0, err
		}

		if !filepath.IsAbs(realpath) {
			realpath = filepath.Join(filepath.Dir(path), realpath)
		}
		path = realpath
	}

	return path, dev, nil
}

func (p *FSImporter) Close() error {
	return p.snapshotter.Delete(p.rootDir)
}

func (p *FSImporter) Root() string {
	return toslash(p.rootDir)
}

// convert paths to the internal format.  For unix nothing changes,
// for windows we apply some edits:
// C:\User\Omar\Plakar -> C:/User/Omar/Plakar -> /C:/User/Omar/Plakar
func toslash(p string) string {
	p = filepath.ToSlash(p)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}
