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
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/location"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/pkg"
)

type pkgerImporter struct {
	cwd          string
	manifest     *pkg.Manifest
	manifestPath string
}

func (imp *pkgerImporter) Origin() string        { return "" }
func (imp *pkgerImporter) Type() string          { return "pkger" }
func (imp *pkgerImporter) Root() string          { return "/" }
func (imp *pkgerImporter) Flags() location.Flags { return 0 }

func absolutify(cwd, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(cwd, path)
}

func mkstruct(p string, ch chan<- *connectors.Record) {
	dir := path.Dir(p)
	for dir != "/" {
		fi := objects.FileInfo{
			Lname: path.Base(dir),
			Lmode: 0700 | os.ModeDir,
		}
		ch <- connectors.NewRecord(dir, "", fi, nil, nil)
		dir = path.Dir(dir)
	}
}

func (imp *pkgerImporter) dofile(p string, ch chan<- *connectors.Record, mustExe bool) error {
	absolute := absolutify(imp.cwd, p)

	relative := absolute
	relative, _ = strings.CutPrefix(relative, imp.cwd)
	relative, _ = strings.CutPrefix(relative, string(os.PathSeparator))
	relative = filepath.ToSlash(relative)
	name := path.Join("/", relative)

	if !strings.HasPrefix(absolute, imp.cwd) {
		return fmt.Errorf("not below the manifest: %s", name)
	}

	fp, err := os.Open(absolute)
	if err != nil {
		ch <- connectors.NewError(name, fmt.Errorf("Failed to open file: %w", err))
		return nil
	}

	fi, err := fp.Stat()
	if err != nil {
		ch <- connectors.NewError(name, fmt.Errorf("Failed to stat file: %w", err))
		return nil
	}

	if mustExe {
		var isexe bool
		if os.Getenv("GOOS") == "windows" || runtime.GOOS == "windows" {
			isexe = strings.HasSuffix(fi.Name(), ".exe")
		} else {
			isexe = (fi.Mode() & 0111) != 0
		}

		if !isexe {
			ch <- connectors.NewError(name, fmt.Errorf("Not executable: %s", absolute))
			return nil
		}
	}

	mkstruct(name, ch)
	ch <- &connectors.Record{
		Pathname: name,
		FileInfo: objects.FileInfoFromStat(fi),
		Reader:   fp,
	}

	return nil
}

func (imp *pkgerImporter) scan(ch chan<- *connectors.Record) error {
	info := objects.NewFileInfo("/", 0, 0700|os.ModeDir, time.Unix(0, 0), 0, 0, 0, 0, 1)
	ch <- &connectors.Record{
		Pathname: "/",
		FileInfo: info,
	}

	if err := imp.dofile(imp.manifestPath, ch, false); err != nil {
		return err
	}
	for _, conn := range imp.manifest.Connectors {
		if err := imp.dofile(conn.Executable, ch, true); err != nil {
			return err
		}
		for _, file := range conn.ExtraFiles {
			if err := imp.dofile(file, ch, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (imp *pkgerImporter) Import(ctx context.Context, records chan<- *connectors.Record, results <-chan *connectors.Result) error {
	defer close(records)
	return imp.scan(records)
}

func (imp *pkgerImporter) Ping(ctx context.Context) error {
	return nil
}

func (imp *pkgerImporter) Close(ctx context.Context) error {
	return nil
}
