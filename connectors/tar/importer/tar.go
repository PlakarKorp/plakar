/*
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

package tar

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/location"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/importer"
)

type TarImporter struct {
	ctx context.Context

	fp  *os.File
	rd  *gzip.Reader
	tar *tar.Reader

	location string
	name     string

	next chan struct{}
}

func init() {
	importer.Register("tar", location.FLAG_LOCALFS|location.FLAG_FILE, NewTarImporter)
	importer.Register("tar+gz", location.FLAG_LOCALFS|location.FLAG_FILE, NewTarImporter)
	importer.Register("tgz", location.FLAG_LOCALFS|location.FLAG_FILE, NewTarImporter)
}

func NewTarImporter(ctx context.Context, opts *importer.Options, name string, config map[string]string) (importer.Importer, error) {
	location := strings.TrimPrefix(config["location"], name+"://")

	fp, err := os.Open(location)
	if err != nil {
		return nil, err
	}

	t := &TarImporter{ctx: ctx, fp: fp, location: location, name: name}

	if name == "tar+gz" || name == "tgz" {
		rd, err := gzip.NewReader(fp)
		if err != nil {
			t.Close(ctx)
			return nil, err
		}
		t.rd = rd
		t.tar = tar.NewReader(t.rd)
	} else {
		t.tar = tar.NewReader(t.fp)
	}

	t.next = make(chan struct{}, 1)

	return t, nil
}

func (t *TarImporter) Type(ctx context.Context) (string, error) { return t.name, nil }
func (t *TarImporter) Root(ctx context.Context) (string, error) { return "/", nil }

func (p *TarImporter) Origin(ctx context.Context) (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return hostname, nil
}

func (t *TarImporter) Scan(ctx context.Context) (<-chan *importer.ScanResult, error) {
	ch := make(chan *importer.ScanResult, 1)
	go t.scan(ch)
	return ch, nil
}

func finfo(hdr *tar.Header) objects.FileInfo {
	f := objects.FileInfo{
		Lname:      path.Base(hdr.Name),
		Lsize:      hdr.Size,
		Lmode:      fs.FileMode(hdr.Mode),
		LmodTime:   hdr.ModTime,
		Ldev:       0, // XXX could use hdr.Devminor / hdr.Devmajor
		Luid:       uint64(hdr.Uid),
		Lgid:       uint64(hdr.Gid),
		Lnlink:     1,
		Lusername:  "",
		Lgroupname: "",
	}

	switch hdr.Typeflag {
	case tar.TypeLink:
		f.Lmode |= fs.ModeSymlink
	case tar.TypeChar:
		f.Lmode |= fs.ModeCharDevice
	case tar.TypeBlock:
		f.Lmode |= fs.ModeDevice
	case tar.TypeDir:
		f.Lmode |= fs.ModeDir
	case tar.TypeFifo:
		f.Lmode |= fs.ModeNamedPipe
	default:
		// other are implicitly regular files.
	}

	return f
}

type entry struct {
	t  *tar.Reader
	ch chan<- struct{}
}

func (e *entry) Read(buf []byte) (int, error) {
	return e.t.Read(buf)
}

func (e *entry) Close() error {
	e.ch <- struct{}{}
	return nil
}

func (t *TarImporter) scan(ch chan<- *importer.ScanResult) {
	defer close(ch)

	info := objects.NewFileInfo("/", 0, 0700|os.ModeDir, time.Unix(0, 0), 0, 0, 0, 0, 1)
	ch <- &importer.ScanResult{
		Record: &importer.ScanRecord{
			Pathname: "/",
			FileInfo: info,
		},
	}

	for {
		hdr, err := t.tar.Next()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				ch <- importer.NewScanError(t.location, err)
			}
			return
		}

		name := path.Join("/", hdr.Name)
		ch <- &importer.ScanResult{
			Record: &importer.ScanRecord{
				Pathname: name,
				Target:   hdr.Linkname,
				FileInfo: finfo(hdr),
				Reader:   &entry{t.tar, t.next},
			},
		}

		select {
		case <-t.next:
			break
		case <-t.ctx.Done():
			return
		}
	}
}

func (t *TarImporter) Close(ctx context.Context) (err error) {
	t.fp.Close()
	if t.rd != nil {
		err = t.rd.Close()
	}

	return err
}
