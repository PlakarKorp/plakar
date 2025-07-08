/*
 * Copyright (c) 2025 Matthieu Masson <matthieu.masson@plakar.io>
 * Copyright (c) 2025 Omar Polo <omar.polo@plakar.io>
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

package pkg

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/importer"
	"github.com/PlakarKorp/plakar/plugins"
)

type pkgerImporter struct {
	cwd          string
	manifest     *plugins.Manifest
	manifestPath string
}

func (imp *pkgerImporter) Origin() string {
	return ""
}

func (imp *pkgerImporter) Type() string {
	return "pkger"
}

func (imp *pkgerImporter) Root() string {
	return "/"
}

func absolutify(cwd, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(cwd, path)
}

func mkstruct(p string, ch chan<- *importer.ScanResult) {
	dir := path.Dir(p)
	for dir != "/" {
		fi := objects.FileInfo{
			Lname: path.Base(dir),
			Lmode: 0700 | os.ModeDir,
		}
		ch <- importer.NewScanRecord(dir, "", fi, nil, nil)
		dir = path.Dir(dir)
	}
}

func (imp *pkgerImporter) dofile(p string, ch chan<- *importer.ScanResult, mustExe bool) {
	absolute := absolutify(imp.cwd, p)

	relative := absolute
	relative, _ = strings.CutPrefix(relative, imp.cwd)
	relative, _ = strings.CutPrefix(relative, string(os.PathSeparator))
	relative = filepath.ToSlash(relative)
	name := path.Join("/", relative)

	if !strings.HasPrefix(absolute, imp.cwd) {
		ch <- importer.NewScanError(name, fmt.Errorf("not below the manifest"))
		return
	}

	fp, err := os.Open(absolute)
	if err != nil {
		ch <- importer.NewScanError(name, fmt.Errorf("Failed to open file: %w", err))
		return
	}

	fi, err := fp.Stat()
	if err != nil {
		ch <- importer.NewScanError(name, fmt.Errorf("Failed to stat file: %w", err))
		return
	}

	if mustExe && (fi.Mode()&0111) == 0 {
		ch <- importer.NewScanError(name, fmt.Errorf("Not executable: %s", absolute))
		return
	}

	mkstruct(name, ch)
	ch <- &importer.ScanResult{
		Record: &importer.ScanRecord{
			Pathname: name,
			FileInfo: objects.FileInfoFromStat(fi),
			Reader:   fp,
		},
	}
}

func (imp *pkgerImporter) scan(ch chan<- *importer.ScanResult) {
	defer close(ch)

	info := objects.NewFileInfo("/", 0, 0700|os.ModeDir, time.Unix(0, 0), 0, 0, 0, 0, 1)
	ch <- &importer.ScanResult{
		Record: &importer.ScanRecord{
			Pathname: "/",
			FileInfo: info,
		},
	}

	imp.dofile(imp.manifestPath, ch, false)
	for _, conn := range imp.manifest.Connectors {
		imp.dofile(conn.Executable, ch, true)
		for _, file := range conn.ExtraFiles {
			imp.dofile(file, ch, false)
		}
	}
}

func (imp *pkgerImporter) Scan() (<-chan *importer.ScanResult, error) {
	ch := make(chan *importer.ScanResult, 1)
	go imp.scan(ch)
	return ch, nil
}

func (imp *pkgerImporter) Close() error {
	return nil
}
