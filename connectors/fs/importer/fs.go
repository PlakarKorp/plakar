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
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PlakarKorp/kloset/snapshot/importer"
)

type FSImporter struct {
	ctx     context.Context
	opts    *importer.Options
	rootDir string

	uidToName map[uint64]string
	gidToName map[uint64]string
	mu        sync.RWMutex
}

func init() {
	importer.Register("fs", NewFSImporter)
}

func NewFSImporter(appCtx context.Context, opts *importer.Options, name string, config map[string]string) (importer.Importer, error) {
	location := config["location"]
	rootDir := strings.TrimPrefix(location, "fs://")

	if !path.IsAbs(rootDir) {
		return nil, fmt.Errorf("not an absolute path %s", location)
	}

	rootDir = path.Clean(rootDir)

	return &FSImporter{
		ctx:       appCtx,
		opts:      opts,
		rootDir:   rootDir,
		uidToName: make(map[uint64]string),
		gidToName: make(map[uint64]string),
	}, nil
}

func (p *FSImporter) Origin() string {
	return p.opts.Hostname
}

func (p *FSImporter) Type() string {
	return "fs"
}

func (p *FSImporter) Scan() (<-chan *importer.ScanResult, error) {
	realp, err := p.realpathFollow(p.rootDir)
	if err != nil {
		return nil, err
	}

	results := make(chan *importer.ScanResult, 1000)
	go p.walkDir_walker(results, p.rootDir, realp, 256)
	return results, nil
}

func (f *FSImporter) walkDir_walker(results chan<- *importer.ScanResult, rootDir, realp string, numWorkers int) {
	jobs := make(chan string, 1000) // Buffered channel to feed paths to workers
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go f.walkDir_worker(jobs, results, &wg)
	}

	// Add prefix directories first
	walkDir_addPrefixDirectories(realp, jobs, results)
	if realp != rootDir {
		jobs <- rootDir
		walkDir_addPrefixDirectories(rootDir, jobs, results)
	}

	err := filepath.WalkDir(realp, func(path string, d fs.DirEntry, err error) error {
		if f.ctx.Err() != nil {
			return err
		}

		if err != nil {
			results <- importer.NewScanError(path, err)
			return nil
		}
		jobs <- path
		return nil
	})
	if err != nil {
		results <- importer.NewScanError(realp, err)
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

func (f *FSImporter) realpathFollow(path string) (resolved string, err error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		realpath, err := os.Readlink(path)
		if err != nil {
			return "", err
		}

		if !filepath.IsAbs(realpath) {
			realpath = filepath.Join(filepath.Dir(path), realpath)
		}
		path = realpath
	}

	return path, nil
}

func (p *FSImporter) Close() error {
	return nil
}

func (p *FSImporter) Root() string {
	return p.rootDir
}
