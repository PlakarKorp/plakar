/*
 * Copyright (c) 2025 Gilles Chehade <gilles@plakar.io>
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

package ftp

import (
	"context"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/exporter"
	"github.com/secsy/goftp"
)

type FTPExporter struct {
	host    string
	rootDir string
	client  *goftp.Client
}

func connectToFTP(host, username, password string) (*goftp.Client, error) {
	config := goftp.Config{}
	if username != "" {
		config.User = username
	}
	if password != "" {
		config.Password = password
	}
	config.Timeout = 10 * time.Second
	return goftp.DialConfig(config, host)
}

func init() {
	exporter.Register("ftp", NewFTPExporter)
}

func NewFTPExporter(ctx context.Context, opts *exporter.Options, name string, config map[string]string) (exporter.Exporter, error) {
	target := config["location"]

	parsed, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	var username string
	if tmp, ok := config["username"]; ok {
		username = tmp
	}

	var password string
	if tmp, ok := config["password"]; ok {
		password = tmp
	}

	client, err := connectToFTP(parsed.Host, username, password)
	if err != nil {
		return nil, err
	}

	return &FTPExporter{
		host:    parsed.Host,
		rootDir: parsed.Path,
		client:  client,
	}, nil
}

func (p *FTPExporter) Root() string {
	return p.rootDir
}

func (p *FTPExporter) CreateDirectory(pathname string) error {
	if pathname == "/" {
		return nil
	}
	_, err := p.client.Mkdir(pathname)
	if err != nil {
		if strings.Contains(err.Error(), "exists") {
			return nil
		}
	}
	return err
}

func (p *FTPExporter) StoreFile(pathname string, fp io.Reader, size int64) error {
	return p.client.Store(pathname, fp)
}

func (p *FTPExporter) SetPermissions(pathname string, fileinfo *objects.FileInfo) error {
	// can't chown/chmod on FTP
	return nil
}

func (p *FTPExporter) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}
