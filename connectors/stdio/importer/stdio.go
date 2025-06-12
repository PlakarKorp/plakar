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

package stdio

import (
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/importer"
)

type StdioImporter struct {
	fileDir string
	ctx     context.Context
	opts    *importer.ImporterOptions
	name    string
}

func init() {
	importer.Register("stdin", NewStdioImporter)
}

func NewStdioImporter(ctx context.Context, opts *importer.ImporterOptions, name string, config map[string]string) (importer.Importer, error) {
	location := config["location"]
	location = strings.TrimPrefix(location, "stdin://")
	if !strings.HasPrefix(location, "/") {
		location = "/" + location
	}
	location = path.Clean(location)

	return &StdioImporter{
		fileDir: location,
		ctx:     ctx,
		name:    name,
		opts:    opts,
	}, nil
}

func (p *StdioImporter) stdioWalker_addPrefixDirectories(results chan<- *importer.ScanResult) {
	directory := filepath.Clean(p.fileDir)
	atoms := strings.Split(directory, string(os.PathSeparator))

	for i := 0; i < len(atoms)-1; i++ {
		subpath := filepath.Join(atoms[0 : i+1]...)

		if !strings.HasPrefix(subpath, "/") {
			subpath = "/" + subpath
		}

		fi := objects.FileInfo{
			Lname:      path.Base(subpath),
			Lmode:      0755 | os.ModeDir,
			Lsize:      0,
			Ldev:       0,
			Lino:       0,
			Luid:       0,
			Lgid:       0,
			Lnlink:     0,
			LmodTime:   time.Now(),
			Lusername:  "",
			Lgroupname: "",
		}
		results <- importer.NewScanRecord(subpath, "", fi, nil, nil)
	}
}

func (p *StdioImporter) Scan() (<-chan *importer.ScanResult, error) {
	results := make(chan *importer.ScanResult)

	go func() {
		defer close(results)
		p.stdioWalker_addPrefixDirectories(results)
		fi := objects.FileInfo{
			Lname:      path.Base(p.fileDir),
			Lmode:      0644,
			Lsize:      -1,
			Ldev:       0,
			Lino:       0,
			Luid:       0,
			Lgid:       0,
			Lnlink:     0,
			LmodTime:   time.Now(),
			Lusername:  "",
			Lgroupname: "",
		}
		results <- importer.NewScanRecord(p.fileDir, "", fi, nil,
			func() (io.ReadCloser, error) { return os.Stdin, nil })
	}()

	return results, nil
}

func (p *StdioImporter) Close() error {
	return nil
}

func (p *StdioImporter) Root() string {
	return "/"
}

func (p *StdioImporter) Origin() string {
	return p.opts.Hostname
}

func (p *StdioImporter) Type() string {
	return p.name
}
