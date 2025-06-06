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

package sftp

import (
	"context"
	"io"
	"net/url"
	"os"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/exporter"
	plakarsftp "github.com/PlakarKorp/plakar/sftp"
	"github.com/pkg/sftp"
)

type SFTPExporter struct {
	location string
	client   *sftp.Client
}

func init() {
	exporter.Register("sftp", NewSFTPExporter)
}

func NewSFTPExporter(ctx context.Context, opt *exporter.Options, name string, config map[string]string) (exporter.Exporter, error) {
	var err error

	target := config["location"]

	parsed, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	client, err := plakarsftp.Connect(parsed, config)
	if err != nil {
		return nil, err
	}

	return &SFTPExporter{
		location: parsed.Path,
		client:   client,
	}, nil
}

func (p *SFTPExporter) Root() string {

	return p.location
}

func (p *SFTPExporter) CreateDirectory(pathname string) error {
	return p.client.MkdirAll(pathname)
}

func (p *SFTPExporter) StoreFile(pathname string, fp io.Reader, size int64) error {
	f, err := p.client.Create(pathname)
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, fp); err != nil {
		//logging.Warn("copy failure: %s: %s", pathname, err)
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		//logging.Warn("sync failure: %s: %s", pathname, err)
	}
	if err := f.Close(); err != nil {
		//logging.Warn("close failure: %s: %s", pathname, err)
	}
	return nil
}

func (p *SFTPExporter) SetPermissions(pathname string, fileinfo *objects.FileInfo) error {
	if err := p.client.Chmod(pathname, fileinfo.Mode()); err != nil {
		return err
	}
	if os.Getuid() == 0 {
		if err := p.client.Chown(pathname, int(fileinfo.Uid()), int(fileinfo.Gid())); err != nil {
			return err
		}
	}
	return nil
}

func (p *SFTPExporter) Close() error {
	return p.client.Close()
}
