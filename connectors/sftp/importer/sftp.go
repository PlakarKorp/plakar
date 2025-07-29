/*
 * Copyright (c) 2025 Gilles Chehade <gilles@poolp.org>
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
	"net/url"

	"github.com/PlakarKorp/kloset/snapshot/importer"
	plakarsftp "github.com/PlakarKorp/plakar/sftp"
	"github.com/pkg/sftp"
)

type SFTPImporter struct {
	rootDir    string
	remoteHost string
	client     *sftp.Client
}

func init() {
	importer.Register("sftp", 0, NewSFTPImporter)
}

func NewSFTPImporter(appCtx context.Context, opts *importer.Options, name string, config map[string]string) (importer.Importer, error) {
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

	return &SFTPImporter{
		rootDir:    parsed.Path,
		remoteHost: parsed.Host,
		client:     client,
	}, nil
}

func (p *SFTPImporter) Origin(ctx context.Context) (string, error) {
	return p.remoteHost, nil
}

func (p *SFTPImporter) Type(ctx context.Context) (string, error) {
	return "sftp", nil
}

func (p *SFTPImporter) Scan(ctx context.Context) (<-chan *importer.ScanResult, error) {
	return p.walkDir_walker(256)
}

func (p *SFTPImporter) Close(ctx context.Context) error {
	return p.client.Close()
}

func (p *SFTPImporter) Root(ctx context.Context) (string, error) {
	return p.rootDir, nil
}
