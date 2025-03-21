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
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/pkg/xattr"
)

func toUnixPath(pathname string) string {
	unixPath := filepath.ToSlash(pathname)
	if len(unixPath) > 1 && unixPath[1] == ':' {
		// Convert drive letter to Unix format (e.g., C: -> /c)
		unixPath = "/" + strings.ToUpper(unixPath[0:2]) + unixPath[2:]
	}
	if !strings.HasPrefix(unixPath, "/") {
		unixPath = "/" + unixPath
	}
	return unixPath
}

// Worker pool to handle file scanning in parallel
func walkDir_worker(jobs <-chan string, results chan<- *importer.ScanResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for pathname := range jobs {
		unixPath := toUnixPath(pathname)

		var fileinfo objects.FileInfo
		var err error

		if pathname == "/" {
			fileinfo = objects.NewFileInfo("/", 0, os.ModeDir, time.Now(), 0, 0, 0, 0, 1)
		} else {
			info, err := os.Lstat(pathname)
			if err != nil {
				results <- importer.NewScanError(unixPath, err)
				continue
			}
			fileinfo = objects.FileInfoFromStat(info)
			if info.Name() == "\\" {
				fileinfo.Lname = pathname
			}
		}

		extendedAttributes, err := xattr.List(pathname)
		if err != nil {
			results <- importer.NewScanError(unixPath, err)
			continue
		}

		if u, err := user.LookupId(fmt.Sprintf("%d", fileinfo.Uid())); err == nil {
			fileinfo.Lusername = u.Username
		}
		if g, err := user.LookupGroupId(fmt.Sprintf("%d", fileinfo.Gid())); err == nil {
			fileinfo.Lgroupname = g.Name
		}

		var originFile string
		if fileinfo.Mode()&os.ModeSymlink != 0 {
			originFile, err = os.Readlink(pathname)
			if err != nil {
				results <- importer.NewScanError(unixPath, err)
				continue
			}
		}
		results <- importer.NewScanRecord(unixPath, originFile, fileinfo, extendedAttributes)
		for _, attr := range extendedAttributes {
			results <- importer.NewScanXattr(filepath.ToSlash(pathname), attr, objects.AttributeExtended)
		}
	}
}

func walkDir_addPrefixDirectories(rootDir string, jobs chan<- string, results chan<- *importer.ScanResult) {
	atoms := strings.Split(rootDir, string(os.PathSeparator))

	jobs <- "/"
	for i := range len(atoms)-1 {
		pathname := strings.Join(atoms[0:i+1], string(os.PathSeparator))

		if _, err := os.Stat(pathname); err != nil {
			results <- importer.NewScanError(pathname, err)
			continue
		}

		jobs <- pathname
	}
}

func walkDir_walker(rootDir string, numWorkers int) (<-chan *importer.ScanResult, error) {
	results := make(chan *importer.ScanResult, 1000) // Larger buffer for results
	jobs := make(chan string, 1000)                  // Buffered channel to feed paths to workers
	var wg sync.WaitGroup

	// Launch worker pool
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go walkDir_worker(jobs, results, &wg)
	}

	// Start walking the directory and sending file paths to workers
	go func() {
		defer close(jobs)

		info, err := os.Lstat(rootDir)
		if err != nil {
			results <- importer.NewScanError(rootDir, err)
			return
		}
		if info.Mode()&os.ModeSymlink != 0 {
			originFile, err := os.Readlink(rootDir)
			if err != nil {
				results <- importer.NewScanError(rootDir, err)
				return
			}

			if !filepath.IsAbs(originFile) {
				originFile = filepath.Join(filepath.Dir(rootDir), originFile)
			}

			rootDir = originFile
		}

		// Add prefix directories first
		walkDir_addPrefixDirectories(rootDir, jobs, results)

		err = filepath.WalkDir(rootDir, func(pathname string, d fs.DirEntry, err error) error {
			if err != nil {
				results <- importer.NewScanError(pathname, err)
				return nil
			}
			jobs <- pathname
			return nil
		})
		if err != nil {
			results <- importer.NewScanError(rootDir, err)
		}
	}()

	// Close the results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	return results, nil
}
