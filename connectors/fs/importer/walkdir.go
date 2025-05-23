//go:build !windows
// +build !windows

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
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/importer"
	"github.com/pkg/xattr"
)

// Worker pool to handle file scanning in parallel
func (f *FSImporter) walkDir_worker(jobs <-chan string, results chan<- *importer.ScanResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		var (
			path string
			ok   bool
		)

		select {
		case path, ok = <-jobs:
			if !ok {
				return
			}
		case <-f.ctx.Done():
			return
		}

		info, err := os.Lstat(path)
		if err != nil {
			results <- importer.NewScanError(path, err)
			continue
		}

		extendedAttributes, err := xattr.List(path)
		if err != nil {
			results <- importer.NewScanError(path, err)
			continue
		}

		fileinfo := objects.FileInfoFromStat(info)
		fileinfo.Lusername, fileinfo.Lgroupname = f.lookupIDs(fileinfo.Uid(), fileinfo.Gid())

		var originFile string
		if fileinfo.Mode()&os.ModeSymlink != 0 {
			originFile, err = os.Readlink(path)
			if err != nil {
				results <- importer.NewScanError(path, err)
				continue
			}
		}
		results <- importer.NewScanRecord(filepath.ToSlash(path), originFile, fileinfo, extendedAttributes,
			func() (io.ReadCloser, error) {
				return os.Open(path)
			})
		for _, attr := range extendedAttributes {
			results <- importer.NewScanXattr(filepath.ToSlash(path), attr, objects.AttributeExtended,
				func() (io.ReadCloser, error) {
					data, err := xattr.Get(path, attr)
					if err != nil {
						return nil, err
					}
					return io.NopCloser(bytes.NewReader(data)), nil
				})
		}
	}
}

func walkDir_addPrefixDirectories(rootDir string, jobs chan<- string, results chan<- *importer.ScanResult) {
	atoms := strings.Split(rootDir, string(os.PathSeparator))

	for i := range len(atoms) - 1 {
		path := filepath.Join(atoms[0 : i+1]...)

		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		if _, err := os.Stat(path); err != nil {
			results <- importer.NewScanError(path, err)
			continue
		}

		jobs <- path
	}
}
